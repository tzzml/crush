package models

// Opencode-compatible types for the /prompt endpoint

// PromptRequest represents the request body for the /prompt endpoint
type PromptRequest struct {
	MessageID string      `json:"messageID,omitempty"`
	Model     *ModelSpec  `json:"model,omitempty"`
	Agent     string      `json:"agent,omitempty"`
	NoReply   bool        `json:"noReply,omitempty"`
	Parts     []PartInput `json:"parts"`
}

// ModelSpec specifies the AI model to use
type ModelSpec struct {
	ProviderID string `json:"providerID"`
	ModelID    string `json:"modelID"`
}

// PartInput represents input part types (request side)
type PartInput interface {
	isPartInput()
}

// TextPartInput represents a text part in the request
type TextPartInput struct {
	Text string `json:"text"`
}

func (TextPartInput) isPartInput() {}

// FilePartInput represents a file attachment in the request
type FilePartInput struct {
	Name string `json:"name"`
	Data string `json:"data"` // base64 encoded
}

func (FilePartInput) isPartInput() {}

// AgentPartInput represents an agent invocation in the request
type AgentPartInput struct {
	Prompt  string     `json:"prompt"`
	Agent   string     `json:"agent,omitempty"`
	Model   *ModelSpec `json:"model,omitempty"`
	Command string     `json:"command,omitempty"`
}

func (AgentPartInput) isPartInput() {}

// SubtaskPartInput represents a subtask in the request
type SubtaskPartInput struct {
	Prompt     string     `json:"prompt"`
	Agent      string     `json:"agent"`
	Descriptor string     `json:"descriptor,omitempty"`
	Model      *ModelSpec `json:"model,omitempty"`
	Command    string     `json:"command,omitempty"`
}

func (SubtaskPartInput) isPartInput() {}

// PromptResponse represents the response from the /prompt endpoint
type PromptResponse struct {
	Info  AssistantMessage `json:"info"`
	Parts []Part           `json:"parts"`
}

// AssistantMessage contains metadata about the assistant's response
type AssistantMessage struct {
	ID         string      `json:"id"`
	SessionID  string      `json:"sessionID"`
	Role       string      `json:"role"` // "assistant"
	Time       MessageTime `json:"time"`
	ParentID   string      `json:"parentID,omitempty"`
	ModelID    string      `json:"modelID,omitempty"`
	ProviderID string      `json:"providerID,omitempty"`
	Agent      string      `json:"agent,omitempty"`
	Cost       float64     `json:"cost,omitempty"`
	Tokens     *TokenInfo  `json:"tokens,omitempty"`
	Finish     string      `json:"finish,omitempty"`
}

// MessageTime contains timestamp information
type MessageTime struct {
	Created   int64 `json:"created"`
	Completed int64 `json:"completed,omitempty"`
}

// TokenInfo contains token usage information
type TokenInfo struct {
	Input     int64           `json:"input"`
	Output    int64           `json:"output"`
	Reasoning int64           `json:"reasoning,omitempty"`
	Cache     *CacheTokenInfo `json:"cache,omitempty"`
}

// CacheTokenInfo contains cache token usage
type CacheTokenInfo struct {
	Read  int64 `json:"read,omitempty"`
	Write int64 `json:"write,omitempty"`
}

// Part represents output part types (response side)
type Part interface {
	isPart()
	GetPartType() string
}

// TextPart represents a text part in the response
type TextPart struct {
	ID        string `json:"id,omitempty"`
	SessionID string `json:"sessionID,omitempty"`
	MessageID string `json:"messageID,omitempty"`
	Type      string `json:"type"` // "text"
	Text      string `json:"text"`
}

func (TextPart) isPart()               {}
func (TextPart) GetPartType() string   { return "text" }

// ReasoningPart represents reasoning/thinking content
type ReasoningPart struct {
	ID        string      `json:"id,omitempty"`
	SessionID string      `json:"sessionID,omitempty"`
	MessageID string      `json:"messageID,omitempty"`
	Type      string      `json:"type"` // "reasoning"
	Text      string      `json:"text"`
	Time      *MessageTime `json:"time,omitempty"`
}

func (ReasoningPart) isPart()               {}
func (ReasoningPart) GetPartType() string   { return "reasoning" }

