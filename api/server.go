package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/crush/api/handlers"
	"github.com/charmbracelet/crush/api/middleware"
	"github.com/charmbracelet/crush/internal/app"
)

// WriteError 写入错误响应
func WriteError(w http.ResponseWriter, code string, message string, statusCode int) {
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error": map[string]interface{}{
			"code":    code,
			"message": message,
		},
	})
}

// WriteJSON 写入 JSON 响应
func WriteJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		slog.Error("Failed to encode JSON response", "error", err)
	}
}

// Server 表示 API 服务器
type Server struct {
	app    *app.App
	server *http.Server
}

// NewServer 创建新的 API 服务器实例
func NewServer(app *app.App, host string, port int) *Server {
	mux := http.NewServeMux()

	// 创建处理器
	h := handlers.New(app)

	// API 路由处理器
	apiHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		slog.Info("API request received", "method", r.Method, "path", path, "raw_path", r.URL.RawPath)

		// Projects API
		if path == "/api/v1/projects" {
			slog.Info("Matched /api/v1/projects", "method", r.Method)
			if r.Method == "GET" {
				slog.Info("Calling HandleListProjects")
				h.HandleListProjects(w, r)
				return
			}
			if r.Method == "POST" {
				slog.Info("Calling HandleCreateProject")
				h.HandleCreateProject(w, r)
				return
			}
			slog.Warn("Method not allowed for /api/v1/projects", "method", r.Method)
		} else {
			slog.Debug("Path does not match /api/v1/projects", "path", path, "expected", "/api/v1/projects")
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

		// Project sessions API
		if strings.HasPrefix(path, "/api/v1/projects/") && strings.HasSuffix(path, "/sessions") {
			// /api/v1/projects/{project_path}/sessions
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
		if strings.HasPrefix(path, "/api/v1/projects/") && strings.Contains(path, "/sessions/") && !strings.Contains(path, "/messages") && !strings.HasSuffix(path, "/sessions") {
			// /api/v1/projects/{project_path}/sessions/{session_id}
			if r.Method == "GET" {
				h.HandleGetSession(w, r)
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

		WriteError(w, "NOT_FOUND", "Endpoint not found", http.StatusNotFound)
	})

	// 注册 API 路由
	// 使用前缀匹配，ServeMux 会匹配所有以该模式开头的路径
	mux.Handle("/api/v1/", apiHandler)
	
	// 添加一个测试路由来验证路由是否工作
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		slog.Info("Health check request received", "path", r.URL.Path)
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
		// 立即输出，不等待
		fmt.Fprintf(os.Stderr, "\n=== 请求到达 ===\n")
		fmt.Fprintf(os.Stderr, "Method: %s\n", r.Method)
		fmt.Fprintf(os.Stderr, "Path: %s\n", r.URL.Path)
		fmt.Fprintf(os.Stderr, "RawPath: %s\n", r.URL.RawPath)
		fmt.Fprintf(os.Stderr, "RequestURI: %s\n", r.RequestURI)
		fmt.Fprintf(os.Stderr, "Host: %s\n", r.Host)
		fmt.Fprintf(os.Stderr, "RemoteAddr: %s\n", r.RemoteAddr)
		fmt.Fprintf(os.Stderr, "================\n\n")
		
		slog.Info("=== 所有请求捕获 ===",
			"method", r.Method,
			"path", r.URL.Path,
			"raw_path", r.URL.RawPath,
			"host", r.Host,
			"remote_addr", r.RemoteAddr,
			"url", r.URL.String(),
			"request_uri", r.RequestURI,
		)
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
		app:    app,
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
