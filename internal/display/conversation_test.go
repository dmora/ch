package display

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"github.com/dmora/ch/internal/history"
	"github.com/dmora/ch/internal/jsonl"
)

func TestNewConversationDisplay(t *testing.T) {
	opts := DefaultConversationDisplayOptions()
	disp := NewConversationDisplay(opts)

	if disp == nil {
		t.Error("NewConversationDisplay returned nil")
	}
	if disp.opts.Writer == nil {
		t.Error("Writer should not be nil")
	}
}

func TestConversationDisplay_Render(t *testing.T) {
	conv := &history.Conversation{
		Meta: history.ConversationMeta{
			ID:          "abc123",
			SessionID:   "abc123",
			Path:        "/path/to/conv.jsonl",
			ProjectPath: "/Users/test/project",
			Timestamp:   time.Now(),
			MessageCount: 2,
		},
		Entries: []*jsonl.RawEntry{
			{
				Type:      jsonl.EntryTypeUser,
				Timestamp: "2024-01-01T10:00:00Z",
				Message:   json.RawMessage(`{"role":"user","content":"Hello"}`),
			},
			{
				Type:      jsonl.EntryTypeAssistant,
				Timestamp: "2024-01-01T10:00:01Z",
				Message:   json.RawMessage(`{"role":"assistant","content":[{"type":"text","text":"Hi!"}]}`),
			},
		},
	}

	t.Run("formatted output", func(t *testing.T) {
		var buf bytes.Buffer
		disp := NewConversationDisplay(ConversationDisplayOptions{Writer: &buf})
		err := disp.Render(conv)
		if err != nil {
			t.Fatalf("Render() error = %v", err)
		}
		output := buf.String()
		if output == "" {
			t.Error("Render() produced empty output")
		}
		if !bytes.Contains(buf.Bytes(), []byte("User")) {
			t.Error("Output should contain 'User'")
		}
	})

	t.Run("JSON output", func(t *testing.T) {
		var buf bytes.Buffer
		disp := NewConversationDisplay(ConversationDisplayOptions{Writer: &buf, JSON: true})
		err := disp.Render(conv)
		if err != nil {
			t.Fatalf("Render() error = %v", err)
		}

		var result map[string]interface{}
		if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
			t.Fatalf("Failed to parse JSON: %v", err)
		}
		if result["id"] != "abc123" {
			t.Errorf("Expected id 'abc123', got %v", result["id"])
		}
	})

	t.Run("with thinking", func(t *testing.T) {
		convWithThinking := &history.Conversation{
			Meta: conv.Meta,
			Entries: []*jsonl.RawEntry{
				{
					Type:      jsonl.EntryTypeAssistant,
					Timestamp: "2024-01-01T10:00:00Z",
					Message:   json.RawMessage(`{"role":"assistant","content":[{"type":"thinking","thinking":"Let me think..."},{"type":"text","text":"Here's my answer."}]}`),
				},
			},
		}

		var buf bytes.Buffer
		disp := NewConversationDisplay(ConversationDisplayOptions{Writer: &buf, ShowThinking: true})
		err := disp.Render(convWithThinking)
		if err != nil {
			t.Fatalf("Render() error = %v", err)
		}
		if !bytes.Contains(buf.Bytes(), []byte("Thinking")) {
			t.Error("Output should contain 'Thinking' when ShowThinking is true")
		}
	})

	t.Run("with tools", func(t *testing.T) {
		convWithTools := &history.Conversation{
			Meta: conv.Meta,
			Entries: []*jsonl.RawEntry{
				{
					Type:      jsonl.EntryTypeAssistant,
					Timestamp: "2024-01-01T10:00:00Z",
					Message:   json.RawMessage(`{"role":"assistant","content":[{"type":"tool_use","name":"read_file","id":"tool1","input":{"path":"/test.txt"}}]}`),
				},
			},
		}

		var buf bytes.Buffer
		disp := NewConversationDisplay(ConversationDisplayOptions{Writer: &buf, ShowTools: true})
		err := disp.Render(convWithTools)
		if err != nil {
			t.Fatalf("Render() error = %v", err)
		}
		if !bytes.Contains(buf.Bytes(), []byte("Tool")) {
			t.Error("Output should contain 'Tool' when ShowTools is true")
		}
	})
}

