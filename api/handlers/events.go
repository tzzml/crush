package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"sync"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/crush/api/models"
	"github.com/charmbracelet/crush/internal/pubsub"
	"github.com/charmbracelet/crush/internal/app"
	"github.com/charmbracelet/crush/internal/message"
	"github.com/charmbracelet/crush/internal/session"
)

// HandleSSE 处理 Server-Sent Events 请求
func (h *Handlers) HandleSSE(w http.ResponseWriter, r *http.Request) {
	// 从 URL 中提取项目路径
	projectPath, err := extractProjectPathFromEvents(r)
	if err != nil {
		WriteError(w, "INVALID_REQUEST", "Failed to extract project path: "+err.Error(), http.StatusBadRequest)
		return
	}

	slog.Info("SSE connection established", "remote_addr", r.RemoteAddr, "project", projectPath)

	// 获取项目的 app 实例
	appInstance, err := h.GetAppForProject(r.Context(), projectPath)
	if err != nil {
		if strings.Contains(err.Error(), "not open") {
			WriteError(w, "APP_NOT_OPENED", "Project app instance is not open. Call open first: "+err.Error(), http.StatusBadRequest)
			return
		}
		WriteError(w, "PROJECT_NOT_FOUND", "Failed to get app for project: "+err.Error(), http.StatusNotFound)
		return
	}

	// 设置SSE头
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Cache-Control")

	// 获取flusher用于实时推送
	flusher, ok := w.(http.Flusher)
	if !ok {
		WriteError(w, "UNSUPPORTED", "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	// 创建事件通道，只订阅指定项目的 app 实例事件
	eventCh := h.createEventChannelForProject(r.Context(), appInstance)

	// 发送初始连接确认
	fmt.Fprintf(w, "event: connected\ndata: {\"status\": \"connected\"}\n\n")
	flusher.Flush()

	// 创建心跳定时器
	heartbeat := time.NewTicker(30 * time.Second)
	defer heartbeat.Stop()

	for {
		select {
		case <-r.Context().Done():
			slog.Info("SSE connection closed by client", "remote_addr", r.RemoteAddr)
			return
		case <-heartbeat.C:
			// 发送心跳保持连接
			fmt.Fprintf(w, ": heartbeat\n\n")
			flusher.Flush()
		case event, ok := <-eventCh:
			if !ok {
				slog.Info("Event channel closed", "remote_addr", r.RemoteAddr)
				return
			}

			// 处理不同类型的事件
			if err := h.handleSSEEvent(w, flusher, event); err != nil {
				slog.Error("Failed to handle SSE event", "error", err, "remote_addr", r.RemoteAddr)
				return
			}
		}
	}
}

// handleSSEEvent 处理单个SSE事件
func (h *Handlers) handleSSEEvent(w http.ResponseWriter, flusher http.Flusher, event tea.Msg) error {
	var eventType string
	var eventData interface{}

	// 处理不同类型的事件
	switch e := event.(type) {
	case pubsub.Event[app.LSPEvent]:
		eventType = string(e.Type)
		eventData = e.Payload
	case pubsub.Event[message.Message]:
		// 消息事件：转换为 API 响应格式
		eventType = string(e.Type)
		eventData = models.MessageToResponse(e.Payload)
	case pubsub.Event[session.Session]:
		// 会话事件：转换为 API 响应格式
		eventType = string(e.Type)
		eventData = models.SessionToResponse(e.Payload)
	default:
		// 使用反射来识别事件类型
		eventType, eventData = h.extractEventData(event)
	}

	// 序列化事件数据
	data, err := json.Marshal(eventData)
	if err != nil {
		// 如果序列化失败，发送简单的事件
		fmt.Fprintf(w, "event: %s\ndata: {\"error\": \"failed to serialize event: %s\"}\n\n", eventType, err.Error())
		flusher.Flush()
		return nil
	}

	// 发送SSE格式的事件
	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", eventType, data)
	flusher.Flush()

	return nil
}

// extractEventData 使用反射提取事件数据
func (h *Handlers) extractEventData(event tea.Msg) (string, interface{}) {
	v := reflect.ValueOf(event)
	if !v.IsValid() || v.Kind() != reflect.Struct {
		return "unknown", map[string]string{"type": fmt.Sprintf("%T", event)}
	}

	// 检查是否是 pubsub.Event 类型
	eventTypeField := v.FieldByName("Type")
	payloadField := v.FieldByName("Payload")

	if !eventTypeField.IsValid() || !payloadField.IsValid() {
		return "unknown", map[string]string{"type": fmt.Sprintf("%T", event)}
	}

	// 获取事件类型
	eventType := ""
	if eventTypeField.Kind() == reflect.String {
		eventType = eventTypeField.String()
	} else {
		eventType = "unknown"
	}

	// 获取 payload
	payload := payloadField.Interface()

	// 尝试识别 payload 类型并转换
	payloadType := reflect.TypeOf(payload)
	if payloadType == nil {
		return eventType, payload
	}

	// 检查是否是 Message 类型
	if payloadType.Name() == "Message" && payloadType.PkgPath() == "github.com/charmbracelet/crush/internal/message" {
		if msg, ok := payload.(message.Message); ok {
			return eventType, models.MessageToResponse(msg)
		}
	}

	// 检查是否是 Session 类型
	if payloadType.Name() == "Session" && payloadType.PkgPath() == "github.com/charmbracelet/crush/internal/session" {
		if sess, ok := payload.(session.Session); ok {
			return eventType, models.SessionToResponse(sess)
		}
	}

	// 其他类型，直接返回 payload
	return eventType, payload
}

// createEventChannelForProject 为指定项目的 app 实例创建事件通道
func (h *Handlers) createEventChannelForProject(ctx context.Context, appInstance *app.App) <-chan tea.Msg {
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
		lspCh := app.SubscribeLSPEvents(ctx)
		
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

// extractProjectPathFromEvents 从 SSE URL 中提取项目路径
// URL 格式: /api/v1/projects/{project_path}/events
func extractProjectPathFromEvents(r *http.Request) (string, error) {
	path := r.URL.Path
	prefix := "/api/v1/projects/"
	if !strings.HasPrefix(path, prefix) {
		return "", fmt.Errorf("invalid path format")
	}

	// 移除前缀
	rest := path[len(prefix):]

	// 查找 /events 的位置
	eventsIndex := strings.Index(rest, "/events")
	if eventsIndex == -1 {
		return "", fmt.Errorf("missing /events in path")
	}

	// 提取项目路径
	projectPath := rest[:eventsIndex]
	if projectPath == "" {
		return "", fmt.Errorf("project path is empty")
	}

	// URL 解码
	decoded, err := url.PathUnescape(projectPath)
	if err != nil {
		return "", fmt.Errorf("failed to decode project path: %w", err)
	}

	return decoded, nil
}