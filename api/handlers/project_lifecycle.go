package handlers

import (
	"context"
	"log/slog"

	"github.com/charmbracelet/crush/api/models"
	"github.com/charmbracelet/crush/internal/projects"
	hertzapp "github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
)

// HandleDisposeProject 处理释放项目 app 实例的请求
//
//	@Summary		释放实例
//	@Description	释放指定项目的 app 实例以释放资源
//	@Tags			Project
//	@Accept			json
//	@Produce		json
//	@Param			directory	query		string	true	"项目路径"
//	@Success		200		{object}	models.DisposeProjectResponse
//	@Failure		400	{object}	map[string]interface{}
//	@Failure		404	{object}	map[string]interface{}
//	@Failure		500	{object}	map[string]interface{}
//	@Router			/instance/dispose [post]
func (h *Handlers) HandleDisposeProject(c context.Context, ctx *hertzapp.RequestContext) {
	projectPath := string(ctx.Query("directory"))
	if projectPath == "" {
		WriteError(c, ctx, "MISSING_DIRECTORY_PARAM", "Directory query parameter is required", consts.StatusBadRequest)
		return
	}

	// 验证项目是否已注册
	projectList, err := projects.List()
	if err != nil {
		WriteError(c, ctx, "INTERNAL_ERROR", "Failed to list projects: "+err.Error(), consts.StatusInternalServerError)
		return
	}

	projectExists := false
	for _, p := range projectList {
		if p.Path == projectPath {
			projectExists = true
			break
		}
	}

	if !projectExists {
		WriteError(c, ctx, "PROJECT_NOT_FOUND", "Project not found: "+projectPath, consts.StatusNotFound)
		return
	}

	err = h.DisposeProject(c, projectPath)
	if err != nil {
		WriteError(c, ctx, "INTERNAL_ERROR", "Failed to dispose project: "+err.Error(), consts.StatusInternalServerError)
		return
	}

	response := models.DisposeProjectResponse{
		ProjectPath: projectPath,
		Status:      "disposed",
		Message:     "Project instance disposed successfully",
	}

	WriteJSON(c, ctx, consts.StatusOK, response)
}

// HandleDisposeAll 处理释放所有项目 app 实例的请求
//
//	@Summary		释放所有项目
//	@Description	释放所有项目的 app 实例以释放资源
//	@Tags			Global
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	models.DisposeAllResponse
//	@Failure		500	{object}	map[string]interface{}
//	@Router			/global/dispose [post]
func (h *Handlers) HandleDisposeAll(c context.Context, ctx *hertzapp.RequestContext) {
	disposedProjects, err := h.DisposeAll(c)
	if err != nil {
		WriteError(c, ctx, "INTERNAL_ERROR", "Failed to dispose all projects: "+err.Error(), consts.StatusInternalServerError)
		return
	}

	slog.Info("Disposed all project app instances", "count", len(disposedProjects))

	response := models.DisposeAllResponse{
		DisposedCount: len(disposedProjects),
		Projects:      disposedProjects,
		Status:        "all_disposed",
	}

	WriteJSON(c, ctx, consts.StatusOK, response)
}
