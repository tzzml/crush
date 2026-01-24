package handlers

import (
	"context"
	"strings"

	"github.com/charmbracelet/crush/api/models"
	hertzapp "github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
)

// HandleGetConfig 获取配置信息 (参考 OpenCode: /config)
//
//	@Summary		获取项目配置
//	@Description	获取指定项目的配置信息
//	@Tags			Config
//	@Accept			json
//	@Produce		json
//	@Param			directory	query		string	true	"项目路径"
//	@Success		200		{object}	models.ConfigResponse
//	@Failure		400		{object}	map[string]interface{}
//	@Failure		404		{object}	map[string]interface{}
//	@Router			/project/config [get]
func (h *Handlers) HandleGetConfig(c context.Context, ctx *hertzapp.RequestContext) {
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

	cfg := appInstance.Config()
	if cfg == nil {
		WriteError(c, ctx, "CONFIG_NOT_FOUND", "Configuration not available", consts.StatusNotFound)
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

	WriteJSON(c, ctx, consts.StatusOK, response)
}

