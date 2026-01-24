package handlers

import (
	"context"
	"path/filepath"

	"github.com/charmbracelet/crush/api/models"
	"github.com/charmbracelet/crush/internal/projects"
	hertzapp "github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
)

// HandleListProjects 处理获取所有项目的请求
//
//	@Summary		获取项目列表
//	@Description	获取所有已注册的项目
//	@Tags			Project
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	models.ProjectsResponse
//	@Failure		500	{object}	map[string]interface{}
//	@Router			/project [get]
func (h *Handlers) HandleListProjects(c context.Context, ctx *hertzapp.RequestContext) {
	projectList, err := projects.List()
	if err != nil {
		WriteError(c, ctx, "INTERNAL_ERROR", "Failed to list projects: "+err.Error(), consts.StatusInternalServerError)
		return
	}

	response := models.ProjectsResponse{
		Projects: make([]models.ProjectResponse, len(projectList)),
	}
	for i, p := range projectList {
		response.Projects[i] = models.ProjectToResponse(p)
	}

	WriteJSON(c, ctx, consts.StatusOK, response)
}

// HandleCreateProject 处理创建项目的请求
//
//	@Summary		创建项目
//	@Description	注册一个新项目
//	@Tags			Project
//	@Accept			json
//	@Produce		json
//	@Param			request	body		models.CreateProjectRequest	true	"创建项目请求"
//	@Success		201		{object}	models.CreateProjectResponse
//	@Failure		400		{object}	map[string]interface{}
//	@Failure		500		{object}	map[string]interface{}
//	@Router			/project [post]
func (h *Handlers) HandleCreateProject(c context.Context, ctx *hertzapp.RequestContext) {
	var req models.CreateProjectRequest
	if err := ctx.BindJSON(&req); err != nil {
		WriteError(c, ctx, "INVALID_REQUEST", "Invalid request body: "+err.Error(), consts.StatusBadRequest)
		return
	}

	if req.Path == "" {
		WriteError(c, ctx, "INVALID_REQUEST", "Project path is required", consts.StatusBadRequest)
		return
	}

	// 如果 data_dir 未提供，使用默认逻辑
	dataDir := req.DataDir
	if dataDir == "" {
		// 使用默认的数据目录：项目路径/.crush
		dataDir = filepath.Join(req.Path, ".crush")
	}

	// 注册项目
	if err := projects.Register(req.Path, dataDir); err != nil {
		WriteError(c, ctx, "INTERNAL_ERROR", "Failed to register project: "+err.Error(), consts.StatusInternalServerError)
		return
	}

	// 获取注册后的项目信息
	projectList, err := projects.List()
	if err != nil {
		WriteError(c, ctx, "INTERNAL_ERROR", "Failed to get project: "+err.Error(), consts.StatusInternalServerError)
		return
	}

	var project *projects.Project
	for _, p := range projectList {
		if p.Path == req.Path {
			project = &p
			break
		}
	}

	if project == nil {
		WriteError(c, ctx, "INTERNAL_ERROR", "Project registered but not found", consts.StatusInternalServerError)
		return
	}

	response := models.CreateProjectResponse{
		Project: models.ProjectToResponse(*project),
	}

	WriteJSON(c, ctx, consts.StatusCreated, response)
}
