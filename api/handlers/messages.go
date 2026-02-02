package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/charmbracelet/crush/api/models"
	internalapp "github.com/charmbracelet/crush/internal/app"
	"github.com/charmbracelet/crush/internal/message"
	hertzapp "github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
)

// HandleListMessages 处理获取会话消息列表的请求
//
//	@Summary		获取消息列表
//	@Description	获取指定会话的所有消息
//	@Tags			Message
//	@Accept			json
//	@Produce		json
//	@Param			directory	query		string	true	"项目路径"
//	@Param			sessionID	path		string	true	"会话ID"
//	@Success		200			{object}	models.MessagesResponse
//	@Failure		400			{object}	map[string]interface{}
//	@Failure		404			{object}	map[string]interface{}
//	@Router			/session/{sessionID}/message [get]
func (h *Handlers) HandleListMessages(c context.Context, ctx *hertzapp.RequestContext) {
	projectPath := string(ctx.Query("directory"))
	if projectPath == "" {
		WriteError(c, ctx, "MISSING_DIRECTORY_PARAM", "Directory query parameter is required", consts.StatusBadRequest)
		return
	}
	sessionID := ctx.Param("sessionID")

	// 获取项目的 app 实例
	appInstance, err := h.GetAppForProject(c, projectPath)
	if err != nil {
		if strings.Contains(err.Error(), "project not found") {
			WriteError(c, ctx, "PROJECT_NOT_FOUND", err.Error(), consts.StatusNotFound)
			return
		}
		WriteError(c, ctx, "INTERNAL_ERROR", "Failed to get or create app for project: "+err.Error(), consts.StatusInternalServerError)
		return
	}

	// 获取消息列表
	messages, err := appInstance.Messages.List(c, sessionID)
	if err != nil {
		WriteError(c, ctx, "INTERNAL_ERROR", "Failed to list messages: "+err.Error(), consts.StatusInternalServerError)
		return
	}

	response := models.MessagesResponse{
		Messages: make([]models.MessageResponse, len(messages)),
		Total:    len(messages),
	}
	for i, m := range messages {
		response.Messages[i] = models.MessageToResponse(m)
	}

	WriteJSON(c, ctx, consts.StatusOK, response)
}

// HandlePrompt 处理发送消息的请求（Opencode SDK 兼容）
//
//	@Summary		Send message (Opencode compatible)
//	@Description	Create and send a new message to a session using Opencode SDK compatible API
//	@Tags			Session
//	@Accept			json
//	@Produce		json
//	@Param			directory	query		string					true	"Project path"
//	@Param			sessionID	path		string					true	"Session ID"
//	@Param			request		body		models.PromptRequest	true	"Prompt request"
//	@Success		200			{object}	models.PromptResponse
//	@Failure		400			{object}	map[string]interface{}
//	@Failure		404			{object}	map[string]interface{}
//	@Router			/session/{sessionID}/prompt [post]
func (h *Handlers) HandlePrompt(c context.Context, ctx *hertzapp.RequestContext) {
	projectPath := string(ctx.Query("directory"))
	if projectPath == "" {
		WriteError(c, ctx, "MISSING_DIRECTORY_PARAM", "Directory query parameter is required", consts.StatusBadRequest)
		return
	}
	sessionID := ctx.Param("sessionID")

	var req models.PromptRequest
	if err := ctx.BindJSON(&req); err != nil {
		WriteError(c, ctx, "INVALID_REQUEST", "Invalid request body: "+err.Error(), consts.StatusBadRequest)
		return
	}

	if len(req.Parts) == 0 {
		WriteError(c, ctx, "INVALID_REQUEST", "Parts array is required", consts.StatusBadRequest)
		return
	}

	// 获取项目的 app 实例
	appInstance, err := h.GetAppForProject(c, projectPath)
	if err != nil {
		if strings.Contains(err.Error(), "project not found") {
			WriteError(c, ctx, "PROJECT_NOT_FOUND", err.Error(), consts.StatusNotFound)
			return
		}
		WriteError(c, ctx, "INTERNAL_ERROR", "Failed to get or create app for project: "+err.Error(), consts.StatusInternalServerError)
		return
	}

	// 自动批准权限请求
	appInstance.Permissions.AutoApproveSession(sessionID)

	// 从 Parts 中提取 prompt 文本
	promptText := models.ExtractPromptTextFromParts(req.Parts)

	if req.NoReply {
		// NoReply 模式 - 仅创建用户消息，不运行 AI
		h.handleNoReplyPrompt(c, ctx, sessionID, req, appInstance)
	} else {
		// 运行 AI 并获取响应
		h.handleSyncPrompt(c, ctx, sessionID, promptText, appInstance)
	}
}

// 消息处理相关常量
const (
	promptTimeout    = 5 * time.Minute        // AI 推理超时时间
	messageWaitDelay = 500 * time.Millisecond // 等待最后消息更新的延迟
	finishWaitDelay  = 200 * time.Millisecond // 消息完成后的等待延迟
)

