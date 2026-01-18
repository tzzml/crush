package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/charmbracelet/crush/api"
	"github.com/charmbracelet/crush/internal/app"
	"github.com/charmbracelet/crush/internal/config"
	"github.com/charmbracelet/crush/internal/db"
	"github.com/charmbracelet/crush/internal/projects"
)

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

// StartServer 启动 API 服务器
func StartServer(port int, host string) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 解析命令行参数（类似 internal/cmd 的方式）
	cwd, err := resolveCwd()
	if err != nil {
		slog.Error("Failed to resolve working directory", "error", err)
		os.Exit(1)
	}

	dataDir := getDataDir()
	debug := getDebugFlag()

	// 初始化配置
	cfg, err := config.Init(cwd, dataDir, debug)
	if err != nil {
		slog.Error("Failed to initialize config", "error", err)
		os.Exit(1)
	}

	// 为服务器模式添加控制台日志输出
	// 创建一个同时输出到控制台和文件的日志处理器
	consoleHandler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level:     slog.LevelInfo,
		AddSource: false,
	})
	fileHandler := slog.Default().Handler()
	
	// 创建多路复用 handler
	multi := &multiHandler{handlers: []slog.Handler{consoleHandler, fileHandler}}
	slog.SetDefault(slog.New(multi))

	// 创建数据目录
	if err := createDotCrushDir(cfg.Options.DataDirectory); err != nil {
		slog.Error("Failed to create data directory", "error", err)
		os.Exit(1)
	}

	// 注册项目
	if err := projects.Register(cwd, cfg.Options.DataDirectory); err != nil {
		slog.Warn("Failed to register project", "error", err)
	}

	// 连接数据库
	conn, err := db.Connect(ctx, cfg.Options.DataDirectory)
	if err != nil {
		slog.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer conn.Close()

	// 创建 app 实例
	appInstance, err := app.New(ctx, conn, cfg)
	if err != nil {
		slog.Error("Failed to create app instance", "error", err)
		os.Exit(1)
	}
	defer appInstance.Shutdown()

	// 创建 API 服务器
	server := api.NewServer(appInstance, host, port)

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
		slog.Error("Error shutting down server", "error", err)
	}
}

// resolveCwd 解析工作目录
func resolveCwd() (string, error) {
	// 检查 --cwd 或 -c 参数
	for i, arg := range os.Args {
		if (arg == "--cwd" || arg == "-c") && i+1 < len(os.Args) {
			cwd := os.Args[i+1]
			if err := os.Chdir(cwd); err != nil {
				return "", fmt.Errorf("failed to change directory: %v", err)
			}
			return cwd, nil
		}
	}
	// 默认使用当前工作目录
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current working directory: %v", err)
	}
	return cwd, nil
}

// getDataDir 获取数据目录
func getDataDir() string {
	for i, arg := range os.Args {
		if (arg == "--data-dir" || arg == "-D") && i+1 < len(os.Args) {
			return os.Args[i+1]
		}
	}
	return ""
}

// getDebugFlag 获取调试标志
func getDebugFlag() bool {
	for _, arg := range os.Args {
		if arg == "--debug" || arg == "-d" {
			return true
		}
	}
	return false
}

// createDotCrushDir 创建 .crush 目录
func createDotCrushDir(dir string) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("failed to create data directory: %q %w", dir, err)
	}

	gitIgnorePath := filepath.Join(dir, ".gitignore")
	if _, err := os.Stat(gitIgnorePath); os.IsNotExist(err) {
		if err := os.WriteFile(gitIgnorePath, []byte("*\n"), 0o644); err != nil {
			return fmt.Errorf("failed to create .gitignore file: %q %w", gitIgnorePath, err)
		}
	}

	return nil
}
