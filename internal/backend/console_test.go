package backend

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/dmora/ch/internal/sync"
)

func TestConsoleBackendName(t *testing.T) {
	be := NewConsoleBackend(DefaultConsoleConfig())
	if be.Name() != "console" {
		t.Errorf("Name() = %s, want console", be.Name())
	}
}

func TestConsoleBackendSendSpanText(t *testing.T) {
	var buf bytes.Buffer
	be := NewConsoleBackend(ConsoleConfig{
		Writer:  &buf,
		Format:  "text",
		NoColor: true,
	})

	span := &sync.Span{
		ID:         "span-123",
		TraceID:    "trace-456",
		Kind:       sync.SpanKindSpan,
		Name:       "test-span",
		StartTime:  time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC),
		SourceFile: "/test/file.jsonl",
		SourceLine: 1,
	}

	err := be.SendSpan(context.Background(), span)
	if err != nil {
		t.Fatalf("SendSpan failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "[SYNC]") {
		t.Error("Output should contain [SYNC]")
	}
	if !strings.Contains(output, "span") {
		t.Error("Output should contain span kind")
	}
	if !strings.Contains(output, "test-span") {
		t.Error("Output should contain span name")
	}

	// Check stats
	stats := be.Stats()
	if stats.SpansSent != 1 {
		t.Errorf("SpansSent = %d, want 1", stats.SpansSent)
	}
}

func TestConsoleBackendSendSpanJSON(t *testing.T) {
	var buf bytes.Buffer
	be := NewConsoleBackend(ConsoleConfig{
		Writer: &buf,
		Format: "json",
	})

	span := &sync.Span{
		ID:         "span-123",
		TraceID:    "trace-456",
		Kind:       sync.SpanKindGeneration,
		Name:       "test-generation",
		StartTime:  time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC),
		Model:      "claude-3",
		SourceFile: "/test/file.jsonl",
		SourceLine: 2,
	}

	err := be.SendSpan(context.Background(), span)
	if err != nil {
		t.Fatalf("SendSpan failed: %v", err)
	}

	output := buf.String()

	// Should be valid JSON
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Errorf("Output is not valid JSON: %v", err)
	}

	// Check fields
	if result["id"] != "span-123" {
		t.Errorf("id = %v, want span-123", result["id"])
	}
	if result["name"] != "test-generation" {
		t.Errorf("name = %v, want test-generation", result["name"])
	}
}

func TestConsoleBackendSendSpanVerbose(t *testing.T) {
	var buf bytes.Buffer
	be := NewConsoleBackend(ConsoleConfig{
		Writer:  &buf,
		Format:  "text",
		Verbose: true,
		NoColor: true,
	})

	span := &sync.Span{
		ID:         "span-123",
		TraceID:    "trace-456",
		Kind:       sync.SpanKindSpan,
		Name:       "test-span",
		Input:      "test input",
		Output:     "test output",
		Model:      "claude-3",
		StartTime:  time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC),
		SourceFile: "/test/file.jsonl",
		SourceLine: 3,
	}

	err := be.SendSpan(context.Background(), span)
	if err != nil {
		t.Fatalf("SendSpan failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "input") {
		t.Error("Verbose output should contain input")
	}
	if !strings.Contains(output, "output") {
		t.Error("Verbose output should contain output")
	}
	if !strings.Contains(output, "model") {
		t.Error("Verbose output should contain model")
	}
	if !strings.Contains(output, "source") {
		t.Error("Verbose output should contain source")
	}
}

func TestConsoleBackendSendBatch(t *testing.T) {
	var buf bytes.Buffer
	be := NewConsoleBackend(ConsoleConfig{
		Writer:  &buf,
		Format:  "text",
		NoColor: true,
	})

	batch := &sync.SpanBatch{
		TraceID:   "trace-123",
		SessionID: "session-456",
		Project:   "test-project",
		Spans: []*sync.Span{
			{
				ID:         "span-1",
				TraceID:    "trace-123",
				Kind:       sync.SpanKindSpan,
				Name:       "span-1",
				StartTime:  time.Now(),
				SourceFile: "/test/file.jsonl",
				SourceLine: 1,
			},
			{
				ID:         "span-2",
				TraceID:    "trace-123",
				Kind:       sync.SpanKindGeneration,
				Name:       "span-2",
				StartTime:  time.Now(),
				SourceFile: "/test/file.jsonl",
				SourceLine: 2,
			},
		},
	}

	err := be.SendBatch(context.Background(), batch)
	if err != nil {
		t.Fatalf("SendBatch failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "[BATCH]") {
		t.Error("Output should contain [BATCH]")
	}
	if !strings.Contains(output, "2 spans") {
		t.Error("Output should contain span count")
	}

	// Check stats
	stats := be.Stats()
	if stats.SpansSent != 2 {
		t.Errorf("SpansSent = %d, want 2", stats.SpansSent)
	}
}

func TestConsoleBackendFlushAndClose(t *testing.T) {
	be := NewConsoleBackend(DefaultConsoleConfig())

	// Flush should be no-op
	err := be.Flush(context.Background())
	if err != nil {
		t.Errorf("Flush failed: %v", err)
	}

	// Close should be no-op
	err = be.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"hello", 10, "hello"},
		{"hello world", 5, "he..."},
		{"", 10, ""},
		{"hello\nworld", 20, "hello world"},
		{"hello\tworld", 20, "hello world"},
	}

	for _, tc := range tests {
		result := truncate(tc.input, tc.maxLen)
		if result != tc.expected {
			t.Errorf("truncate(%q, %d) = %q, want %q", tc.input, tc.maxLen, result, tc.expected)
		}
	}
}