// handleSyncPrompt 处理同步消息响应（Opencode 兼容）
func (h *Handlers) handleSyncPrompt(c context.Context, ctx *hertzapp.RequestContext, sessionID, prompt string, appInstance *internalapp.App) {
	assistantMsg, err := h.waitForAIResponse(c, sessionID, prompt, appInstance)
	if err != nil {
		switch err.Error() {
		case "request_cancelled":
			WriteError(c, ctx, "REQUEST_CANCELLED", "Request cancelled", consts.StatusRequestTimeout)
		case "timeout":
			WriteError(c, ctx, "TIMEOUT", "Request timeout", consts.StatusRequestTimeout)
		default:
			WriteError(c, ctx, "INTERNAL_ERROR", "Failed to run agent: "+err.Error(), consts.StatusInternalServerError)
		}
		return
	}

	// 使用 Opencode 兼容的响应格式
	response := models.MessageToPromptResponse(assistantMsg)
	WriteJSON(c, ctx, consts.StatusOK, response)
}

// waitForAIResponse 等待 AI 响应完成
// 返回 assistant 消息和错误（如果有）
func (h *Handlers) waitForAIResponse(c context.Context, sessionID, prompt string, appInstance *internalapp.App) (message.Message, error) {
	var assistantMsg message.Message

	// 运行 AI（AgentCoordinator 内部会创建用户消息）
	done := make(chan struct {
		result interface{}
		err    error
	}, 1)

	go func() {
		if appInstance.AgentCoordinator == nil {
			done <- struct {
				result interface{}
				err    error
			}{nil, fmt.Errorf("agent coordinator not initialized")}
			return
		}

		// 执行AI推理
		result, err := appInstance.AgentCoordinator.Run(c, sessionID, prompt)
		if err != nil {
			slog.Error("AI run failed", "session_id", sessionID, "error", err)
		}
		done <- struct {
			result interface{}
			err    error
		}{result, err}
	}()

	// 订阅消息事件
	messageEvents := appInstance.Messages.Subscribe(c)
	timeout := time.After(promptTimeout)

	// 标记 Run() 是否已完成
	runCompleted := false

	for {
		select {
		case <-c.Done():
			return assistantMsg, fmt.Errorf("request_cancelled")

		case <-timeout:
			return assistantMsg, fmt.Errorf("timeout")

		case result := <-done:
			if result.err != nil {
				return assistantMsg, result.err
			}
			// Run() 已完成,但不立即返回,继续等待消息完成
			runCompleted = true
			slog.Debug("AI Run completed, waiting for message finish", "session_id", sessionID)

		case event := <-messageEvents:
			msg := event.Payload
			if msg.SessionID == sessionID && msg.Role == message.Assistant {
				assistantMsg = msg
				// 检查消息是否完成
				if msg.FinishPart() != nil {
					// 消息已完成,等待可能的最后更新
					time.Sleep(finishWaitDelay)
					return assistantMsg, nil
				}
				// 如果 Run 已完成但消息还没有 finish part,继续等待
				if runCompleted {
					slog.Debug("Run completed but message not finished yet",
						"session_id", sessionID,
						"message_id", msg.ID,
						"parts_count", len(msg.Parts))
				}
			}
		}
	}
}

// handleNoReplyPrompt 处理 NoReply 模式的消息（仅创建用户消息）
func (h *Handlers) handleNoReplyPrompt(c context.Context, ctx *hertzapp.RequestContext, sessionID string, req models.PromptRequest, appInstance *internalapp.App) {
	// 创建用户消息参数
	params := message.CreateMessageParams{
		Role:  message.User,
		Parts: models.PartsToMessageParts(sessionID, req.Parts),
	}

	// 如果请求中指定了模型，使用它
	if req.Model != nil {
		params.Provider = req.Model.ProviderID
		params.Model = req.Model.ModelID
	}

	// 保存消息
	createdMsg, err := appInstance.Messages.Create(c, sessionID, params)
	if err != nil {
		WriteError(c, ctx, "INTERNAL_ERROR", "Failed to create message: "+err.Error(), consts.StatusInternalServerError)
		return
	}

	// 返回创建的消息（使用 Opencode 格式）
	response := models.MessageToPromptResponse(createdMsg)

	WriteJSON(c, ctx, consts.StatusOK, response)
}

