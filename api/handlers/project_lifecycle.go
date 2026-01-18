package handlers

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/charmbracelet/crush/api/models"
)

// HandleOpenProject 处理打开项目 app 实例的请求
func (h *Handlers) HandleOpenProject(w http.ResponseWriter, r *http.Request) {
	projectPath, err := extractProjectPathFromLifecycle(r)
	if err != nil {
		WriteError(w, "INVALID_REQUEST", "Failed to extract project path: "+err.Error(), http.StatusBadRequest)
		return
	}

	err = h.Open(r.Context(), projectPath)
	if err != nil {
		if strings.Contains(err.Error(), "project not found") {
			WriteError(w, "PROJECT_NOT_FOUND", err.Error(), http.StatusNotFound)
			return
		}
		WriteError(w, "INTERNAL_ERROR", "Failed to open project: "+err.Error(), http.StatusInternalServerError)
		return
	}

	response := models.OpenProjectResponse{
		ProjectPath: projectPath,
		Status:      "opened",
	}

	WriteJSON(w, http.StatusOK, response)
}

// HandleCloseProject 处理关闭项目 app 实例的请求
func (h *Handlers) HandleCloseProject(w http.ResponseWriter, r *http.Request) {
	projectPath, err := extractProjectPathFromLifecycle(r)
	if err != nil {
		WriteError(w, "INVALID_REQUEST", "Failed to extract project path: "+err.Error(), http.StatusBadRequest)
		return
	}

	err = h.Close(r.Context(), projectPath)
	if err != nil {
		WriteError(w, "INTERNAL_ERROR", "Failed to close project: "+err.Error(), http.StatusInternalServerError)
		return
	}

	response := models.CloseProjectResponse{
		ProjectPath: projectPath,
		Status:      "closed",
	}

	WriteJSON(w, http.StatusOK, response)
}

// extractProjectPathFromLifecycle 从生命周期 API URL 中提取项目路径
// URL 格式: /api/v1/projects/{project_path}/open
//           /api/v1/projects/{project_path}/close
func extractProjectPathFromLifecycle(r *http.Request) (string, error) {
	path := r.URL.Path
	prefix := "/api/v1/projects/"
	if !strings.HasPrefix(path, prefix) {
		return "", http.ErrMissingFile
	}

	// 移除前缀
	rest := path[len(prefix):]

	// 查找生命周期操作的后缀（/open, /close）
	var suffix string
	if strings.HasSuffix(rest, "/open") {
		suffix = "/open"
	} else if strings.HasSuffix(rest, "/close") {
		suffix = "/close"
	} else {
		return "", http.ErrMissingFile
	}

	// 提取项目路径
	projectPath := rest[:len(rest)-len(suffix)]
	if projectPath == "" {
		return "", http.ErrMissingFile
	}

	// URL 解码
	decoded, err := url.PathUnescape(projectPath)
	if err != nil {
		return "", err
	}

	return decoded, nil
}
