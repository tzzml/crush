package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"sync"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/crush/api/models"
	internalapp "github.com/charmbracelet/crush/internal/app"
	"github.com/charmbracelet/crush/internal/message"
	"github.com/charmbracelet/crush/internal/pubsub"
	"github.com/charmbracelet/crush/internal/session"
	hertzapp "github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
)

// messagePartState 用于跟踪消息每个 part 的状态，以便精确计算增量
type messagePartState struct {
	// key 为 part 索引，value 为该 part 已发送的文本长度
	lengths map[int]int
	// 已处理过的 parts 数量，用于检测新增 non-text parts (如 tool_call)
	partsCount int
}

func newMessagePartState() *messagePartState {
	return &messagePartState{
		lengths: make(map[int]int),
	}
}

// HandleSSE 处理 Server-Sent Events 请求
//
//	@Summary		订阅服务器事件
//	@Description	订阅项目的实时事件流
//	@Tags			Event
//	@Accept			json
//	@Produce		text/event-stream
//	@Param			directory	query		string	true	"项目路径"
//	@Success		200		{object}	models.SSEEvent	"Event stream"
//	@Failure		400		{object}	map[string]interface{}
//	@Failure		404		{object}	map[string]interface{}
//	@Router			/event [get]
func (h *Handlers) HandleSSE(c context.Context, ctx *hertzapp.RequestContext) {
	// 从查询参数中获取项目路径
	projectPath := string(ctx.Query("directory"))
	if projectPath == "" {
		WriteError(c, ctx, "MISSING_DIRECTORY_PARAM", "Directory query parameter is required", consts.StatusBadRequest)
		return
	}

	remoteAddr := string(ctx.GetHeader("X-Real-IP"))
	if remoteAddr == "" {
		remoteAddr = ctx.RemoteAddr().String()
	}

	slog.Info("SSE connection established", "remote_addr", remoteAddr, "project", projectPath)

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

	// 设置SSE头
	ctx.SetContentType("text/event-stream")
	ctx.Response.Header.Set("Cache-Control", "no-cache")
	ctx.Response.Header.Set("Connection", "keep-alive")
	ctx.Response.Header.Set("Access-Control-Allow-Origin", "*")
	ctx.Response.Header.Set("Access-Control-Allow-Headers", "Cache-Control")
	ctx.Response.Header.Set("Transfer-Encoding", "chunked")

	slog.Info("SSE connection established", "remote_addr", remoteAddr, "project", projectPath)

	// 创建事件通道，只订阅指定项目的 app 实例事件
	eventCh := h.createEventChannelForProject(c, appInstance)

	// 使用 io.Pipe 进行流式传输
	pr, pw := io.Pipe()
	ctx.Response.SetBodyStream(pr, -1)

	// 在新的 goroutine 中处理事件并写入 Response
	go func() {
		defer pw.Close()

		slog.Info("Starting SSE writer goroutine", "remote_addr", remoteAddr)

		// 创建消息状态缓存，用于计算增量
		messageStates := make(map[string]*messagePartState)

		// 发送初始连接确认
		initEvent := models.SSEEvent{
			Type: "server.connected",
			Properties: map[string]string{
				"status": "connected",
			},
		}
		initData, _ := json.Marshal(initEvent)
		if _, err := fmt.Fprintf(pw, "event: server.connected\ndata: %s\n\n", initData); err != nil {
			slog.Error("Failed to write init event", "error", err, "remote_addr", remoteAddr)
			return
		}
		slog.Info("Init event written to pipe", "remote_addr", remoteAddr)

		// 创建心跳定时器
		heartbeat := time.NewTicker(30 * time.Second)
		defer heartbeat.Stop()

		for {
			select {
			case <-c.Done():
				slog.Info("SSE connection closed by client", "remote_addr", remoteAddr)
				return
			case <-heartbeat.C:
				// 发送心跳保持连接
				if _, err := fmt.Fprintf(pw, ": heartbeat\n\n"); err != nil {
					slog.Info("Client disconnected (heartbeat write failed)", "remote_addr", remoteAddr)
					return
				}
			case event, ok := <-eventCh:
				if !ok {
					slog.Info("Event channel closed", "remote_addr", remoteAddr)
					return
				}

				// 处理并发送事件（带增量计算）
				if err := h.writeSSEEventWithDelta(pw, event, messageStates); err != nil {
					slog.Error("Failed to write SSE event", "error", err, "remote_addr", remoteAddr)
					return
				}
			}
		}
	}()
}

