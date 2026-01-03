package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/dmora/ch/internal/display"
	"github.com/dmora/ch/internal/history"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List conversations",
	Long: `List conversations from Claude Code history.

By default, lists conversations from the current directory's project.
Use -g/--global to list from all projects.`,
	Aliases: []string{"ls", "l"},
	RunE:    runList,
}

var (
	listAgents  bool
	listProject string
	listLimit   int
	listGlobal  bool
	listJSON    bool
)

func init() {
	listCmd.Flags().BoolVarP(&listAgents, "agents", "a", true, "Include agent/subagent conversations (default: true)")
	listCmd.Flags().StringVarP(&listProject, "project", "p", "", "Filter by project path")
	listCmd.Flags().IntVarP(&listLimit, "limit", "n", 50, "Limit number of results")
	listCmd.Flags().BoolVarP(&listGlobal, "global", "g", false, "List from all projects")
	listCmd.Flags().BoolVar(&listJSON, "json", false, "Output as JSON")
}

func runList(cmd *cobra.Command, args []string) error {
	opts := history.ScannerOptions{
		ProjectsDir:   cfg.ProjectsDir,
		IncludeAgents: listAgents,
		Limit:         listLimit,
		SortByTime:    true,
	}

	// Determine project filter
	if listProject != "" {
		opts.ProjectPath = listProject
	} else if !listGlobal {
		// Default to current directory's project
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("getting current directory: %w", err)
		}
		opts.ProjectPath = cwd
	}

	scanner := history.NewScanner(opts)
	conversations, err := scanner.ScanAll()
	if err != nil {
		return fmt.Errorf("scanning conversations: %w", err)
	}

	// If not showing agents, count them for each main conversation
	if !listAgents {
		for _, c := range conversations {
			if !c.IsAgent {
				projectDir := filepath.Dir(c.Path)
				c.AgentCount = scanner.CountAgents(projectDir, c.SessionID)
			}
		}
	}

	// Count projects for global view
	projectCount := 0
	if listGlobal || listProject == "" {
		projects, _ := history.ListProjects(cfg.ProjectsDir)
		projectCount = len(projects)
	}

	// Get display project path
	displayProject := opts.ProjectPath
	if displayProject == "" && !listGlobal {
		displayProject, _ = os.Getwd()
	}

	// Render table
	table := display.NewConversationTable(display.TableOptions{
		Writer:       os.Stdout,
		ShowAgent:    listAgents,
		JSON:         listJSON,
		ProjectPath:  displayProject,
		IsGlobal:     listGlobal,
		ProjectCount: projectCount,
	})

	return table.Render(conversations)
}
