package history

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConversation(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "ch-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	projectDir := filepath.Join(tmpDir, "-test-project")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("Failed to create project dir: %v", err)
	}

	convFile := filepath.Join(projectDir, "abc123.jsonl")
	content := `{"type":"user","timestamp":"2024-01-01T10:00:00Z","sessionId":"abc123","message":{"role":"user","content":"Hello"}}
{"type":"assistant","timestamp":"2024-01-01T10:00:01Z","sessionId":"abc123","message":{"role":"assistant","model":"claude-3","content":[{"type":"text","text":"Hi!"}]}}
`
	if err := os.WriteFile(convFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}

	conv, err := LoadConversation(convFile)
	if err != nil {
		t.Fatalf("LoadConversation() error = %v", err)
	}

	if conv.Meta.ID != "abc123" {
		t.Errorf("Meta.ID = %q, want %q", conv.Meta.ID, "abc123")
	}
	if conv.Meta.MessageCount != 2 {
		t.Errorf("Meta.MessageCount = %d, want 2", conv.Meta.MessageCount)
	}
	if len(conv.Entries) != 2 {
		t.Errorf("len(Entries) = %d, want 2", len(conv.Entries))
	}
}

func TestLoadConversation_NonexistentFile(t *testing.T) {
	_, err := LoadConversation("/nonexistent/file.jsonl")
	if err == nil {
		t.Error("Expected error for nonexistent file")
	}
}

func TestLoadConversation_Agent(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "ch-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	projectDir := filepath.Join(tmpDir, "-test-project")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("Failed to create project dir: %v", err)
	}

	agentFile := filepath.Join(projectDir, "agent-xyz789.jsonl")
	content := `{"type":"assistant","timestamp":"2024-01-01T10:00:00Z","sessionId":"parent123","agentId":"xyz789","isSidechain":true,"message":{"role":"assistant","content":[{"type":"text","text":"Agent response"}]}}
`
	if err := os.WriteFile(agentFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}

	conv, err := LoadConversation(agentFile)
	if err != nil {
		t.Fatalf("LoadConversation() error = %v", err)
	}

	if !conv.Meta.IsAgent {
		t.Error("Expected IsAgent to be true")
	}
	if conv.Meta.ID != "xyz789" {
		t.Errorf("Meta.ID = %q, want %q", conv.Meta.ID, "xyz789")
	}
	if conv.Meta.ParentSessionID != "parent123" {
		t.Errorf("Meta.ParentSessionID = %q, want %q", conv.Meta.ParentSessionID, "parent123")
	}
}

func TestConversation_GetMessages(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "ch-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	projectDir := filepath.Join(tmpDir, "-test-project")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("Failed to create project dir: %v", err)
	}

	convFile := filepath.Join(projectDir, "abc123.jsonl")
	content := `{"type":"file-history-snapshot","snapshot":{}}
{"type":"user","message":{"role":"user","content":"Hello"}}
{"type":"assistant","message":{"role":"assistant","content":"Hi!"}}
{"type":"system","message":{"role":"system","content":"System message"}}
`
	if err := os.WriteFile(convFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}

	conv, err := LoadConversation(convFile)
	if err != nil {
		t.Fatalf("LoadConversation() error = %v", err)
	}

	messages := conv.GetMessages()
	if len(messages) != 3 {
		t.Errorf("GetMessages() returned %d messages, want 3", len(messages))
	}

	userMessages := conv.GetUserMessages()
	if len(userMessages) != 1 {
		t.Errorf("GetUserMessages() returned %d messages, want 1", len(userMessages))
	}

	assistantMessages := conv.GetAssistantMessages()
	if len(assistantMessages) != 1 {
		t.Errorf("GetAssistantMessages() returned %d messages, want 1", len(assistantMessages))
	}
}

func TestScanConversationMeta_WithModel(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "ch-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	projectDir := filepath.Join(tmpDir, "-test-project")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("Failed to create project dir: %v", err)
	}

	convFile := filepath.Join(projectDir, "abc123.jsonl")
	content := `{"type":"user","message":{"role":"user","content":"Hello"}}
{"type":"assistant","message":{"role":"assistant","model":"claude-opus-4-5","content":[{"type":"text","text":"Hi!"}]}}
`
	if err := os.WriteFile(convFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}

	meta, err := ScanConversationMeta(convFile)
	if err != nil {
		t.Fatalf("ScanConversationMeta() error = %v", err)
	}

	if meta.Model != "claude-opus-4-5" {
		t.Errorf("Model = %q, want %q", meta.Model, "claude-opus-4-5")
	}
}

func TestScanConversationMeta_NoTimestamp(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "ch-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	projectDir := filepath.Join(tmpDir, "-test-project")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("Failed to create project dir: %v", err)
	}

	convFile := filepath.Join(projectDir, "abc123.jsonl")
	content := `{"type":"user","message":{"role":"user","content":"Hello"}}
`
	if err := os.WriteFile(convFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}

	meta, err := ScanConversationMeta(convFile)
	if err != nil {
		t.Fatalf("ScanConversationMeta() error = %v", err)
	}

	// Timestamp should be file mtime
	if meta.Timestamp.IsZero() {
		t.Error("Timestamp should not be zero")
	}
}

func TestScanConversationMeta_NonexistentFile(t *testing.T) {
	_, err := ScanConversationMeta("/nonexistent/file.jsonl")
	if err == nil {
		t.Error("Expected error for nonexistent file")
	}
}
