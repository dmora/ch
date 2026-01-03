// Package config provides configuration handling for the ch CLI.
package config

import (
	"os"
	"path/filepath"
)

// Config holds the application configuration.
type Config struct {
	// ProjectsDir is the directory containing Claude project history.
	ProjectsDir string

	// ClaudeBin is the path to the Claude CLI binary.
	ClaudeBin string
}

// DefaultConfig returns the default configuration.
func DefaultConfig() *Config {
	home, _ := os.UserHomeDir()

	return &Config{
		ProjectsDir: filepath.Join(home, ".claude", "projects"),
		ClaudeBin:   "claude",
	}
}

// Load loads configuration from environment variables and defaults.
func Load() *Config {
	cfg := DefaultConfig()

	// Override from environment
	if dir := os.Getenv("CLAUDE_PROJECTS_DIR"); dir != "" {
		cfg.ProjectsDir = dir
	}
	if bin := os.Getenv("CLAUDE_BIN"); bin != "" {
		cfg.ClaudeBin = bin
	}

	return cfg
}

// Validate validates the configuration.
func (c *Config) Validate() error {
	// Check if projects directory exists
	if _, err := os.Stat(c.ProjectsDir); os.IsNotExist(err) {
		// Not an error - user might not have any history yet
		return nil
	}
	return nil
}

// GetCurrentProjectDir returns the Claude project directory for the current working directory.
func (c *Config) GetCurrentProjectDir() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	// Encode the path
	encoded := encodeProjectPath(cwd)
	return filepath.Join(c.ProjectsDir, encoded), nil
}

// encodeProjectPath encodes a filesystem path to a Claude project directory name.
func encodeProjectPath(path string) string {
	// Replace path separators with dashes
	result := ""
	for _, c := range path {
		if c == filepath.Separator {
			result += "-"
		} else if c == ':' {
			// Skip Windows drive colon
			continue
		} else {
			result += string(c)
		}
	}
	return result
}
