package models

import (
	"encoding/base64"
	"strings"

	internalmsg "github.com/charmbracelet/crush/internal/message"
)

// MessageToPromptResponse converts a Zorkagent internal Message to Opencode-compatible PromptResponse
func MessageToPromptResponse(msg internalmsg.Message) PromptResponse {
	info := AssistantMessage{
		ID:        msg.ID,
		SessionID: msg.SessionID,
		Role:      "assistant",
		Time: MessageTime{
			Created: msg.CreatedAt,
		},
		ModelID:    msg.Model,
		ProviderID: msg.Provider,
	}

	// Set completed time and finish reason if available
	if finish := msg.FinishPart(); finish != nil {
		if finish.Time > 0 {
			info.Time.Completed = finish.Time
		}
		if finish.Reason != "" {
			info.Finish = string(finish.Reason)
		}
	}

	// Convert parts
	parts := make([]Part, 0, len(msg.Parts))
	for _, p := range msg.Parts {
		part := contentPartToPart(p, msg)
		if part != nil {
			parts = append(parts, part)
		}
	}

	return PromptResponse{
		Info:  info,
		Parts: parts,
	}
}

// contentPartToPart converts internal ContentPart to Opencode Part
func contentPartToPart(contentPart internalmsg.ContentPart, _ internalmsg.Message) Part {
	switch p := contentPart.(type) {
	case internalmsg.TextContent:
		return TextPart{
			Type: "text",
			Text: p.Text,
		}

	case internalmsg.ReasoningContent:
		reasoningPart := ReasoningPart{
			Type: "reasoning",
			Text: p.Thinking,
		}
		if p.StartedAt > 0 || p.FinishedAt > 0 {
			reasoningPart.Time = &MessageTime{
				Created:   p.StartedAt,
				Completed: p.FinishedAt,
			}
		}
		return reasoningPart

	case internalmsg.BinaryContent:
		return FilePart{
			Type:     "file",
			Name:     p.Path,
			Data:     base64.StdEncoding.EncodeToString(p.Data),
			MIMEType: p.MIMEType,
		}

	case internalmsg.ToolCall:
		return ToolPart{
			Type:  "tool",
			Name:  p.Name,
			Input: p.Input,
		}

	case internalmsg.ToolResult:
		resultPart := ToolResultPart{
			Type: "tool_result",
			Name: p.Name,
		}
		if p.IsError {
			resultPart.Error = p.Content
		} else {
			resultPart.Output = p.Content
		}
		return resultPart

	case internalmsg.Finish:
		// Finish is represented in the info.finish field, not as a separate part
		return nil

	default:
		// Unknown part type, skip it
		return nil
	}
}

// ExtractPromptTextFromParts extracts the text content from PartInput array
// This is used to get the prompt text for the internal AI runner
func ExtractPromptTextFromParts(parts []PartInput) string {
	var text strings.Builder
	for _, part := range parts {
		if p, ok := part.(TextPartInput); ok {
			text.WriteString(p.Text)
		}
	}
	return text.String()
}

// PartsToMessage converts Opencode PartInput array to internal message parts
func PartsToMessageParts(sessionID string, parts []PartInput) []internalmsg.ContentPart {
	contentParts := make([]internalmsg.ContentPart, 0, len(parts))

	for _, part := range parts {
		switch p := part.(type) {
		case TextPartInput:
			contentParts = append(contentParts, internalmsg.TextContent{
				Text: p.Text,
			})

		case FilePartInput:
			data, err := base64.StdEncoding.DecodeString(p.Data)
			if err != nil {
				// Skip invalid base64 data
				continue
			}
			contentParts = append(contentParts, internalmsg.BinaryContent{
				Path:     p.Name,
				MIMEType: detectMIMEType(p.Name),
				Data:     data,
			})

		case AgentPartInput:
			// Agent parts are handled as text with special formatting
			text := p.Prompt
			if p.Agent != "" {
				text = "[Agent: " + p.Agent + "] " + text
			}
			contentParts = append(contentParts, internalmsg.TextContent{
				Text: text,
			})

		case SubtaskPartInput:
			// Subtask parts are handled as text with special formatting
			text := p.Prompt
			if p.Agent != "" {
				text = "[Subtask: " + p.Agent + "] " + text
			}
			contentParts = append(contentParts, internalmsg.TextContent{
				Text: text,
			})
		}
	}

	return contentParts
}

// detectMIMEType detects MIME type based on file extension
func detectMIMEType(filename string) string {
	if strings.HasSuffix(filename, ".txt") {
		return "text/plain"
	} else if strings.HasSuffix(filename, ".json") {
		return "application/json"
	} else if strings.HasSuffix(filename, ".xml") {
		return "application/xml"
	} else if strings.HasSuffix(filename, ".html") {
		return "text/html"
	} else if strings.HasSuffix(filename, ".css") {
		return "text/css"
	} else if strings.HasSuffix(filename, ".js") {
		return "application/javascript"
	} else if strings.HasSuffix(filename, ".ts") {
		return "application/typescript"
	} else if strings.HasSuffix(filename, ".md") {
		return "text/markdown"
	} else if strings.HasSuffix(filename, ".pdf") {
		return "application/pdf"
	} else if strings.HasSuffix(filename, ".png") {
		return "image/png"
	} else if strings.HasSuffix(filename, ".jpg") || strings.HasSuffix(filename, ".jpeg") {
		return "image/jpeg"
	} else if strings.HasSuffix(filename, ".gif") {
		return "image/gif"
	} else if strings.HasSuffix(filename, ".svg") {
		return "image/svg+xml"
	} else if strings.HasSuffix(filename, ".mp3") {
		return "audio/mpeg"
	} else if strings.HasSuffix(filename, ".mp4") {
		return "video/mp4"
	} else if strings.HasSuffix(filename, ".zip") {
		return "application/zip"
	}
	// Default to binary
	return "application/octet-stream"
}
