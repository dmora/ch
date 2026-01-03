package jsonl

import (
	"strings"
	"testing"
)

func TestNewParserFromReader(t *testing.T) {
	input := `{"type":"user","message":{"role":"user","content":"Hello"}}
{"type":"assistant","message":{"role":"assistant","content":"Hi!"}}
`
	parser := NewParserFromReader(strings.NewReader(input))

	// First entry
	entry1, err := parser.Next()
	if err != nil {
		t.Fatalf("Next() error = %v", err)
	}
	if entry1.Type != EntryTypeUser {
		t.Errorf("entry1.Type = %q, want %q", entry1.Type, EntryTypeUser)
	}

	// Second entry
	entry2, err := parser.Next()
	if err != nil {
		t.Fatalf("Next() error = %v", err)
	}
	if entry2.Type != EntryTypeAssistant {
		t.Errorf("entry2.Type = %q, want %q", entry2.Type, EntryTypeAssistant)
	}

	// EOF
	entry3, err := parser.Next()
	if err != nil {
		t.Fatalf("Next() error = %v", err)
	}
	if entry3 != nil {
		t.Error("Expected nil at EOF")
	}
}

func TestParser_SkipsEmptyLines(t *testing.T) {
	input := `{"type":"user"}

{"type":"assistant"}
`
	parser := NewParserFromReader(strings.NewReader(input))

	entry1, _ := parser.Next()
	if entry1.Type != EntryTypeUser {
		t.Errorf("entry1.Type = %q, want %q", entry1.Type, EntryTypeUser)
	}

	entry2, _ := parser.Next()
	if entry2.Type != EntryTypeAssistant {
		t.Errorf("entry2.Type = %q, want %q", entry2.Type, EntryTypeAssistant)
	}
}

func TestParser_ParseAll(t *testing.T) {
	input := `{"type":"user"}
{"type":"assistant"}
{"type":"system"}
`
	parser := NewParserFromReader(strings.NewReader(input))
	entries, err := parser.ParseAll()
	if err != nil {
		t.Fatalf("ParseAll() error = %v", err)
	}
	if len(entries) != 3 {
		t.Errorf("len(entries) = %d, want 3", len(entries))
	}
}

func TestParseEntry(t *testing.T) {
	line := []byte(`{"type":"user","timestamp":"2024-01-01T00:00:00Z"}`)
	entry, err := ParseEntry(line)
	if err != nil {
		t.Fatalf("ParseEntry() error = %v", err)
	}
	if entry.Type != EntryTypeUser {
		t.Errorf("Type = %q, want %q", entry.Type, EntryTypeUser)
	}
}

func TestParseEntry_Invalid(t *testing.T) {
	line := []byte(`not valid json`)
	_, err := ParseEntry(line)
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}

func TestParseMessage(t *testing.T) {
	entry := &RawEntry{
		Type:    EntryTypeUser,
		Message: []byte(`{"role":"user","content":"Hello"}`),
	}

	msg, err := ParseMessage(entry)
	if err != nil {
		t.Fatalf("ParseMessage() error = %v", err)
	}
	if msg.Role != "user" {
		t.Errorf("Role = %q, want %q", msg.Role, "user")
	}
	if len(msg.Content) != 1 {
		t.Fatalf("len(Content) = %d, want 1", len(msg.Content))
	}
	if msg.Content[0].Text != "Hello" {
		t.Errorf("Content[0].Text = %q, want %q", msg.Content[0].Text, "Hello")
	}
}

func TestParseMessage_NilMessage(t *testing.T) {
	entry := &RawEntry{
		Type:    EntryTypeUser,
		Message: nil,
	}

	msg, err := ParseMessage(entry)
	if err != nil {
		t.Fatalf("ParseMessage() error = %v", err)
	}
	if msg != nil {
		t.Error("Expected nil message")
	}
}

func TestParser_NextRaw(t *testing.T) {
	input := `{"type":"user"}
{"type":"assistant"}
`
	parser := NewParserFromReader(strings.NewReader(input))

	line1, err := parser.NextRaw()
	if err != nil {
		t.Fatalf("NextRaw() error = %v", err)
	}
	if string(line1) != `{"type":"user"}` {
		t.Errorf("line1 = %q, want %q", string(line1), `{"type":"user"}`)
	}

	line2, err := parser.NextRaw()
	if err != nil {
		t.Fatalf("NextRaw() error = %v", err)
	}
	if string(line2) != `{"type":"assistant"}` {
		t.Errorf("line2 = %q, want %q", string(line2), `{"type":"assistant"}`)
	}
}
