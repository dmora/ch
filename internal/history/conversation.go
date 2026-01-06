package history

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/dmora/ch/internal/jsonl"
)

// ConversationMeta contains lightweight metadata for listing conversations.
type ConversationMeta struct {
	ID              string    // UUID/AgentID from filename
	SessionID       string    // Session ID (for agents, points to parent)
	Path            string    // Full file path
	Project         string    // Project directory name (encoded)
	ProjectPath     string    // Decoded project path
	Timestamp       time.Time // From first entry or file mtime
	Preview         string    // First ~100 chars of first user message
	MessageCount    int       // Number of user+assistant messages
	IsAgent         bool      // Is this an agent/sidechain conversation
	AgentCount      int       // Number of agents spawned (for main conversations)
	ParentSessionID string    // Parent session ID (for agents only)
	FileSize        int64     // For stats
	Model           string    // Model used (from first assistant message)
}

// Conversation represents a fully loaded conversation with all messages.
type Conversation struct {
	Meta    ConversationMeta
	Entries []*jsonl.RawEntry
}

// ScanConversationMeta scans a JSONL file to extract metadata efficiently.
// It only parses the minimum necessary to extract preview and counts.
func ScanConversationMeta(path string) (*ConversationMeta, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return nil, err
	}

	meta := initMetaFromPath(path, info)
	parser := jsonl.NewParserFromReader(file)
	state := &metaScanState{}

	for {
		entry, err := parser.Next()
		if err != nil || entry == nil {
			break
		}
		updateMetaFromEntry(meta, entry, state)
	}

	return meta, nil
}

// initMetaFromPath creates initial metadata from file path and info.
func initMetaFromPath(path string, info os.FileInfo) *ConversationMeta {
	filename := filepath.Base(path)
	projectDir := filepath.Base(filepath.Dir(path))

	meta := &ConversationMeta{
		Path:        path,
		Project:     projectDir,
		ProjectPath: DecodeProjectPath(projectDir),
		FileSize:    info.Size(),
		Timestamp:   info.ModTime(),
		IsAgent:     IsAgentFile(filename),
	}

	if meta.IsAgent {
		meta.ID = ExtractAgentID(filename)
	} else {
		meta.ID = ExtractSessionID(filename)
		meta.SessionID = meta.ID
	}

	return meta
}

// metaScanState tracks scanning progress across entries.
type metaScanState struct {
	firstUserFound   bool
	firstTimestamp   time.Time
}

// updateMetaFromEntry updates metadata from a single entry.
func updateMetaFromEntry(meta *ConversationMeta, entry *jsonl.RawEntry, state *metaScanState) {
	updateSessionInfo(meta, entry)
	updateTimestamp(meta, entry, state)
	updateMessageStats(meta, entry, state)
}

// updateSessionInfo extracts session info from entry.
func updateSessionInfo(meta *ConversationMeta, entry *jsonl.RawEntry) {
	if entry.SessionID == "" {
		return
	}
	if meta.SessionID == "" {
		meta.SessionID = entry.SessionID
	}
	if meta.IsAgent && meta.ParentSessionID == "" {
		meta.ParentSessionID = entry.SessionID
	}
}

// updateTimestamp updates timestamp from first entry with timestamp.
func updateTimestamp(meta *ConversationMeta, entry *jsonl.RawEntry, state *metaScanState) {
	if entry.Timestamp == "" || !state.firstTimestamp.IsZero() {
		return
	}
	if t, err := time.Parse(time.RFC3339, entry.Timestamp); err == nil {
		state.firstTimestamp = t
		meta.Timestamp = t
	}
}

// updateMessageStats updates message count, preview, and model.
func updateMessageStats(meta *ConversationMeta, entry *jsonl.RawEntry, state *metaScanState) {
	if entry.Type.IsUserOrAssistant() {
		meta.MessageCount++
	}

	if entry.Type == jsonl.EntryTypeUser && !state.firstUserFound {
		meta.Preview = jsonl.ExtractPreview(entry.Message, 100)
		state.firstUserFound = true
	}

	if entry.Type == jsonl.EntryTypeAssistant && meta.Model == "" && entry.Message != nil {
		var msg jsonl.Message
		if json.Unmarshal(entry.Message, &msg) == nil && msg.Model != "" {
			meta.Model = msg.Model
		}
	}
}

// LoadConversation fully loads a conversation from a JSONL file.
func LoadConversation(path string) (*Conversation, error) {
	meta, err := ScanConversationMeta(path)
	if err != nil {
		return nil, err
	}

	parser, err := jsonl.NewParser(path)
	if err != nil {
		return nil, err
	}
	defer parser.Close()

	entries, err := parser.ParseAll()
	if err != nil {
		return nil, err
	}

	return &Conversation{
		Meta:    *meta,
		Entries: entries,
	}, nil
}

// GetMessages returns only the message entries (user, assistant, system).
func (c *Conversation) GetMessages() []*jsonl.RawEntry {
	var messages []*jsonl.RawEntry
	for _, entry := range c.Entries {
		if entry.Type.IsMessage() {
			messages = append(messages, entry)
		}
	}
	return messages
}

// GetUserMessages returns only user message entries.
func (c *Conversation) GetUserMessages() []*jsonl.RawEntry {
	var messages []*jsonl.RawEntry
	for _, entry := range c.Entries {
		if entry.Type == jsonl.EntryTypeUser {
			messages = append(messages, entry)
		}
	}
	return messages
}

// GetAssistantMessages returns only assistant message entries.
func (c *Conversation) GetAssistantMessages() []*jsonl.RawEntry {
	var messages []*jsonl.RawEntry
	for _, entry := range c.Entries {
		if entry.Type == jsonl.EntryTypeAssistant {
			messages = append(messages, entry)
		}
	}
	return messages
}

// GetSummaries returns only summary type entries.
func (c *Conversation) GetSummaries() []*jsonl.RawEntry {
	var summaries []*jsonl.RawEntry
	for _, entry := range c.Entries {
		if entry.Type == jsonl.EntryTypeSummary {
			summaries = append(summaries, entry)
		}
	}
	return summaries
}

// ParseMessageEntry parses a raw entry into a Message struct.
func ParseMessageEntry(entry *jsonl.RawEntry) (*jsonl.Message, error) {
	return jsonl.ParseMessage(entry)
}

// ExtractMessageText extracts text content from a parsed message.
func ExtractMessageText(msg *jsonl.Message) string {
	return jsonl.ExtractText(msg)
}
