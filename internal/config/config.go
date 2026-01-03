// Package config provides configuration handling for the ch CLI.
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config holds the application configuration.
type Config struct {
	// ProjectsDir is the directory containing Claude project history.
	ProjectsDir string `yaml:"projects_dir"`

	// ClaudeBin is the path to the Claude CLI binary.
	ClaudeBin string `yaml:"claude_bin"`

	// Sync contains sync-specific configuration.
	Sync SyncConfig `yaml:"sync"`
}

// SyncConfig holds sync-specific configuration.
type SyncConfig struct {
	// Enabled controls whether sync is active.
	Enabled bool `yaml:"enabled"`

	// Backend specifies the sync backend ("console", "langfuse").
	Backend string `yaml:"backend"`

	// DBPath is the path to the sync state database.
	DBPath string `yaml:"db_path"`

	// Workers is the number of parallel sync workers.
	Workers int `yaml:"workers"`

	// DryRun if true, shows what would be synced without persisting.
	DryRun bool `yaml:"dry_run"`

	// Console backend settings.
	Console ConsoleConfig `yaml:"console"`
}

// ConsoleConfig holds console backend settings.
type ConsoleConfig struct {
	// Verbose shows full span details.
	Verbose bool `yaml:"verbose"`

	// Format is "text" or "json".
	Format string `yaml:"format"`
}

// DefaultConfig returns the default configuration.
func DefaultConfig() *Config {
	home, _ := os.UserHomeDir()

	return &Config{
		ProjectsDir: filepath.Join(home, ".claude", "projects"),
		ClaudeBin:   "claude",
		Sync: SyncConfig{
			Enabled: true,
			Backend: "console",
			DBPath:  DefaultSyncDBPath(),
			Workers: 4,
			DryRun:  false,
			Console: ConsoleConfig{
				Verbose: false,
				Format:  "text",
			},
		},
	}
}

// DataDir returns the path to the ch data directory.
func DataDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".ch")
}

// ConfigPath returns the path to the config file.
func ConfigPath() string {
	return filepath.Join(DataDir(), "config.yaml")
}

// DefaultSyncDBPath returns the default sync database path.
func DefaultSyncDBPath() string {
	return filepath.Join(DataDir(), "sync.db")
}

// LoadFromFile loads configuration from a YAML file.
func LoadFromFile(path string) (*Config, error) {
	cfg := DefaultConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil // Use defaults if no config file
		}
		return nil, fmt.Errorf("reading config: %w", err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	return cfg, nil
}

// Load loads configuration from file and environment variables.
// Environment variables override file values.
func Load() *Config {
	// Load from file first (ignores errors, uses defaults)
	cfg, _ := LoadFromFile(ConfigPath())

	// Override from environment
	if dir := os.Getenv("CLAUDE_PROJECTS_DIR"); dir != "" {
		cfg.ProjectsDir = dir
	}
	if bin := os.Getenv("CLAUDE_BIN"); bin != "" {
		cfg.ClaudeBin = bin
	}

	// Sync-specific environment overrides
	if db := os.Getenv("CH_SYNC_DB"); db != "" {
		cfg.Sync.DBPath = db
	}
	if backend := os.Getenv("CH_SYNC_BACKEND"); backend != "" {
		cfg.Sync.Backend = backend
	}

	// Ensure defaults for sync config
	if cfg.Sync.DBPath == "" {
		cfg.Sync.DBPath = DefaultSyncDBPath()
	}
	if cfg.Sync.Backend == "" {
		cfg.Sync.Backend = "console"
	}
	if cfg.Sync.Workers <= 0 {
		cfg.Sync.Workers = 4
	}
	if cfg.Sync.Console.Format == "" {
		cfg.Sync.Console.Format = "text"
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
