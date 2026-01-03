package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dmora/ch/internal/display"
	"github.com/dmora/ch/internal/history"
	"github.com/spf13/cobra"
)

var showCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show a specific conversation",
	Long: `Show the contents of a specific conversation.

The id can be:
  - A full session UUID (e.g., 9dbf1107-d255-4d17-a544-aadb594fc786)
  - A short ID (e.g., 9dbf1107)
  - An agent ID (e.g., agent-d0e14239 or just d0e14239)`,
	Args:    cobra.ExactArgs(1),
	Aliases: []string{"s", "view"},
	RunE:    runShow,
}

var (
	showThinking bool
	showTools    bool
	showJSON     bool
	showRaw      bool
)

func init() {
	showCmd.Flags().BoolVar(&showThinking, "thinking", true, "Include thinking blocks (default: true)")
	showCmd.Flags().BoolVar(&showTools, "tools", false, "Include tool calls")
	showCmd.Flags().BoolVar(&showJSON, "json", false, "Output as JSON")
	showCmd.Flags().BoolVar(&showRaw, "raw", false, "Output raw JSONL")
}

func runShow(cmd *cobra.Command, args []string) error {
	id := args[0]

	// Find the conversation file
	path, err := findConversationFile(id)
	if err != nil {
		return err
	}

	// Load the conversation
	conv, err := history.LoadConversation(path)
	if err != nil {
		return fmt.Errorf("loading conversation: %w", err)
	}

	// Count agents for main conversations
	agentCount := 0
	if !conv.Meta.IsAgent {
		projectDir := filepath.Dir(path)
		scanner := history.NewScanner(history.ScannerOptions{ProjectsDir: cfg.ProjectsDir})
		agentCount = scanner.CountAgents(projectDir, conv.Meta.SessionID)
	}

	// Display
	disp := display.NewConversationDisplay(display.ConversationDisplayOptions{
		Writer:       os.Stdout,
		ShowThinking: showThinking,
		ShowTools:    showTools,
		JSON:         showJSON,
		Raw:          showRaw,
		AgentCount:   agentCount,
	})

	return disp.Render(conv)
}

// findConversationFile finds a conversation file by ID.
func findConversationFile(id string) (string, error) {
	// Check if it's an agent ID
	isAgent := strings.HasPrefix(id, "agent-")
	if isAgent {
		id = strings.TrimPrefix(id, "agent-")
	}

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

			// Check for match
			if isAgent {
				// Look for agent-{id}.jsonl
				if name == fmt.Sprintf("agent-%s.jsonl", id) {
					return filepath.Join(project.Dir, name), nil
				}
				// Partial match
				if strings.HasPrefix(name, "agent-") && strings.Contains(name, id) {
					return filepath.Join(project.Dir, name), nil
				}
			} else {
				// Look for {id}.jsonl or partial match
				baseName := strings.TrimSuffix(name, ".jsonl")
				if !strings.HasPrefix(baseName, "agent-") {
					if baseName == id || strings.HasPrefix(baseName, id) {
						return filepath.Join(project.Dir, name), nil
					}
				}
			}
		}
	}

	return "", fmt.Errorf("conversation not found: %s", id)
}
