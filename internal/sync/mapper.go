package sync

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/dmora/ch/internal/jsonl"
)

// Mapper converts JSONL entries to spans.
type Mapper struct {
	filePath string
	lineNum  int
}

// NewMapper creates a new span mapper for a file.
func NewMapper(filePath string) *Mapper {
	return &Mapper{
		filePath: filePath,
		lineNum:  0,
	}
}

// MapEntry converts a JSONL entry to a span.
// Returns nil if the entry should not produce a span.
func (m *Mapper) MapEntry(entry *jsonl.RawEntry, lineNum int) (*Span, error) {
	m.lineNum = lineNum

	switch entry.Type {
	case jsonl.EntryTypeUser:
		return m.mapUserMessage(entry)
	case jsonl.EntryTypeAssistant:
		return m.mapAssistantMessage(entry)
	case jsonl.EntryTypeSummary:
		return m.mapSummary(entry)
	case jsonl.EntryTypeSystem:
		return m.mapSystemMessage(entry)
	default:
		// Skip file-history-snapshot, queue-operation, etc.
		return nil, nil
	}
}

// mapUserMessage maps a user message entry to a span.
func (m *Mapper) mapUserMessage(entry *jsonl.RawEntry) (*Span, error) {
	msg, err := jsonl.ParseMessage(entry)
	if err != nil {
		return nil, fmt.Errorf("parsing user message: %w", err)
	}

	var text string
	if msg != nil {
		text = jsonl.ExtractText(msg)
	}
	timestamp := m.parseTimestamp(entry.Timestamp)

	return &Span{
		ID:         m.generateSpanID(entry),
		TraceID:    entry.SessionID,
		Kind:       SpanKindSpan,
		Name:       "user-message",
		StartTime:  timestamp,
		EndTime:    timestamp,
		Input:      text,
		SourceFile: m.filePath,
		SourceLine: m.lineNum,
		Metadata: map[string]interface{}{
			"uuid":        entry.UUID,
			"parent_uuid": entry.ParentUUID,
		},
	}, nil
}

// mapAssistantMessage maps an assistant message to a generation span.
func (m *Mapper) mapAssistantMessage(entry *jsonl.RawEntry) (*Span, error) {
	msg, err := jsonl.ParseMessage(entry)
	if err != nil {
		return nil, fmt.Errorf("parsing assistant message: %w", err)
	}

	var text, model string
	if msg != nil {
		text = jsonl.ExtractText(msg)
		model = msg.Model
	}
	timestamp := m.parseTimestamp(entry.Timestamp)

	span := &Span{
		ID:         m.generateSpanID(entry),
		TraceID:    entry.SessionID,
		Kind:       SpanKindGeneration,
		Name:       "assistant-generation",
		StartTime:  timestamp,
		EndTime:    timestamp,
		Output:     text,
		Model:      model,
		SourceFile: m.filePath,
		SourceLine: m.lineNum,
		Metadata:   make(map[string]interface{}),
	}

	// Add thinking if present
	if msg != nil {
		if thinking := jsonl.ExtractThinking(msg); thinking != "" {
			span.Metadata["thinking"] = thinking
		}

		// Add tool calls summary if present
		if tools := jsonl.ExtractToolCalls(msg); len(tools) > 0 {
			span.Metadata["tool_calls"] = tools
		}
	}

	if entry.UUID != "" {
		span.Metadata["uuid"] = entry.UUID
	}

	return span, nil
}

// mapSummary maps a summary entry to a span.
func (m *Mapper) mapSummary(entry *jsonl.RawEntry) (*Span, error) {
	timestamp := m.parseTimestamp(entry.Timestamp)

	return &Span{
		ID:         m.generateSpanID(entry),
		TraceID:    entry.SessionID,
		Kind:       SpanKindSpan,
		Name:       "context-summary",
		StartTime:  timestamp,
		EndTime:    timestamp,
		Output:     entry.Summary,
		SourceFile: m.filePath,
		SourceLine: m.lineNum,
	}, nil
}

// mapSystemMessage maps a system message entry to a span.
func (m *Mapper) mapSystemMessage(entry *jsonl.RawEntry) (*Span, error) {
	msg, err := jsonl.ParseMessage(entry)
	if err != nil {
		return nil, fmt.Errorf("parsing system message: %w", err)
	}

	text := jsonl.ExtractText(msg)
	timestamp := m.parseTimestamp(entry.Timestamp)

	return &Span{
		ID:         m.generateSpanID(entry),
		TraceID:    entry.SessionID,
		Kind:       SpanKindSpan,
		Name:       "system-message",
		StartTime:  timestamp,
		EndTime:    timestamp,
		Input:      text,
		SourceFile: m.filePath,
		SourceLine: m.lineNum,
	}, nil
}

// generateSpanID creates a unique span ID from entry data.
func (m *Mapper) generateSpanID(entry *jsonl.RawEntry) string {
	// Use UUID if available
	if entry.UUID != "" {
		return entry.UUID
	}

	// Otherwise hash the content
	h := sha256.New()
	h.Write([]byte(m.filePath))
	h.Write([]byte(fmt.Sprintf("%d", m.lineNum)))
	h.Write([]byte(entry.Timestamp))
	h.Write(entry.Message)
	return hex.EncodeToString(h.Sum(nil))[:16]
}

// parseTimestamp parses an RFC3339 timestamp, falling back to now.
func (m *Mapper) parseTimestamp(ts string) time.Time {
	if ts == "" {
		return time.Now()
	}
	t, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		// Try RFC3339Nano
		t, err = time.Parse(time.RFC3339Nano, ts)
		if err != nil {
			return time.Now()
		}
	}
	return t
}

// GenerateMessageHash creates a hash for deduplication.
func GenerateMessageHash(entry *jsonl.RawEntry) string {
	h := sha256.New()
	h.Write([]byte(entry.Type))
	h.Write([]byte(entry.SessionID))
	h.Write([]byte(entry.Timestamp))
	h.Write([]byte(entry.UUID))
	h.Write(entry.Message)
	return hex.EncodeToString(h.Sum(nil))[:32]
}
