package cli

import (
	"os"

	"github.com/dmora/ch/internal/display"
	"github.com/dmora/ch/internal/history"
	"github.com/spf13/cobra"
)

var projectsCmd = &cobra.Command{
	Use:     "projects",
	Short:   "List all projects",
	Long:    `List all Claude Code projects with conversation history.`,
	Aliases: []string{"proj", "p"},
	RunE:    runProjects,
}

var projectsJSON bool

func init() {
	projectsCmd.Flags().BoolVar(&projectsJSON, "json", false, "Output as JSON")
}

func runProjects(cmd *cobra.Command, args []string) error {
	projects, err := history.ListProjects(cfg.ProjectsDir)
	if err != nil {
		return err
	}

	table := display.NewProjectTable(display.TableOptions{
		Writer: os.Stdout,
		JSON:   projectsJSON,
	})

	return table.Render(projects)
}
