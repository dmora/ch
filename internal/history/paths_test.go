package history

import (
	"testing"
)

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
			expected: "-Users-foo-github-com-bar",
		},
		{
			name:     "empty path",
			path:     "",
			expected: "-",
		},
		{
			name:     "already has dash",
			path:     "-Users/foo",
			expected: "-Users-foo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EncodeProjectPath(tt.path)
			if result != tt.expected {
				t.Errorf("EncodeProjectPath(%q) = %q, want %q", tt.path, result, tt.expected)
			}
		})
	}
}

func TestDecodeProjectPath(t *testing.T) {
	tests := []struct {
		name     string
		encoded  string
		expected string
	}{
		{
			name:     "simple path",
			encoded:  "-Users-foo-Projects",
			expected: "/Users/foo/Projects",
		},
		{
			name:     "empty becomes root",
			encoded:  "-",
			expected: "/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DecodeProjectPath(tt.encoded)
			if result != tt.expected {
				t.Errorf("DecodeProjectPath(%q) = %q, want %q", tt.encoded, result, tt.expected)
			}
		})
	}
}

func TestIsAgentFile(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		expected bool
	}{
		{"agent file", "agent-abc123.jsonl", true},
		{"main conversation", "abc123-def456.jsonl", false},
		{"no extension", "agent-abc123", false},
		{"wrong prefix", "conversation-abc123.jsonl", false},
		{"just agent prefix", "agent-.jsonl", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsAgentFile(tt.filename); got != tt.expected {
				t.Errorf("IsAgentFile(%q) = %v, want %v", tt.filename, got, tt.expected)
			}
		})
	}
}

func TestIsConversationFile(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		expected bool
	}{
		{"jsonl file", "abc123.jsonl", true},
		{"agent file", "agent-abc123.jsonl", true},
		{"json file", "abc123.json", false},
		{"txt file", "abc123.txt", false},
		{"no extension", "abc123", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsConversationFile(tt.filename); got != tt.expected {
				t.Errorf("IsConversationFile(%q) = %v, want %v", tt.filename, got, tt.expected)
			}
		})
	}
}

func TestExtractSessionID(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		expected string
	}{
		{"uuid file", "abc123-def456.jsonl", "abc123-def456"},
		{"simple id", "abc123.jsonl", "abc123"},
		{"agent file returns empty", "agent-abc123.jsonl", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ExtractSessionID(tt.filename); got != tt.expected {
				t.Errorf("ExtractSessionID(%q) = %q, want %q", tt.filename, got, tt.expected)
			}
		})
	}
}

func TestExtractAgentID(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		expected string
	}{
		{"agent file", "agent-abc123.jsonl", "abc123"},
		{"main file returns empty", "abc123.jsonl", ""},
		{"complex agent id", "agent-abc123-def456.jsonl", "abc123-def456"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ExtractAgentID(tt.filename); got != tt.expected {
				t.Errorf("ExtractAgentID(%q) = %q, want %q", tt.filename, got, tt.expected)
			}
		})
	}
}

func TestShortID(t *testing.T) {
	tests := []struct {
		name     string
		id       string
		expected string
	}{
		{"long uuid", "abc12345-6789-0abc-def0-123456789abc", "abc12345"},
		{"exactly 8", "abc12345", "abc12345"},
		{"short id", "abc", "abc"},
		{"empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ShortID(tt.id); got != tt.expected {
				t.Errorf("ShortID(%q) = %q, want %q", tt.id, got, tt.expected)
			}
		})
	}
}

func TestDefaultProjectsDir(t *testing.T) {
	dir := DefaultProjectsDir()
	if dir == "" {
		t.Error("DefaultProjectsDir() returned empty string")
	}
	// Should contain .claude/projects
	if !containsSubstring(dir, ".claude") || !containsSubstring(dir, "projects") {
		t.Errorf("DefaultProjectsDir() = %q, expected to contain .claude/projects", dir)
	}
}

func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstringHelper(s, substr))
}

func containsSubstringHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestGetProjectDir(t *testing.T) {
	projectsDir := "/home/user/.claude/projects"
	path := "/Users/foo/Projects"

	result := GetProjectDir(projectsDir, path)
	expected := "/home/user/.claude/projects/-Users-foo-Projects"

	if result != expected {
		t.Errorf("GetProjectDir() = %q, want %q", result, expected)
	}
}
