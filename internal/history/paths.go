// Package history provides utilities for working with Claude Code conversation history.
package history

import (
	"os"
	"path/filepath"
	"strings"
)

// DefaultProjectsDir returns the default Claude projects directory.
func DefaultProjectsDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".claude", "projects")
}

// EncodeProjectPath encodes a filesystem path to a Claude project directory name.
// For example: /Users/davidmora/Projects/foo -> -Users-davidmora-Projects-foo
// Note: Claude Code also replaces dots with dashes.
func EncodeProjectPath(path string) string {
	// Replace path separators with dashes
	encoded := strings.ReplaceAll(path, string(filepath.Separator), "-")
	// Replace dots with dashes (Claude Code does this)
	encoded = strings.ReplaceAll(encoded, ".", "-")
	// Handle Windows drive letters if needed
	encoded = strings.ReplaceAll(encoded, ":", "")
	// Ensure it starts with a dash if it doesn't already
	if !strings.HasPrefix(encoded, "-") {
		encoded = "-" + encoded
	}
	return encoded
}

// DecodeProjectPath decodes a Claude project directory name to a filesystem path.
// For example: -Users-davidmora-Projects-foo -> /Users/davidmora/Projects/foo
func DecodeProjectPath(encoded string) string {
	// Remove leading dash
	decoded := strings.TrimPrefix(encoded, "-")
	// Replace dashes with path separators
	decoded = strings.ReplaceAll(decoded, "-", string(filepath.Separator))
	// Add leading separator for absolute paths
	if !strings.HasPrefix(decoded, string(filepath.Separator)) {
		decoded = string(filepath.Separator) + decoded
	}
	return decoded
}

// GetProjectDir returns the Claude project directory for the given path.
func GetProjectDir(projectsDir, path string) string {
	return filepath.Join(projectsDir, EncodeProjectPath(path))
}

// GetCurrentProjectDir returns the Claude project directory for the current working directory.
func GetCurrentProjectDir() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return GetProjectDir(DefaultProjectsDir(), cwd), nil
}

// ProjectDirExists checks if a Claude project directory exists for the given path.
func ProjectDirExists(projectsDir, path string) bool {
	dir := GetProjectDir(projectsDir, path)
	info, err := os.Stat(dir)
	return err == nil && info.IsDir()
}

// IsAgentFile returns true if the filename indicates an agent conversation.
func IsAgentFile(filename string) bool {
	return strings.HasPrefix(filename, "agent-") && strings.HasSuffix(filename, ".jsonl")
}

// IsConversationFile returns true if the filename indicates a conversation file.
func IsConversationFile(filename string) bool {
	return strings.HasSuffix(filename, ".jsonl")
}

// ExtractSessionID extracts the session ID from a main conversation filename.
// For example: 9dbf1107-d255-4d17-a544-aadb594fc786.jsonl -> 9dbf1107-d255-4d17-a544-aadb594fc786
func ExtractSessionID(filename string) string {
	if IsAgentFile(filename) {
		return ""
	}
	return strings.TrimSuffix(filename, ".jsonl")
}

// ExtractAgentID extracts the agent ID from an agent conversation filename.
// For example: agent-d0e14239.jsonl -> d0e14239
func ExtractAgentID(filename string) string {
	if !IsAgentFile(filename) {
		return ""
	}
	name := strings.TrimPrefix(filename, "agent-")
	return strings.TrimSuffix(name, ".jsonl")
}

// ShortID returns a shortened version of a UUID for display.
func ShortID(id string) string {
	if len(id) <= 8 {
		return id
	}
	return id[:8]
}
