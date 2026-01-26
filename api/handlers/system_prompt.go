package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"

	"github.com/charmbracelet/crush/api/models"
	hertzapp "github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
)

// HandleGetSystemPrompt 获取系统提示词
//
//	@Summary		获取系统提示词
//	@Description	获取指定项目当前的系统提示词内容
//	@Tags			Prompt
//	@Accept			json
//	@Produce		json
//	@Param			directory	query		string	true	"项目路径"
//	@Success		200		{object}	models.GetSystemPromptResponse
//	@Failure		400		{object}	map[string]interface{}
//	@Failure		404		{object}	map[string]interface{}
//	@Failure		500		{object}	map[string]interface{}
//	@Router			/system-prompt [get]
func (h *Handlers) HandleGetSystemPrompt(c context.Context, ctx *hertzapp.RequestContext) {
	projectPath := string(ctx.Query("directory"))
	if projectPath == "" {
		WriteError(c, ctx, "MISSING_DIRECTORY_PARAM", "Directory query parameter is required", consts.StatusBadRequest)
		return
	}

	// 获取项目的 app 实例
	appInstance, err := h.GetAppForProject(c, projectPath)
	if err != nil {
		if strings.Contains(err.Error(), "project not found") {
			WriteError(c, ctx, "PROJECT_NOT_FOUND", err.Error(), consts.StatusNotFound)
			return
		}
		slog.Error("Failed to get app instance", "project", projectPath, "error", err)
		WriteError(c, ctx, "INTERNAL_ERROR", "Failed to get app instance: "+err.Error(), consts.StatusInternalServerError)
		return
	}

	// 使用反射访问器获取当前的系统提示词
	accessor := newCoordinatorAccessor()
	systemPrompt, err := accessor.getSystemPrompt(appInstance.AgentCoordinator)
	if err != nil {
		slog.Error("Failed to access system prompt through reflection",
			"project", projectPath,
			"error", err)
		WriteError(c, ctx, "COORDINATOR_ACCESS_FAILED",
			"Failed to access system prompt: "+err.Error()+
				". This may indicate internal implementation has changed.",
			consts.StatusInternalServerError)
		return
	}

	// 返回成功响应
	WriteJSON(c, ctx, consts.StatusOK, models.GetSystemPromptResponse{
		SystemPrompt: systemPrompt,
		Length:       len(systemPrompt),
		IsCustom:     systemPrompt != "",
	})
}

// HandleUpdateSystemPrompt 更新系统提示词
//
//	@Summary		更新系统提示词
//	@Description	动态修改指定项目的系统提示词，无需重启服务。更新后立即对后续对话生效。
//	@Tags			Prompt
//	@Accept			json
//	@Produce		json
//	@Param			directory	query		string	true	"项目路径"
//	@Param			request		body		models.UpdateSystemPromptRequest	true	"系统提示词"
//	@Success		200		{object}	models.UpdateSystemPromptResponse
//	@Failure		400		{object}	map[string]interface{}
//	@Failure		404		{object}	map[string]interface{}
//	@Failure		500		{object}	map[string]interface{}
//	@Router			/system-prompt [put]
func (h *Handlers) HandleUpdateSystemPrompt(c context.Context, ctx *hertzapp.RequestContext) {
	projectPath := string(ctx.Query("directory"))
	if projectPath == "" {
		WriteError(c, ctx, "MISSING_DIRECTORY_PARAM", "Directory query parameter is required", consts.StatusBadRequest)
		return
	}

	// 解析请求体
	var req models.UpdateSystemPromptRequest
	if err := json.Unmarshal(ctx.Request.Body(), &req); err != nil {
		slog.Error("Failed to parse request body", "error", err)
		WriteError(c, ctx, "INVALID_REQUEST_BODY", "Invalid request body format", consts.StatusBadRequest)
		return
	}

	// 验证系统提示词不为空
	if strings.TrimSpace(req.SystemPrompt) == "" {
		WriteError(c, ctx, "EMPTY_SYSTEM_PROMPT", "System prompt cannot be empty or whitespace only", consts.StatusBadRequest)
		return
	}

	// 获取项目的 app 实例
	appInstance, err := h.GetAppForProject(c, projectPath)
	if err != nil {
		if strings.Contains(err.Error(), "project not found") {
			WriteError(c, ctx, "PROJECT_NOT_FOUND", err.Error(), consts.StatusNotFound)
			return
		}
		slog.Error("Failed to get app instance", "project", projectPath, "error", err)
		WriteError(c, ctx, "INTERNAL_ERROR", "Failed to get app instance: "+err.Error(), consts.StatusInternalServerError)
		return
	}

	// 使用反射访问器获取 SessionAgent
	accessor := newCoordinatorAccessor()
	sessionAgent, err := accessor.getSessionAgent(appInstance.AgentCoordinator)
	if err != nil {
		slog.Error("Failed to access session agent through reflection",
			"project", projectPath,
			"error", err)
		WriteError(c, ctx, "COORDINATOR_ACCESS_FAILED",
			"Failed to access session agent: "+err.Error()+
				". This may indicate internal implementation has changed.",
			consts.StatusInternalServerError)
		return
	}

	// 更新系统提示词
	sessionAgent.SetSystemPrompt(req.SystemPrompt)

	slog.Info("System prompt updated successfully",
		"project", projectPath,
		"prompt_length", len(req.SystemPrompt))

	// 返回成功响应
	WriteJSON(c, ctx, consts.StatusOK, models.UpdateSystemPromptResponse{
		Success:      true,
		SystemPrompt: req.SystemPrompt,
		Message:      "System prompt updated successfully",
	})
}
