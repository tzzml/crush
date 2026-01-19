package handlers

import (
	"net/http"
	"strings"

	"github.com/charmbracelet/crush/api/models"
)

// HandleGetConfig 获取配置信息 (参考 OpenCode: /config)
func (h *Handlers) HandleGetConfig(w http.ResponseWriter, r *http.Request) {
	projectPath, err := extractProjectPathFromConfig(r)
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

	cfg := appInstance.Config()
	if cfg == nil {
		WriteError(w, "CONFIG_NOT_FOUND", "Configuration not available", http.StatusNotFound)
		return
	}

	response := models.ConfigResponse{
		WorkingDir: cfg.WorkingDir(),
		DataDir:    cfg.Options.DataDirectory,
		Debug:      cfg.Options.Debug,
		Configured: cfg.IsConfigured(),
	}

	// 添加 provider 信息（不暴露敏感的 API Key）
	if cfg.Providers != nil && cfg.Providers.Len() > 0 {
		response.Providers = make([]models.ProviderInfo, 0, cfg.Providers.Len())
		for name, p := range cfg.Providers.Seq2() {
			response.Providers = append(response.Providers, models.ProviderInfo{
				Name:       name,
				Type:       string(p.Type),
				Configured: p.APIKey != "" || p.BaseURL != "",
			})
		}
	}

	WriteJSON(w, http.StatusOK, response)
}

// extractProjectPathFromConfig 从配置 API URL 中提取项目路径
func extractProjectPathFromConfig(r *http.Request) (string, error) {
	return extractProjectPathGeneric(r, "/config")
}
