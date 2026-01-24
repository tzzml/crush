package handlers

import (
	"context"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/crush/api/models"
	"github.com/charmbracelet/crush/internal/projects"
	hertzapp "github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
)

// HandleGetCurrentProject 处理获取当前项目的请求
//
//	@Summary		获取当前项目
//	@Description	获取当前活跃的项目。提供 directory 参数时返回该目录的项目，否则返回最近访问的项目
//	@Tags			Project
//	@Accept			json
//	@Produce		json
//	@Param			directory	query		string	false	"项目路径（可选，不传则返回最近访问的项目）"
//	@Success		200			{object}	models.CurrentProjectResponse
//	@Failure		404			{object}	map[string]interface{}	"没有可用的项目"
//	@Failure		500			{object}	map[string]interface{}	"服务器内部错误"
//	@Router			/project/current [get]
func (h *Handlers) HandleGetCurrentProject(c context.Context, ctx *hertzapp.RequestContext) {
	directory := string(ctx.Query("directory"))

	// 如果提供了 directory 参数，获取该目录的项目
	if directory != "" {
		appInstance, err := h.GetAppForProject(c, directory)
		if err != nil {
			if strings.Contains(err.Error(), "project not found") {
				WriteError(c, ctx, "PROJECT_NOT_FOUND", err.Error(), consts.StatusNotFound)
				return
			}
			WriteError(c, ctx, "INTERNAL_ERROR", "Failed to get app for project: "+err.Error(), consts.StatusInternalServerError)
			return
		}

		cfg := appInstance.Config()
		project := projects.Project{
			Path:         directory,
			DataDir:      filepath.Join(cfg.WorkingDir(), ".crush"),
			LastAccessed: time.Now(),
		}

		pr := models.ProjectToResponse(project)
		response := models.CurrentProjectResponse{
			Project: &pr,
		}

		WriteJSON(c, ctx, consts.StatusOK, response)
		return
	}

	// 如果没有提供 directory 参数，返回最近访问的项目（原有逻辑）
	projectList, err := projects.List()
	if err != nil {
		WriteError(c, ctx, "INTERNAL_ERROR", "Failed to list projects: "+err.Error(), consts.StatusInternalServerError)
		return
	}

	// 如果没有项目，返回 404
	if len(projectList) == 0 {
		WriteError(c, ctx, "NO_PROJECTS", "No projects available. Please register a project first.", consts.StatusNotFound)
		return
	}

	// 第一个项目就是当前项目（最近访问的）
	currentProject := projectList[0]
	pr := models.ProjectToResponse(currentProject)

	response := models.CurrentProjectResponse{
		Project: &pr,
	}

	WriteJSON(c, ctx, consts.StatusOK, response)
}
