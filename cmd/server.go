package cmd

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/charmbracelet/crush/api"
	"github.com/charmbracelet/crush/internal/config"
	"github.com/charmbracelet/crush/internal/projects"
	"github.com/spf13/cobra"
)

// @title           Zork Agent API
// @version         1.0
// @description     AI 项目管理 API 服务
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.url    http://www.swagger.io/support
// @contact.email  support@swagger.io

// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html

// @host      localhost:8080
// @BasePath  /

// @schemes http https

// multiHandler 实现 slog.Handler 接口，同时将日志写入多个 handler
type multiHandler struct {
	handlers []slog.Handler
}

func (m *multiHandler) Enabled(ctx context.Context, level slog.Level) bool {
	// 只要有一个 handler 启用就返回 true
	for _, h := range m.handlers {
		if h.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

func (m *multiHandler) Handle(ctx context.Context, record slog.Record) error {
	// 将记录写入所有 handler
	for _, h := range m.handlers {
		if h.Enabled(ctx, record.Level) {
			if err := h.Handle(ctx, record); err != nil {
				return err
			}
		}
	}
	return nil
}

func (m *multiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	handlers := make([]slog.Handler, len(m.handlers))
	for i, h := range m.handlers {
		handlers[i] = h.WithAttrs(attrs)
	}
	return &multiHandler{handlers: handlers}
}

func (m *multiHandler) WithGroup(name string) slog.Handler {
	handlers := make([]slog.Handler, len(m.handlers))
	for i, h := range m.handlers {
		handlers[i] = h.WithGroup(name)
	}
	return &multiHandler{handlers: handlers}
}

// simpleHandler 实现简洁的日志格式，类似 nginx 日志
type simpleHandler struct {
	mu     sync.Mutex
	writer io.Writer
	level  slog.Level
}

func newSimpleHandler(w io.Writer, level slog.Level) *simpleHandler {
	return &simpleHandler{
		writer: w,
		level:  level,
	}
}

func (h *simpleHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return level >= h.level
}

func (h *simpleHandler) Handle(ctx context.Context, record slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	// 格式化时间 [2026-01-18 23:24:13]
	timeStr := record.Time.Format("[2006-01-02 15:04:05]")

	// 格式化级别
	levelStr := record.Level.String()
	switch record.Level {
	case slog.LevelDebug:
		levelStr = "DEBUG"
	case slog.LevelInfo:
		levelStr = "INFO"
	case slog.LevelWarn:
		levelStr = "WARN"
	case slog.LevelError:
		levelStr = "ERROR"
	}

	// 构建日志行
	var parts []string
	parts = append(parts, timeStr, levelStr, record.Message)

	// 添加其他字段
	record.Attrs(func(attr slog.Attr) bool {
		if attr.Key != "" {
			parts = append(parts, fmt.Sprintf("%s=%v", attr.Key, attr.Value.Any()))
		}
		return true
	})

	// 输出日志行
	_, err := fmt.Fprintln(h.writer, strings.Join(parts, " "))
	return err
}

func (h *simpleHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	// 简化实现：不处理属性继承
	return h
}

func (h *simpleHandler) WithGroup(name string) slog.Handler {
	// 简化实现：不处理分组
	return h
}

// StartServer 启动 API 服务器
func StartServer(cmd *cobra.Command, port int, host string) {
	debug, _ := cmd.Flags().GetBool("debug")
	dataDir, _ := cmd.Flags().GetString("data-dir")

	cwd, err := ResolveCwd(cmd)
	if err != nil {
		slog.Error("Failed to resolve working directory", "error", err)
		os.Exit(1)
	}

	// 初始化配置
	cfg, err := config.Init(cwd, dataDir, debug)
	if err != nil {
		slog.Error("Failed to initialize config", "error", err)
		os.Exit(1)
	}

	// 为服务器模式添加控制台日志输出
	// 创建一个简洁格式的控制台日志处理器
	consoleHandler := newSimpleHandler(os.Stderr, slog.LevelInfo)
	fileHandler := slog.Default().Handler()

	// 创建多路复用 handler
	multi := &multiHandler{handlers: []slog.Handler{consoleHandler, fileHandler}}
	slog.SetDefault(slog.New(multi))

	// 创建数据目录
	if err := createDotZorkAgentDir(cfg.Options.DataDirectory); err != nil {
		slog.Error("Failed to create data directory", "error", err)
		os.Exit(1)
	}

	// 注册项目
	if err := projects.Register(cwd, cfg.Options.DataDirectory); err != nil {
		slog.Warn("Failed to register project", "error", err)
	}

	// 创建 API 服务器（不再需要默认 app 实例）
	server := api.NewServer(host, port)

	// 设置信号处理
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// 在 goroutine 中启动服务器
	go func() {
		slog.Info("=== 准备启动 API 服务器 ===", "host", host, "port", port)
		slog.Info("Server instance created", "server", fmt.Sprintf("%+v", server))
		if err := server.Start(); err != nil && err != http.ErrServerClosed {
			slog.Error("API server error", "error", err)
			os.Exit(1)
		}
		slog.Info("API server goroutine exited")
	}()

	// 等待一小段时间确保服务器启动
	time.Sleep(100 * time.Millisecond)
	slog.Info("服务器应该已经启动，等待请求...")

	// 等待中断信号
	<-sigChan
	slog.Info("Shutting down API server...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		// Hertz Shutdown may return an error even on graceful shutdown
		// Only log if it's not a context error or http.ErrServerClosed
		if err != nil && err != context.DeadlineExceeded && err != context.Canceled {
			slog.Error("Error shutting down server", "error", err)
		}
	}
	slog.Info("API server shutdown complete")
}
