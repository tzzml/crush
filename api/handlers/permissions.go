package handlers

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

	"github.com/charmbracelet/crush/api/models"
)

// HandleListPermissions 获取待处理的权限请求列表 (参考 OpenCode: /permission)
func (h *Handlers) HandleListPermissions(w http.ResponseWriter, r *http.Request) {
	projectPath, err := extractProjectPathFromPermissions(r)
	if err != nil {
		WriteError(w, "INVALID_REQUEST", "Failed to extract project path: "+err.Error(), http.StatusBadRequest)
		return
	}

	// 获取项目的 app 实例
	appInstance, err := h.GetAppForProject(r.Context(), projectPath)
	if err != nil {
		if strings.Contains(err.Error(), "not open") {
			WriteError(w, "APP_NOT_OPENED", "Project app instance is not open. Call open first: "+err.Error(), http.StatusBadRequest)
			return
		}
		WriteError(w, "PROJECT_NOT_FOUND", "Failed to get app for project: "+err.Error(), http.StatusNotFound)
		return
	}

	// 返回权限服务状态
	skipRequests := appInstance.Permissions.SkipRequests()

	response := models.PermissionsResponse{
		SkipRequests: skipRequests,
	}

	WriteJSON(w, http.StatusOK, response)
}

// HandleReplyPermission 回复权限请求 (参考 OpenCode: /permission/{requestID}/reply)
func (h *Handlers) HandleReplyPermission(w http.ResponseWriter, r *http.Request) {
	projectPath, requestID, err := extractProjectAndPermissionID(r)
	if err != nil {
		WriteError(w, "INVALID_REQUEST", "Failed to extract request ID: "+err.Error(), http.StatusBadRequest)
		return
	}

	var req models.PermissionReplyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, "INVALID_REQUEST", "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	// 获取项目的 app 实例
	appInstance, err := h.GetAppForProject(r.Context(), projectPath)
	if err != nil {
		if strings.Contains(err.Error(), "not open") {
			WriteError(w, "APP_NOT_OPENED", "Project app instance is not open. Call open first: "+err.Error(), http.StatusBadRequest)
			return
		}
		WriteError(w, "PROJECT_NOT_FOUND", "Failed to get app for project: "+err.Error(), http.StatusNotFound)
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

	WriteJSON(w, http.StatusOK, map[string]string{
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

// extractProjectPathFromPermissions 从权限 API URL 中提取项目路径
func extractProjectPathFromPermissions(r *http.Request) (string, error) {
	return extractProjectPathGeneric(r, "/permissions")
}

// extractProjectAndPermissionID 从 URL 中提取项目路径和权限请求 ID
func extractProjectAndPermissionID(r *http.Request) (projectPath, requestID string, err error) {
	path := r.URL.Path
	prefix := "/api/v1/projects/"
	if !strings.HasPrefix(path, prefix) {
		return "", "", http.ErrMissingFile
	}

	rest := path[len(prefix):]

	// 查找 /permissions/ 的位置
	permIdx := strings.Index(rest, "/permissions/")
	if permIdx == -1 {
		return "", "", http.ErrMissingFile
	}

	projectPath = rest[:permIdx]
	projectPath, err = url.PathUnescape(projectPath)
	if err != nil {
		return "", "", err
	}

	// 提取 requestID 和 /reply 后缀
	permPart := rest[permIdx+len("/permissions/"):]
	replyIdx := strings.Index(permPart, "/reply")
	if replyIdx == -1 {
		return "", "", http.ErrMissingFile
	}

	requestID = permPart[:replyIdx]
	return projectPath, requestID, nil
}
