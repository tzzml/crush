package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sync"

	internalapp "github.com/charmbracelet/crush/internal/app"
	"github.com/charmbracelet/crush/internal/config"
	"github.com/charmbracelet/crush/internal/db"
	"github.com/charmbracelet/crush/internal/projects"
)

// AppManager 管理不同项目的 app 实例
type AppManager struct {
	apps map[string]*internalapp.App
	mu   sync.RWMutex
}

var globalAppManager = &AppManager{
	apps: make(map[string]*internalapp.App),
}

// createAppInstance 创建 app 实例的辅助方法
func (am *AppManager) createAppInstance(ctx context.Context, projectPath string) (*internalapp.App, error) {
	// 获取项目信息
	projectList, err := projects.List()
	if err != nil {
		return nil, fmt.Errorf("failed to list projects: %w", err)
	}

	var project *projects.Project
	for _, p := range projectList {
		if p.Path == projectPath {
			project = &p
			break
		}
	}

	if project == nil {
		return nil, fmt.Errorf("project not found: %s", projectPath)
	}

	// 加载配置
	cfg, err := config.Load(project.Path, project.DataDir, false)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// 确保数据目录存在
	if err := os.MkdirAll(project.DataDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	// 连接数据库
	conn, err := db.Connect(ctx, project.DataDir)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// 创建 app 实例
	appInstance, err := internalapp.New(ctx, conn, cfg)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to create app instance: %w", err)
	}

	return appInstance, nil
}

// DisposeProject 释放单个项目的 app 实例（幂等性：如果已释放则直接返回成功）
func (h *Handlers) DisposeProject(ctx context.Context, projectPath string) error {
	globalAppManager.mu.Lock()
	defer globalAppManager.mu.Unlock()

	appInstance, ok := globalAppManager.apps[projectPath]
	if !ok {
		// 幂等性：如果已经释放，直接返回成功
		slog.Info("Project app instance already disposed", "project", projectPath)
		return nil
	}

	// 关闭 app 实例
	appInstance.Shutdown()
	delete(globalAppManager.apps, projectPath)
	slog.Info("Disposed project app instance", "project", projectPath)

	return nil
}

// GetAppForProject 获取项目的 app 实例（如果不存在则自动创建）
func (h *Handlers) GetAppForProject(ctx context.Context, projectPath string) (*internalapp.App, error) {
	// Fast path: read lock to check if already exists
	globalAppManager.mu.RLock()
	appInstance, ok := globalAppManager.apps[projectPath]
	globalAppManager.mu.RUnlock()

	if ok {
		return appInstance, nil
	}

	// Slow path: acquire write lock and create
	globalAppManager.mu.Lock()
	defer globalAppManager.mu.Unlock()

	// Double-check pattern: another goroutine might have created it while we waited
	if appInstance, ok := globalAppManager.apps[projectPath]; ok {
		return appInstance, nil
	}

	// Auto-create the instance
	appInstance, err := globalAppManager.createAppInstance(ctx, projectPath)
	if err != nil {
		return nil, err
	}

	globalAppManager.apps[projectPath] = appInstance
	slog.Info("Auto-created project app instance", "project", projectPath)

	return appInstance, nil
}

// GetAppForSession 通过会话ID查找对应的app实例
func (h *Handlers) GetAppForSession(ctx context.Context, sessionID string) (*internalapp.App, error) {
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

// DisposeAll 释放所有项目的 app 实例
func (h *Handlers) DisposeAll(ctx context.Context) ([]string, error) {
	globalAppManager.mu.Lock()
	defer globalAppManager.mu.Unlock()

	disposedProjects := make([]string, 0, len(globalAppManager.apps))

	for path, appInstance := range globalAppManager.apps {
		appInstance.Shutdown()
		disposedProjects = append(disposedProjects, path)
		slog.Info("Shutdown app instance", "path", path)
	}

	globalAppManager.apps = make(map[string]*internalapp.App)

	return disposedProjects, nil
}
