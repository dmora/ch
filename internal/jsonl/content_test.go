package jsonl

import (
	"encoding/json"
	"testing"
)

func TestExtractText(t *testing.T) {
	msg := &Message{
		Content: []ContentBlock{
			{Type: BlockTypeText, Text: "Hello"},
			{Type: BlockTypeThinking, Thinking: "Let me think..."},
			{Type: BlockTypeText, Text: "World"},
		},
	}

	text := ExtractText(msg)
	if text != "Hello\nWorld" {
		t.Errorf("ExtractText() = %q, want %q", text, "Hello\nWorld")
	}
}

func TestExtractText_Nil(t *testing.T) {
	text := ExtractText(nil)
	if text != "" {
		t.Errorf("ExtractText(nil) = %q, want %q", text, "")
	}
}

func TestExtractThinking(t *testing.T) {
	msg := &Message{
		Content: []ContentBlock{
			{Type: BlockTypeText, Text: "Hello"},
			{Type: BlockTypeThinking, Thinking: "Thought 1"},
			{Type: BlockTypeThinking, Thinking: "Thought 2"},
		},
	}

	thinking := ExtractThinking(msg)
	if thinking != "Thought 1\nThought 2" {
		t.Errorf("ExtractThinking() = %q, want %q", thinking, "Thought 1\nThought 2")
	}
}

func TestExtractToolCalls(t *testing.T) {
	msg := &Message{
		Content: []ContentBlock{
			{Type: BlockTypeText, Text: "I'll use some tools"},
			{Type: BlockTypeToolUse, Name: "read_file", ID: "1"},
			{Type: BlockTypeToolUse, Name: "write_file", ID: "2"},
		},
	}

	tools := ExtractToolCalls(msg)
	if len(tools) != 2 {
		t.Fatalf("len(ExtractToolCalls()) = %d, want 2", len(tools))
	}
	if tools[0] != "read_file" {
		t.Errorf("tools[0] = %q, want %q", tools[0], "read_file")
	}
	if tools[1] != "write_file" {
		t.Errorf("tools[1] = %q, want %q", tools[1], "write_file")
	}
}

func TestExtractPreview(t *testing.T) {
	tests := []struct {
		name     string
		message  string
		maxLen   int
		expected string
	}{
		{
			name:     "short text",
			message:  `{"role":"user","content":"Hello world"}`,
			maxLen:   100,
			expected: "Hello world",
		},
		{
			name:     "long text truncated",
			message:  `{"role":"user","content":"This is a very long message that should be truncated because it exceeds the maximum length"}`,
			maxLen:   50,
			expected: "This is a very long message that should be trun...",
		},
		{
			name:     "with newlines",
			message:  `{"role":"user","content":"Line 1\nLine 2\nLine 3"}`,
			maxLen:   100,
			expected: "Line 1 Line 2 Line 3",
		},
		{
			name:     "with tabs",
			message:  `{"role":"user","content":"Col1\tCol2\tCol3"}`,
			maxLen:   100,
			expected: "Col1 Col2 Col3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractPreview(json.RawMessage(tt.message), tt.maxLen)
			if result != tt.expected {
				t.Errorf("ExtractPreview() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestExtractPreview_ArrayContent(t *testing.T) {
	message := `{"role":"user","content":[{"type":"text","text":"Hello from array"}]}`
	result := ExtractPreview(json.RawMessage(message), 100)
	if result != "Hello from array" {
		t.Errorf("ExtractPreview() = %q, want %q", result, "Hello from array")
	}
}

func TestExtractPreview_Nil(t *testing.T) {
	result := ExtractPreview(nil, 100)
	if result != "" {
		t.Errorf("ExtractPreview(nil) = %q, want %q", result, "")
	}
}

func TestHasToolCalls(t *testing.T) {
	tests := []struct {
		name     string
		msg      *Message
		expected bool
	}{
		{
			name: "has tool calls",
			msg: &Message{
				Content: []ContentBlock{
					{Type: BlockTypeToolUse, Name: "read_file"},
				},
			},
			expected: true,
		},
		{
			name: "no tool calls",
			msg: &Message{
				Content: []ContentBlock{
					{Type: BlockTypeText, Text: "Hello"},
				},
			},
			expected: false,
		},
		{
			name:     "nil message",
			msg:      nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := HasToolCalls(tt.msg); got != tt.expected {
				t.Errorf("HasToolCalls() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestHasThinking(t *testing.T) {
	tests := []struct {
		name     string
		msg      *Message
		expected bool
	}{
		{
			name: "has thinking",
			msg: &Message{
				Content: []ContentBlock{
					{Type: BlockTypeThinking, Thinking: "Let me think..."},
				},
			},
			expected: true,
		},
		{
			name: "no thinking",
			msg: &Message{
				Content: []ContentBlock{
					{Type: BlockTypeText, Text: "Hello"},
				},
			},
			expected: false,
		},
		{
			name:     "nil message",
			msg:      nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := HasThinking(tt.msg); got != tt.expected {
				t.Errorf("HasThinking() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestExtractToolCallDetails(t *testing.T) {
	msg := &Message{
		Content: []ContentBlock{
			{Type: BlockTypeToolUse, ID: "tool1", Name: "read_file", Input: json.RawMessage(`{"path":"/test.txt"}`)},
			{Type: BlockTypeText, Text: "Some text"},
			{Type: BlockTypeToolUse, ID: "tool2", Name: "write_file", Input: json.RawMessage(`{"path":"/out.txt","content":"hello"}`)},
		},
	}

	calls := ExtractToolCallDetails(msg)
	if len(calls) != 2 {
		t.Fatalf("len(calls) = %d, want 2", len(calls))
	}

	if calls[0].ID != "tool1" {
		t.Errorf("calls[0].ID = %q, want %q", calls[0].ID, "tool1")
	}
	if calls[0].Name != "read_file" {
		t.Errorf("calls[0].Name = %q, want %q", calls[0].Name, "read_file")
	}
	if calls[0].Input["path"] != "/test.txt" {
		t.Errorf("calls[0].Input[path] = %v, want %q", calls[0].Input["path"], "/test.txt")
	}
}

func TestExtractToolResults(t *testing.T) {
	msg := &Message{
		Content: []ContentBlock{
			{Type: BlockTypeToolResult, ToolUseID: "tool1", Content: json.RawMessage(`"file contents"`)},
			{Type: BlockTypeToolResult, ToolUseID: "tool2", IsError: true, Content: json.RawMessage(`"error message"`)},
		},
	}

	results := ExtractToolResults(msg)
	if len(results) != 2 {
		t.Fatalf("len(results) = %d, want 2", len(results))
	}

	if results[0].ToolUseID != "tool1" {
		t.Errorf("results[0].ToolUseID = %q, want %q", results[0].ToolUseID, "tool1")
	}
	if results[0].Content != "file contents" {
		t.Errorf("results[0].Content = %q, want %q", results[0].Content, "file contents")
	}
	if results[0].IsError {
		t.Error("results[0].IsError should be false")
	}

	if !results[1].IsError {
		t.Error("results[1].IsError should be true")
	}
}
