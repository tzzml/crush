package handlers

import (
	"encoding/json"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/crush/api/models"
	"github.com/charmbracelet/crush/internal/projects"
)

// HandleListProjects 处理获取所有项目的请求
func (h *Handlers) HandleListProjects(w http.ResponseWriter, r *http.Request) {
	projectList, err := projects.List()
	if err != nil {
		WriteError(w, "INTERNAL_ERROR", "Failed to list projects: "+err.Error(), http.StatusInternalServerError)
		return
	}

	response := models.ProjectsResponse{
		Projects: make([]models.ProjectResponse, len(projectList)),
	}
	for i, p := range projectList {
		response.Projects[i] = models.ProjectToResponse(p)
	}

	WriteJSON(w, http.StatusOK, response)
}

// HandleCreateProject 处理创建项目的请求
func (h *Handlers) HandleCreateProject(w http.ResponseWriter, r *http.Request) {
	var req models.CreateProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, "INVALID_REQUEST", "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	if req.Path == "" {
		WriteError(w, "INVALID_REQUEST", "Project path is required", http.StatusBadRequest)
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
		WriteError(w, "INTERNAL_ERROR", "Failed to register project: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 获取注册后的项目信息
	projectList, err := projects.List()
	if err != nil {
		WriteError(w, "INTERNAL_ERROR", "Failed to get project: "+err.Error(), http.StatusInternalServerError)
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
		WriteError(w, "INTERNAL_ERROR", "Project registered but not found", http.StatusInternalServerError)
		return
	}

	response := models.CreateProjectResponse{
		Project: models.ProjectToResponse(*project),
	}

	WriteJSON(w, http.StatusCreated, response)
}

// extractProjectPath 从 URL 中提取项目路径
func extractProjectPath(r *http.Request) (string, error) {
	// URL 格式: /api/v1/projects/{project_path}/sessions
	// project_path 需要 URL 解码
	path := r.URL.Path
	prefix := "/api/v1/projects/"
	if !strings.HasPrefix(path, prefix) {
		return "", http.ErrMissingFile
	}

	// 移除前缀
	rest := path[len(prefix):]

	// 找到 /sessions 的位置
	idx := strings.Index(rest, "/sessions")
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
