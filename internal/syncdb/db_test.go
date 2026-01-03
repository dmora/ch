package syncdb

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOpenClose(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer db.Close()

	// Verify file was created
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("Database file was not created")
	}
}

func TestSyncState(t *testing.T) {
	tmpDir := t.TempDir()
	db, err := Open(filepath.Join(tmpDir, "test.db"))
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer db.Close()

	// Get state for non-existent file
	state, err := db.GetState("/path/to/file.jsonl")
	if err != nil {
		t.Fatalf("GetState failed: %v", err)
	}
	if state != nil {
		t.Error("Expected nil state for non-existent file")
	}

	// Save state
	newState := &SyncState{
		FilePath:     "/path/to/file.jsonl",
		LastOffset:   1000,
		LastSize:     2000,
		LastMtime:    1234567890,
		TraceID:      "trace-123",
		MessageCount: 50,
		LastSyncAt:   1234567890,
		Backend:      "console",
	}
	if err := db.SaveState(newState); err != nil {
		t.Fatalf("SaveState failed: %v", err)
	}

	// Get state
	state, err = db.GetState("/path/to/file.jsonl")
	if err != nil {
		t.Fatalf("GetState failed: %v", err)
	}
	if state == nil {
		t.Fatal("Expected non-nil state")
	}
	if state.LastOffset != 1000 {
		t.Errorf("LastOffset = %d, want 1000", state.LastOffset)
	}
	if state.MessageCount != 50 {
		t.Errorf("MessageCount = %d, want 50", state.MessageCount)
	}

	// Update state
	newState.LastOffset = 2000
	newState.MessageCount = 100
	if err := db.SaveState(newState); err != nil {
		t.Fatalf("SaveState (update) failed: %v", err)
	}

	state, err = db.GetState("/path/to/file.jsonl")
	if err != nil {
		t.Fatalf("GetState failed: %v", err)
	}
	if state.LastOffset != 2000 {
		t.Errorf("LastOffset = %d, want 2000", state.LastOffset)
	}

	// Delete state
	if err := db.DeleteState("/path/to/file.jsonl"); err != nil {
		t.Fatalf("DeleteState failed: %v", err)
	}

	state, err = db.GetState("/path/to/file.jsonl")
	if err != nil {
		t.Fatalf("GetState failed: %v", err)
	}
	if state != nil {
		t.Error("Expected nil state after delete")
	}
}

func TestSyncedMessages(t *testing.T) {
	tmpDir := t.TempDir()
	db, err := Open(filepath.Join(tmpDir, "test.db"))
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer db.Close()

	filePath := "/path/to/file.jsonl"
	hash := "abc123"

	// Check not synced
	synced, err := db.IsSynced(filePath, hash)
	if err != nil {
		t.Fatalf("IsSynced failed: %v", err)
	}
	if synced {
		t.Error("Expected not synced")
	}

	// Record synced
	if err := db.RecordSyncedMessage(filePath, hash, "span-1"); err != nil {
		t.Fatalf("RecordSyncedMessage failed: %v", err)
	}

	// Check synced
	synced, err = db.IsSynced(filePath, hash)
	if err != nil {
		t.Fatalf("IsSynced failed: %v", err)
	}
	if !synced {
		t.Error("Expected synced")
	}

	// Clear messages
	if err := db.ClearFileMessages(filePath); err != nil {
		t.Fatalf("ClearFileMessages failed: %v", err)
	}

	// Check not synced again
	synced, err = db.IsSynced(filePath, hash)
	if err != nil {
		t.Fatalf("IsSynced failed: %v", err)
	}
	if synced {
		t.Error("Expected not synced after clear")
	}
}

func TestStats(t *testing.T) {
	tmpDir := t.TempDir()
	db, err := Open(filepath.Join(tmpDir, "test.db"))
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer db.Close()

	// Initial stats
	stats, err := db.Stats()
	if err != nil {
		t.Fatalf("Stats failed: %v", err)
	}
	if stats.TrackedFiles != 0 {
		t.Errorf("TrackedFiles = %d, want 0", stats.TrackedFiles)
	}

	// Add some data
	db.SaveState(&SyncState{
		FilePath:     "/file1.jsonl",
		LastOffset:   100,
		LastSize:     200,
		LastMtime:    1234567890,
		MessageCount: 10,
		LastSyncAt:   1234567890,
		Backend:      "console",
	})
	db.SaveState(&SyncState{
		FilePath:     "/file2.jsonl",
		LastOffset:   100,
		LastSize:     200,
		LastMtime:    1234567890,
		MessageCount: 20,
		LastSyncAt:   1234567890,
		Backend:      "console",
	})
	db.RecordSyncedMessage("/file1.jsonl", "hash1", "span1")
	db.RecordSyncedMessage("/file1.jsonl", "hash2", "span2")

	stats, err = db.Stats()
	if err != nil {
		t.Fatalf("Stats failed: %v", err)
	}
	if stats.TrackedFiles != 2 {
		t.Errorf("TrackedFiles = %d, want 2", stats.TrackedFiles)
	}
	if stats.SyncedMessages != 2 {
		t.Errorf("SyncedMessages = %d, want 2", stats.SyncedMessages)
	}
	if stats.TotalMessages != 30 {
		t.Errorf("TotalMessages = %d, want 30", stats.TotalMessages)
	}
}

func TestConcurrentWrites(t *testing.T) {
	tmpDir := t.TempDir()
	db, err := Open(filepath.Join(tmpDir, "test.db"))
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer db.Close()

	// Test concurrent writes don't cause errors
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(n int) {
			state := &SyncState{
				FilePath:     filepath.Join("/file", string(rune('0'+n))),
				LastOffset:   int64(n * 100),
				LastSize:     int64(n * 200),
				LastMtime:    1234567890,
				MessageCount: n * 10,
				LastSyncAt:   1234567890,
				Backend:      "console",
			}
			if err := db.SaveState(state); err != nil {
				t.Errorf("Concurrent SaveState failed: %v", err)
			}
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	stats, err := db.Stats()
	if err != nil {
		t.Fatalf("Stats failed: %v", err)
	}
	if stats.TrackedFiles != 10 {
		t.Errorf("TrackedFiles = %d, want 10", stats.TrackedFiles)
	}
}
