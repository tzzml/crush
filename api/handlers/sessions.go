package handlers

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"github.com/charmbracelet/crush/api/models"
)

// HandleListSessions 处理获取项目下所有会话的请求
func (h *Handlers) HandleListSessions(w http.ResponseWriter, r *http.Request) {
	projectPath, err := extractProjectPath(r)
	if err != nil {
		WriteError(w, "INVALID_PROJECT_PATH", "Failed to extract project path: "+err.Error(), http.StatusBadRequest)
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

	// 获取所有会话
	sessions, err := appInstance.Sessions.List(r.Context())
	if err != nil {
		WriteError(w, "INTERNAL_ERROR", "Failed to list sessions: "+err.Error(), http.StatusInternalServerError)
		return
	}

	response := models.SessionsResponse{
		Sessions: make([]models.SessionResponse, len(sessions)),
		Total:    len(sessions),
	}
	for i, s := range sessions {
		response.Sessions[i] = models.SessionToResponse(s)
	}

	WriteJSON(w, http.StatusOK, response)
}

// HandleCreateSession 处理创建会话的请求
func (h *Handlers) HandleCreateSession(w http.ResponseWriter, r *http.Request) {
	projectPath, err := extractProjectPath(r)
	if err != nil {
		WriteError(w, "INVALID_REQUEST", err.Error(), http.StatusBadRequest)
		return
	}

	var req models.CreateSessionRequest
	if err := decodeJSON(r, &req); err != nil {
		WriteError(w, "INVALID_REQUEST", "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	if req.Title == "" {
		WriteError(w, "INVALID_REQUEST", "Title is required", http.StatusBadRequest)
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


	// 创建会话
	session, err := appInstance.Sessions.Create(r.Context(), req.Title)
	if err != nil {
		WriteError(w, "INTERNAL_ERROR", "Failed to create session: "+err.Error(), http.StatusInternalServerError)
		return
	}

	slog.Info("Session created", "project", projectPath, "session_id", session.ID)

	response := models.CreateSessionResponse{
		Session: models.SessionToResponse(session),
	}

	WriteJSON(w, http.StatusCreated, response)
}

// HandleGetSession 处理获取单个会话的请求
func (h *Handlers) HandleGetSession(w http.ResponseWriter, r *http.Request) {
	projectPath, sessionID, err := extractProjectAndSessionID(r)
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

	// 获取会话详情
	session, err := appInstance.Sessions.Get(r.Context(), sessionID)
	if err != nil {
		WriteError(w, "SESSION_NOT_FOUND", "Session not found: "+err.Error(), http.StatusNotFound)
		return
	}

	response := models.SessionDetailResponse{
		Session: models.SessionToResponse(session),
	}

	WriteJSON(w, http.StatusOK, response)
}

// HandleDeleteSession 处理删除会话的请求
func (h *Handlers) HandleDeleteSession(w http.ResponseWriter, r *http.Request) {
	projectPath, sessionID, err := extractProjectAndSessionID(r)
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

	// 删除会话
	err = appInstance.Sessions.Delete(r.Context(), sessionID)
	if err != nil {
		WriteError(w, "SESSION_NOT_FOUND", "Session not found: "+err.Error(), http.StatusNotFound)
		return
	}

	WriteJSON(w, http.StatusOK, map[string]string{"message": "Session deleted successfully"})
}

// extractSessionID 从 URL 中提取会话 ID
func extractSessionID(r *http.Request) string {
	path := r.URL.Path
	prefix := "/api/v1/sessions/"
	if !strings.HasPrefix(path, prefix) {
		return ""
	}

	rest := path[len(prefix):]

	// 如果后面还有路径（如 /messages），需要截断
	if idx := strings.Index(rest, "/"); idx != -1 {
		return rest[:idx]
	}

	return rest
}

// extractProjectAndSessionID 从 URL 中提取项目路径和会话 ID
// URL 格式: /api/v1/projects/{project_path}/sessions/{session_id}
func extractProjectAndSessionID(r *http.Request) (projectPath, sessionID string, err error) {
	path := r.URL.Path
	prefix := "/api/v1/projects/"
	if !strings.HasPrefix(path, prefix) {
		return "", "", fmt.Errorf("invalid path format")
	}

	// 移除前缀
	rest := path[len(prefix):]

	// 查找 /sessions/ 的位置
	sessionsIndex := strings.Index(rest, "/sessions/")
	if sessionsIndex == -1 {
		return "", "", fmt.Errorf("missing /sessions/ in path")
	}

	// 提取项目路径
	projectPath = rest[:sessionsIndex]
	if projectPath == "" {
		return "", "", fmt.Errorf("project path is empty")
	}

	// URL 解码项目路径
	projectPath, err = url.QueryUnescape(projectPath)
	if err != nil {
		return "", "", fmt.Errorf("failed to decode project path: %w", err)
	}

	// 提取会话ID部分
	sessionPart := rest[sessionsIndex+len("/sessions/"):]
	// 移除可能的尾部斜杠
	sessionPart = strings.TrimSuffix(sessionPart, "/")
	if sessionPart == "" {
		return "", "", fmt.Errorf("session ID is empty")
	}

	sessionID = sessionPart
	return projectPath, sessionID, nil
}
