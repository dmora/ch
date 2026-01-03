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
	searchCmd.Flags().BoolVarP(&searchAgents, "agents", "a", false, "Include agent conversations")
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
		opts.ProjectPath = searchProject
	} else if !searchGlobal {
		// Default to current directory's project
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("getting current directory: %w", err)
		}
		opts.ProjectPath = cwd
	}

	results, err := history.Search(query, opts)
	if err != nil {
		return fmt.Errorf("searching: %w", err)
	}

	// Render results
	table := display.NewSearchResultTable(display.TableOptions{
		Writer: os.Stdout,
		JSON:   searchJSON,
	})

	return table.Render(results)
}