func TestConversationDisplay_RenderRaw(t *testing.T) {
	// Create a temp file with test content
	conv := &history.Conversation{
		Meta: history.ConversationMeta{
			Path: "/nonexistent/path.jsonl",
		},
	}

	var buf bytes.Buffer
	disp := NewConversationDisplay(ConversationDisplayOptions{Writer: &buf, Raw: true})

	// This will fail because the file doesn't exist
	err := disp.Render(conv)
	if err == nil {
		t.Error("Expected error for nonexistent file in raw mode")
	}
}

func TestRenderAgentList(t *testing.T) {
	agents := []*history.ConversationMeta{
		{
			ID:           "xyz789",
			Timestamp:    time.Now(),
			MessageCount: 5,
			Preview:      "Agent task 1",
		},
		{
			ID:           "abc456",
			Timestamp:    time.Now().Add(-time.Hour),
			MessageCount: 3,
			Preview:      "Agent task 2",
		},
	}

	t.Run("table output", func(t *testing.T) {
		var buf bytes.Buffer
		err := RenderAgentList(&buf, agents, "parent123", false, "")
		if err != nil {
			t.Fatalf("RenderAgentList() error = %v", err)
		}
		output := buf.String()
		if !bytes.Contains(buf.Bytes(), []byte("xyz789")) {
			t.Errorf("Expected 'xyz789' in output, got: %s", output)
		}
	})

	t.Run("JSON output", func(t *testing.T) {
		var buf bytes.Buffer
		err := RenderAgentList(&buf, agents, "parent123", true, "")
		if err != nil {
			t.Fatalf("RenderAgentList() error = %v", err)
		}

		var result []map[string]interface{}
		if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
			t.Fatalf("Failed to parse JSON: %v", err)
		}
		if len(result) != 2 {
			t.Errorf("Expected 2 agents, got %d", len(result))
		}
	})

	t.Run("empty list", func(t *testing.T) {
		var buf bytes.Buffer
		err := RenderAgentList(&buf, []*history.ConversationMeta{}, "parent123", false, "")
		if err != nil {
			t.Fatalf("RenderAgentList() error = %v", err)
		}
		if !bytes.Contains(buf.Bytes(), []byte("No agents")) {
			t.Error("Expected 'No agents' message for empty list")
		}
	})

	t.Run("empty list with filter", func(t *testing.T) {
		var buf bytes.Buffer
		err := RenderAgentList(&buf, []*history.ConversationMeta{}, "parent123", false, "test-type")
		if err != nil {
			t.Fatalf("RenderAgentList() error = %v", err)
		}
		if !bytes.Contains(buf.Bytes(), []byte("test-type")) {
			t.Error("Expected filter type in 'No agents' message")
		}
	})
}

func TestRenderStats(t *testing.T) {
	stats := &Stats{
		ProjectCount:       5,
		ConversationCount:  100,
		AgentCount:         50,
		TotalMessages:      1000,
		TotalSize:          1024 * 1024 * 100, // 100 MB
		OldestConversation: "2024-01-01 10:00",
		NewestConversation: "2024-06-01 10:00",
	}

	t.Run("table output", func(t *testing.T) {
		var buf bytes.Buffer
		err := RenderStats(&buf, stats, false)
		if err != nil {
			t.Fatalf("RenderStats() error = %v", err)
		}
		output := buf.String()
		if !bytes.Contains(buf.Bytes(), []byte("Projects:")) {
			t.Errorf("Expected 'Projects:' in output, got: %s", output)
		}
	})

	t.Run("JSON output", func(t *testing.T) {
		var buf bytes.Buffer
		err := RenderStats(&buf, stats, true)
		if err != nil {
			t.Fatalf("RenderStats() error = %v", err)
		}

		var result map[string]interface{}
		if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
			t.Fatalf("Failed to parse JSON: %v", err)
		}
		if result["project_count"].(float64) != 5 {
			t.Errorf("Expected project_count 5, got %v", result["project_count"])
		}
	})
}

func TestDefaultConversationDisplayOptions(t *testing.T) {
	opts := DefaultConversationDisplayOptions()
	if opts.Writer == nil {
		t.Error("Writer should not be nil")
	}
	if opts.ShowThinking {
		t.Error("ShowThinking should be false by default")
	}
	if opts.ShowTools {
		t.Error("ShowTools should be false by default")
	}
}
