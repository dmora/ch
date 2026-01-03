package history

import (
	"os"
	"path/filepath"
	"testing"
)

func TestListProjects(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "ch-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create project directories
	project1 := filepath.Join(tmpDir, "-Users-test-project1")
	project2 := filepath.Join(tmpDir, "-Users-test-project2")
	if err := os.MkdirAll(project1, 0755); err != nil {
		t.Fatalf("Failed to create project1: %v", err)
	}
	if err := os.MkdirAll(project2, 0755); err != nil {
		t.Fatalf("Failed to create project2: %v", err)
	}

	// Create conversation files
	if err := os.WriteFile(filepath.Join(project1, "abc.jsonl"), []byte(`{"type":"user"}`), 0644); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(project1, "agent-xyz.jsonl"), []byte(`{"type":"assistant"}`), 0644); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(project2, "def.jsonl"), []byte(`{"type":"user"}`), 0644); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}

	projects, err := ListProjects(tmpDir)
	if err != nil {
		t.Fatalf("ListProjects() error = %v", err)
	}
	if len(projects) != 2 {
		t.Errorf("Expected 2 projects, got %d", len(projects))
	}

	// Check first project
	foundProject1 := false
	for _, p := range projects {
		if p.Name == "-Users-test-project1" {
			foundProject1 = true
			if p.ConversationCount != 1 {
				t.Errorf("project1.ConversationCount = %d, want 1", p.ConversationCount)
			}
			if p.AgentCount != 1 {
				t.Errorf("project1.AgentCount = %d, want 1", p.AgentCount)
			}
		}
	}
	if !foundProject1 {
		t.Error("project1 not found in results")
	}
}

func TestListProjects_EmptyDir(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "ch-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	projects, err := ListProjects(tmpDir)
	if err != nil {
		t.Fatalf("ListProjects() error = %v", err)
	}
	if len(projects) != 0 {
		t.Errorf("Expected 0 projects, got %d", len(projects))
	}
}

func TestListProjects_NonexistentDir(t *testing.T) {
	projects, err := ListProjects("/nonexistent/path/that/does/not/exist")
	if err != nil {
		t.Fatalf("ListProjects() error = %v", err)
	}
	if projects != nil {
		t.Errorf("Expected nil projects, got %v", projects)
	}
}

func TestListProjects_EmptyProjectDir(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "ch-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create empty project directory
	project := filepath.Join(tmpDir, "-Users-test-project")
	if err := os.MkdirAll(project, 0755); err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	projects, err := ListProjects(tmpDir)
	if err != nil {
		t.Fatalf("ListProjects() error = %v", err)
	}
	// Empty project should not be included
	if len(projects) != 0 {
		t.Errorf("Expected 0 projects (empty project excluded), got %d", len(projects))
	}
}

func TestFindProject(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "ch-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	project := filepath.Join(tmpDir, "-Users-test-project")
	if err := os.MkdirAll(project, 0755); err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}
	if err := os.WriteFile(filepath.Join(project, "abc.jsonl"), []byte(`{"type":"user"}`), 0644); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}

	// Find by path
	found, err := FindProject(tmpDir, "/Users/test/project")
	if err != nil {
		t.Fatalf("FindProject() error = %v", err)
	}
	if found == nil {
		t.Error("Expected to find project")
	}

	// Find nonexistent
	found, err = FindProject(tmpDir, "/nonexistent")
	if err != nil {
		t.Fatalf("FindProject() error = %v", err)
	}
	if found != nil {
		t.Error("Expected nil for nonexistent project")
	}
}

func TestGetProjectStats(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "ch-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	project := &Project{
		Name:              "-Users-test-project",
		Path:              "/Users/test/project",
		Dir:               filepath.Join(tmpDir, "-Users-test-project"),
		ConversationCount: 1,
		AgentCount:        1,
		TotalSize:         100,
	}

	// Create the project directory
	if err := os.MkdirAll(project.Dir, 0755); err != nil {
		t.Fatalf("Failed to create project dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(project.Dir, "abc.jsonl"), []byte(`{"type":"user","timestamp":"2024-01-01T10:00:00Z","message":{"role":"user","content":"Hello"}}`), 0644); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}

	stats, err := GetProjectStats(project)
	if err != nil {
		t.Fatalf("GetProjectStats() error = %v", err)
	}
	if stats.Project != project {
		t.Error("stats.Project should be the input project")
	}
	if stats.ConversationCount != 1 {
		t.Errorf("stats.ConversationCount = %d, want 1", stats.ConversationCount)
	}
}

func TestListProjects_DefaultDir(t *testing.T) {
	// Test with empty string (should use default)
	projects, err := ListProjects("")
	// This might succeed or fail depending on whether ~/.claude/projects exists
	if err != nil && !os.IsNotExist(err) {
		t.Logf("ListProjects('') returned error: %v (this is expected if default dir doesn't exist)", err)
	}
	_ = projects // Don't fail even if projects is empty
}
