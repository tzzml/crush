package handlers

import (
	"encoding/json"
	"net/http"
)

// WriteError 写入错误响应
func WriteError(w http.ResponseWriter, code string, message string, statusCode int) {
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error": map[string]interface{}{
			"code":    code,
			"message": message,
		},
	})
}

// WriteJSON 写入 JSON 响应
func WriteJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		// 如果编码失败，尝试写入错误
		w.WriteHeader(http.StatusInternalServerError)
	}
}
