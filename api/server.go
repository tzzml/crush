package api

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/charmbracelet/crush/api/handlers"
	"github.com/charmbracelet/crush/api/middleware"
	hertzserver "github.com/cloudwego/hertz/pkg/app/server"
	hertzapp "github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
)

// Server 表示 Hertz API 服务器
type Server struct {
	*hertzserver.Hertz
	handlers *handlers.Handlers
}

// NewServer 创建新的 Hertz API 服务器实例
func NewServer(host string, port int) *Server {
	addr := fmt.Sprintf("%s:%d", host, port)

	// 创建 Hertz 服务器
	h := hertzserver.New(
		hertzserver.WithHostPorts(addr),
		hertzserver.WithReadTimeout(30*time.Second),
		hertzserver.WithWriteTimeout(30*time.Second),
		hertzserver.WithIdleTimeout(120*time.Second),
	)

	// 创建 handlers
	handlersInstance := handlers.New()

	slog.Info("Hertz server created", "addr", addr)

	return &Server{
		Hertz:    h,
		handlers: handlersInstance,
	}
}

// Start 启动 Hertz 服务器并注册路由
func (s *Server) Start() error {
	// 全局中间件
	s.Use(
		middleware.LoggingMiddleware(),
		middleware.CORSMiddleware(),
		middleware.JSONMiddleware(),
	)

	// Swagger 路由
	s.GET("/", handlers.HandleIndexRedirect)
	s.GET("/swagger", handlers.HandleSwaggerUI)
	s.GET("/swagger/doc.json", handlers.HandleSwaggerJSON)
	s.GET("/swagger/openapi3.json", handlers.HandleOpenAPI3JSON) // OpenAPI 3.0
	s.GET("/redoc", handlers.HandleRedoc) // Redoc UI with OpenAPI 3.0

	// API 路由
	{
		// 路径信息
		s.GET("/path", s.handlers.HandleGetPath)

		// 项目管理
		s.GET("/project", s.handlers.HandleListProjects)
		s.GET("/project/current", s.handlers.HandleGetCurrentProject)
		s.POST("/project", s.handlers.HandleCreateProject)

		// 实例生命周期 - 释放单个项目实例
		s.POST("/instance/dispose", s.handlers.HandleDisposeProject)
		s.GET("/project/config", s.handlers.HandleGetConfig)

		// 系统提示词管理
		s.GET("/system-prompt", s.handlers.HandleGetSystemPrompt)
		s.PUT("/system-prompt", s.handlers.HandleUpdateSystemPrompt)

		s.GET("/project/permissions", s.handlers.HandleListPermissions)
		s.POST("/project/permissions/:requestID/reply", s.handlers.HandleReplyPermission)

		// 全局操作
		s.GET("/global/health", func(c context.Context, ctx *hertzapp.RequestContext) {
			ctx.JSON(consts.StatusOK, map[string]interface{}{
				"healthy": true,
				"version": "1.0.0",
			})
		})
		s.POST("/global/dispose", s.handlers.HandleDisposeAll)

		// 文件系统操作
		s.GET("/find", s.handlers.HandleSearchContent)          // 搜索文本内容
		s.GET("/find/file", s.handlers.HandleSearchFile)        // 搜索文件名
		s.GET("/file", s.handlers.HandleListFiles)              // 列出目录内容
		s.GET("/file/content", s.handlers.HandleGetFileContent) // 读取文件内容
		s.GET("/file/status", s.handlers.HandleGetGitStatus)    // 获取 Git 状态

		// LSP 和 MCP 状态
		s.GET("/lsp", s.handlers.HandleGetLSPStatus)            // 获取 LSP 状态
		s.GET("/mcp", s.handlers.HandleGetMCPStatus)            // 获取 MCP 状态

		// 会话管理 - 使用查询参数指定项目
		s.GET("/session", s.handlers.HandleListSessions)
		s.POST("/session", s.handlers.HandleCreateSession)
		s.GET("/session/:id", s.handlers.HandleGetSession)
		s.PUT("/session/:id", s.handlers.HandleUpdateSession)
		s.DELETE("/session/:id", s.handlers.HandleDeleteSession)
		s.POST("/session/:id/abort", s.handlers.HandleAbortSession)
		s.GET("/session/status", s.handlers.HandleGetSessionStatus)

		// 消息管理 - 使用查询参数指定项目
		s.GET("/session/:sessionID/message", s.handlers.HandleListMessages)
		s.POST("/session/:sessionID/prompt", s.handlers.HandlePrompt)
		s.GET("/message/:id", s.handlers.HandleGetMessage)

		// SSE 事件流 - 需要单独处理，跳过 JSON 中间件
		s.GET("/event", func(c context.Context, ctx *hertzapp.RequestContext) {
			// 临时清除 Content-Type，让 HandleSSE 自己设置
			ctx.Response.Header.Del("Content-Type")
			s.handlers.HandleSSE(c, ctx)
		})

		// 健康检查 (兼容旧路径)
		s.GET("/health", func(c context.Context, ctx *hertzapp.RequestContext) {
			ctx.JSON(consts.StatusOK, map[string]string{"status": "ok"})
		})
	}

	slog.Info("=== Hertz 服务器启动 ===")
	return s.Run()
}

// Shutdown 优雅关闭服务器
func (s *Server) Shutdown(ctx context.Context) error {
	slog.Info("Shutting down Hertz server")
	return s.Hertz.Shutdown(ctx)
}
