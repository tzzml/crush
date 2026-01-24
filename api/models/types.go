package models

import (
	"encoding/base64"
	"time"

	"github.com/charmbracelet/crush/internal/message"
	"github.com/charmbracelet/crush/internal/permission"
	"github.com/charmbracelet/crush/internal/projects"
	"github.com/charmbracelet/crush/internal/session"
)

// API 响应包装器

type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

type ErrorDetail struct {
	Code    string                 `json:"code"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// Projects API

type ProjectsResponse struct {
	Projects []ProjectResponse `json:"projects"`
}

type ProjectResponse struct {
	Path         string    `json:"path"`
	DataDir      string    `json:"data_dir"`
	LastAccessed time.Time `json:"last_accessed"`
}

type CreateProjectRequest struct {
	Path    string `json:"path"`
	DataDir string `json:"data_dir,omitempty"`
}

type CreateProjectResponse struct {
	Project ProjectResponse `json:"project"`
}

// Project Lifecycle API

type CurrentProjectResponse struct {
	Project *ProjectResponse `json:"project"`
}

type DisposeProjectResponse struct {
	ProjectPath string `json:"project_path"`
	Status      string `json:"status"`
	Message     string `json:"message"`
}

type DisposeAllResponse struct {
	DisposedCount int      `json:"disposed_count"`
	Projects      []string `json:"projects"`
	Status        string   `json:"status"`
}

// Sessions API

type SessionsResponse struct {
	Sessions []SessionResponse `json:"sessions"`
	Total    int               `json:"total"`
}

type SessionResponse struct {
	ID               string         `json:"id"`
	ParentSessionID  string         `json:"parent_session_id,omitempty"`
	Title            string         `json:"title"`
	MessageCount     int64          `json:"message_count"`
	PromptTokens     int64          `json:"prompt_tokens"`
	CompletionTokens int64          `json:"completion_tokens"`
	Cost             float64        `json:"cost"`
	SummaryMessageID string         `json:"summary_message_id,omitempty"`
	Todos            []TodoResponse `json:"todos"`
	CreatedAt        int64          `json:"created_at"`
	UpdatedAt        int64          `json:"updated_at"`
}

type TodoResponse struct {
	Content    string `json:"content"`
	Status     string `json:"status"`
	ActiveForm string `json:"active_form"`
}

type CreateSessionRequest struct {
	Title string `json:"title"`
}

type CreateSessionResponse struct {
	Session SessionResponse `json:"session"`
}

type UpdateSessionRequest struct {
	Title string `json:"title,omitempty"`
}

type UpdateSessionResponse struct {
	Session SessionResponse `json:"session"`
}

type SessionDetailResponse struct {
	Session SessionResponse `json:"session"`
}

// Messages API

type MessagesResponse struct {
	Messages []MessageResponse `json:"messages"`
	Total    int               `json:"total"`
}

type MessageResponse struct {
	ID               string                   `json:"id"`
	SessionID        string                   `json:"session_id"`
	Role             string                   `json:"role"`
	Content          string                   `json:"content"`
	Model            string                   `json:"model,omitempty"`
	Provider         string                   `json:"provider,omitempty"`
	IsSummaryMessage bool                     `json:"is_summary_message"`
	CreatedAt        int64                    `json:"created_at"`
	UpdatedAt        int64                    `json:"updated_at"`
	FinishedAt       *int64                   `json:"finished_at,omitempty"`
	FinishReason     *string                  `json:"finish_reason,omitempty"`
	Parts            []map[string]interface{} `json:"parts,omitempty"`
}

type CreateMessageRequest struct {
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream,omitempty"`
}

type CreateMessageResponse struct {
	Message MessageResponse `json:"message"`
	Session SessionResponse `json:"session"`
}

type MessageDetailResponse struct {
	Message MessageResponse `json:"message"`
}

// SSE 事件类型

type SSEEvent struct {
	Type      string          `json:"type"`
	MessageID string          `json:"message_id,omitempty"`
	Content   string          `json:"content,omitempty"`
	Message   MessageResponse `json:"message,omitempty"`
	Session   SessionResponse `json:"session,omitempty"`
	Error     *ErrorDetail    `json:"error,omitempty"`
}

// Config API

type ConfigResponse struct {
	WorkingDir string         `json:"working_dir"`
	DataDir    string         `json:"data_dir"`
	Debug      bool           `json:"debug"`
	Configured bool           `json:"configured"`
	Providers  []ProviderInfo `json:"providers,omitempty"`
}

type ProviderInfo struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	Configured bool   `json:"configured"`
}

// Permission API

type PermissionsResponse struct {
	SkipRequests bool                `json:"skip_requests"`
	Pending      []PermissionRequest `json:"pending,omitempty"`
}

type PermissionRequest struct {
	ID          string `json:"id"`
	SessionID   string `json:"session_id"`
	ToolCallID  string `json:"tool_call_id"`
	ToolName    string `json:"tool_name"`
	Description string `json:"description"`
	Action      string `json:"action"`
	Params      any    `json:"params,omitempty"`
	Path        string `json:"path"`
}

type PermissionReplyRequest struct {
	Granted    bool `json:"granted"`
	Persistent bool `json:"persistent,omitempty"`
}

