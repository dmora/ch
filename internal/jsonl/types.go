// Package jsonl provides types and utilities for parsing Claude Code JSONL conversation files.
package jsonl

import "encoding/json"

// EntryType represents the type of a JSONL entry.
type EntryType string

const (
	EntryTypeUser         EntryType = "user"
	EntryTypeAssistant    EntryType = "assistant"
	EntryTypeSummary      EntryType = "summary"
	EntryTypeSystem       EntryType = "system"
	EntryTypeFileSnapshot EntryType = "file-history-snapshot"
	EntryTypeQueueOp      EntryType = "queue-operation"
)

// ContentBlockType represents the type of a content block within a message.
type ContentBlockType string

const (
	BlockTypeText       ContentBlockType = "text"
	BlockTypeThinking   ContentBlockType = "thinking"
	BlockTypeToolUse    ContentBlockType = "tool_use"
	BlockTypeToolResult ContentBlockType = "tool_result"
	BlockTypeImage      ContentBlockType = "image"
)

// RawEntry represents a raw JSON entry with minimal parsing.
// The Message field is kept as json.RawMessage for deferred parsing.
type RawEntry struct {
	Type        EntryType       `json:"type"`
	Timestamp   string          `json:"timestamp,omitempty"`
	UUID        string          `json:"uuid,omitempty"`
	ParentUUID  string          `json:"parentUuid,omitempty"`
	SessionID   string          `json:"sessionId,omitempty"`
	IsSidechain bool            `json:"isSidechain,omitempty"`
	AgentID     string          `json:"agentId,omitempty"`
	CWD         string          `json:"cwd,omitempty"`
	Message     json.RawMessage `json:"message,omitempty"`
	Summary     string          `json:"summary,omitempty"`
}

// Message represents a fully parsed message with role and content blocks.
type Message struct {
	Role    string         `json:"role"`
	Model   string         `json:"model,omitempty"`
	Content []ContentBlock `json:"-"` // Custom unmarshaling
}

// UnmarshalJSON implements custom JSON unmarshaling to handle content as string or array.
func (m *Message) UnmarshalJSON(data []byte) error {
	// Use an alias to avoid recursion
	type Alias Message
	aux := &struct {
		Content json.RawMessage `json:"content"`
		*Alias
	}{
		Alias: (*Alias)(m),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	// Try to unmarshal content as string first
	var contentStr string
	if err := json.Unmarshal(aux.Content, &contentStr); err == nil {
		// Content is a string, wrap in a text block
		m.Content = []ContentBlock{{Type: BlockTypeText, Text: contentStr}}
		return nil
	}

	// Try to unmarshal as array of content blocks
	var contentBlocks []ContentBlock
	if err := json.Unmarshal(aux.Content, &contentBlocks); err == nil {
		m.Content = contentBlocks
		return nil
	}

	// If neither works, leave content empty
	m.Content = nil
	return nil
}

// ContentBlock represents a single content block within a message.
type ContentBlock struct {
	Type      ContentBlockType `json:"type"`
	Text      string           `json:"text,omitempty"`
	Thinking  string           `json:"thinking,omitempty"`
	ID        string           `json:"id,omitempty"`
	Name      string           `json:"name,omitempty"`
	Input     json.RawMessage  `json:"input,omitempty"`
	ToolUseID string           `json:"tool_use_id,omitempty"`
	Content   json.RawMessage  `json:"content,omitempty"`
	IsError   bool             `json:"is_error,omitempty"`
}

// IsUserOrAssistant returns true if the entry type is user or assistant.
func (e EntryType) IsUserOrAssistant() bool {
	return e == EntryTypeUser || e == EntryTypeAssistant
}

// IsMessage returns true if the entry type represents a conversation message.
func (e EntryType) IsMessage() bool {
	return e == EntryTypeUser || e == EntryTypeAssistant || e == EntryTypeSystem
}