// handleStreamMessage 处理流式消息响应
func (h *Handlers) handleStreamMessage(c context.Context, ctx *hertzapp.RequestContext, sessionID, prompt string, appInstance *internalapp.App) {
	// 设置 SSE 响应头
	ctx.SetContentType("text/event-stream")
	ctx.Response.Header.Set("Cache-Control", "no-cache")
	ctx.Response.Header.Set("Connection", "keep-alive")
	ctx.Response.Header.Set("X-Accel-Buffering", "no") // 禁用 nginx 缓冲

	// 发送开始事件（用户消息 ID 暂时为空，等 AgentCoordinator 创建后会通过事件更新）
	writeSSE(ctx, models.SSEEvent{
		Type: "start",
	})
	ctx.Flush()

	// 运行 AI（AgentCoordinator 内部会创建用户消息）
	done := make(chan struct {
		result interface{}
		err    error
	}, 1)

	go func() {
		if appInstance.AgentCoordinator == nil {
			done <- struct {
				result interface{}
				err    error
			}{nil, fmt.Errorf("agent coordinator not initialized")}
			return
		}

		result, err := appInstance.AgentCoordinator.Run(c, sessionID, prompt)
		done <- struct {
			result interface{}
			err    error
		}{result, err}
	}()

	// 订阅消息事件
	messageEvents := appInstance.Messages.Subscribe(c)
	messageReadBytes := make(map[string]int)
	var assistantMsg message.Message
	var streamErr error

	timeout := time.After(promptTimeout)

eventLoop:
	for {
		select {
		case <-c.Done():
			streamErr = fmt.Errorf("request cancelled")
			break eventLoop
		case <-timeout:
			streamErr = fmt.Errorf("request timeout")
			break eventLoop
		case result := <-done:
			if result.err != nil {
				streamErr = result.err
				break eventLoop
			}
			// 等待最后的消息更新
			time.Sleep(messageWaitDelay)
			break eventLoop
		case event := <-messageEvents:
			msg := event.Payload
			if msg.SessionID == sessionID && msg.Role == message.Assistant {
				assistantMsg = msg
				content := msg.Content().String()
				readBytes := messageReadBytes[msg.ID]

				if len(content) > readBytes {
					messageReadBytes[msg.ID] = len(content)

					// 发送内容块
					writeSSE(ctx, models.SSEEvent{
						Type: "message.updated",
						Properties: map[string]interface{}{
							"info": models.MessageToResponse(assistantMsg),
						},
					})
					ctx.Flush()
				}

				// 检查消息是否完成
				if msg.FinishPart() != nil {
					time.Sleep(finishWaitDelay)
					break eventLoop
				}
			}
		}
	}

	// 处理错误
	if streamErr != nil {
		writeSSEError(ctx, streamErr.Error())
		return
	}

	// 获取更新后的会话信息
	updatedSession, _ := appInstance.Sessions.Get(c, sessionID)

	// 发送完成事件
	writeSSE(ctx, models.SSEEvent{
		Type: "message.created",
		Properties: map[string]interface{}{
			"info":    models.MessageToResponse(assistantMsg),
			"session": models.SessionToResponse(updatedSession),
		},
	})
	ctx.Flush()
}

// HandleGetMessage 处理获取单个消息的请求
//
//	@Summary		获取消息详情
//	@Description	获取指定消息的详细信息
//	@Tags			Message
//	@Accept			json
//	@Produce		json
//	@Param			directory	query		string	true	"项目路径"
//	@Param			id			path		string	true	"消息ID"
//	@Success		200			{object}	models.MessageDetailResponse
//	@Failure		400			{object}	map[string]interface{}
//	@Failure		404			{object}	map[string]interface{}
//	@Router			/message/{id} [get]
func (h *Handlers) HandleGetMessage(c context.Context, ctx *hertzapp.RequestContext) {
	projectPath := string(ctx.Query("directory"))
	if projectPath == "" {
		WriteError(c, ctx, "MISSING_DIRECTORY_PARAM", "Directory query parameter is required", consts.StatusBadRequest)
		return
	}
	messageID := ctx.Param("id")

	// 获取项目的 app 实例
	appInstance, err := h.GetAppForProject(c, projectPath)
	if err != nil {
		if strings.Contains(err.Error(), "project not found") {
			WriteError(c, ctx, "PROJECT_NOT_FOUND", err.Error(), consts.StatusNotFound)
			return
		}
		WriteError(c, ctx, "INTERNAL_ERROR", "Failed to get or create app for project: "+err.Error(), consts.StatusInternalServerError)
		return
	}

	// 获取消息
	msg, err := appInstance.Messages.Get(c, messageID)
	if err != nil {
		WriteError(c, ctx, "MESSAGE_NOT_FOUND", "Message not found: "+err.Error(), consts.StatusNotFound)
		return
	}

	response := models.MessageDetailResponse{
		Message: models.MessageToResponse(msg),
	}

	WriteJSON(c, ctx, consts.StatusOK, response)
}

// writeSSE 写入 SSE 事件
func writeSSE(ctx *hertzapp.RequestContext, event models.SSEEvent) {
	data, _ := json.Marshal(event)
	ctx.Response.SetBodyString(fmt.Sprintf("event: %s\ndata: %s\n\n", event.Type, data))
}

// writeSSEError 写入 SSE 错误事件
func writeSSEError(ctx *hertzapp.RequestContext, message string) {
	event := models.SSEEvent{
		Type: "error",
		Properties: map[string]interface{}{
			"code":    "INTERNAL_ERROR",
			"message": message,
		},
	}
	writeSSE(ctx, event)
}
