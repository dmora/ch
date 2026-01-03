package syncdb

import (
	"database/sql"
	"time"
)

// SyncState represents the sync state for a single file.
type SyncState struct {
	FilePath     string
	LastOffset   int64
	LastSize     int64
	LastMtime    int64
	TraceID      string
	MessageCount int
	LastSyncAt   int64
	Backend      string
}

// GetState retrieves the sync state for a file.
func (d *DB) GetState(filePath string) (*SyncState, error) {
	row := d.db.QueryRow(`
		SELECT file_path, last_offset, last_size, last_mtime,
			   trace_id, message_count, last_sync_at, backend
		FROM sync_state
		WHERE file_path = ?
	`, filePath)

	var state SyncState
	var traceID sql.NullString
	err := row.Scan(
		&state.FilePath,
		&state.LastOffset,
		&state.LastSize,
		&state.LastMtime,
		&traceID,
		&state.MessageCount,
		&state.LastSyncAt,
		&state.Backend,
	)
	if err == sql.ErrNoRows {
		return nil, nil // No state yet
	}
	if err != nil {
		return nil, err
	}
	if traceID.Valid {
		state.TraceID = traceID.String
	}
	return &state, nil
}

// SaveState saves the sync state for a file.
func (d *DB) SaveState(state *SyncState) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	_, err := d.db.Exec(`
		INSERT OR REPLACE INTO sync_state
		(file_path, last_offset, last_size, last_mtime, trace_id,
		 message_count, last_sync_at, backend)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`,
		state.FilePath,
		state.LastOffset,
		state.LastSize,
		state.LastMtime,
		state.TraceID,
		state.MessageCount,
		state.LastSyncAt,
		state.Backend,
	)
	return err
}

// DeleteState removes the sync state for a file.
func (d *DB) DeleteState(filePath string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	_, err := d.db.Exec("DELETE FROM sync_state WHERE file_path = ?", filePath)
	return err
}

// RecordSyncedMessage records that a message has been synced.
func (d *DB) RecordSyncedMessage(filePath, messageHash, spanID string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	_, err := d.db.Exec(`
		INSERT OR REPLACE INTO synced_messages (file_path, message_hash, span_id, synced_at)
		VALUES (?, ?, ?, ?)
	`, filePath, messageHash, spanID, time.Now().Unix())
	return err
}

// IsSynced checks if a message has been synced.
func (d *DB) IsSynced(filePath, messageHash string) (bool, error) {
	var count int
	err := d.db.QueryRow(`
		SELECT COUNT(*) FROM synced_messages
		WHERE file_path = ? AND message_hash = ?
	`, filePath, messageHash).Scan(&count)
	return count > 0, err
}

// ClearFileMessages clears all synced messages for a file.
// Used when compaction is detected.
func (d *DB) ClearFileMessages(filePath string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	_, err := d.db.Exec("DELETE FROM synced_messages WHERE file_path = ?", filePath)
	return err
}

// RecordError records a sync error.
func (d *DB) RecordError(filePath, errorMsg string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	_, err := d.db.Exec(`
		INSERT INTO sync_errors (file_path, error_message, occurred_at)
		VALUES (?, ?, ?)
	`, filePath, errorMsg, time.Now().Unix())
	return err
}

// GetAllStates returns all sync states.
func (d *DB) GetAllStates() ([]*SyncState, error) {
	rows, err := d.db.Query(`
		SELECT file_path, last_offset, last_size, last_mtime,
			   trace_id, message_count, last_sync_at, backend
		FROM sync_state
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var states []*SyncState
	for rows.Next() {
		var state SyncState
		var traceID sql.NullString
		err := rows.Scan(
			&state.FilePath,
			&state.LastOffset,
			&state.LastSize,
			&state.LastMtime,
			&traceID,
			&state.MessageCount,
			&state.LastSyncAt,
			&state.Backend,
		)
		if err != nil {
			return nil, err
		}
		if traceID.Valid {
			state.TraceID = traceID.String
		}
		states = append(states, &state)
	}
	return states, rows.Err()
}
