package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/dmora/ch/internal/display"
	"github.com/dmora/ch/internal/history"
	"github.com/spf13/cobra"
)

var agentsCmd = &cobra.Command{
	Use:   "agents <id>",
	Short: "List agents spawned by a conversation",
	Long: `List all agent/subagent conversations spawned by a main conversation.

The id should be a main conversation ID (not an agent ID).`,
	Args:    cobra.ExactArgs(1),
	Aliases: []string{"agent", "ag"},
	RunE:    runAgents,
}

var (
	agentsJSON   bool
	agentsFilter string
)

func init() {
	agentsCmd.Flags().BoolVar(&agentsJSON, "json", false, "Output as JSON")
	agentsCmd.Flags().StringVarP(&agentsFilter, "filter", "f", "", "Filter by agent type (exact match)")
}

func runAgents(cmd *cobra.Command, args []string) error {
	id := args[0]

	// Find the conversation file
	path, err := findConversationFile(id)
	if err != nil {
		return err
	}

	// Load the conversation to get the session ID
	conv, err := history.LoadConversation(path)
	if err != nil {
		return fmt.Errorf("loading conversation: %w", err)
	}

	if conv.Meta.IsAgent {
		return fmt.Errorf("cannot list agents for an agent conversation; use the parent conversation ID")
	}

	sessionID := conv.Meta.SessionID
	if sessionID == "" {
		sessionID = conv.Meta.ID
	}

	// Find agents for this session
	projectDir := filepath.Dir(path)
	scanner := history.NewScanner(history.ScannerOptions{
		ProjectsDir: cfg.ProjectsDir,
	})

	var agents []*history.ConversationMeta
	if agentsFilter != "" {
		// Filter by agent type
		agents, err = scanner.FindAgentsWithType(projectDir, sessionID, agentsFilter)
		if err != nil {
			return fmt.Errorf("finding agents: %w", err)
		}
	} else {
		agents, err = scanner.FindAgents(projectDir, sessionID)
		if err != nil {
			return fmt.Errorf("finding agents: %w", err)
		}
	}

	// Render with filter context
	return display.RenderAgentList(os.Stdout, agents, sessionID, agentsJSON, agentsFilter)
}
