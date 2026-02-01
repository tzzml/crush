package handlers

import (
	"context"
	"os"
	"path/filepath"

	hertzapp "github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
)

// HandleGetPath 处理获取路径信息的请求 (参考 OpenCode: /path)
//
//	@Summary		获取路径信息
//	@Description	获取当前工作目录和相关路径信息
//	@Tags			Project
//	@Accept			json
//	@Produce		json
//	@Param			directory	query		string	true	"项目路径"
//	@Success		200		{object}	map[string]interface{}
//	@Failure		400	{object}	map[string]interface{}
//	@Failure		404	{object}	map[string]interface{}
//	@Router			/path [get]
func (h *Handlers) HandleGetPath(c context.Context, ctx *hertzapp.RequestContext) {
	directory := string(ctx.Query("directory"))
	if directory == "" {
		WriteError(c, ctx, "MISSING_DIRECTORY_PARAM", "Directory query parameter is required", consts.StatusBadRequest)
		return
	}

	// 获取项目的 app 实例
	appInstance, err := h.GetAppForProject(c, directory)
	if err != nil {
		if err.Error() == "project not found" {
			WriteError(c, ctx, "PROJECT_NOT_FOUND", err.Error(), consts.StatusNotFound)
			return
		}
		WriteError(c, ctx, "INTERNAL_ERROR", "Failed to get app for project: "+err.Error(), consts.StatusInternalServerError)
		return
	}

	cfg := appInstance.Config()
	if cfg == nil {
		WriteError(c, ctx, "CONFIG_NOT_FOUND", "Configuration not available", consts.StatusNotFound)
		return
	}

	// 尝试获取 home 目录
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = ""
	}

	// 构建路径响应
	response := map[string]interface{}{
		"home":      homeDir,                                            // 用户主目录
		"state":     filepath.Join(cfg.WorkingDir(), ".crush", "state"), // 状态目录
		"config":    filepath.Join(cfg.WorkingDir(), ".crush"),          // 配置目录
		"worktree":  cfg.WorkingDir(),                                   // 工作树/项目目录
		"directory": cfg.WorkingDir(),                                   // 当前目录
	}

	WriteJSON(c, ctx, consts.StatusOK, response)
}