// writeSSEEventWithDelta 将 SSE 事件写入 io.Writer，对消息更新事件计算增量
func (h *Handlers) writeSSEEventWithDelta(w io.Writer, event tea.Msg, messageStates map[string]*messagePartState) error {
	// 处理不同类型的事件
	switch e := event.(type) {
	case pubsub.Event[internalapp.LSPEvent]:
		// LSP 事件 - 直接发送，无需增量计算
		return h.writeLSPEvent(w, e)

	case pubsub.Event[message.Message]:
		// 消息事件 - 需要增量计算
		return h.writeMessageEventWithDelta(w, e, messageStates)

	case pubsub.Event[session.Session]:
		// 会话事件 - 直接发送，无需增量计算
		return h.writeSessionEvent(w, e)

	default:
		// Unknown event type, ignore
		return nil
	}
}

// writeLSPEvent 写入 LSP 事件
func (h *Handlers) writeLSPEvent(w io.Writer, e pubsub.Event[internalapp.LSPEvent]) error {
	var resp models.SSEEvent

	switch e.Payload.Type {
	case internalapp.LSPEventStateChanged:
		resp.Type = "lsp.server.state_changed"
		resp.Properties = map[string]interface{}{
			"name":             e.Payload.Name,
			"state":            e.Payload.State,
			"error":            e.Payload.Error,
			"diagnostic_count": e.Payload.DiagnosticCount,
		}
	case internalapp.LSPEventDiagnosticsChanged:
		resp.Type = "lsp.client.diagnostics"
		resp.Properties = map[string]interface{}{
			"serverID":         e.Payload.Name,
			"diagnostic_count": e.Payload.DiagnosticCount,
		}
	default:
		return nil
	}

	return h.sendSSEEvent(w, resp)
}

// writeSessionEvent 写入会话事件
func (h *Handlers) writeSessionEvent(w io.Writer, e pubsub.Event[session.Session]) error {
	var resp models.SSEEvent
	sessResp := models.SessionToResponse(e.Payload)

	switch e.Type {
	case pubsub.CreatedEvent:
		resp.Type = "session.created"
		resp.Properties = map[string]interface{}{
			"info": sessResp,
		}
	case pubsub.UpdatedEvent:
		resp.Type = "session.updated"
		resp.Properties = map[string]interface{}{
			"info": sessResp,
		}
	case pubsub.DeletedEvent:
		resp.Type = "session.deleted"
		resp.Properties = map[string]interface{}{
			"sessionID": sessResp.ID,
		}
	default:
		return nil
	}

	return h.sendSSEEvent(w, resp)
}

// writeMessageEventWithDelta 写入消息事件，对更新事件计算增量
func (h *Handlers) writeMessageEventWithDelta(w io.Writer, e pubsub.Event[message.Message], messageStates map[string]*messagePartState) error {
	msg := e.Payload
	msgID := msg.ID

	switch e.Type {
	case pubsub.CreatedEvent:
		// 消息创建事件 - 初始化状态并发送创建事件
		messageStates[msgID] = newMessagePartState()
		resp := models.SSEEvent{
			Type: "message.created",
			Properties: map[string]interface{}{
				"info": models.MessageToResponse(msg),
			},
		}
		return h.sendSSEEvent(w, resp)

	case pubsub.UpdatedEvent:
		// 消息更新事件 - 计算增量并发送 message.part.updated
		return h.processMessageUpdate(w, msg, messageStates)

	case pubsub.DeletedEvent:
		// 消息删除事件 - 清理状态并发送删除事件
		delete(messageStates, msgID)
		resp := models.SSEEvent{
			Type: "message.removed",
			Properties: map[string]interface{}{
				"messageID": msgID,
				"sessionID": msg.SessionID,
			},
		}
		return h.sendSSEEvent(w, resp)

	default:
		return nil
	}
}

