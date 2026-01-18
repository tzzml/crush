package models

import (
	"time"

	"github.com/charmbracelet/crush/internal/message"
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

type OpenProjectResponse struct {
	ProjectPath string `json:"project_path"`
	Status      string `json:"status"`
}

type CloseProjectResponse struct {
	ProjectPath string `json:"project_path"`
	Status      string `json:"status"`
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

type SessionDetailResponse struct {
	Session SessionResponse `json:"session"`
}

// Messages API

type MessagesResponse struct {
	Messages []MessageResponse `json:"messages"`
	Total    int               `json:"total"`
}

type MessageResponse struct {
	ID               string `json:"id"`
	SessionID        string `json:"session_id"`
	Role             string `json:"role"`
	Content          string `json:"content"`
	Model            string `json:"model,omitempty"`
	Provider         string `json:"provider,omitempty"`
	IsSummaryMessage bool   `json:"is_summary_message"`
	CreatedAt        int64  `json:"created_at"`
	UpdatedAt        int64  `json:"updated_at"`
	FinishedAt       *int64 `json:"finished_at,omitempty"`
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
	if finish := m.FinishPart(); finish != nil && finish.Time > 0 {
		ts := finish.Time
		finishedAt = &ts
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
	}
}
