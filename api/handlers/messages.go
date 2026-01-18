package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/charmbracelet/crush/api/models"
	"github.com/charmbracelet/crush/internal/app"
	"github.com/charmbracelet/crush/internal/message"
)

// HandleListMessages 处理获取会话消息列表的请求
func (h *Handlers) HandleListMessages(w http.ResponseWriter, r *http.Request) {
	projectPath, sessionID, err := extractProjectAndSessionIDFromMessages(r)
	if err != nil {
		WriteError(w, "INVALID_REQUEST", err.Error(), http.StatusBadRequest)
		return
	}

	// 获取项目的 app 实例
	appInstance, err := h.GetAppForProject(r.Context(), projectPath)
	if err != nil {
		WriteError(w, "PROJECT_NOT_FOUND", "Failed to get app for project: "+err.Error(), http.StatusNotFound)
		return
	}

	// 获取消息列表
	messages, err := appInstance.Messages.List(r.Context(), sessionID)
	if err != nil {
		WriteError(w, "INTERNAL_ERROR", "Failed to list messages: "+err.Error(), http.StatusInternalServerError)
		return
	}

	response := models.MessagesResponse{
		Messages: make([]models.MessageResponse, len(messages)),
		Total:    len(messages),
	}
	for i, m := range messages {
		response.Messages[i] = models.MessageToResponse(m)
	}

	WriteJSON(w, http.StatusOK, response)
}