// processMessageUpdate 处理消息更新，计算并发送增量事件
func (h *Handlers) processMessageUpdate(w io.Writer, msg message.Message, messageStates map[string]*messagePartState) error {
	msgID := msg.ID

	// 获取或创建状态
	state, exists := messageStates[msgID]
	if !exists {
		state = newMessagePartState()
		messageStates[msgID] = state
	}

	// 遍历所有 parts，检测增量
	var eventsToSend []models.SSEEvent

	for i, part := range msg.Parts {
		partIndex := i

		switch p := part.(type) {
		case message.ReasoningContent:
			// 计算 reasoning 增量（使用 rune 处理多字节字符）
			thinkingRunes := []rune(p.Thinking)
			lastLen := state.lengths[partIndex]
			currentLen := len(thinkingRunes)
			if currentLen > lastLen {
				deltaRunes := thinkingRunes[lastLen:]
				delta := string(deltaRunes)
				state.lengths[partIndex] = currentLen

				// 构建 part 更新事件
				partData := map[string]interface{}{
					"type":     "reasoning",
					"id":       fmt.Sprintf("%s-part-%d", msgID, partIndex),
					"thinking": p.Thinking,
				}
				if p.StartedAt > 0 {
					partData["started_at"] = p.StartedAt
				}
				if p.FinishedAt > 0 {
					partData["finished_at"] = p.FinishedAt
				}

				evt := models.SSEEvent{
					Type: "message.part.updated",
					Properties: map[string]interface{}{
						"messageID": msgID,
						"sessionID": msg.SessionID,
						"partIndex": partIndex,
						"part":      partData,
						"delta":     delta,
					},
				}
				eventsToSend = append(eventsToSend, evt)
			}

		case message.TextContent:
			// 计算 text 增量（使用 rune 处理多字节字符）
			textRunes := []rune(p.Text)
			lastLen := state.lengths[partIndex]
			currentLen := len(textRunes)
			if currentLen > lastLen {
				deltaRunes := textRunes[lastLen:]
				delta := string(deltaRunes)
				state.lengths[partIndex] = currentLen

				// 构建 part 更新事件
				partData := map[string]interface{}{
					"type": "text",
					"id":   fmt.Sprintf("%s-part-%d", msgID, partIndex),
					"text": p.Text,
				}

				evt := models.SSEEvent{
					Type: "message.part.updated",
					Properties: map[string]interface{}{
						"messageID": msgID,
						"sessionID": msg.SessionID,
						"partIndex": partIndex,
						"part":      partData,
						"delta":     delta,
					},
				}
				eventsToSend = append(eventsToSend, evt)
			}

		case message.ToolCall:
			// ToolCall 事件 - 当有新的 tool call 时发送
			if partIndex >= state.partsCount {
				partData := map[string]interface{}{
					"type":              "tool_call",
					"id":                p.ID,
					"name":              p.Name,
					"input":             p.Input,
					"finished":          p.Finished,
					"provider_executed": p.ProviderExecuted,
				}

				evt := models.SSEEvent{
					Type: "message.part.updated",
					Properties: map[string]interface{}{
						"messageID": msgID,
						"sessionID": msg.SessionID,
						"partIndex": partIndex,
						"part":      partData,
						"delta":     "",
					},
				}
				eventsToSend = append(eventsToSend, evt)
			}

		case message.Finish:
			// Finish 事件 - 当消息完成时发送
			if partIndex >= state.partsCount {
				partData := map[string]interface{}{
					"type":   "finish",
					"reason": string(p.Reason),
					"time":   p.Time,
				}
				if p.Message != "" {
					partData["message"] = p.Message
				}
				if p.Details != "" {
					partData["details"] = p.Details
				}

				evt := models.SSEEvent{
					Type: "message.part.updated",
					Properties: map[string]interface{}{
						"messageID": msgID,
						"sessionID": msg.SessionID,
						"partIndex": partIndex,
						"part":      partData,
						"delta":     "",
					},
				}
				eventsToSend = append(eventsToSend, evt)
			}
		}
	}

	// 更新 parts 计数
	state.partsCount = len(msg.Parts)

	// 发送所有事件
	for _, evt := range eventsToSend {
		if err := h.sendSSEEvent(w, evt); err != nil {
			return err
		}
	}

	// 同时发送完整的 message.updated 事件（供需要完整消息的客户端使用）
	fullResp := models.SSEEvent{
		Type: "message.updated",
		Properties: map[string]interface{}{
			"info": models.MessageToResponse(msg),
		},
	}
	return h.sendSSEEvent(w, fullResp)
}