// FilePart represents a file attachment in the response
type FilePart struct {
	ID        string `json:"id,omitempty"`
	SessionID string `json:"sessionID,omitempty"`
	MessageID string `json:"messageID,omitempty"`
	Type      string `json:"type"` // "file"
	Name      string `json:"name"`
	Data      string `json:"data"` // base64 encoded
	MIMEType  string `json:"mimeType,omitempty"`
}

func (FilePart) isPart()               {}
func (FilePart) GetPartType() string   { return "file" }

// ToolPart represents a tool call
type ToolPart struct {
	ID        string `json:"id,omitempty"`
	SessionID string `json:"sessionID,omitempty"`
	MessageID string `json:"messageID,omitempty"`
	Type      string `json:"type"` // "tool"
	Name      string `json:"name"`
	Input     string `json:"input"`
}

func (ToolPart) isPart()               {}
func (ToolPart) GetPartType() string   { return "tool" }

// ToolResultPart represents the result of a tool call
type ToolResultPart struct {
	ID        string `json:"id,omitempty"`
	SessionID string `json:"sessionID,omitempty"`
	MessageID string `json:"messageID,omitempty"`
	Type      string `json:"type"` // "tool_result"
	Name      string `json:"name"`
	Output    string `json:"output,omitempty"`
	Error     string `json:"error,omitempty"`
}

func (ToolResultPart) isPart()               {}
func (ToolResultPart) GetPartType() string   { return "tool_result" }

// StepStartPart marks the start of a step
type StepStartPart struct {
	ID        string `json:"id,omitempty"`
	SessionID string `json:"sessionID,omitempty"`
	MessageID string `json:"messageID,omitempty"`
	Type      string `json:"type"` // "step_start"
	Name      string `json:"name"`
	Input     string `json:"input,omitempty"`
}

func (StepStartPart) isPart()               {}
func (StepStartPart) GetPartType() string   { return "step_start" }

// StepFinishPart marks the completion of a step
type StepFinishPart struct {
	ID        string `json:"id,omitempty"`
	SessionID string `json:"sessionID,omitempty"`
	MessageID string `json:"messageID,omitempty"`
	Type      string `json:"type"` // "step_finish"
	Name      string `json:"name"`
	Output    string `json:"output,omitempty"`
	Error     string `json:"error,omitempty"`
}

func (StepFinishPart) isPart()               {}
func (StepFinishPart) GetPartType() string   { return "step_finish" }

// SnapshotPart represents a code snapshot
type SnapshotPart struct {
	ID        string `json:"id,omitempty"`
	SessionID string `json:"sessionID,omitempty"`
	MessageID string `json:"messageID,omitempty"`
	Type      string `json:"type"` // "snapshot"
	Content   string `json:"content"`
}

func (SnapshotPart) isPart()               {}
func (SnapshotPart) GetPartType() string   { return "snapshot" }

// PatchPart represents a code patch
type PatchPart struct {
	ID        string `json:"id,omitempty"`
	SessionID string `json:"sessionID,omitempty"`
	MessageID string `json:"messageID,omitempty"`
	Type      string `json:"type"` // "patch"`
	Content   string `json:"content"`
}

func (PatchPart) isPart()               {}
func (PatchPart) GetPartType() string   { return "patch" }

// AgentPart represents an agent invocation
type AgentPart struct {
	ID        string `json:"id,omitempty"`
	SessionID string `json:"sessionID,omitempty"`
	MessageID string `json:"messageID,omitempty"`
	Type      string `json:"type"` // "agent"
	Prompt    string `json:"prompt"`
	Agent     string `json:"agent"`
}

func (AgentPart) isPart()               {}
func (AgentPart) GetPartType() string   { return "agent" }

// RetryPart indicates a retry operation
type RetryPart struct {
	ID        string `json:"id,omitempty"`
	SessionID string `json:"sessionID,omitempty"`
	MessageID string `json:"messageID,omitempty"`
	Type      string `json:"type"` // "retry"
	Reason    string `json:"reason,omitempty"`
}

func (RetryPart) isPart()               {}
func (RetryPart) GetPartType() string   { return "retry" }

// CompactionPart indicates message compaction
type CompactionPart struct {
	ID        string `json:"id,omitempty"`
	SessionID string `json:"sessionID,omitempty"`
	MessageID string `json:"messageID,omitempty"`
	Type      string `json:"type"` // "compaction"`
}

func (CompactionPart) isPart()               {}
func (CompactionPart) GetPartType() string   { return "compaction" }
