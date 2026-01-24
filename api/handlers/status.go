package handlers

import (
	"context"
	"strings"

	"github.com/charmbracelet/crush/api/models"
	hertzapp "github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
)

// HandleGetLSPStatus 处理获取 LSP 状态的请求 (参考 OpenCode: /lsp)
//
//	@Summary		获取 LSP 状态
//	@Description	获取所有 LSP 服务器的状态
//	@Tags			LSP
//	@Accept			json
//	@Produce		json
//	@Param			directory	query		string	true	"项目路径"
//	@Success		200			{array}		models.LSPStatus
//	@Failure		400			{object}	map[string]interface{}
//	@Failure		404			{object}	map[string]interface{}
//	@Router			/lsp [get]
func (h *Handlers) HandleGetLSPStatus(c context.Context, ctx *hertzapp.RequestContext) {
	directory := string(ctx.Query("directory"))
	if directory == "" {
		WriteError(c, ctx, "MISSING_DIRECTORY_PARAM", "Directory query parameter is required", consts.StatusBadRequest)
		return
	}

	// 获取项目的 app 实例
	appInstance, err := h.GetAppForProject(c, directory)
	if err != nil {
		if strings.Contains(err.Error(), "project not found") {
			WriteError(c, ctx, "PROJECT_NOT_FOUND", err.Error(), consts.StatusNotFound)
			return
		}
		WriteError(c, ctx, "INTERNAL_ERROR", "Failed to get app for project: "+err.Error(), consts.StatusInternalServerError)
		return
	}

	// 获取 LSP 客户端列表
	lspClients := appInstance.LSPClients
	var lspStatusList []models.LSPStatus

	// 遍历所有 LSP 客户端
	// csync.Map 使用 Seq2() 方法遍历
	for name, _ := range lspClients.Seq2() {
		status := models.LSPStatus{
			ID:     name,
			Name:   name,
			Root:   directory,
			Status: "connected", // 如果在 Map 中，说明已连接
		}
		lspStatusList = append(lspStatusList, status)
	}

	WriteJSON(c, ctx, consts.StatusOK, lspStatusList)
}

// HandleGetMCPStatus 处理获取 MCP 状态的请求 (参考 OpenCode: /mcp)
//
//	@Summary		获取 MCP 状态
//	@Description	获取所有 Model Context Protocol (MCP) 服务器的状态
//	@Tags			MCP
//	@Accept			json
//	@Produce		json
//	@Param			directory	query		string	true	"项目路径"
//	@Success		200			{object}	map[string]models.MCPStatus
//	@Failure		400			{object}	map[string]interface{}
//	@Failure		404			{object}	map[string]interface{}
//	@Router			/mcp [get]
func (h *Handlers) HandleGetMCPStatus(c context.Context, ctx *hertzapp.RequestContext) {
	directory := string(ctx.Query("directory"))
	if directory == "" {
		WriteError(c, ctx, "MISSING_DIRECTORY_PARAM", "Directory query parameter is required", consts.StatusBadRequest)
		return
	}

	// 获取项目的 app 实例
	appInstance, err := h.GetAppForProject(c, directory)
	if err != nil {
		if strings.Contains(err.Error(), "project not found") {
			WriteError(c, ctx, "PROJECT_NOT_FOUND", err.Error(), consts.StatusNotFound)
			return
		}
		WriteError(c, ctx, "INTERNAL_ERROR", "Failed to get app for project: "+err.Error(), consts.StatusInternalServerError)
		return
	}

	// 获取配置的 MCP 列表
	cfg := appInstance.Config()
	mcpConfigs := cfg.MCP

	// 构建 MCP 状态响应
	mcpStatusMap := make(map[string]models.MCPStatus)

	for name := range mcpConfigs {
		// 简单判断 MCP 状态
		// 实际实现中可能需要检查 MCP 连接状态
		status := models.MCPStatusConnected{
			Status: "connected",
		}

		// 将状态转换为通用格式
		mcpStatusMap[name] = models.MCPStatus{
			"status": status.Status,
		}
	}

	WriteJSON(c, ctx, consts.StatusOK, mcpStatusMap)
}
