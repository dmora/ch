package cli

import (
	"fmt"
	"os"

	"github.com/dmora/ch/internal/display"
	"github.com/dmora/ch/internal/history"
	"github.com/spf13/cobra"
)

var searchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search across conversations",
	Long: `Search for text across all conversations.

Searches through all message content (user and assistant messages).
By default, searches in the current directory's project.`,
	Args:    cobra.MinimumNArgs(1),
	Aliases: []string{"grep", "find"},
	RunE:    runSearch,
}

var (
	searchProject       string
	searchLimit         int
	searchGlobal        bool
	searchCaseSensitive bool
	searchJSON          bool
	searchAgents        bool
)

func init() {
	searchCmd.Flags().StringVarP(&searchProject, "project", "p", "", "Filter by project path")
	searchCmd.Flags().IntVarP(&searchLimit, "limit", "n", 20, "Limit number of results")
	searchCmd.Flags().BoolVarP(&searchGlobal, "global", "g", false, "Search in all projects")
	searchCmd.Flags().BoolVarP(&searchCaseSensitive, "case-sensitive", "c", false, "Case-sensitive search")
	searchCmd.Flags().BoolVar(&searchJSON, "json", false, "Output as JSON")
	searchCmd.Flags().BoolVarP(&searchAgents, "agents", "a", true, "Include agent conversations (default: true)")
}

func runSearch(cmd *cobra.Command, args []string) error {
	query := args[0]
	if len(args) > 1 {
		// Join multiple args with space
		for _, arg := range args[1:] {
			query += " " + arg
		}
	}

	opts := history.SearchOptions{
		ProjectsDir:   cfg.ProjectsDir,
		IncludeAgents: searchAgents,
		Limit:         searchLimit,
		CaseSensitive: searchCaseSensitive,
	}

	// Determine project filter
	if searchProject != "" {
		// Try to resolve project path (supports fuzzy matching)
		resolvedPath, ambiguous, err := history.ResolveProjectPath(cfg.ProjectsDir, searchProject)
		if err != nil {
			return fmt.Errorf("resolving project: %w", err)
		}
		if len(ambiguous) > 0 {
			// Multiple matches - show them to user
			fmt.Fprintf(os.Stdout, "%s\n\n", display.Dim(fmt.Sprintf("Multiple projects match '%s':", searchProject)))
			for i, p := range ambiguous {
				fmt.Fprintf(os.Stdout, "  %d. %s\n", i+1, p.Path)
			}
			fmt.Fprintf(os.Stdout, "\n%s\n", display.Dim("Please use a more specific project path or name."))
			return nil
		}
		opts.ProjectPath = resolvedPath
		// Update searchProject for display
		searchProject = resolvedPath
	} else if !searchGlobal {
		// Default to current directory's project
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("getting current directory: %w", err)
		}
		opts.ProjectPath = cwd
	}

	// Show search context
	if !searchJSON {
		scope := "current project"
		if searchGlobal {
			scope = "all projects"
		} else if searchProject != "" {
			scope = searchProject
		}
		fmt.Fprintf(os.Stdout, "%s \"%s\" %s\n\n", display.Dim("Searching for"), display.Match(query), display.Dim("in "+scope+"..."))
	}

	results, err := history.Search(query, opts)
	if err != nil {
		return fmt.Errorf("searching: %w", err)
	}

	// Render results
	table := display.NewSearchResultTable(display.TableOptions{
		Writer: os.Stdout,
		JSON:   searchJSON,
		Query:  query,
	})

	return table.Render(results)
}
