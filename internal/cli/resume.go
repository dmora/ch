package cli

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/dmora/ch/internal/history"
	"github.com/spf13/cobra"
)

var resumeCmd = &cobra.Command{
	Use:   "resume <id>",
	Short: "Resume a conversation in Claude Code",
	Long: `Resume a conversation by launching Claude Code with the --resume flag.

The id can be:
  - A full session UUID
  - A short ID (first 8 characters)`,
	Args:    cobra.ExactArgs(1),
	Aliases: []string{"r", "continue"},
	RunE:    runResume,
}

func runResume(cmd *cobra.Command, args []string) error {
	id := args[0]

	// Find the conversation to get the full session ID
	path, err := findConversationFile(id)
	if err != nil {
		return err
	}

	// Load conversation to get the session ID
	conv, err := history.LoadConversation(path)
	if err != nil {
		return fmt.Errorf("loading conversation: %w", err)
	}

	// Can't resume agent conversations directly
	if conv.Meta.IsAgent {
		return fmt.Errorf("cannot resume agent conversations directly; resume the parent conversation instead")
	}

	sessionID := conv.Meta.SessionID
	if sessionID == "" {
		sessionID = conv.Meta.ID
	}

	// Change to the project directory
	if conv.Meta.ProjectPath != "" {
		if err := os.Chdir(conv.Meta.ProjectPath); err != nil {
			// Not fatal - try to resume anyway
			fmt.Fprintf(os.Stderr, "Warning: could not change to project directory: %v\n", err)
		}
	}

	// Execute claude with --resume
	claudeCmd := exec.Command(cfg.ClaudeBin, "--resume", sessionID)
	claudeCmd.Stdin = os.Stdin
	claudeCmd.Stdout = os.Stdout
	claudeCmd.Stderr = os.Stderr

	fmt.Printf("Resuming conversation %s...\n", history.ShortID(sessionID))

	if err := claudeCmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		return fmt.Errorf("running claude: %w", err)
	}

	return nil
}

// findSessionID finds the full session ID for a given short or full ID.
func findSessionID(id string) (string, error) {
	// Remove agent- prefix if present
	id = strings.TrimPrefix(id, "agent-")

	// Search in all projects
	projects, err := history.ListProjects(cfg.ProjectsDir)
	if err != nil {
		return "", fmt.Errorf("listing projects: %w", err)
	}

	for _, project := range projects {
		entries, err := os.ReadDir(project.Dir)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}

			name := entry.Name()
			if !strings.HasSuffix(name, ".jsonl") {
				continue
			}

			// Skip agent files
			if strings.HasPrefix(name, "agent-") {
				continue
			}

			baseName := strings.TrimSuffix(name, ".jsonl")
			if baseName == id || strings.HasPrefix(baseName, id) {
				return baseName, nil
			}
		}
	}

	return "", fmt.Errorf("conversation not found: %s", id)
}
