package sync

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	gosync "sync"
	"time"

	"github.com/dmora/ch/internal/history"
	"github.com/dmora/ch/internal/jsonl"
	"github.com/dmora/ch/internal/syncdb"
)

// Syncer coordinates the sync process.
type Syncer struct {
	db          *syncdb.DB
	backend     Backend
	projectsDir string
	workers     int
	dryRun      bool
}

// SyncerOptions configures the syncer.
type SyncerOptions struct {
	DBPath      string
	Backend     Backend
	ProjectsDir string
	Workers     int
	DryRun      bool
}

// NewSyncer creates a new syncer.
func NewSyncer(opts SyncerOptions) (*Syncer, error) {
	if opts.Workers <= 0 {
		opts.Workers = 4
	}

	var db *syncdb.DB
	var err error

	if !opts.DryRun {
		db, err = syncdb.Open(opts.DBPath)
		if err != nil {
			return nil, fmt.Errorf("opening sync database: %w", err)
		}
	}

	return &Syncer{
		db:          db,
		backend:     opts.Backend,
		projectsDir: opts.ProjectsDir,
		workers:     opts.Workers,
		dryRun:      opts.DryRun,
	}, nil
}

// Close releases syncer resources.
func (s *Syncer) Close() error {
	if err := s.backend.Close(); err != nil {
		return err
	}
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// SyncResult holds the result of a sync operation.
type SyncResult struct {
	FilesScanned int
	FilesUpdated int
	SpansSynced  int
	Errors       []error
	Duration     time.Duration
}

// SyncAll syncs all conversation files.
func (s *Syncer) SyncAll(ctx context.Context) (*SyncResult, error) {
	start := time.Now()
	result := &SyncResult{}

	// Find all JSONL files
	files, err := s.findFiles()
	if err != nil {
		return nil, fmt.Errorf("finding files: %w", err)
	}
	result.FilesScanned = len(files)

	// Process files with worker pool
	type workItem struct {
		path    string
		err     error
		spans   int
		updated bool
	}

	fileChan := make(chan string, len(files))
	resultChan := make(chan workItem, len(files))

	var wg gosync.WaitGroup
	for i := 0; i < s.workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for path := range fileChan {
				spans, updated, err := s.syncFile(ctx, path)
				resultChan <- workItem{path: path, spans: spans, updated: updated, err: err}
			}
		}()
	}

	// Send work
	for _, f := range files {
		fileChan <- f
	}
	close(fileChan)

	// Collect results
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	for item := range resultChan {
		if item.err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("%s: %w", item.path, item.err))
		} else {
			result.SpansSynced += item.spans
			if item.updated {
				result.FilesUpdated++
			}
		}
	}

	result.Duration = time.Since(start)
	return result, nil
}

// SyncFile syncs a single file.
func (s *Syncer) SyncFile(ctx context.Context, path string) (int, error) {
	spans, _, err := s.syncFile(ctx, path)
	return spans, err
}

// syncFile syncs a single file and returns (spans synced, was updated, error).
func (s *Syncer) syncFile(ctx context.Context, path string) (int, bool, error) {
	// Get file info
	info, err := os.Stat(path)
	if err != nil {
		return 0, false, fmt.Errorf("stat file: %w", err)
	}

	currentSize := info.Size()
	currentMtime := info.ModTime().Unix()

	// Get existing state (only if not dry run)
	var state *syncdb.SyncState
	if !s.dryRun && s.db != nil {
		state, err = s.db.GetState(path)
		if err != nil {
			return 0, false, fmt.Errorf("getting state: %w", err)
		}
	}

	// Determine sync strategy
	var offset int64 = 0
	needsResync := false

	if state == nil {
		// New file: full sync
		needsResync = true
	} else if currentSize < state.LastSize {
		// File shrunk: compaction detected, full resync
		needsResync = true
		if !s.dryRun && s.db != nil {
			s.db.ClearFileMessages(path)
			s.db.DeleteState(path)
		}
	} else if currentMtime == state.LastMtime && currentSize == state.LastSize {
		// No changes
		return 0, false, nil
	} else {
		// Incremental sync from last offset
		offset = state.LastOffset
	}

	_ = needsResync // Used for logging if needed

	// Open file and seek to offset
	file, err := os.Open(path)
	if err != nil {
		return 0, false, fmt.Errorf("opening file: %w", err)
	}
	defer file.Close()

	if offset > 0 {
		if _, err := file.Seek(offset, io.SeekStart); err != nil {
			return 0, false, fmt.Errorf("seeking to offset: %w", err)
		}
	}

	// Parse and map entries
	parser := jsonl.NewParserFromReader(file)
	mapper := NewMapper(path)

	lineNum := 0
	if state != nil {
		lineNum = state.MessageCount
	}

	spansProcessed := 0
	var traceID string

	for {
		entry, err := parser.Next()
		if err != nil {
			return spansProcessed, spansProcessed > 0, fmt.Errorf("parsing entry: %w", err)
		}
		if entry == nil {
			break // EOF
		}
		lineNum++

		// Extract trace ID from first entry with sessionID
		if traceID == "" && entry.SessionID != "" {
			traceID = entry.SessionID
		}

		// Map to span
		span, err := mapper.MapEntry(entry, lineNum)
		if err != nil {
			// Log error but continue
			if s.db != nil {
				s.db.RecordError(path, err.Error())
			}
			continue
		}
		if span == nil {
			continue // Entry doesn't produce a span
		}

		// Check if already synced (for resumability) - only if not dry run
		if !s.dryRun && s.db != nil {
			hash := GenerateMessageHash(entry)
			synced, _ := s.db.IsSynced(path, hash)
			if synced {
				continue
			}
		}

		// Send to backend
		if err := s.backend.SendSpan(ctx, span); err != nil {
			return spansProcessed, spansProcessed > 0, fmt.Errorf("sending span: %w", err)
		}

		// Record synced message (only if not dry run)
		if !s.dryRun && s.db != nil {
			hash := GenerateMessageHash(entry)
			s.db.RecordSyncedMessage(path, hash, span.ID)
		}

		spansProcessed++
	}

	// Update state (only if not dry run)
	if !s.dryRun && s.db != nil {
		newOffset, _ := file.Seek(0, io.SeekCurrent)
		newState := &syncdb.SyncState{
			FilePath:     path,
			LastOffset:   newOffset,
			LastSize:     currentSize,
			LastMtime:    currentMtime,
			TraceID:      traceID,
			MessageCount: lineNum,
			LastSyncAt:   time.Now().Unix(),
			Backend:      s.backend.Name(),
		}
		if err := s.db.SaveState(newState); err != nil {
			return spansProcessed, true, fmt.Errorf("saving state: %w", err)
		}
	}

	return spansProcessed, spansProcessed > 0, nil
}

// findFiles finds all JSONL files in the projects directory.
func (s *Syncer) findFiles() ([]string, error) {
	var files []string

	entries, err := os.ReadDir(s.projectsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		projectDir := filepath.Join(s.projectsDir, entry.Name())
		projectFiles, err := os.ReadDir(projectDir)
		if err != nil {
			continue
		}

		for _, f := range projectFiles {
			if f.IsDir() || !history.IsConversationFile(f.Name()) {
				continue
			}
			files = append(files, filepath.Join(projectDir, f.Name()))
		}
	}

	return files, nil
}

// Stats returns sync database statistics.
func (s *Syncer) Stats() (*syncdb.Stats, error) {
	if s.db == nil {
		return &syncdb.Stats{}, nil
	}
	return s.db.Stats()
}
