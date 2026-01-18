package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sync"

	"github.com/charmbracelet/crush/internal/app"
	"github.com/charmbracelet/crush/internal/config"
	"github.com/charmbracelet/crush/internal/db"
	"github.com/charmbracelet/crush/internal/projects"
)

// AppManager 管理不同项目的 app 实例
type AppManager struct {
	apps map[string]*app.App
	mu   sync.RWMutex
}

var globalAppManager = &AppManager{
	apps: make(map[string]*app.App),
}

// Open 创建并启动项目的 app 实例（幂等性：如果已打开则直接返回成功）
func (h *Handlers) Open(ctx context.Context, projectPath string) error {
	globalAppManager.mu.Lock()
	defer globalAppManager.mu.Unlock()

	// 检查是否已经打开（幂等性：如果已打开则直接返回成功）
	if _, ok := globalAppManager.apps[projectPath]; ok {
		slog.Info("Project app instance already open, returning success", "project", projectPath)
		return nil
	}

	// 获取项目信息
	projectList, err := projects.List()
	if err != nil {
		return fmt.Errorf("failed to list projects: %w", err)
	}

	var project *projects.Project
	for _, p := range projectList {
		if p.Path == projectPath {
			project = &p
			break
		}
	}

	if project == nil {
		return fmt.Errorf("project not found: %s", projectPath)
	}

	// 加载配置
	cfg, err := config.Load(project.Path, project.DataDir, false)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// 确保数据目录存在
	if err := os.MkdirAll(project.DataDir, 0o755); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}

	// 连接数据库
	conn, err := db.Connect(ctx, project.DataDir)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	// 创建 app 实例
	appInstance, err := app.New(ctx, conn, cfg)
	if err != nil {
		conn.Close()
		return fmt.Errorf("failed to create app instance: %w", err)
	}

	globalAppManager.apps[projectPath] = appInstance
	slog.Info("Opened project app instance", "project", projectPath)

	return nil
}

// Close 关闭并清理项目的 app 实例（幂等性：如果已关闭则直接返回成功）
func (h *Handlers) Close(ctx context.Context, projectPath string) error {
	globalAppManager.mu.Lock()
	defer globalAppManager.mu.Unlock()

	appInstance, ok := globalAppManager.apps[projectPath]
	if !ok {
		// 幂等性：如果已经关闭，直接返回成功
		slog.Info("Project app instance already closed, returning success", "project", projectPath)
		return nil
	}

	// 关闭 app 实例
	appInstance.Shutdown()
	delete(globalAppManager.apps, projectPath)
	slog.Info("Closed project app instance", "project", projectPath)

	return nil
}

// GetAppForProject 获取已打开的 app 实例（如果未打开则返回错误）
func (h *Handlers) GetAppForProject(ctx context.Context, projectPath string) (*app.App, error) {
	globalAppManager.mu.RLock()
	defer globalAppManager.mu.RUnlock()

	appInstance, ok := globalAppManager.apps[projectPath]
	if !ok {
		return nil, fmt.Errorf("app instance not open for project: %s (call open first)", projectPath)
	}

	return appInstance, nil
}

// GetAppForSession 通过会话ID查找对应的app实例
func (h *Handlers) GetAppForSession(ctx context.Context, sessionID string) (*app.App, error) {
	globalAppManager.mu.RLock()
	defer globalAppManager.mu.RUnlock()

	// 遍历所有app实例，查找包含该会话的实例
	for _, appInstance := range globalAppManager.apps {
		_, err := appInstance.Sessions.Get(ctx, sessionID)
		if err == nil {
			// 找到了会话
			return appInstance, nil
		}
	}

	return nil, fmt.Errorf("session not found in any project")
}

// Cleanup 清理所有 app 实例
func (h *Handlers) Cleanup() {
	globalAppManager.mu.Lock()
	defer globalAppManager.mu.Unlock()

	for path, app := range globalAppManager.apps {
		app.Shutdown()
		slog.Info("Shutdown app instance", "path", path)
	}
	globalAppManager.apps = make(map[string]*app.App)
}
