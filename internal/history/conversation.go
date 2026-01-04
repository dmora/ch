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

	parser := jsonl.NewParserFromReader(file)
	var firstUserFound bool
	var firstTimestamp time.Time

	for {
		entry, err := parser.Next()
		if err != nil || entry == nil {
			break
		}

		// Extract session info from first entry
		if meta.SessionID == "" && entry.SessionID != "" {
			meta.SessionID = entry.SessionID
		}
		if meta.IsAgent && meta.ParentSessionID == "" && entry.SessionID != "" {
			meta.ParentSessionID = entry.SessionID
		}

		// Track timestamp from first entry
		if entry.Timestamp != "" && firstTimestamp.IsZero() {
			if t, err := time.Parse(time.RFC3339, entry.Timestamp); err == nil {
				firstTimestamp = t
				meta.Timestamp = t
			}
		}

		// Count messages
		if entry.Type.IsUserOrAssistant() {
			meta.MessageCount++
		}

		// Extract preview from first user message
		if entry.Type == jsonl.EntryTypeUser && !firstUserFound {
			meta.Preview = jsonl.ExtractPreview(entry.Message, 100)
			firstUserFound = true
		}

		// Extract model from first assistant message
		if entry.Type == jsonl.EntryTypeAssistant && meta.Model == "" && entry.Message != nil {
			var msg jsonl.Message
			if json.Unmarshal(entry.Message, &msg) == nil && msg.Model != "" {
				meta.Model = msg.Model
			}
		}
	}

	return meta, nil
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
