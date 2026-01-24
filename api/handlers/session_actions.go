package handlers

import (
	"context"
	"strings"

	hertzapp "github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
)

// HandleAbortSession 中止会话的 AI 处理 (参考 OpenCode: /session/{sessionID}/abort)
//
//	@Summary		中止会话
//	@Description	中止指定会话的 AI 处理
//	@Tags			Session
//	@Accept			json
//	@Produce		json
//	@Param			directory	query		string	true	"项目路径"
//	@Param			session_id	path		string	true	"会话ID"
//	@Success		200			{object}	map[string]string
//	@Failure		400			{object}	map[string]interface{}
//	@Failure		404			{object}	map[string]interface{}
//	@Router			/session/{session_id}/abort [post]
func (h *Handlers) HandleAbortSession(c context.Context, ctx *hertzapp.RequestContext) {
	projectPath := string(ctx.Query("directory"))
	if projectPath == "" {
		WriteError(c, ctx, "MISSING_DIRECTORY_PARAM", "Directory query parameter is required", consts.StatusBadRequest)
		return
	}

	sessionID := ctx.Param("session_id")
	if sessionID == "" {
		WriteError(c, ctx, "INVALID_REQUEST", "Session ID is required", consts.StatusBadRequest)
		return
	}

	// 获取项目的 app 实例
	appInstance, err := h.GetAppForProject(c, projectPath)
	if err != nil {
		if strings.Contains(err.Error(), "project not found") {
			WriteError(c, ctx, "PROJECT_NOT_FOUND", err.Error(), consts.StatusNotFound)
			return
		}
		WriteError(c, ctx, "INTERNAL_ERROR", "Failed to get or create app for project: "+err.Error(), consts.StatusInternalServerError)
		return
	}

	// 验证会话存在
	_, err = appInstance.Sessions.Get(c, sessionID)
	if err != nil {
		WriteError(c, ctx, "SESSION_NOT_FOUND", "Session not found: "+err.Error(), consts.StatusNotFound)
		return
	}

	// 中止所有正在进行的 agent 处理
	// 注意：当前 internal 层的 AgentCoordinator.CancelAll() 会取消所有会话
	// 如果需要只取消特定会话，可能需要扩展 internal 层
	if appInstance.AgentCoordinator != nil {
		appInstance.AgentCoordinator.CancelAll()
	}

	WriteJSON(c, ctx, consts.StatusOK, map[string]string{
		"status":     "aborted",
		"session_id": sessionID,
	})
}

// HandleGetSessionStatus 获取会话状态 (参考 OpenCode: /session/status)
//
//	@Summary		获取会话状态
//	@Description	获取项目的会话状态信息
//	@Tags			Session
//	@Accept			json
//	@Produce		json
//	@Param			directory	query		string	true	"项目路径"
//	@Success		200			{object}	map[string]interface{}
//	@Failure		400			{object}	map[string]interface{}
//	@Failure		404			{object}	map[string]interface{}
//	@Router			/session/status [get]
func (h *Handlers) HandleGetSessionStatus(c context.Context, ctx *hertzapp.RequestContext) {
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
		WriteError(c, ctx, "INTERNAL_ERROR", "Failed to get or create app for project: "+err.Error(), consts.StatusInternalServerError)
		return
	}

	// 获取所有会话并统计
	sessions, err := appInstance.Sessions.List(c)
	if err != nil {
		WriteError(c, ctx, "INTERNAL_ERROR", "Failed to list sessions: "+err.Error(), consts.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"total_sessions": len(sessions),
		"app_configured": appInstance.Config().IsConfigured(),
		"agent_ready":    appInstance.AgentCoordinator != nil,
	}

	WriteJSON(c, ctx, consts.StatusOK, response)
}