// sendSSEEvent 发送 SSE 事件
func (h *Handlers) sendSSEEvent(w io.Writer, resp models.SSEEvent) error {
	data, err := json.Marshal(resp)
	if err != nil {
		// 如果序列化失败，发送 error 事件
		errResp := models.SSEEvent{
			Type: "error",
			Properties: map[string]string{
				"message": fmt.Sprintf("failed to serialize event: %s", err.Error()),
			},
		}
		data, _ = json.Marshal(errResp)
	}

	// 发送SSE格式的事件
	_, err = fmt.Fprintf(w, "event: %s\ndata: %s\n\n", resp.Type, data)
	return err
}

// createEventChannelForProject 为指定项目的 app 实例创建事件通道
func (h *Handlers) createEventChannelForProject(ctx context.Context, appInstance *internalapp.App) <-chan tea.Msg {
	eventCh := make(chan tea.Msg, 100)

	var wg sync.WaitGroup

	// 订阅该项目的 sessions 事件
	wg.Add(1)
	go func() {
		defer wg.Done()
		sessionsCh := appInstance.Sessions.Subscribe(ctx)
		for {
			select {
			case <-ctx.Done():
				return
			case event, ok := <-sessionsCh:
				if !ok {
					return
				}
				select {
				case eventCh <- event:
				case <-ctx.Done():
					return
				default:
					// 通道已满，跳过此事件
					continue
				}
			}
		}
	}()

	// 订阅该项目的 messages 事件
	wg.Add(1)
	go func() {
		defer wg.Done()
		messagesCh := appInstance.Messages.Subscribe(ctx)
		for {
			select {
			case <-ctx.Done():
				return
			case event, ok := <-messagesCh:
				if !ok {
					return
				}
				select {
				case eventCh <- event:
				case <-ctx.Done():
					return
				default:
					// 通道已满，跳过此事件
					continue
				}
			}
		}
	}()

	// 订阅该项目的 LSP 事件
	// LSP 事件在 internal 中是全局的，但我们可以通过 app 实例的 LSPClients 来过滤
	// 只发送属于该项目的 LSP 客户端事件
	wg.Add(1)
	go func() {
		defer wg.Done()
		lspCh := internalapp.SubscribeLSPEvents(ctx)

		// 获取该项目的 LSP 客户端名称集合
		projectLSPNames := make(map[string]bool)
		for name := range appInstance.LSPClients.Seq2() {
			projectLSPNames[name] = true
		}

		for {
			select {
			case <-ctx.Done():
				return
			case event, ok := <-lspCh:
				if !ok {
					return
				}

				// 过滤：只发送属于该项目的 LSP 事件
				// event 已经是 pubsub.Event[app.LSPEvent] 类型
				lspName := event.Payload.Name
				// 检查该 LSP 客户端是否属于当前项目
				if !projectLSPNames[lspName] {
					// 不属于当前项目，跳过
					continue
				}

				select {
				case eventCh <- event:
				case <-ctx.Done():
					return
				default:
					// 通道已满，跳过此事件
					continue
				}
			}
		}
	}()

	// 在后台等待所有goroutine完成，然后关闭通道
	go func() {
		wg.Wait()
		close(eventCh)
	}()

	return eventCh
}
