package handlers

import (
	"context"
	"strings"

	"github.com/charmbracelet/crush/api/models"
	hertzapp "github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
)

// HandleListPermissions 获取待处理的权限请求列表 (参考 OpenCode: /permission)
//
//	@Summary		获取权限请求列表
//	@Description	获取指定项目的权限请求列表
//	@Tags			Permission
//	@Accept			json
//	@Produce		json
//	@Param			directory	query		string	true	"项目路径"
//	@Success		200		{object}	models.PermissionsResponse
//	@Failure		400		{object}	map[string]interface{}
//	@Failure		404		{object}	map[string]interface{}
//	@Router			/project/permissions [get]
func (h *Handlers) HandleListPermissions(c context.Context, ctx *hertzapp.RequestContext) {
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

	// 返回权限服务状态
	skipRequests := appInstance.Permissions.SkipRequests()

	response := models.PermissionsResponse{
		SkipRequests: skipRequests,
	}

	WriteJSON(c, ctx, consts.StatusOK, response)
}

// HandleReplyPermission 回复权限请求 (参考 OpenCode: /permission/{requestID}/reply)
//
//	@Summary		回复权限请求
//	@Description	批准或拒绝特定的权限请求
//	@Tags			Permission
//	@Accept			json
//	@Produce		json
//	@Param			directory	query		string							true	"项目路径"
//	@Param			id			path		string							true	"权限请求ID"
//	@Param			request		body		models.PermissionReplyRequest	true	"权限回复请求"
//	@Success		200			{object}	map[string]string
//	@Failure		400			{object}	map[string]interface{}
//	@Failure		404			{object}	map[string]interface{}
//	@Router			/project/permissions/{id}/reply [post]
func (h *Handlers) HandleReplyPermission(c context.Context, ctx *hertzapp.RequestContext) {
	projectPath := string(ctx.Query("directory"))
	if projectPath == "" {
		WriteError(c, ctx, "MISSING_DIRECTORY_PARAM", "Directory query parameter is required", consts.StatusBadRequest)
		return
	}

	requestID := ctx.Param("id")
	if requestID == "" {
		WriteError(c, ctx, "MISSING_REQUEST_ID", "Request ID path parameter is required", consts.StatusBadRequest)
		return
	}

	var req models.PermissionReplyRequest
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

	// 注意：这里需要找到对应的 PermissionRequest 才能调用 Grant/Deny
	// 由于 internal 层的限制，我们需要通过订阅事件来获取待处理的请求
	// 这是一个简化的实现，实际中可能需要在 Handlers 中维护待处理请求的映射

	// 创建一个模拟的 PermissionRequest（实际应该从待处理队列中获取）
	permReq := models.PermissionRequest{
		ID: requestID,
	}

	if req.Granted {
		if req.Persistent {
			appInstance.Permissions.GrantPersistent(permReq.ToInternal())
		} else {
			appInstance.Permissions.Grant(permReq.ToInternal())
		}
	} else {
		appInstance.Permissions.Deny(permReq.ToInternal())
	}

	WriteJSON(c, ctx, consts.StatusOK, map[string]string{
		"status":     "replied",
		"request_id": requestID,
		"granted":    boolToString(req.Granted),
	})
}

func boolToString(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
