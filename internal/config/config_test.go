package config

import (
	"os"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.ProjectsDir == "" {
		t.Error("ProjectsDir should not be empty")
	}
	if cfg.ClaudeBin != "claude" {
		t.Errorf("ClaudeBin = %q, want %q", cfg.ClaudeBin, "claude")
	}
}

func TestLoad(t *testing.T) {
	// Test default values
	cfg := Load()
	if cfg.ProjectsDir == "" {
		t.Error("ProjectsDir should not be empty")
	}
}

func TestLoad_WithEnvVars(t *testing.T) {
	// Set environment variables
	os.Setenv("CLAUDE_PROJECTS_DIR", "/custom/projects")
	os.Setenv("CLAUDE_BIN", "/custom/bin/claude")
	defer func() {
		os.Unsetenv("CLAUDE_PROJECTS_DIR")
		os.Unsetenv("CLAUDE_BIN")
	}()

	cfg := Load()
	if cfg.ProjectsDir != "/custom/projects" {
		t.Errorf("ProjectsDir = %q, want %q", cfg.ProjectsDir, "/custom/projects")
	}
	if cfg.ClaudeBin != "/custom/bin/claude" {
		t.Errorf("ClaudeBin = %q, want %q", cfg.ClaudeBin, "/custom/bin/claude")
	}
}

func TestConfig_Validate(t *testing.T) {
	cfg := DefaultConfig()
	err := cfg.Validate()
	// Validate should not return an error for default config
	// even if the directory doesn't exist
	if err != nil {
		t.Errorf("Validate() error = %v", err)
	}
}

func TestEncodeProjectPath(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "simple path",
			path:     "/Users/foo/Projects",
			expected: "-Users-foo-Projects",
		},
		{
			name:     "path with dots",
			path:     "/Users/foo/github.com/bar",
			expected: "-Users-foo-github.com-bar",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := encodeProjectPath(tt.path)
			if result != tt.expected {
				t.Errorf("encodeProjectPath(%q) = %q, want %q", tt.path, result, tt.expected)
			}
		})
	}
}
