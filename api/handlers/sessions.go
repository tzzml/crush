package handlers

import (
	"context"
	"log/slog"
	"strings"

	"github.com/charmbracelet/crush/api/models"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
)

// HandleListSessions 处理获取项目下所有会话的请求
//
//	@Summary		获取会话列表
//	@Description	获取指定项目的所有会话
//	@Tags			Session
//	@Accept			json
//	@Produce		json
//	@Param			directory	query		string	true	"项目路径"
//	@Success		200			{object}	models.SessionsResponse
//	@Failure		400			{object}	map[string]interface{}
//	@Failure		404			{object}	map[string]interface{}
//	@Router			/session [get]
func (h *Handlers) HandleListSessions(c context.Context, ctx *app.RequestContext) {
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

	// 获取所有会话
	sessions, err := appInstance.Sessions.List(c)
	if err != nil {
		WriteError(c, ctx, "INTERNAL_ERROR", "Failed to list sessions: "+err.Error(), consts.StatusInternalServerError)
		return
	}

	response := models.SessionsResponse{
		Sessions: make([]models.SessionResponse, len(sessions)),
		Total:    len(sessions),
	}
	for i, s := range sessions {
		response.Sessions[i] = models.SessionToResponse(s)
	}

	WriteJSON(c, ctx, consts.StatusOK, response)
}

// HandleCreateSession 处理创建会话的请求
//
//	@Summary		创建会话
//	@Description	在指定项目中创建新会话
//	@Tags			Session
//	@Accept			json
//	@Produce		json
//	@Param			project		query		string						true	"项目路径"
//	@Param			request		body		models.CreateSessionRequest	true	"创建会话请求"
//	@Success		201			{object}	models.CreateSessionResponse
//	@Failure		400			{object}	map[string]interface{}
//	@Failure		404			{object}	map[string]interface{}
//	@Router			/session [post]
func (h *Handlers) HandleCreateSession(c context.Context, ctx *app.RequestContext) {
	projectPath := string(ctx.Query("directory"))
	if projectPath == "" {
		WriteError(c, ctx, "MISSING_DIRECTORY_PARAM", "Directory query parameter is required", consts.StatusBadRequest)
		return
	}

	var req models.CreateSessionRequest
	if err := ctx.BindJSON(&req); err != nil {
		WriteError(c, ctx, "INVALID_REQUEST", "Invalid request body: "+err.Error(), consts.StatusBadRequest)
		return
	}

	if req.Title == "" {
		WriteError(c, ctx, "INVALID_REQUEST", "Title is required", consts.StatusBadRequest)
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

	// 创建会话
	session, err := appInstance.Sessions.Create(c, req.Title)
	if err != nil {
		WriteError(c, ctx, "INTERNAL_ERROR", "Failed to create session: "+err.Error(), consts.StatusInternalServerError)
		return
	}

	slog.Info("Session created", "project", projectPath, "session_id", session.ID)

	response := models.CreateSessionResponse{
		Session: models.SessionToResponse(session),
	}

	WriteJSON(c, ctx, consts.StatusCreated, response)
}

// HandleGetSession 处理获取单个会话的请求
//
//	@Summary		获取会话详情
//	@Description	获取指定会话的详细信息
//	@Tags			Session
//	@Accept			json
//	@Produce		json
//	@Param			directory	query		string	true	"项目路径"
//	@Param			id			path		string	true	"会话ID"
//	@Success		200			{object}	models.SessionDetailResponse
//	@Failure		400			{object}	map[string]interface{}
//	@Failure		404			{object}	map[string]interface{}
//	@Router			/session/{id} [get]
func (h *Handlers) HandleGetSession(c context.Context, ctx *app.RequestContext) {
	projectPath := string(ctx.Query("directory"))
	if projectPath == "" {
		WriteError(c, ctx, "MISSING_DIRECTORY_PARAM", "Directory query parameter is required", consts.StatusBadRequest)
		return
	}

	sessionID := ctx.Param("id")

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

	// 获取会话详情
	session, err := appInstance.Sessions.Get(c, sessionID)
	if err != nil {
		WriteError(c, ctx, "SESSION_NOT_FOUND", "Session not found: "+err.Error(), consts.StatusNotFound)
		return
	}

	response := models.SessionDetailResponse{
		Session: models.SessionToResponse(session),
	}

	WriteJSON(c, ctx, consts.StatusOK, response)
}

// HandleDeleteSession 处理删除会话的请求
//
//	@Summary		删除会话
//	@Description	删除指定的会话及其所有消息
//	@Tags			Session
//	@Accept			json
//	@Produce		json
//	@Param			directory	query		string	true	"项目路径"
//	@Param			id			path		string	true	"会话ID"
//	@Success		200			{object}	map[string]string
//	@Failure		400			{object}	map[string]interface{}
//	@Failure		404			{object}	map[string]interface{}
//	@Router			/session/{id} [delete]
func (h *Handlers) HandleDeleteSession(c context.Context, ctx *app.RequestContext) {
	projectPath := string(ctx.Query("directory"))
	if projectPath == "" {
		WriteError(c, ctx, "MISSING_DIRECTORY_PARAM", "Directory query parameter is required", consts.StatusBadRequest)
		return
	}

	sessionID := ctx.Param("id")

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

	// 删除会话
	err = appInstance.Sessions.Delete(c, sessionID)
	if err != nil {
		WriteError(c, ctx, "SESSION_NOT_FOUND", "Session not found: "+err.Error(), consts.StatusNotFound)
		return
	}

	WriteJSON(c, ctx, consts.StatusOK, map[string]string{"message": "Session deleted successfully"})
}

// HandleUpdateSession 处理更新会话的请求
//
//	@Summary		更新会话
//	@Description	更新指定会话的信息
//	@Tags			Session
//	@Accept			json
//	@Produce		json
//	@Param			project		query		string							true	"项目路径"
//	@Param			id			path		string							true	"会话ID"
//	@Param			request		body		models.UpdateSessionRequest	true	"更新会话请求"
//	@Success		200			{object}	models.UpdateSessionResponse
//	@Failure		400			{object}	map[string]interface{}
//	@Failure		404			{object}	map[string]interface{}
//	@Router			/session/{id} [put]
func (h *Handlers) HandleUpdateSession(c context.Context, ctx *app.RequestContext) {
	projectPath := string(ctx.Query("directory"))
	if projectPath == "" {
		WriteError(c, ctx, "MISSING_DIRECTORY_PARAM", "Directory query parameter is required", consts.StatusBadRequest)
		return
	}

	sessionID := ctx.Param("id")

	var req models.UpdateSessionRequest
	if err := ctx.BindJSON(&req); err != nil {
		WriteError(c, ctx, "INVALID_REQUEST", "Invalid request body: "+err.Error(), consts.StatusBadRequest)
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

	// 获取现有会话
	session, err := appInstance.Sessions.Get(c, sessionID)
	if err != nil {
		WriteError(c, ctx, "SESSION_NOT_FOUND", "Session not found: "+err.Error(), consts.StatusNotFound)
		return
	}

	// 更新会话字段
	if req.Title != "" {
		session.Title = req.Title
	}

	// 保存更新后的会话
	updatedSession, err := appInstance.Sessions.Save(c, session)
	if err != nil {
		WriteError(c, ctx, "INTERNAL_ERROR", "Failed to update session: "+err.Error(), consts.StatusInternalServerError)
		return
	}

	response := models.UpdateSessionResponse{
		Session: models.SessionToResponse(updatedSession),
	}

	WriteJSON(c, ctx, consts.StatusOK, response)
}
