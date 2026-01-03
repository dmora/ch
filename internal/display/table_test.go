package display

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"github.com/dmora/ch/internal/history"
)

func TestConversationTable_Render(t *testing.T) {
	conversations := []*history.ConversationMeta{
		{
			ID:           "abc123-def456-789",
			SessionID:    "abc123-def456-789",
			Path:         "/path/to/conv.jsonl",
			Project:      "-Users-test-project",
			ProjectPath:  "/Users/test/project",
			Timestamp:    time.Now().Add(-1 * time.Hour),
			Preview:      "Hello, how are you?",
			MessageCount: 10,
			IsAgent:      false,
			AgentCount:   3,
		},
		{
			ID:           "def789",
			SessionID:    "abc123-def456-789",
			Path:         "/path/to/agent.jsonl",
			Project:      "-Users-test-project",
			ProjectPath:  "/Users/test/project",
			Timestamp:    time.Now().Add(-2 * time.Hour),
			Preview:      "Agent task",
			MessageCount: 5,
			IsAgent:      true,
		},
	}

	t.Run("table output", func(t *testing.T) {
		var buf bytes.Buffer
		table := NewConversationTable(TableOptions{Writer: &buf})
		err := table.Render(conversations)
		if err != nil {
			t.Fatalf("Render() error = %v", err)
		}
		output := buf.String()
		if output == "" {
			t.Error("Render() produced empty output")
		}
	})

	t.Run("JSON output", func(t *testing.T) {
		var buf bytes.Buffer
		table := NewConversationTable(TableOptions{Writer: &buf, JSON: true})
		err := table.Render(conversations)
		if err != nil {
			t.Fatalf("Render() error = %v", err)
		}

		var result []map[string]interface{}
		if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
			t.Fatalf("JSON unmarshal error = %v", err)
		}
		if len(result) != 2 {
			t.Errorf("JSON result length = %d, want 2", len(result))
		}
	})

	t.Run("empty list", func(t *testing.T) {
		var buf bytes.Buffer
		table := NewConversationTable(TableOptions{Writer: &buf})
		err := table.Render([]*history.ConversationMeta{})
		if err != nil {
			t.Fatalf("Render() error = %v", err)
		}
		output := buf.String()
		if output == "" {
			t.Error("Render() should produce some output for empty list")
		}
	})
}

func TestProjectTable_Render(t *testing.T) {
	projects := []*history.Project{
		{
			Name:              "-Users-test-project",
			Path:              "/Users/test/project",
			Dir:               "/home/.claude/projects/-Users-test-project",
			ConversationCount: 10,
			AgentCount:        5,
			TotalSize:         1024000,
		},
	}

	t.Run("table output", func(t *testing.T) {
		var buf bytes.Buffer
		table := NewProjectTable(TableOptions{Writer: &buf})
		err := table.Render(projects)
		if err != nil {
			t.Fatalf("Render() error = %v", err)
		}
		output := buf.String()
		if output == "" {
			t.Error("Render() produced empty output")
		}
	})

	t.Run("JSON output", func(t *testing.T) {
		var buf bytes.Buffer
		table := NewProjectTable(TableOptions{Writer: &buf, JSON: true})
		err := table.Render(projects)
		if err != nil {
			t.Fatalf("Render() error = %v", err)
		}

		var result []map[string]interface{}
		if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
			t.Fatalf("JSON unmarshal error = %v", err)
		}
		if len(result) != 1 {
			t.Errorf("JSON result length = %d, want 1", len(result))
		}
	})
}

func TestSearchResultTable_Render(t *testing.T) {
	results := []*history.SearchResult{
		{
			Meta: &history.ConversationMeta{
				ID:          "abc123",
				ProjectPath: "/Users/test/project",
				Path:        "/path/to/conv.jsonl",
			},
			MatchCount: 5,
			Previews:   []string{"...match 1...", "...match 2..."},
		},
	}

	t.Run("table output", func(t *testing.T) {
		var buf bytes.Buffer
		table := NewSearchResultTable(TableOptions{Writer: &buf})
		err := table.Render(results)
		if err != nil {
			t.Fatalf("Render() error = %v", err)
		}
		output := buf.String()
		if output == "" {
			t.Error("Render() produced empty output")
		}
	})

	t.Run("JSON output", func(t *testing.T) {
		var buf bytes.Buffer
		table := NewSearchResultTable(TableOptions{Writer: &buf, JSON: true})
		err := table.Render(results)
		if err != nil {
			t.Fatalf("Render() error = %v", err)
		}

		var result []map[string]interface{}
		if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
			t.Fatalf("JSON unmarshal error = %v", err)
		}
		if len(result) != 1 {
			t.Errorf("JSON result length = %d, want 1", len(result))
		}
	})
}

func TestTruncateString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{"short string", "hello", 10, "hello"},
		{"exact length", "hello", 5, "hello"},
		{"truncate", "hello world", 8, "hello..."},
		{"with newlines", "hello\nworld", 20, "hello world"},
		{"with tabs", "hello\tworld", 20, "hello world"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateString(tt.input, tt.maxLen)
			if result != tt.expected {
				t.Errorf("truncateString(%q, %d) = %q, want %q", tt.input, tt.maxLen, result, tt.expected)
			}
		})
	}
}

func TestDefaultTableOptions(t *testing.T) {
	opts := DefaultTableOptions()
	if opts.Writer == nil {
		t.Error("DefaultTableOptions().Writer should not be nil")
	}
}
