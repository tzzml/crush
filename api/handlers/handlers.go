package handlers

import (
	"github.com/charmbracelet/crush/internal/app"
)

// Handlers 包含所有 API 处理器
type Handlers struct {
	app *app.App
}

// New 创建新的处理器实例
func New(app *app.App) *Handlers {
	return &Handlers{
		app: app,
	}
}
