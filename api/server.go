package api

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/charmbracelet/crush/api/handlers"
	"github.com/charmbracelet/crush/api/middleware"
)

// WriteError 写入错误响应 (委托给 handlers 包)
func WriteError(w http.ResponseWriter, code string, message string, statusCode int) {
	handlers.WriteError(w, code, message, statusCode)
}

// WriteJSON 写入 JSON 响应 (委托给 handlers 包)
func WriteJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	handlers.WriteJSON(w, statusCode, data)
}

// Server 表示 API 服务器
type Server struct {
	server *http.Server
}

// NewServer 创建新的 API 服务器实例
func NewServer(host string, port int) *Server {
	mux := http.NewServeMux()

	// 创建处理器（不再需要 app 实例）
	h := handlers.New()

	// API 路由处理器
	apiHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// Projects API
		if path == "/api/v1/projects" {
			if r.Method == "GET" {
				h.HandleListProjects(w, r)
				return
			}
			if r.Method == "POST" {
				h.HandleCreateProject(w, r)
				return
			}
			slog.Warn("Method not allowed for /api/v1/projects", "method", r.Method)
		}

		// Project Lifecycle API - open/close/connect
		if strings.HasPrefix(path, "/api/v1/projects/") && strings.HasSuffix(path, "/open") {
			// /api/v1/projects/{project_path}/open
			if r.Method == "POST" {
				h.HandleOpenProject(w, r)
				return
			}
		}
		if strings.HasPrefix(path, "/api/v1/projects/") && strings.HasSuffix(path, "/close") {
			// /api/v1/projects/{project_path}/close
			if r.Method == "POST" {
				h.HandleCloseProject(w, r)
				return
			}
		}

		// Sessions API - 需要解析 project_path
		if strings.HasPrefix(path, "/api/v1/projects/") && strings.HasSuffix(path, "/sessions") {
			if r.Method == "GET" {
				h.HandleListSessions(w, r)
				return
			}
			if r.Method == "POST" {
				h.HandleCreateSession(w, r)
				return
			}
		}

		// Project session detail API
		if strings.HasPrefix(path, "/api/v1/projects/") && strings.Contains(path, "/sessions/") && !strings.Contains(path, "/messages") && !strings.HasSuffix(path, "/sessions") && !strings.Contains(path, "/abort") {
			// /api/v1/projects/{project_path}/sessions/{session_id}
			if r.Method == "GET" {
				h.HandleGetSession(w, r)
				return
			}
			if r.Method == "PUT" || r.Method == "PATCH" {
				h.HandleUpdateSession(w, r)
				return
			}
			if r.Method == "DELETE" {
				h.HandleDeleteSession(w, r)
				return
			}
		}

		// Project messages API
		if strings.HasPrefix(path, "/api/v1/projects/") && strings.Contains(path, "/sessions/") && strings.HasSuffix(path, "/messages") {
			// /api/v1/projects/{project_path}/sessions/{session_id}/messages
			if r.Method == "GET" {
				h.HandleListMessages(w, r)
				return
			}
			if r.Method == "POST" {
				h.HandleCreateMessage(w, r)
				return
			}
		}

		// Project message detail API
		if strings.HasPrefix(path, "/api/v1/projects/") && strings.Contains(path, "/messages/") {
			// /api/v1/projects/{project_path}/messages/{message_id}
			if r.Method == "GET" {
				h.HandleGetMessage(w, r)
				return
			}
		}

		// Health check API
		if path == "/api/v1/health" {
			if r.Method == "GET" {
				WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
				return
			}
		}

		// Config API - /api/v1/projects/{project_path}/config
		if strings.HasPrefix(path, "/api/v1/projects/") && strings.HasSuffix(path, "/config") {
			if r.Method == "GET" {
				h.HandleGetConfig(w, r)
				return
			}
		}

		// Permissions API - /api/v1/projects/{project_path}/permissions
		if strings.HasPrefix(path, "/api/v1/projects/") && strings.HasSuffix(path, "/permissions") {
			if r.Method == "GET" {
				h.HandleListPermissions(w, r)
				return
			}
		}

		// Permission Reply API - /api/v1/projects/{project_path}/permissions/{requestID}/reply
		if strings.HasPrefix(path, "/api/v1/projects/") && strings.Contains(path, "/permissions/") && strings.HasSuffix(path, "/reply") {
			if r.Method == "POST" {
				h.HandleReplyPermission(w, r)
				return
			}
		}

		// Session Abort API - /api/v1/projects/{project_path}/sessions/{session_id}/abort
		if strings.HasPrefix(path, "/api/v1/projects/") && strings.Contains(path, "/sessions/") && strings.HasSuffix(path, "/abort") {
			if r.Method == "POST" {
				h.HandleAbortSession(w, r)
				return
			}
		}

		// Session Status API - /api/v1/projects/{project_path}/sessions/status
		if strings.HasPrefix(path, "/api/v1/projects/") && strings.HasSuffix(path, "/sessions/status") {
			if r.Method == "GET" {
				h.HandleGetSessionStatus(w, r)
				return
			}
		}

		WriteError(w, "NOT_FOUND", "Endpoint not found", http.StatusNotFound)
	})

	// 注册 API 路由
	// 使用前缀匹配，ServeMux 会匹配所有以该模式开头的路径
	mux.Handle("/api/v1/", apiHandler)
	
	// 添加一个测试路由来验证路由是否工作
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		// 设置正确的Content-Type，因为JSONMiddleware会覆盖这个
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("HEALTH_OK_FROM_HANDLER"))
	})
	
	slog.Info("API routes registered", "pattern", "/api/v1/")

	// 应用中间件
	handler := middleware.LoggingMiddleware(mux)
	handler = middleware.CORSMiddleware(handler)
	handler = middleware.JSONMiddleware(handler)
	
	// 添加一个根级别的调试 handler 来捕获所有请求
	debugHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// SSE 路由特殊处理，不经过中间件
		// /api/v1/projects/{project_path}/events
		if strings.HasPrefix(r.URL.Path, "/api/v1/projects/") && strings.HasSuffix(r.URL.Path, "/events") && r.Method == "GET" {
			h.HandleSSE(w, r)
			return
		}

		handler.ServeHTTP(w, r)
	})

	addr := fmt.Sprintf("%s:%d", host, port)
	httpServer := &http.Server{
		Addr:         addr,
		Handler:      debugHandler,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
	
	slog.Info("HTTP server configured", "addr", addr, "handler_type", fmt.Sprintf("%T", debugHandler))

	return &Server{
		server: httpServer,
	}
}

// Start 启动 API 服务器
func (s *Server) Start() error {
	slog.Info("=== 服务器启动 ===", "addr", s.server.Addr)
	slog.Info("Server handler type", "type", fmt.Sprintf("%T", s.server.Handler))
	slog.Info("Server starting to listen on", "addr", s.server.Addr)
	
	// 验证 handler 不为 nil
	if s.server.Handler == nil {
		slog.Error("Server handler is nil!")
		return fmt.Errorf("server handler is nil")
	}
	
	slog.Info("About to call ListenAndServe...")
	if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("Server listen error", "error", err)
		return err
	}
	slog.Info("Server stopped")
	return nil
}

// Shutdown 优雅关闭服务器
func (s *Server) Shutdown(ctx context.Context) error {
	slog.Info("Shutting down API server")
	return s.server.Shutdown(ctx)
}
