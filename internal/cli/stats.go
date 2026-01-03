package cli

import (
	"os"
	"time"

	"github.com/dmora/ch/internal/display"
	"github.com/dmora/ch/internal/history"
	"github.com/spf13/cobra"
)

var statsCmd = &cobra.Command{
	Use:     "stats",
	Short:   "Show usage statistics",
	Long:    `Show aggregate usage statistics across all Claude Code projects.`,
	Aliases: []string{"stat", "st"},
	RunE:    runStats,
}

var statsJSON bool

func init() {
	statsCmd.Flags().BoolVar(&statsJSON, "json", false, "Output as JSON")
}

func runStats(cmd *cobra.Command, args []string) error {
	projects, err := history.ListProjects(cfg.ProjectsDir)
	if err != nil {
		return err
	}

	stats := &display.Stats{
		ProjectCount: len(projects),
	}

	var oldest, newest time.Time

	for _, p := range projects {
		stats.ConversationCount += p.ConversationCount
		stats.AgentCount += p.AgentCount
		stats.TotalSize += p.TotalSize
	}

	// Scan for message counts and timestamps
	scanner := history.NewScanner(history.ScannerOptions{
		ProjectsDir:   cfg.ProjectsDir,
		IncludeAgents: true,
	})

	conversations, err := scanner.ScanAll()
	if err == nil {
		for _, c := range conversations {
			stats.TotalMessages += c.MessageCount
			if oldest.IsZero() || c.Timestamp.Before(oldest) {
				oldest = c.Timestamp
			}
			if newest.IsZero() || c.Timestamp.After(newest) {
				newest = c.Timestamp
			}
		}
	}

	if !oldest.IsZero() {
		stats.OldestConversation = oldest.Format("2006-01-02 15:04")
	}
	if !newest.IsZero() {
		stats.NewestConversation = newest.Format("2006-01-02 15:04")
	}

	return display.RenderStats(os.Stdout, stats, statsJSON)
}
