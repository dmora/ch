// Package syncdb provides SQLite-based sync state management.
package syncdb

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	_ "modernc.org/sqlite"
)

// DB wraps the SQLite database connection.
type DB struct {
	db *sql.DB
	mu sync.Mutex // Serialize write operations
}

// Open opens or creates the sync database.
func Open(path string) (*DB, error) {
	// Ensure parent directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("creating database directory: %w", err)
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	// Enable WAL mode for better concurrency
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("setting WAL mode: %w", err)
	}

	// Set busy timeout to 5 seconds
	if _, err := db.Exec("PRAGMA busy_timeout=5000"); err != nil {
		db.Close()
		return nil, fmt.Errorf("setting busy timeout: %w", err)
	}

	// Create tables
	if err := createTables(db); err != nil {
		db.Close()
		return nil, err
	}

	return &DB{db: db}, nil
}

// Close closes the database connection.
func (d *DB) Close() error {
	return d.db.Close()
}

// createTables creates the required tables if they don't exist.
func createTables(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS sync_state (
		file_path TEXT PRIMARY KEY,
		last_offset INTEGER NOT NULL,
		last_size INTEGER NOT NULL,
		last_mtime INTEGER NOT NULL,
		trace_id TEXT,
		message_count INTEGER DEFAULT 0,
		last_sync_at INTEGER NOT NULL,
		backend TEXT NOT NULL
	);

	CREATE TABLE IF NOT EXISTS synced_messages (
		file_path TEXT,
		message_hash TEXT,
		span_id TEXT,
		synced_at INTEGER,
		PRIMARY KEY (file_path, message_hash)
	);

	CREATE INDEX IF NOT EXISTS idx_synced_messages_file
		ON synced_messages(file_path);

	CREATE TABLE IF NOT EXISTS sync_errors (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		file_path TEXT NOT NULL,
		error_message TEXT NOT NULL,
		occurred_at INTEGER NOT NULL
	);
	`

	_, err := db.Exec(schema)
	if err != nil {
		return fmt.Errorf("creating tables: %w", err)
	}
	return nil
}

// Stats holds database statistics.
type Stats struct {
	TrackedFiles   int
	SyncedMessages int
	TotalMessages  int
}

// Stats returns database statistics.
func (d *DB) Stats() (*Stats, error) {
	var stats Stats

	row := d.db.QueryRow("SELECT COUNT(*) FROM sync_state")
	if err := row.Scan(&stats.TrackedFiles); err != nil {
		return nil, err
	}

	row = d.db.QueryRow("SELECT COUNT(*) FROM synced_messages")
	if err := row.Scan(&stats.SyncedMessages); err != nil {
		return nil, err
	}

	row = d.db.QueryRow("SELECT COALESCE(SUM(message_count), 0) FROM sync_state")
	if err := row.Scan(&stats.TotalMessages); err != nil {
		return nil, err
	}

	return &stats, nil
}
