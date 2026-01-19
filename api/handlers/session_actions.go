package handlers

import (
	"net/http"
	"net/url"
	"strings"
)

// HandleAbortSession 中止会话的 AI 处理 (参考 OpenCode: /session/{sessionID}/abort)
func (h *Handlers) HandleAbortSession(w http.ResponseWriter, r *http.Request) {
	projectPath, sessionID, err := extractProjectAndSessionIDFromAction(r)
	if err != nil {
		WriteError(w, "INVALID_REQUEST", err.Error(), http.StatusBadRequest)
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

	// 验证会话存在
	_, err = appInstance.Sessions.Get(r.Context(), sessionID)
	if err != nil {
		WriteError(w, "SESSION_NOT_FOUND", "Session not found: "+err.Error(), http.StatusNotFound)
		return
	}

	// 中止所有正在进行的 agent 处理
	// 注意：当前 internal 层的 AgentCoordinator.CancelAll() 会取消所有会话
	// 如果需要只取消特定会话，可能需要扩展 internal 层
	if appInstance.AgentCoordinator != nil {
		appInstance.AgentCoordinator.CancelAll()
	}

	WriteJSON(w, http.StatusOK, map[string]string{
		"status":     "aborted",
		"session_id": sessionID,
	})
}

// HandleGetSessionStatus 获取会话状态 (参考 OpenCode: /session/status)
func (h *Handlers) HandleGetSessionStatus(w http.ResponseWriter, r *http.Request) {
	projectPath, err := extractProjectPathGeneric(r, "/sessions/status")
	if err != nil {
		WriteError(w, "INVALID_REQUEST", err.Error(), http.StatusBadRequest)
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

	// 获取所有会话并统计
	sessions, err := appInstance.Sessions.List(r.Context())
	if err != nil {
		WriteError(w, "INTERNAL_ERROR", "Failed to list sessions: "+err.Error(), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"total_sessions": len(sessions),
		"app_configured": appInstance.Config().IsConfigured(),
		"agent_ready":    appInstance.AgentCoordinator != nil,
	}

	WriteJSON(w, http.StatusOK, response)
}

// extractProjectAndSessionIDFromAction 从带动作的会话 URL 中提取项目路径和会话 ID
// URL 格式: /api/v1/projects/{project_path}/sessions/{session_id}/abort
func extractProjectAndSessionIDFromAction(r *http.Request) (projectPath, sessionID string, err error) {
	path := r.URL.Path
	prefix := "/api/v1/projects/"
	if !strings.HasPrefix(path, prefix) {
		return "", "", http.ErrMissingFile
	}

	rest := path[len(prefix):]

	// 查找 /sessions/ 的位置
	sessIdx := strings.Index(rest, "/sessions/")
	if sessIdx == -1 {
		return "", "", http.ErrMissingFile
	}

	projectPath = rest[:sessIdx]
	projectPath, err = url.PathUnescape(projectPath)
	if err != nil {
		return "", "", err
	}

	// 提取 session_id 和动作后缀
	sessPart := rest[sessIdx+len("/sessions/"):]
	
	// 查找动作后缀 (/abort, /fork, /summarize 等)
	actionIdx := strings.LastIndex(sessPart, "/")
	if actionIdx == -1 {
		return "", "", http.ErrMissingFile
	}

	sessionID = sessPart[:actionIdx]
	return projectPath, sessionID, nil
}

// extractProjectPathGeneric 通用的项目路径提取函数
func extractProjectPathGeneric(r *http.Request, suffix string) (string, error) {
	path := r.URL.Path
	prefix := "/api/v1/projects/"
	if !strings.HasPrefix(path, prefix) {
		return "", http.ErrMissingFile
	}

	rest := path[len(prefix):]

	// 找到 suffix 的位置
	idx := strings.Index(rest, suffix)
	if idx == -1 {
		return "", http.ErrMissingFile
	}

	projectPath := rest[:idx]

	// URL 解码
	decoded, err := url.PathUnescape(projectPath)
	if err != nil {
		return "", err
	}

	return decoded, nil
}
