package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestE2E runs end-to-end tests for the ch CLI.
// These tests build the binary and run actual commands.
func TestE2E(t *testing.T) {
	// Build the binary
	tmpDir, err := os.MkdirTemp("", "ch-e2e-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	binaryPath := filepath.Join(tmpDir, "ch")
	cmd := exec.Command("go", "build", "-o", binaryPath, "./cmd/ch")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to build binary: %v\n%s", err, output)
	}

	// Set up test data directory
	testProjectsDir := filepath.Join(tmpDir, "projects")
	if err := os.MkdirAll(testProjectsDir, 0755); err != nil {
		t.Fatalf("Failed to create test projects dir: %v", err)
	}

	// Create test project
	testProject := filepath.Join(testProjectsDir, "-test-project")
	if err := os.MkdirAll(testProject, 0755); err != nil {
		t.Fatalf("Failed to create test project: %v", err)
	}

	// Create test conversation
	convFile := filepath.Join(testProject, "abc12345-def6-7890-abcd-ef1234567890.jsonl")
	convContent := `{"type":"user","timestamp":"2024-01-01T10:00:00Z","sessionId":"abc12345-def6-7890-abcd-ef1234567890","message":{"role":"user","content":"Hello, can you help me with Go programming?"}}
{"type":"assistant","timestamp":"2024-01-01T10:00:01Z","sessionId":"abc12345-def6-7890-abcd-ef1234567890","message":{"role":"assistant","content":[{"type":"text","text":"Of course! I'd be happy to help with Go programming."}]}}
{"type":"user","timestamp":"2024-01-01T10:00:02Z","sessionId":"abc12345-def6-7890-abcd-ef1234567890","message":{"role":"user","content":"How do I create a goroutine?"}}
{"type":"assistant","timestamp":"2024-01-01T10:00:03Z","sessionId":"abc12345-def6-7890-abcd-ef1234567890","message":{"role":"assistant","content":[{"type":"text","text":"To create a goroutine in Go, you use the 'go' keyword before a function call."}]}}
`
	if err := os.WriteFile(convFile, []byte(convContent), 0644); err != nil {
		t.Fatalf("Failed to write conversation file: %v", err)
	}

	// Create agent file
	agentFile := filepath.Join(testProject, "agent-xyz789.jsonl")
	agentContent := `{"type":"assistant","timestamp":"2024-01-01T10:01:00Z","sessionId":"abc12345-def6-7890-abcd-ef1234567890","agentId":"xyz789","isSidechain":true,"message":{"role":"assistant","content":[{"type":"text","text":"Agent processing..."}]}}
`
	if err := os.WriteFile(agentFile, []byte(agentContent), 0644); err != nil {
		t.Fatalf("Failed to write agent file: %v", err)
	}

	// Helper function to run ch command
	runCh := func(args ...string) (string, error) {
		cmd := exec.Command(binaryPath, args...)
		cmd.Env = append(os.Environ(), "CLAUDE_PROJECTS_DIR="+testProjectsDir)
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		err := cmd.Run()
		if err != nil {
			return stdout.String() + stderr.String(), err
		}
		return stdout.String(), nil
	}

	// Test: ch --help
	t.Run("help", func(t *testing.T) {
		output, err := runCh("--help")
		if err != nil {
			t.Fatalf("ch --help failed: %v", err)
		}
		if !strings.Contains(output, "conversation history") {
			t.Errorf("Expected 'conversation history' in help output, got: %s", output)
		}
	})

	// Test: ch --version
	t.Run("version", func(t *testing.T) {
		output, err := runCh("--version")
		if err != nil {
			t.Fatalf("ch --version failed: %v", err)
		}
		if !strings.Contains(output, "ch version") {
			t.Errorf("Expected 'ch version' in output, got: %s", output)
		}
	})

	// Test: ch list -g
	t.Run("list_global", func(t *testing.T) {
		output, err := runCh("list", "-g")
		if err != nil {
			t.Fatalf("ch list -g failed: %v\n%s", err, output)
		}
		if !strings.Contains(output, "abc12345") {
			t.Errorf("Expected 'abc12345' in list output, got: %s", output)
		}
	})

	// Test: ch list -g --json
	t.Run("list_json", func(t *testing.T) {
		output, err := runCh("list", "-g", "--json")
		if err != nil {
			t.Fatalf("ch list -g --json failed: %v\n%s", err, output)
		}

		var results []map[string]interface{}
		if err := json.Unmarshal([]byte(output), &results); err != nil {
			t.Fatalf("Failed to parse JSON output: %v\n%s", err, output)
		}
		if len(results) != 1 {
			t.Errorf("Expected 1 conversation, got %d", len(results))
		}
	})

	// Test: ch list -g -a (include agents)
	t.Run("list_with_agents", func(t *testing.T) {
		output, err := runCh("list", "-g", "-a")
		if err != nil {
			t.Fatalf("ch list -g -a failed: %v\n%s", err, output)
		}
		if !strings.Contains(output, "agent-xyz789") || !strings.Contains(output, "abc12345") {
			t.Errorf("Expected both main and agent conversations in output, got: %s", output)
		}
	})

	// Test: ch show
	t.Run("show", func(t *testing.T) {
		output, err := runCh("show", "abc12345")
		if err != nil {
			t.Fatalf("ch show failed: %v\n%s", err, output)
		}
		if !strings.Contains(output, "Hello, can you help me with Go programming") {
			t.Errorf("Expected user message in show output, got: %s", output)
		}
		if !strings.Contains(output, "help with Go programming") {
			t.Errorf("Expected assistant response in show output, got: %s", output)
		}
	})

	// Test: ch show --json
	t.Run("show_json", func(t *testing.T) {
		output, err := runCh("show", "abc12345", "--json")
		if err != nil {
			t.Fatalf("ch show --json failed: %v\n%s", err, output)
		}

		var result map[string]interface{}
		if err := json.Unmarshal([]byte(output), &result); err != nil {
			t.Fatalf("Failed to parse JSON output: %v\n%s", err, output)
		}
		if result["id"] == nil {
			t.Error("Expected 'id' field in JSON output")
		}
	})

	// Test: ch search
	t.Run("search", func(t *testing.T) {
		output, err := runCh("search", "goroutine", "-g")
		if err != nil {
			t.Fatalf("ch search failed: %v\n%s", err, output)
		}
		if !strings.Contains(output, "abc12345") {
			t.Errorf("Expected conversation ID in search results, got: %s", output)
		}
	})

	// Test: ch search --json
	t.Run("search_json", func(t *testing.T) {
		output, err := runCh("search", "goroutine", "-g", "--json")
		if err != nil {
			t.Fatalf("ch search --json failed: %v\n%s", err, output)
		}

		var results []map[string]interface{}
		if err := json.Unmarshal([]byte(output), &results); err != nil {
			t.Fatalf("Failed to parse JSON output: %v\n%s", err, output)
		}
		if len(results) == 0 {
			t.Error("Expected at least one search result")
		}
	})

	// Test: ch search - no matches
	t.Run("search_no_matches", func(t *testing.T) {
		output, err := runCh("search", "nonexistent_term_xyz", "-g")
		if err != nil {
			t.Fatalf("ch search failed: %v\n%s", err, output)
		}
		if !strings.Contains(output, "No matches found") {
			t.Errorf("Expected 'No matches found' in output, got: %s", output)
		}
	})

	// Test: ch projects
	t.Run("projects", func(t *testing.T) {
		output, err := runCh("projects")
		if err != nil {
			t.Fatalf("ch projects failed: %v\n%s", err, output)
		}
		if !strings.Contains(output, "test") {
			t.Errorf("Expected 'test' project in output, got: %s", output)
		}
	})

	// Test: ch projects --json
	t.Run("projects_json", func(t *testing.T) {
		output, err := runCh("projects", "--json")
		if err != nil {
			t.Fatalf("ch projects --json failed: %v\n%s", err, output)
		}

		var results []map[string]interface{}
		if err := json.Unmarshal([]byte(output), &results); err != nil {
			t.Fatalf("Failed to parse JSON output: %v\n%s", err, output)
		}
		if len(results) == 0 {
			t.Error("Expected at least one project")
		}
	})

	// Test: ch stats
	t.Run("stats", func(t *testing.T) {
		output, err := runCh("stats")
		if err != nil {
			t.Fatalf("ch stats failed: %v\n%s", err, output)
		}
		if !strings.Contains(output, "Projects:") {
			t.Errorf("Expected 'Projects:' in stats output, got: %s", output)
		}
		if !strings.Contains(output, "Conversations:") {
			t.Errorf("Expected 'Conversations:' in stats output, got: %s", output)
		}
	})

	// Test: ch stats --json
	t.Run("stats_json", func(t *testing.T) {
		output, err := runCh("stats", "--json")
		if err != nil {
			t.Fatalf("ch stats --json failed: %v\n%s", err, output)
		}

		var result map[string]interface{}
		if err := json.Unmarshal([]byte(output), &result); err != nil {
			t.Fatalf("Failed to parse JSON output: %v\n%s", err, output)
		}
		if result["project_count"] == nil {
			t.Error("Expected 'project_count' field in JSON output")
		}
	})

	// Test: ch agents
	t.Run("agents", func(t *testing.T) {
		output, err := runCh("agents", "abc12345")
		if err != nil {
			t.Fatalf("ch agents failed: %v\n%s", err, output)
		}
		if !strings.Contains(output, "xyz789") {
			t.Errorf("Expected agent 'xyz789' in output, got: %s", output)
		}
	})

	// Test: ch agents --json
	t.Run("agents_json", func(t *testing.T) {
		output, err := runCh("agents", "abc12345", "--json")
		if err != nil {
			t.Fatalf("ch agents --json failed: %v\n%s", err, output)
		}

		var results []map[string]interface{}
		if err := json.Unmarshal([]byte(output), &results); err != nil {
			t.Fatalf("Failed to parse JSON output: %v\n%s", err, output)
		}
		if len(results) == 0 {
			t.Error("Expected at least one agent")
		}
	})

	// Test: invalid command
	t.Run("invalid_command", func(t *testing.T) {
		_, err := runCh("invalidcommand")
		if err == nil {
			t.Error("Expected error for invalid command")
		}
	})

	// Test: show nonexistent conversation
	t.Run("show_nonexistent", func(t *testing.T) {
		_, err := runCh("show", "nonexistent123")
		if err == nil {
			t.Error("Expected error for nonexistent conversation")
		}
	})
}