// HandleCreateMessage 处理发送消息的请求（支持同步和流式）
func (h *Handlers) HandleCreateMessage(w http.ResponseWriter, r *http.Request) {
	projectPath, sessionID, err := extractProjectAndSessionIDFromMessages(r)
	if err != nil {
		WriteError(w, "INVALID_REQUEST", err.Error(), http.StatusBadRequest)
		return
	}

	slog.Info("HandleCreateMessage called", "project_path", projectPath, "session_id", sessionID, "full_url", r.URL.String(), "raw_path", r.URL.Path)

	var req models.CreateMessageRequest
	if err := decodeJSON(r, &req); err != nil {
		WriteError(w, "INVALID_REQUEST", "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	if req.Prompt == "" {
		WriteError(w, "INVALID_REQUEST", "Prompt is required", http.StatusBadRequest)
		return
	}

	// 获取项目的 app 实例
	appInstance, err := h.GetAppForProject(r.Context(), projectPath)
	if err != nil {
		WriteError(w, "PROJECT_NOT_FOUND", "Failed to get app for project: "+err.Error(), http.StatusNotFound)
		return
	}

	slog.Info("Got app instance for project", "project_path", projectPath, "app_instance", fmt.Sprintf("%p", appInstance))

	// 不再预先检查会话是否存在，直接尝试创建消息
	// 如果会话不存在，消息创建会失败并返回适当的错误
	slog.Info("Proceeding to create message without session pre-check", "session_id", sessionID)

	// 自动批准权限请求
	appInstance.Permissions.AutoApproveSession(sessionID)

	if req.Stream {
		// 流式响应
		h.handleStreamMessage(w, r, sessionID, req.Prompt, appInstance)
	} else {
		// 同步响应
		h.handleSyncMessage(w, r, sessionID, req.Prompt, appInstance)
	}
}

// handleSyncMessage 处理同步消息响应
func (h *Handlers) handleSyncMessage(w http.ResponseWriter, r *http.Request, sessionID, prompt string, appInstance *app.App) {
	ctx := r.Context()

	// 创建用户消息
	userMsg, err := appInstance.Messages.Create(ctx, sessionID, message.CreateMessageParams{
		Role:             message.User,
		Parts:            []message.ContentPart{message.TextContent{Text: prompt}},
		Model:            "",
		Provider:         "",
		IsSummaryMessage: false,
	})
	if err != nil {
		WriteError(w, "INTERNAL_ERROR", "Failed to create user message: "+err.Error(), http.StatusInternalServerError)
		return
	}
	slog.Info("Created user message", "message_id", userMsg.ID, "session_id", sessionID)

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

		promptPreview := prompt
		if len(prompt) > 50 {
			promptPreview = prompt[:50] + "..."
		}
		slog.Info("Starting AI run", "session_id", sessionID, "prompt", promptPreview)
		result, err := appInstance.AgentCoordinator.Run(ctx, sessionID, prompt)
		slog.Info("AI run completed", "session_id", sessionID, "error", err)
		done <- struct {
			result interface{}
			err    error
		}{result, err}
	}()

	// 订阅消息事件
	messageEvents := appInstance.Messages.Subscribe(ctx)
	var assistantMsg message.Message

	timeout := time.After(5 * time.Minute) // 5分钟超时

	for {
		select {
		case <-ctx.Done():
			WriteError(w, "REQUEST_CANCELLED", "Request cancelled", http.StatusRequestTimeout)
			return
		case <-timeout:
			WriteError(w, "TIMEOUT", "Request timeout", http.StatusRequestTimeout)
			return
		case result := <-done:
			if result.err != nil {
				WriteError(w, "INTERNAL_ERROR", "Failed to run agent: "+result.err.Error(), http.StatusInternalServerError)
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
	updatedSession, _ := appInstance.Sessions.Get(ctx, sessionID)

	response := models.CreateMessageResponse{
		Message: models.MessageToResponse(assistantMsg),
		Session: models.SessionToResponse(updatedSession),
	}

	WriteJSON(w, http.StatusOK, response)
}

// handleStreamMessage 处理流式消息响应
func (h *Handlers) handleStreamMessage(w http.ResponseWriter, r *http.Request, sessionID, prompt string, appInstance *app.App) {
	ctx := r.Context()

	// 设置 SSE 响应头
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // 禁用 nginx 缓冲

	flusher, ok := w.(http.Flusher)
	if !ok {
		WriteError(w, "INTERNAL_ERROR", "Streaming not supported", http.StatusInternalServerError)
		return
	}

	// 创建用户消息
	userMsg, err := appInstance.Messages.Create(ctx, sessionID, message.CreateMessageParams{
		Role:             message.User,
		Parts:            []message.ContentPart{message.TextContent{Text: prompt}},
		Model:            "",
		Provider:         "",
		IsSummaryMessage: false,
	})
	if err != nil {
		writeSSEError(w, "Failed to create user message: "+err.Error())
		return
	}

	// 发送开始事件
	writeSSE(w, models.SSEEvent{
		Type:      "start",
		MessageID: userMsg.ID,
	})
	flusher.Flush()

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

		result, err := appInstance.AgentCoordinator.Run(ctx, sessionID, prompt)
		done <- struct {
			result interface{}
			err    error
		}{result, err}
	}()

	// 订阅消息事件
	messageEvents := appInstance.Messages.Subscribe(ctx)
	messageReadBytes := make(map[string]int)
	var assistantMsg message.Message

	timeout := time.After(5 * time.Minute)

	for {
		select {
		case <-ctx.Done():
			writeSSEError(w, "Request cancelled")
			return
		case <-timeout:
			writeSSEError(w, "Request timeout")
			return
		case result := <-done:
			if result.err != nil {
				writeSSEError(w, "Failed to run agent: "+result.err.Error())
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
					writeSSE(w, models.SSEEvent{
						Type:    "chunk",
						Content: chunk,
					})
					flusher.Flush()
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
	updatedSession, _ := appInstance.Sessions.Get(ctx, sessionID)

	// 发送完成事件
	writeSSE(w, models.SSEEvent{
		Type:      "done",
		MessageID: assistantMsg.ID,
		Message:   models.MessageToResponse(assistantMsg),
		Session:   models.SessionToResponse(updatedSession),
	})
	flusher.Flush()
}

// HandleGetMessage 处理获取单个消息的请求
func (h *Handlers) HandleGetMessage(w http.ResponseWriter, r *http.Request) {
	projectPath, messageID, err := extractProjectAndMessageID(r)
	if err != nil {
		WriteError(w, "INVALID_REQUEST", err.Error(), http.StatusBadRequest)
		return
	}

	// 获取项目的 app 实例
	appInstance, err := h.GetAppForProject(r.Context(), projectPath)
	if err != nil {
		WriteError(w, "PROJECT_NOT_FOUND", "Failed to get app for project: "+err.Error(), http.StatusNotFound)
		return
	}

	// 获取消息
	msg, err := appInstance.Messages.Get(r.Context(), messageID)
	if err != nil {
		WriteError(w, "MESSAGE_NOT_FOUND", "Message not found: "+err.Error(), http.StatusNotFound)
		return
	}

	response := models.MessageDetailResponse{
		Message: models.MessageToResponse(msg),
	}

	WriteJSON(w, http.StatusOK, response)
}

// extractSessionIDFromMessagesPath 从消息路径中提取会话 ID
// extractProjectAndMessageID 从 URL 中提取项目路径和消息 ID
// URL 格式: /api/v1/projects/{project_path}/messages/{message_id}
func extractProjectAndMessageID(r *http.Request) (projectPath, messageID string, err error) {
	path := r.URL.Path
	prefix := "/api/v1/projects/"
	if !strings.HasPrefix(path, prefix) {
		return "", "", fmt.Errorf("invalid path format")
	}

	// 移除前缀
	rest := path[len(prefix):]

	// 查找 /messages/ 的位置
	messagesIndex := strings.Index(rest, "/messages/")
	if messagesIndex == -1 {
		return "", "", fmt.Errorf("missing /messages/ in path")
	}

	// 提取项目路径
	projectPath = rest[:messagesIndex]
	if projectPath == "" {
		return "", "", fmt.Errorf("project path is empty")
	}

	// URL 解码项目路径
	projectPath, err = url.QueryUnescape(projectPath)
	if err != nil {
		return "", "", fmt.Errorf("failed to decode project path: %w", err)
	}

	// 提取消息ID部分
	messagePart := rest[messagesIndex+len("/messages/"):]
	// 移除可能的尾部斜杠
	messagePart = strings.TrimSuffix(messagePart, "/")
	if messagePart == "" {
		return "", "", fmt.Errorf("message ID is empty")
	}

	messageID = messagePart
	return projectPath, messageID, nil
}

// extractProjectAndSessionIDFromMessages 从消息URL中提取项目路径和会话ID
// URL 格式: /api/v1/projects/{project_path}/sessions/{session_id}/messages
func extractProjectAndSessionIDFromMessages(r *http.Request) (projectPath, sessionID string, err error) {
	path := r.URL.Path
	prefix := "/api/v1/projects/"
	if !strings.HasPrefix(path, prefix) {
		return "", "", fmt.Errorf("invalid path format")
	}

	// 移除前缀
	rest := path[len(prefix):]

	// 查找 /sessions/ 的位置
	sessionsIndex := strings.Index(rest, "/sessions/")
	if sessionsIndex == -1 {
		return "", "", fmt.Errorf("missing /sessions/ in path")
	}

	// 提取项目路径
	projectPath = rest[:sessionsIndex]
	if projectPath == "" {
		return "", "", fmt.Errorf("project path is empty")
	}

	// URL 解码项目路径
	projectPath, err = url.QueryUnescape(projectPath)
	if err != nil {
		return "", "", fmt.Errorf("failed to decode project path: %w", err)
	}

	// 查找 /messages 的位置
	messagesIndex := strings.Index(rest, "/messages")
	if messagesIndex == -1 {
		return "", "", fmt.Errorf("missing /messages in path")
	}

	// 提取会话ID部分
	sessionPart := rest[sessionsIndex+len("/sessions/") : messagesIndex]
	// 移除可能的尾部斜杠
	sessionPart = strings.TrimSuffix(sessionPart, "/")
	if sessionPart == "" {
		return "", "", fmt.Errorf("session ID is empty")
	}

	sessionID = sessionPart
	return projectPath, sessionID, nil
}

func extractSessionIDFromMessagesPath(r *http.Request) string {
	path := r.URL.Path
	prefix := "/api/v1/sessions/"
	if !strings.HasPrefix(path, prefix) {
		return ""
	}

	rest := path[len(prefix):]
	idx := strings.Index(rest, "/messages")
	if idx == -1 {
		return ""
	}

	return rest[:idx]
}

// decodeJSON 解码 JSON 请求体
func decodeJSON(r *http.Request, v interface{}) error {
	return json.NewDecoder(r.Body).Decode(v)
}

// writeSSE 写入 SSE 事件
func writeSSE(w io.Writer, event models.SSEEvent) {
	data, _ := json.Marshal(event)
	fmt.Fprintf(w, "event: %s\n", event.Type)
	fmt.Fprintf(w, "data: %s\n\n", data)
}

// writeSSEError 写入 SSE 错误事件
func writeSSEError(w io.Writer, message string) {
	event := models.SSEEvent{
		Type: "error",
		Error: &models.ErrorDetail{
			Code:    "INTERNAL_ERROR",
			Message: message,
		},
	}
	writeSSE(w, event)
}
