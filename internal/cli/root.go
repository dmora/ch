// Package cli implements the ch command-line interface.
package cli

import (
	"github.com/dmora/ch/internal/config"
	"github.com/dmora/ch/internal/display"
	"github.com/spf13/cobra"
)

var (
	// Version is set at build time.
	Version = "dev"

	// cfg is the global configuration.
	cfg *config.Config
)

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}

// SetVersion sets the version string for the CLI.
// Must be called before Execute() to take effect.
func SetVersion(v string) {
	Version = v
	rootCmd.Version = v
}

var rootCmd = &cobra.Command{
	Use:   "ch",
	Short: "Claude History - view Claude Code conversation history",
	Long: `ch is a memory-efficient CLI tool for viewing Claude Code conversation history,
including subagent conversations.

Examples:
  ch list                    # List recent conversations
  ch list -g                 # List from all projects
  ch show abc123             # Show specific conversation
  ch search "docker"         # Search across conversations
  ch agents abc123           # List agents spawned by a conversation
  ch projects                # List all projects
  ch stats                   # Show usage statistics`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Load configuration
		cfg = config.Load()

		// Set up colors
		display.DisableColorIfNotTTY()
	},
	Version: Version,
}

func init() {
	// Add subcommands
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(showCmd)
	rootCmd.AddCommand(searchCmd)
	rootCmd.AddCommand(resumeCmd)
	rootCmd.AddCommand(agentsCmd)
	rootCmd.AddCommand(projectsCmd)
	rootCmd.AddCommand(statsCmd)
	rootCmd.AddCommand(syncCmd)
}
