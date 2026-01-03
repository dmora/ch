package history

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultScannerOptions(t *testing.T) {
	opts := DefaultScannerOptions()

	if opts.ProjectsDir == "" {
		t.Error("ProjectsDir should not be empty")
	}
	if opts.Workers != 4 {
		t.Errorf("Workers = %d, want 4", opts.Workers)
	}
	if !opts.SortByTime {
		t.Error("SortByTime should be true")
	}
}

func TestNewScanner(t *testing.T) {
	opts := ScannerOptions{
		ProjectsDir: "/tmp/test",
		Workers:     0,
	}
	scanner := NewScanner(opts)

	if scanner.opts.Workers != 4 {
		t.Errorf("Workers should default to 4, got %d", scanner.opts.Workers)
	}
}

func TestScanner_ScanAll_EmptyDir(t *testing.T) {
	// Create a temp directory
	tmpDir, err := os.MkdirTemp("", "ch-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	scanner := NewScanner(ScannerOptions{
		ProjectsDir: tmpDir,
	})

	results, err := scanner.ScanAll()
	if err != nil {
		t.Fatalf("ScanAll() error = %v", err)
	}
	if len(results) != 0 {
		t.Errorf("Expected empty results, got %d", len(results))
	}
}

func TestScanner_ScanAll_WithConversations(t *testing.T) {
	// Create a temp directory structure
	tmpDir, err := os.MkdirTemp("", "ch-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a project directory
	projectDir := filepath.Join(tmpDir, "-test-project")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("Failed to create project dir: %v", err)
	}

	// Create a conversation file
	convFile := filepath.Join(projectDir, "abc123.jsonl")
	content := `{"type":"user","message":{"role":"user","content":"Hello"}}
{"type":"assistant","message":{"role":"assistant","content":"Hi!"}}
`
	if err := os.WriteFile(convFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write conversation file: %v", err)
	}

	scanner := NewScanner(ScannerOptions{
		ProjectsDir: tmpDir,
	})

	results, err := scanner.ScanAll()
	if err != nil {
		t.Fatalf("ScanAll() error = %v", err)
	}
	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}
}

func TestScanner_ScanAll_ExcludesAgents(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "ch-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	projectDir := filepath.Join(tmpDir, "-test-project")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("Failed to create project dir: %v", err)
	}

	// Create main conversation
	mainFile := filepath.Join(projectDir, "abc123.jsonl")
	if err := os.WriteFile(mainFile, []byte(`{"type":"user"}`), 0644); err != nil {
		t.Fatalf("Failed to write main file: %v", err)
	}

	// Create agent conversation
	agentFile := filepath.Join(projectDir, "agent-def456.jsonl")
	if err := os.WriteFile(agentFile, []byte(`{"type":"assistant","isSidechain":true}`), 0644); err != nil {
		t.Fatalf("Failed to write agent file: %v", err)
	}

	// Without IncludeAgents
	scanner := NewScanner(ScannerOptions{
		ProjectsDir:   tmpDir,
		IncludeAgents: false,
	})

	results, _ := scanner.ScanAll()
	if len(results) != 1 {
		t.Errorf("Without agents: expected 1 result, got %d", len(results))
	}

	// With IncludeAgents
	scanner = NewScanner(ScannerOptions{
		ProjectsDir:   tmpDir,
		IncludeAgents: true,
	})

	results, _ = scanner.ScanAll()
	if len(results) != 2 {
		t.Errorf("With agents: expected 2 results, got %d", len(results))
	}
}

func TestScanner_Limit(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "ch-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	projectDir := filepath.Join(tmpDir, "-test-project")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("Failed to create project dir: %v", err)
	}

	// Create multiple conversations
	for i := 0; i < 5; i++ {
		file := filepath.Join(projectDir, filepath.Base(string(rune('a'+i))+"bc123.jsonl"))
		if err := os.WriteFile(file, []byte(`{"type":"user"}`), 0644); err != nil {
			t.Fatalf("Failed to write file: %v", err)
		}
	}

	scanner := NewScanner(ScannerOptions{
		ProjectsDir: tmpDir,
		Limit:       2,
	})

	results, _ := scanner.ScanAll()
	if len(results) != 2 {
		t.Errorf("Expected 2 results with limit, got %d", len(results))
	}
}
