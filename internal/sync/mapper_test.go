package sync

import (
	"testing"

	"github.com/dmora/ch/internal/jsonl"
)

func TestMapperUserMessage(t *testing.T) {
	mapper := NewMapper("/test/file.jsonl")

	entry := &jsonl.RawEntry{
		Type:      "user",
		SessionID: "session-123",
		Timestamp: "2025-01-01T12:00:00Z",
	}

	span, err := mapper.MapEntry(entry, 1)
	if err != nil {
		t.Fatalf("MapEntry failed: %v", err)
	}
	if span == nil {
		t.Fatal("Expected non-nil span")
	}

	if span.Kind != SpanKindSpan {
		t.Errorf("Kind = %s, want span", span.Kind)
	}
	if span.Name != "user-message" {
		t.Errorf("Name = %s, want user-message", span.Name)
	}
	if span.TraceID != "session-123" {
		t.Errorf("TraceID = %s, want session-123", span.TraceID)
	}
	if span.SourceFile != "/test/file.jsonl" {
		t.Errorf("SourceFile = %s, want /test/file.jsonl", span.SourceFile)
	}
	if span.SourceLine != 1 {
		t.Errorf("SourceLine = %d, want 1", span.SourceLine)
	}
}

func TestMapperAssistantMessage(t *testing.T) {
	mapper := NewMapper("/test/file.jsonl")

	entry := &jsonl.RawEntry{
		Type:      "assistant",
		SessionID: "session-456",
		Timestamp: "2025-01-01T12:00:00Z",
	}

	span, err := mapper.MapEntry(entry, 2)
	if err != nil {
		t.Fatalf("MapEntry failed: %v", err)
	}
	if span == nil {
		t.Fatal("Expected non-nil span")
	}

	if span.Kind != SpanKindGeneration {
		t.Errorf("Kind = %s, want generation", span.Kind)
	}
	if span.Name != "assistant-generation" {
		t.Errorf("Name = %s, want assistant-generation", span.Name)
	}
}

func TestMapperSystemMessage(t *testing.T) {
	mapper := NewMapper("/test/file.jsonl")

	entry := &jsonl.RawEntry{
		Type:      "system",
		SessionID: "session-789",
		Timestamp: "2025-01-01T12:00:00Z",
	}

	span, err := mapper.MapEntry(entry, 3)
	if err != nil {
		t.Fatalf("MapEntry failed: %v", err)
	}
	if span == nil {
		t.Fatal("Expected non-nil span")
	}

	if span.Kind != SpanKindSpan {
		t.Errorf("Kind = %s, want span", span.Kind)
	}
	if span.Name != "system-message" {
		t.Errorf("Name = %s, want system-message", span.Name)
	}
}

func TestMapperSummary(t *testing.T) {
	mapper := NewMapper("/test/file.jsonl")

	entry := &jsonl.RawEntry{
		Type:      "summary",
		SessionID: "session-abc",
		Timestamp: "2025-01-01T12:00:00Z",
	}

	span, err := mapper.MapEntry(entry, 4)
	if err != nil {
		t.Fatalf("MapEntry failed: %v", err)
	}
	if span == nil {
		t.Fatal("Expected non-nil span")
	}

	if span.Kind != SpanKindSpan {
		t.Errorf("Kind = %s, want span", span.Kind)
	}
	if span.Name != "context-summary" {
		t.Errorf("Name = %s, want context-summary", span.Name)
	}
}

func TestMapperUnknownType(t *testing.T) {
	mapper := NewMapper("/test/file.jsonl")

	entry := &jsonl.RawEntry{
		Type:      "unknown-type",
		SessionID: "session-xyz",
		Timestamp: "2025-01-01T12:00:00Z",
	}

	span, err := mapper.MapEntry(entry, 5)
	if err != nil {
		t.Fatalf("MapEntry failed: %v", err)
	}
	// Unknown types should return nil (skip)
	if span != nil {
		t.Error("Expected nil span for unknown type")
	}
}

func TestGenerateMessageHash(t *testing.T) {
	entry1 := &jsonl.RawEntry{
		Type:      "user",
		SessionID: "session-1",
		Timestamp: "2025-01-01T12:00:00Z",
	}
	entry2 := &jsonl.RawEntry{
		Type:      "user",
		SessionID: "session-1",
		Timestamp: "2025-01-01T12:00:00Z",
	}
	entry3 := &jsonl.RawEntry{
		Type:      "user",
		SessionID: "session-2", // Different session
		Timestamp: "2025-01-01T12:00:00Z",
	}

	hash1 := GenerateMessageHash(entry1)
	hash2 := GenerateMessageHash(entry2)
	hash3 := GenerateMessageHash(entry3)

	// Same entries should produce same hash
	if hash1 != hash2 {
		t.Error("Same entries should produce same hash")
	}

	// Different entries should produce different hash
	if hash1 == hash3 {
		t.Error("Different entries should produce different hash")
	}

	// Hash should be non-empty
	if hash1 == "" {
		t.Error("Hash should not be empty")
	}
}

func TestSpanIDDeterminism(t *testing.T) {
	// Test that same entry produces same span ID
	mapper := NewMapper("/test/file.jsonl")

	entry := &jsonl.RawEntry{
		Type:      "user",
		SessionID: "session-1",
		Timestamp: "2025-01-01T12:00:00Z",
	}

	span1, _ := mapper.MapEntry(entry, 1)
	span2, _ := mapper.MapEntry(entry, 1)

	if span1.ID != span2.ID {
		t.Error("Same entry should produce same span ID")
	}

	// Different line number should produce different ID
	span3, _ := mapper.MapEntry(entry, 2)
	if span1.ID == span3.ID {
		t.Error("Different line numbers should produce different span IDs")
	}
}
