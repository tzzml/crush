package handlers

import (
	"context"
	"encoding/json"
	"fmt"
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

	// 创建事件通道，只订阅指定项目的 app 实例事件
	eventCh := h.createEventChannelForProject(c, appInstance)

	// 发送初始连接确认
	// Opencode style: server.connected
	initEvent := models.SSEEvent{
		Type: "server.connected",
		Properties: map[string]string{
			"status": "connected",
		},
	}
	initData, _ := json.Marshal(initEvent)
	ctx.Response.SetBodyString(fmt.Sprintf("event: server.connected\ndata: %s\n\n", initData))
	ctx.Flush()

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
			ctx.Response.SetBodyString(": heartbeat\n\n")
			ctx.Flush()
		case event, ok := <-eventCh:
			if !ok {
				slog.Info("Event channel closed", "remote_addr", remoteAddr)
				return
			}

			// 处理不同类型的事件
			if err := h.handleSSEEvent(ctx, event); err != nil {
				slog.Error("Failed to handle SSE event", "error", err, "remote_addr", remoteAddr)
				return
			}
		}
	}
}

// handleSSEEvent 处理单个SSE事件
func (h *Handlers) handleSSEEvent(ctx *hertzapp.RequestContext, event tea.Msg) error {
	var resp models.SSEEvent

	// 处理不同类型的事件
	switch e := event.(type) {
	case pubsub.Event[internalapp.LSPEvent]:
		// LSP 事件
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

	case pubsub.Event[message.Message]:
		// 消息事件
		msgResp := models.MessageToResponse(e.Payload)
		switch e.Type {
		case pubsub.CreatedEvent:
			resp.Type = "message.created"
			resp.Properties = map[string]interface{}{
				"info": msgResp,
			}
		case pubsub.UpdatedEvent:
			resp.Type = "message.updated"
			resp.Properties = map[string]interface{}{
				"info": msgResp,
			}
		case pubsub.DeletedEvent:
			resp.Type = "message.removed"
			resp.Properties = map[string]interface{}{
				"messageID": msgResp.ID,
				"sessionID": msgResp.SessionID,
			}
		default:
			return nil
		}

	case pubsub.Event[session.Session]:
		// 会话事件
		sessResp := models.SessionToResponse(e.Payload)
		switch e.Type {
		case pubsub.CreatedEvent:
			resp.Type = "session.created"
			resp.Properties = map[string]interface{}{
				"session": sessResp,
			}
		case pubsub.UpdatedEvent:
			resp.Type = "session.updated"
			resp.Properties = map[string]interface{}{
				"session": sessResp,
			}
		case pubsub.DeletedEvent:
			resp.Type = "session.deleted"
			resp.Properties = map[string]interface{}{
				"session": sessResp,
			}
		default:
			return nil
		}

	default:
		// Unknown event type, ignore
		return nil
	}

	// 序列化事件数据
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
	ctx.Response.SetBodyString(fmt.Sprintf("event: %s\ndata: %s\n\n", resp.Type, data))
	ctx.Flush()

	return nil
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
