package models

// UpdateSystemPromptRequest 更新系统提示词的请求体
type UpdateSystemPromptRequest struct {
	// SystemPrompt 新的系统提示词内容
	SystemPrompt string `json:"system_prompt" binding:"required"`
}

// UpdateSystemPromptResponse 更新系统提示词的响应体
type UpdateSystemPromptResponse struct {
	// Success 是否成功更新
	Success bool `json:"success"`

	// SystemPrompt 更新后的系统提示词内容
	SystemPrompt string `json:"system_prompt"`

	// Message 操作结果消息（可选）
	Message string `json:"message,omitempty"`
}

// GetSystemPromptResponse 获取系统提示词的响应体
type GetSystemPromptResponse struct {
	// SystemPrompt 当前的系统提示词内容
	SystemPrompt string `json:"system_prompt"`

	// Length 系统提示词的字符数
	Length int `json:"length"`

	// IsCustom 是否为自定义提示词（非空表示已自定义）
	IsCustom bool `json:"is_custom"`
}
