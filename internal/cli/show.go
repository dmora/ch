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
	showPrompt   bool
	showResult   bool
)

func init() {
	showCmd.Flags().BoolVar(&showThinking, "thinking", true, "Include thinking blocks (default: true)")
	showCmd.Flags().BoolVar(&showTools, "tools", true, "Include tool calls (default: true)")
	showCmd.Flags().BoolVar(&showJSON, "json", false, "Output as JSON")
	showCmd.Flags().BoolVar(&showRaw, "raw", false, "Output raw JSONL")
	showCmd.Flags().BoolVar(&showPrompt, "prompt", false, "Show only the prompt that spawned this agent (agents only)")
	showCmd.Flags().BoolVar(&showResult, "result", false, "Show only the final result from this agent (agents only)")
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

	// Handle --prompt flag (agents only)
	if showPrompt {
		if !conv.Meta.IsAgent {
			return fmt.Errorf("--prompt flag only works for agent conversations")
		}
		return showAgentPrompt(conv, path)
	}

	// Handle --result flag (agents only)
	if showResult {
		if !conv.Meta.IsAgent {
			return fmt.Errorf("--result flag only works for agent conversations")
		}
		return showAgentResult(conv)
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

// showAgentPrompt displays the prompt that was used to spawn an agent.
func showAgentPrompt(conv *history.Conversation, agentPath string) error {
	projectDir := filepath.Dir(agentPath)
	parentSessionID := conv.Meta.ParentSessionID
	if parentSessionID == "" {
		return fmt.Errorf("cannot find parent session for this agent")
	}

	// Find parent conversation file
	parentPath := filepath.Join(projectDir, parentSessionID+".jsonl")

	// Check if parent exists
	if _, err := os.Stat(parentPath); os.IsNotExist(err) {
		return fmt.Errorf("parent conversation not found at %s.\nThe parent may have been deleted or compacted.\nTry: ch show %s to view the full agent conversation instead", parentPath, "agent-"+conv.Meta.ID)
	}

	// Extract agent info from parent
	info, err := history.ExtractAgentInfo(parentPath, conv.Meta.ID)
	if err != nil {
		return fmt.Errorf("extracting agent info: %w", err)
	}

	if info == nil {
		return fmt.Errorf("could not find Task tool call that spawned this agent.\nThe parent conversation may have been compacted (Task tool calls removed).\nTry: ch show %s to view the full agent conversation instead", "agent-"+conv.Meta.ID)
	}

	// Display prompt
	fmt.Fprintf(os.Stdout, "\n%s %s\n", display.Title("Agent Prompt"), display.ID("agent-"+conv.Meta.ID))
	if info.SubagentType != "" {
		fmt.Fprintf(os.Stdout, "%s %s\n", display.Dim("Type:"), display.Match(info.SubagentType))
	}
	if info.Description != "" {
		fmt.Fprintf(os.Stdout, "%s %s\n", display.Dim("Description:"), info.Description)
	}
	fmt.Fprintf(os.Stdout, "\n%s\n", display.Section("Prompt:"))
	if info.Prompt != "" {
		fmt.Fprintln(os.Stdout, info.Prompt)
	} else {
		fmt.Fprintln(os.Stdout, display.Dim("(no prompt found)"))
	}

	return nil
}

// showAgentResult displays the final result from an agent.
func showAgentResult(conv *history.Conversation) error {
	assistantMsgs := conv.GetAssistantMessages()
	if len(assistantMsgs) == 0 {
		return fmt.Errorf("no assistant messages found in agent conversation")
	}

	lastMsg := assistantMsgs[len(assistantMsgs)-1]

	// Parse and extract text
	msg, err := history.ParseMessageEntry(lastMsg)
	if err != nil {
		return fmt.Errorf("parsing message: %w", err)
	}

	text := history.ExtractMessageText(msg)

	// Display result
	fmt.Fprintf(os.Stdout, "\n%s %s\n", display.Title("Agent Result"), display.ID("agent-"+conv.Meta.ID))
	fmt.Fprintf(os.Stdout, "%s %s\n\n", display.Dim("Messages:"), display.Number(fmt.Sprintf("%d", conv.Meta.MessageCount)))

	if text != "" {
		fmt.Fprintln(os.Stdout, text)
	} else {
		fmt.Fprintln(os.Stdout, display.Dim("(no text content in final response)"))
	}

	return nil
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
