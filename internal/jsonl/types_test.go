package jsonl

import (
	"encoding/json"
	"testing"
)

func TestEntryType_IsUserOrAssistant(t *testing.T) {
	tests := []struct {
		name     string
		entry    EntryType
		expected bool
	}{
		{"user", EntryTypeUser, true},
		{"assistant", EntryTypeAssistant, true},
		{"system", EntryTypeSystem, false},
		{"summary", EntryTypeSummary, false},
		{"file-snapshot", EntryTypeFileSnapshot, false},
		{"queue-op", EntryTypeQueueOp, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.entry.IsUserOrAssistant(); got != tt.expected {
				t.Errorf("IsUserOrAssistant() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestEntryType_IsMessage(t *testing.T) {
	tests := []struct {
		name     string
		entry    EntryType
		expected bool
	}{
		{"user", EntryTypeUser, true},
		{"assistant", EntryTypeAssistant, true},
		{"system", EntryTypeSystem, true},
		{"summary", EntryTypeSummary, false},
		{"file-snapshot", EntryTypeFileSnapshot, false},
		{"queue-op", EntryTypeQueueOp, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.entry.IsMessage(); got != tt.expected {
				t.Errorf("IsMessage() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestMessage_UnmarshalJSON_StringContent(t *testing.T) {
	data := `{"role":"user","content":"Hello, world!"}`

	var msg Message
	if err := json.Unmarshal([]byte(data), &msg); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if msg.Role != "user" {
		t.Errorf("Role = %q, want %q", msg.Role, "user")
	}
	if len(msg.Content) != 1 {
		t.Fatalf("Content length = %d, want 1", len(msg.Content))
	}
	if msg.Content[0].Type != BlockTypeText {
		t.Errorf("Content[0].Type = %q, want %q", msg.Content[0].Type, BlockTypeText)
	}
	if msg.Content[0].Text != "Hello, world!" {
		t.Errorf("Content[0].Text = %q, want %q", msg.Content[0].Text, "Hello, world!")
	}
}

func TestMessage_UnmarshalJSON_ArrayContent(t *testing.T) {
	data := `{"role":"assistant","model":"claude-3","content":[{"type":"text","text":"Hello!"},{"type":"thinking","thinking":"Let me think..."}]}`

	var msg Message
	if err := json.Unmarshal([]byte(data), &msg); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if msg.Role != "assistant" {
		t.Errorf("Role = %q, want %q", msg.Role, "assistant")
	}
	if msg.Model != "claude-3" {
		t.Errorf("Model = %q, want %q", msg.Model, "claude-3")
	}
	if len(msg.Content) != 2 {
		t.Fatalf("Content length = %d, want 2", len(msg.Content))
	}
	if msg.Content[0].Text != "Hello!" {
		t.Errorf("Content[0].Text = %q, want %q", msg.Content[0].Text, "Hello!")
	}
	if msg.Content[1].Thinking != "Let me think..." {
		t.Errorf("Content[1].Thinking = %q, want %q", msg.Content[1].Thinking, "Let me think...")
	}
}

func TestMessage_UnmarshalJSON_EmptyContent(t *testing.T) {
	data := `{"role":"user"}`

	var msg Message
	if err := json.Unmarshal([]byte(data), &msg); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if msg.Role != "user" {
		t.Errorf("Role = %q, want %q", msg.Role, "user")
	}
	if msg.Content != nil {
		t.Errorf("Content = %v, want nil", msg.Content)
	}
}

func TestRawEntry_Unmarshal(t *testing.T) {
	data := `{"type":"user","timestamp":"2024-01-01T00:00:00Z","sessionId":"abc123","message":{"role":"user","content":"test"}}`

	var entry RawEntry
	if err := json.Unmarshal([]byte(data), &entry); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if entry.Type != EntryTypeUser {
		t.Errorf("Type = %q, want %q", entry.Type, EntryTypeUser)
	}
	if entry.Timestamp != "2024-01-01T00:00:00Z" {
		t.Errorf("Timestamp = %q, want %q", entry.Timestamp, "2024-01-01T00:00:00Z")
	}
	if entry.SessionID != "abc123" {
		t.Errorf("SessionID = %q, want %q", entry.SessionID, "abc123")
	}
	if entry.Message == nil {
		t.Error("Message should not be nil")
	}
}

func TestRawEntry_Agent(t *testing.T) {
	data := `{"type":"assistant","sessionId":"parent123","agentId":"agent456","isSidechain":true}`

	var entry RawEntry
	if err := json.Unmarshal([]byte(data), &entry); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if entry.Type != EntryTypeAssistant {
		t.Errorf("Type = %q, want %q", entry.Type, EntryTypeAssistant)
	}
	if entry.SessionID != "parent123" {
		t.Errorf("SessionID = %q, want %q", entry.SessionID, "parent123")
	}
	if entry.AgentID != "agent456" {
		t.Errorf("AgentID = %q, want %q", entry.AgentID, "agent456")
	}
	if !entry.IsSidechain {
		t.Error("IsSidechain should be true")
	}
}
