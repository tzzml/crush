package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/url"
	"strconv"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
)

// WriteError 写入错误响应
func WriteError(c context.Context, ctx *app.RequestContext, code string, message string, statusCode int) {
	response := map[string]interface{}{
		"error": map[string]interface{}{
			"code":    code,
			"message": message,
		},
	}

	// 先编码到 buffer，避免部分写入问题
	buf := &bytes.Buffer{}
	if err := json.NewEncoder(buf).Encode(response); err != nil {
		slog.Error("Failed to encode error response", "error", err)
		ctx.JSON(consts.StatusInternalServerError, map[string]string{
			"error":   "INTERNAL_ERROR",
			"message": "Failed to encode response",
		})
		return
	}

	ctx.SetStatusCode(statusCode)
	ctx.Response.SetBody(buf.Bytes())
}

// WriteJSON 写入 JSON 响应
func WriteJSON(c context.Context, ctx *app.RequestContext, statusCode int, data interface{}) {
	// 先编码到 buffer，避免部分写入问题
	buf := &bytes.Buffer{}
	if err := json.NewEncoder(buf).Encode(data); err != nil {
		slog.Error("Failed to encode JSON response", "error", err)
		WriteError(c, ctx, "INTERNAL_ERROR", "Failed to encode response", consts.StatusInternalServerError)
		return
	}

	ctx.SetStatusCode(statusCode)
	ctx.Response.SetBody(buf.Bytes())
}

// ParsePaginationParams 解析分页参数
func ParsePaginationParams(ctx *app.RequestContext, defaultLimit, maxLimit int) (limit, offset int) {
	limit = defaultLimit
	offset = 0

	if l := ctx.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
			if limit > maxLimit {
				limit = maxLimit
			}
		}
	}

	if o := ctx.Query("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	return limit, offset
}

// ExtractProjectPath 从 Hertz 请求上下文中提取并解码项目路径
func ExtractProjectPath(ctx *app.RequestContext) (string, error) {
	projectPath := ctx.Param("path")
	projectPath, err := url.PathUnescape(projectPath)
	if err != nil {
		return "", err
	}

	if projectPath == "" {
		return "", ErrInvalidPath
	}

	return projectPath, nil
}

// ErrInvalidPath 表示项目路径无效的错误
var ErrInvalidPath = &PathError{"invalid project path"}

// PathError 表示路径相关错误
type PathError struct {
	Msg string
}

func (e *PathError) Error() string {
	return e.Msg
}


