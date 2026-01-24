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
//	@Param			session_id	path		string	true	"会话ID"
//	@Success		200			{object}	models.MessagesResponse
//	@Failure		400			{object}	map[string]interface{}
//	@Failure		404			{object}	map[string]interface{}
//	@Router			/session/{session_id}/message [get]
func (h *Handlers) HandleListMessages(c context.Context, ctx *hertzapp.RequestContext) {
	projectPath := string(ctx.Query("directory"))
	if projectPath == "" {
		WriteError(c, ctx, "MISSING_DIRECTORY_PARAM", "Directory query parameter is required", consts.StatusBadRequest)
		return
	}
	sessionID := ctx.Param("session_id")

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

// HandleCreateMessage 处理发送消息的请求（支持同步和流式）
//
//	@Summary		创建消息
//	@Description	在指定会话中创建新消息，支持同步和流式响应
//	@Tags			Message
//	@Accept			json
//	@Produce		json
//	@Param			project		query		string						true	"项目路径"
//	@Param			session_id	path		string						true	"会话ID"
//	@Param			request		body		models.CreateMessageRequest	true	"创建消息请求"
//	@Success		200			{object}	models.CreateMessageResponse
//	@Failure		400			{object}	map[string]interface{}
//	@Failure		404			{object}	map[string]interface{}
//	@Router			/session/{session_id}/message [post]
func (h *Handlers) HandleCreateMessage(c context.Context, ctx *hertzapp.RequestContext) {
	projectPath := string(ctx.Query("directory"))
	if projectPath == "" {
		WriteError(c, ctx, "MISSING_DIRECTORY_PARAM", "Directory query parameter is required", consts.StatusBadRequest)
		return
	}
	sessionID := ctx.Param("session_id")

	var req models.CreateMessageRequest
	if err := ctx.BindJSON(&req); err != nil {
		WriteError(c, ctx, "INVALID_REQUEST", "Invalid request body: "+err.Error(), consts.StatusBadRequest)
		return
	}

	if req.Prompt == "" {
		WriteError(c, ctx, "INVALID_REQUEST", "Prompt is required", consts.StatusBadRequest)
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

	if req.Stream {
		// 流式响应
		h.handleStreamMessage(c, ctx, sessionID, req.Prompt, appInstance)
	} else {
		// 同步响应
		h.handleSyncMessage(c, ctx, sessionID, req.Prompt, appInstance)
	}
}

// handleSyncMessage 处理同步消息响应
func (h *Handlers) handleSyncMessage(c context.Context, ctx *hertzapp.RequestContext, sessionID, prompt string, appInstance *internalapp.App) {
	// 创建用户消息
	_, err := appInstance.Messages.Create(c, sessionID, message.CreateMessageParams{
		Role:             message.User,
		Parts:            []message.ContentPart{message.TextContent{Text: prompt}},
		Model:            "",
		Provider:         "",
		IsSummaryMessage: false,
	})
	if err != nil {
		WriteError(c, ctx, "INTERNAL_ERROR", "Failed to create user message: "+err.Error(), consts.StatusInternalServerError)
		return
	}

	// 运行 AI
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
	var assistantMsg message.Message

	timeout := time.After(5 * time.Minute) // 5分钟超时

	for {
		select {
		case <-c.Done():
			WriteError(c, ctx, "REQUEST_CANCELLED", "Request cancelled", consts.StatusRequestTimeout)
			return
		case <-timeout:
			WriteError(c, ctx, "TIMEOUT", "Request timeout", consts.StatusRequestTimeout)
			return
		case result := <-done:
			if result.err != nil {
				WriteError(c, ctx, "INTERNAL_ERROR", "Failed to run agent: "+result.err.Error(), consts.StatusInternalServerError)
				return
			}
			// 等待最后的消息更新
			time.Sleep(500 * time.Millisecond)
			goto done
		case event := <-messageEvents:
			msg := event.Payload
			if msg.SessionID == sessionID && msg.Role == message.Assistant {
				assistantMsg = msg
				// 检查消息是否完成
				if msg.FinishPart() != nil {
					// 消息已完成
					time.Sleep(200 * time.Millisecond) // 等待可能的最后更新
					goto done
				}
			}
		}
	}

done:
	// 获取更新后的会话信息
	updatedSession, _ := appInstance.Sessions.Get(c, sessionID)

	response := models.CreateMessageResponse{
		Message: models.MessageToResponse(assistantMsg),
		Session: models.SessionToResponse(updatedSession),
	}

	WriteJSON(c, ctx, consts.StatusOK, response)
}

// handleStreamMessage 处理流式消息响应
func (h *Handlers) handleStreamMessage(c context.Context, ctx *hertzapp.RequestContext, sessionID, prompt string, appInstance *internalapp.App) {
	// 设置 SSE 响应头
	ctx.SetContentType("text/event-stream")
	ctx.Response.Header.Set("Cache-Control", "no-cache")
	ctx.Response.Header.Set("Connection", "keep-alive")
	ctx.Response.Header.Set("X-Accel-Buffering", "no") // 禁用 nginx 缓冲

	// 创建用户消息
	userMsg, err := appInstance.Messages.Create(c, sessionID, message.CreateMessageParams{
		Role:             message.User,
		Parts:            []message.ContentPart{message.TextContent{Text: prompt}},
		Model:            "",
		Provider:         "",
		IsSummaryMessage: false,
	})
	if err != nil {
		writeSSEError(ctx, "Failed to create user message: "+err.Error())
		return
	}

	// 发送开始事件
	writeSSE(ctx, models.SSEEvent{
		Type:      "start",
		MessageID: userMsg.ID,
	})
	ctx.Flush()

	// 运行 AI
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

	timeout := time.After(5 * time.Minute)

	for {
		select {
		case <-c.Done():
			writeSSEError(ctx, "Request cancelled")
			return
		case <-timeout:
			writeSSEError(ctx, "Request timeout")
			return
		case result := <-done:
			if result.err != nil {
				writeSSEError(ctx, "Failed to run agent: "+result.err.Error())
				return
			}
			// 等待最后的消息更新
			time.Sleep(500 * time.Millisecond)
			goto streamDone
		case event := <-messageEvents:
			msg := event.Payload
			if msg.SessionID == sessionID && msg.Role == message.Assistant {
				assistantMsg = msg
				content := msg.Content().String()
				readBytes := messageReadBytes[msg.ID]

				if len(content) > readBytes {
					chunk := content[readBytes:]
					messageReadBytes[msg.ID] = len(content)

					// 发送内容块
					writeSSE(ctx, models.SSEEvent{
						Type:    "chunk",
						Content: chunk,
					})
					ctx.Flush()
				}

				// 检查消息是否完成
				if msg.FinishPart() != nil {
					time.Sleep(200 * time.Millisecond)
					goto streamDone
				}
			}
		}
	}

streamDone:
	// 获取更新后的会话信息
	updatedSession, _ := appInstance.Sessions.Get(c, sessionID)

	// 发送完成事件
	writeSSE(ctx, models.SSEEvent{
		Type:      "done",
		MessageID: assistantMsg.ID,
		Message:   models.MessageToResponse(assistantMsg),
		Session:   models.SessionToResponse(updatedSession),
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
		Error: &models.ErrorDetail{
			Code:    "INTERNAL_ERROR",
			Message: message,
		},
	}
	writeSSE(ctx, event)
}
