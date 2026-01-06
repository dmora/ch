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

// shouldRecord returns true if database operations should be performed.
func (s *Syncer) shouldRecord() bool {
	return !s.dryRun && s.db != nil
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

// syncStrategy holds the determined sync approach for a file.
type syncStrategy struct {
	offset      int64
	lineNum     int
	needsResync bool
}

// determineSyncStrategy decides how to sync a file based on its state.
func (s *Syncer) determineSyncStrategy(path string, currentSize, currentMtime int64) (*syncStrategy, error) {
	strategy := &syncStrategy{}

	if !s.shouldRecord() {
		// No database: always full sync
		strategy.needsResync = true
		return strategy, nil
	}

	state, err := s.db.GetState(path)
	if err != nil {
		return nil, fmt.Errorf("getting state: %w", err)
	}

	if state == nil {
		// New file: full sync
		strategy.needsResync = true
		return strategy, nil
	}

	if currentSize < state.LastSize {
		// File shrunk: compaction detected, full resync
		strategy.needsResync = true
		s.db.ClearFileMessages(path)
		s.db.DeleteState(path)
		return strategy, nil
	}

	if currentMtime == state.LastMtime && currentSize == state.LastSize {
		// No changes
		return nil, nil
	}

	// Incremental sync from last offset
	strategy.offset = state.LastOffset
	strategy.lineNum = state.MessageCount
	return strategy, nil
}

// processAndSendEntry processes a single entry, checking deduplication and sending to backend.
// Returns true if the entry was sent (not skipped due to deduplication).
func (s *Syncer) processAndSendEntry(ctx context.Context, entry *jsonl.RawEntry, span *Span, path string) (bool, error) {
	if s.shouldRecord() {
		hash := GenerateMessageHash(entry)
		synced, _ := s.db.IsSynced(path, hash)
		if synced {
			return false, nil
		}
	}

	if err := s.backend.SendSpan(ctx, span); err != nil {
		return false, fmt.Errorf("sending span: %w", err)
	}

	if s.shouldRecord() {
		hash := GenerateMessageHash(entry)
		s.db.RecordSyncedMessage(path, hash, span.ID)
	}

	return true, nil
}

// syncFile syncs a single file and returns (spans synced, was updated, error).
func (s *Syncer) syncFile(ctx context.Context, path string) (int, bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, false, fmt.Errorf("stat file: %w", err)
	}

	currentSize := info.Size()
	currentMtime := info.ModTime().Unix()

	strategy, err := s.determineSyncStrategy(path, currentSize, currentMtime)
	if err != nil {
		return 0, false, err
	}
	if strategy == nil {
		return 0, false, nil // No changes
	}

	file, err := os.Open(path)
	if err != nil {
		return 0, false, fmt.Errorf("opening file: %w", err)
	}
	defer file.Close()

	if strategy.offset > 0 {
		if _, err := file.Seek(strategy.offset, io.SeekStart); err != nil {
			return 0, false, fmt.Errorf("seeking to offset: %w", err)
		}
	}

	spansProcessed, traceID, lineNum, err := s.processEntries(ctx, file, path, strategy.lineNum)
	if err != nil {
		return spansProcessed, spansProcessed > 0, err
	}

	if err := s.saveState(file, path, currentSize, currentMtime, traceID, lineNum); err != nil {
		return spansProcessed, true, err
	}

	return spansProcessed, spansProcessed > 0, nil
}

// processEntries reads and processes all entries from the file.
func (s *Syncer) processEntries(ctx context.Context, file *os.File, path string, startLineNum int) (int, string, int, error) {
	parser := jsonl.NewParserFromReader(file)
	mapper := NewMapper(path)

	lineNum := startLineNum
	spansProcessed := 0
	var traceID string

	for {
		entry, err := parser.Next()
		if err != nil {
			return spansProcessed, traceID, lineNum, fmt.Errorf("parsing entry: %w", err)
		}
		if entry == nil {
			break
		}
		lineNum++

		if traceID == "" && entry.SessionID != "" {
			traceID = entry.SessionID
		}

		span, err := mapper.MapEntry(entry, lineNum)
		if err != nil {
			if s.db != nil {
				s.db.RecordError(path, err.Error())
			}
			continue
		}
		if span == nil {
			continue
		}

		sent, err := s.processAndSendEntry(ctx, entry, span, path)
		if err != nil {
			return spansProcessed, traceID, lineNum, err
		}
		if sent {
			spansProcessed++
		}
	}

	return spansProcessed, traceID, lineNum, nil
}

// saveState persists the sync state to the database.
func (s *Syncer) saveState(file *os.File, path string, currentSize, currentMtime int64, traceID string, lineNum int) error {
	if !s.shouldRecord() {
		return nil
	}

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
		return fmt.Errorf("saving state: %w", err)
	}
	return nil
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
