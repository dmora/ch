package history

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultSearchOptions(t *testing.T) {
	opts := DefaultSearchOptions()

	if opts.ProjectsDir == "" {
		t.Error("ProjectsDir should not be empty")
	}
	if opts.Workers != 4 {
		t.Errorf("Workers = %d, want 4", opts.Workers)
	}
}

func TestSearch_NoMatches(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "ch-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	projectDir := filepath.Join(tmpDir, "-test-project")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("Failed to create project dir: %v", err)
	}

	// Create a conversation file without the search term
	convFile := filepath.Join(projectDir, "abc123.jsonl")
	content := `{"type":"user","message":{"role":"user","content":"Hello world"}}
`
	if err := os.WriteFile(convFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write conversation file: %v", err)
	}

	results, err := Search("notfound", SearchOptions{
		ProjectsDir: tmpDir,
	})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(results) != 0 {
		t.Errorf("Expected 0 results, got %d", len(results))
	}
}

func TestSearch_WithMatches(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "ch-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	projectDir := filepath.Join(tmpDir, "-test-project")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("Failed to create project dir: %v", err)
	}

	// Create a conversation file with the search term
	convFile := filepath.Join(projectDir, "abc123.jsonl")
	content := `{"type":"user","message":{"role":"user","content":"Hello docker world"}}
{"type":"assistant","message":{"role":"assistant","content":"I can help with docker!"}}
`
	if err := os.WriteFile(convFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write conversation file: %v", err)
	}

	results, err := Search("docker", SearchOptions{
		ProjectsDir: tmpDir,
	})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}
	if results[0].MatchCount != 2 {
		t.Errorf("Expected 2 matches, got %d", results[0].MatchCount)
	}
}

func TestSearch_CaseInsensitive(t *testing.T) {
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
	content := `{"type":"user","message":{"role":"user","content":"Hello DOCKER world"}}
`
	if err := os.WriteFile(convFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write conversation file: %v", err)
	}

	// Case insensitive search
	results, _ := Search("docker", SearchOptions{
		ProjectsDir:   tmpDir,
		CaseSensitive: false,
	})
	if len(results) != 1 {
		t.Errorf("Case insensitive: expected 1 result, got %d", len(results))
	}

	// Case sensitive search
	results, _ = Search("docker", SearchOptions{
		ProjectsDir:   tmpDir,
		CaseSensitive: true,
	})
	if len(results) != 0 {
		t.Errorf("Case sensitive: expected 0 results, got %d", len(results))
	}
}

func TestSearch_Limit(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "ch-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	projectDir := filepath.Join(tmpDir, "-test-project")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("Failed to create project dir: %v", err)
	}

	// Create multiple conversation files with the search term
	for i := 0; i < 5; i++ {
		file := filepath.Join(projectDir, string(rune('a'+i))+"bc.jsonl")
		content := `{"type":"user","message":{"role":"user","content":"Hello docker world"}}
`
		if err := os.WriteFile(file, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write file: %v", err)
		}
	}

	results, _ := Search("docker", SearchOptions{
		ProjectsDir: tmpDir,
		Limit:       2,
	})
	if len(results) > 2 {
		t.Errorf("Expected max 2 results with limit, got %d", len(results))
	}
}

func TestQuickSearch(t *testing.T) {
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
	content := `{"type":"user","message":{"role":"user","content":"Hello docker world"}}
`
	if err := os.WriteFile(convFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write conversation file: %v", err)
	}

	results, err := QuickSearch("docker", SearchOptions{
		ProjectsDir: tmpDir,
	})
	if err != nil {
		t.Fatalf("QuickSearch() error = %v", err)
	}
	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}
}

func TestExtractPreviewFromText(t *testing.T) {
	tests := []struct {
		name          string
		text          string
		query         string
		caseSensitive bool
		maxLen        int
		wantEmpty     bool
	}{
		{
			name:          "simple match",
			text:          "Hello docker world",
			query:         "docker",
			caseSensitive: false,
			maxLen:        100,
			wantEmpty:     false,
		},
		{
			name:          "case insensitive match",
			text:          "Hello DOCKER world",
			query:         "docker",
			caseSensitive: false,
			maxLen:        100,
			wantEmpty:     false,
		},
		{
			name:          "case sensitive no match",
			text:          "Hello DOCKER world",
			query:         "docker",
			caseSensitive: true,
			maxLen:        100,
			wantEmpty:     true,
		},
		{
			name:          "no match",
			text:          "Hello world",
			query:         "docker",
			caseSensitive: false,
			maxLen:        100,
			wantEmpty:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractPreviewFromText(tt.text, tt.query, tt.caseSensitive, tt.maxLen)
			if tt.wantEmpty && result != "" {
				t.Errorf("Expected empty result, got %q", result)
			}
			if !tt.wantEmpty && result == "" {
				t.Error("Expected non-empty result, got empty")
			}
		})
	}
}
