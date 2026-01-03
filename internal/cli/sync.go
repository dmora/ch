package cli

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/dmora/ch/internal/backend"
	"github.com/dmora/ch/internal/display"
	"github.com/dmora/ch/internal/sync"
	"github.com/dmora/ch/internal/syncdb"
	"github.com/spf13/cobra"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync conversations to observability backend",
	Long: `Sync Claude Code conversation history to an observability backend.

Supports incremental sync with compaction detection. Uses SQLite to track
sync state and avoid re-sending already synced messages.

Examples:
  ch sync                    # Sync all conversations
  ch sync --dry-run          # Show what would be synced
  ch sync --verbose          # Show detailed span information
  ch sync --file <path>      # Sync a specific file
  ch sync status             # Show sync status`,
	RunE: runSync,
}

var (
	syncDryRun  bool
	syncVerbose bool
	syncJSON    bool
	syncFile    string
)

func init() {
	syncCmd.Flags().BoolVar(&syncDryRun, "dry-run", false, "Show what would be synced without persisting")
	syncCmd.Flags().BoolVarP(&syncVerbose, "verbose", "v", false, "Show detailed span information")
	syncCmd.Flags().BoolVar(&syncJSON, "json", false, "Output as JSON")
	syncCmd.Flags().StringVar(&syncFile, "file", "", "Sync a specific file")

	// Add subcommands
	syncCmd.AddCommand(syncStatusCmd)
}

func runSync(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Create backend based on config
	var be sync.Backend
	switch cfg.Sync.Backend {
	case "console", "":
		be = backend.NewConsoleBackend(backend.ConsoleConfig{
			Writer:  os.Stdout,
			Verbose: syncVerbose || cfg.Sync.Console.Verbose,
			Format:  pickFormat(syncJSON, cfg.Sync.Console.Format),
			NoColor: !display.IsColorEnabled(),
		})
	default:
		return fmt.Errorf("unknown backend: %s", cfg.Sync.Backend)
	}
	defer be.Close()

	// Create syncer
	syncer, err := sync.NewSyncer(sync.SyncerOptions{
		DBPath:      cfg.Sync.DBPath,
		Backend:     be,
		ProjectsDir: cfg.ProjectsDir,
		Workers:     cfg.Sync.Workers,
		DryRun:      syncDryRun || cfg.Sync.DryRun,
	})
	if err != nil {
		return fmt.Errorf("creating syncer: %w", err)
	}
	defer syncer.Close()

	// Sync
	var result *sync.SyncResult
	if syncFile != "" {
		// Single file sync
		spans, err := syncer.SyncFile(ctx, syncFile)
		if err != nil {
			return err
		}
		result = &sync.SyncResult{
			FilesScanned: 1,
			FilesUpdated: 1,
			SpansSynced:  spans,
		}
	} else {
		// Full sync
		result, err = syncer.SyncAll(ctx)
		if err != nil {
			return err
		}
	}

	// Print summary
	printSyncSummary(result, syncDryRun || cfg.Sync.DryRun)

	// Report errors
	if len(result.Errors) > 0 {
		fmt.Fprintf(os.Stderr, "\n%s\n", display.Dim("Errors:"))
		for _, e := range result.Errors {
			fmt.Fprintf(os.Stderr, "  %s\n", e)
		}
	}

	return nil
}

func printSyncSummary(result *sync.SyncResult, dryRun bool) {
	prefix := ""
	if dryRun {
		prefix = "[DRY RUN] "
	}

	fmt.Printf("\n%s%s\n", prefix, display.Dim("Sync Summary"))
	fmt.Printf("  Files scanned: %d\n", result.FilesScanned)
	fmt.Printf("  Files updated: %d\n", result.FilesUpdated)
	fmt.Printf("  Spans synced:  %d\n", result.SpansSynced)
	fmt.Printf("  Duration:      %s\n", result.Duration.Round(time.Millisecond))
}

func pickFormat(jsonFlag bool, configFormat string) string {
	if jsonFlag {
		return "json"
	}
	return configFormat
}

// sync status subcommand
var syncStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show sync status",
	Long:  `Show the current sync status including tracked files and message counts.`,
	RunE:  runSyncStatus,
}

func runSyncStatus(cmd *cobra.Command, args []string) error {
	// Open database directly for status
	db, err := syncdb.Open(cfg.Sync.DBPath)
	if err != nil {
		return fmt.Errorf("opening sync database: %w", err)
	}
	defer db.Close()

	stats, err := db.Stats()
	if err != nil {
		return fmt.Errorf("getting stats: %w", err)
	}

	fmt.Printf("%s\n", display.Dim("Sync Status"))
	fmt.Printf("  Database:        %s\n", cfg.Sync.DBPath)
	fmt.Printf("  Backend:         %s\n", cfg.Sync.Backend)
	fmt.Printf("  Tracked files:   %d\n", stats.TrackedFiles)
	fmt.Printf("  Synced messages: %d\n", stats.SyncedMessages)
	fmt.Printf("  Total messages:  %d\n", stats.TotalMessages)

	return nil
}
