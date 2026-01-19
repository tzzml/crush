package handlers

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
)

// WriteError 写入错误响应
func WriteError(w http.ResponseWriter, code string, message string, statusCode int) {
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
		http.Error(w, `{"error":{"code":"INTERNAL_ERROR","message":"Failed to encode response"}}`, http.StatusInternalServerError)
		return
	}
	
	w.WriteHeader(statusCode)
	w.Write(buf.Bytes())
}

// WriteJSON 写入 JSON 响应
func WriteJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	// 先编码到 buffer，避免部分写入问题
	buf := &bytes.Buffer{}
	if err := json.NewEncoder(buf).Encode(data); err != nil {
		slog.Error("Failed to encode JSON response", "error", err)
		WriteError(w, "INTERNAL_ERROR", "Failed to encode response", http.StatusInternalServerError)
		return
	}
	
	w.WriteHeader(statusCode)
	w.Write(buf.Bytes())
}

// ParsePaginationParams 解析分页参数
func ParsePaginationParams(r *http.Request, defaultLimit, maxLimit int) (limit, offset int) {
	limit = defaultLimit
	offset = 0
	
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
			if limit > maxLimit {
				limit = maxLimit
			}
		}
	}
	
	if o := r.URL.Query().Get("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}
	
	return limit, offset
}