// ToInternal 转换为内部 permission.PermissionRequest 类型
func (p PermissionRequest) ToInternal() permission.PermissionRequest {
	return permission.PermissionRequest{
		ID:          p.ID,
		SessionID:   p.SessionID,
		ToolCallID:  p.ToolCallID,
		ToolName:    p.ToolName,
		Description: p.Description,
		Action:      p.Action,
		Params:      p.Params,
		Path:        p.Path,
	}
}

// Health API

type HealthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version,omitempty"`
}

// 辅助函数：转换内部类型到 API 响应类型

func ProjectToResponse(p projects.Project) ProjectResponse {
	return ProjectResponse{
		Path:         p.Path,
		DataDir:      p.DataDir,
		LastAccessed: p.LastAccessed,
	}
}

func SessionToResponse(s session.Session) SessionResponse {
	todos := make([]TodoResponse, len(s.Todos))
	for i, t := range s.Todos {
		todos[i] = TodoResponse{
			Content:    t.Content,
			Status:     string(t.Status),
			ActiveForm: t.ActiveForm,
		}
	}

	var parentSessionID string
	if s.ParentSessionID != "" {
		parentSessionID = s.ParentSessionID
	}

	var summaryMessageID string
	if s.SummaryMessageID != "" {
		summaryMessageID = s.SummaryMessageID
	}

	return SessionResponse{
		ID:               s.ID,
		ParentSessionID:  parentSessionID,
		Title:            s.Title,
		MessageCount:     s.MessageCount,
		PromptTokens:     s.PromptTokens,
		CompletionTokens: s.CompletionTokens,
		Cost:             s.Cost,
		SummaryMessageID: summaryMessageID,
		Todos:            todos,
		CreatedAt:        s.CreatedAt,
		UpdatedAt:        s.UpdatedAt,
	}
}

func MessageToResponse(m message.Message) MessageResponse {
	var model, provider string
	if m.Model != "" {
		model = string(m.Model)
	}
	if m.Provider != "" {
		provider = m.Provider
	}

	var finishedAt *int64
	var finishReason *string
	if finish := m.FinishPart(); finish != nil {
		if finish.Time > 0 {
			ts := finish.Time
			finishedAt = &ts
		}
		if finish.Reason != "" {
			reason := string(finish.Reason)
			finishReason = &reason
		}
	}

	// Convert Parts to JSON-serializable format
	parts := make([]map[string]interface{}, 0, len(m.Parts))
	for _, part := range m.Parts {
		partMap := partToMap(part)
		if partMap != nil {
			parts = append(parts, partMap)
		}
	}

	return MessageResponse{
		ID:               m.ID,
		SessionID:        m.SessionID,
		Role:             string(m.Role),
		Content:          m.Content().String(),
		Model:            model,
		Provider:         provider,
		IsSummaryMessage: m.IsSummaryMessage,
		CreatedAt:        m.CreatedAt,
		UpdatedAt:        m.UpdatedAt,
		FinishedAt:       finishedAt,
		FinishReason:     finishReason,
		Parts:            parts,
	}
}

// partToMap converts a ContentPart to a map for JSON serialization
func partToMap(part message.ContentPart) map[string]interface{} {
	result := make(map[string]interface{})
	
	switch p := part.(type) {
	case message.TextContent:
		result["type"] = "text"
		result["text"] = p.Text
	case message.ReasoningContent:
		result["type"] = "reasoning"
		result["thinking"] = p.Thinking
		if p.Signature != "" {
			result["signature"] = p.Signature
		}
		if p.ThoughtSignature != "" {
			result["thought_signature"] = p.ThoughtSignature
		}
		if p.ToolID != "" {
			result["tool_id"] = p.ToolID
		}
		if p.StartedAt > 0 {
			result["started_at"] = p.StartedAt
		}
		if p.FinishedAt > 0 {
			result["finished_at"] = p.FinishedAt
		}
	case message.ImageURLContent:
		result["type"] = "image_url"
		result["url"] = p.URL
		if p.Detail != "" {
			result["detail"] = p.Detail
		}
	case message.BinaryContent:
		result["type"] = "binary"
		result["path"] = p.Path
		result["mime_type"] = p.MIMEType
		// Data is []byte, encode as base64 string for JSON
		if len(p.Data) > 0 {
			result["data"] = base64.StdEncoding.EncodeToString(p.Data)
		}
	case message.ToolCall:
		result["type"] = "tool_call"
		result["id"] = p.ID
		result["name"] = p.Name
		result["input"] = p.Input
		result["provider_executed"] = p.ProviderExecuted
		result["finished"] = p.Finished
	case message.ToolResult:
		result["type"] = "tool_result"
		result["tool_call_id"] = p.ToolCallID
		result["name"] = p.Name
		result["content"] = p.Content
		if p.Data != "" {
			result["data"] = p.Data
		}
		if p.MIMEType != "" {
			result["mime_type"] = p.MIMEType
		}
		if p.Metadata != "" {
			result["metadata"] = p.Metadata
		}
		result["is_error"] = p.IsError
	case message.Finish:
		result["type"] = "finish"
		result["reason"] = string(p.Reason)
		result["time"] = p.Time
		if p.Message != "" {
			result["message"] = p.Message
		}
		if p.Details != "" {
			result["details"] = p.Details
		}
	default:
		return nil
	}
	
	return result
}
