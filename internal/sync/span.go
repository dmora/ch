// Package sync provides conversation sync to observability backends.
package sync

import (
	"context"
	"time"
)

// SpanKind indicates the type of span.
type SpanKind string

const (
	SpanKindTrace      SpanKind = "trace"      // Root span for a conversation
	SpanKindGeneration SpanKind = "generation" // LLM generation (assistant message)
	SpanKindSpan       SpanKind = "span"       // User message, tool call, etc.
)

// Span represents a telemetry span for export.
type Span struct {
	// Identity
	ID       string `json:"id"`        // Unique span ID (derived from entry UUID or hash)
	TraceID  string `json:"trace_id"`  // Trace ID (conversation session ID)
	ParentID string `json:"parent_id"` // Parent span ID (for nested spans)

	// Timing
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`

	// Classification
	Kind SpanKind `json:"kind"`
	Name string   `json:"name"` // e.g., "user-message", "assistant-generation", "tool-Read"

	// Content
	Input  string `json:"input,omitempty"`  // User message or tool input
	Output string `json:"output,omitempty"` // Assistant response or tool output

	// Metadata
	Metadata map[string]interface{} `json:"metadata,omitempty"`

	// LLM-specific (for generation spans)
	Model       string `json:"model,omitempty"`
	TokensIn    int    `json:"tokens_in,omitempty"`
	TokensOut   int    `json:"tokens_out,omitempty"`
	IsStreaming bool   `json:"is_streaming,omitempty"`

	// Tool-specific
	ToolName   string `json:"tool_name,omitempty"`
	ToolResult string `json:"tool_result,omitempty"`
	IsError    bool   `json:"is_error,omitempty"`

	// Source info
	SourceFile string `json:"source_file"` // Original JSONL file path
	SourceLine int    `json:"source_line"` // Line number in JSONL file
}

// SpanBatch represents a batch of spans to be sent to backend.
type SpanBatch struct {
	TraceID   string    `json:"trace_id"`
	SessionID string    `json:"session_id"`
	Project   string    `json:"project"`
	Spans     []*Span   `json:"spans"`
	CreatedAt time.Time `json:"created_at"`
}

// Backend defines the interface for sync backends.
type Backend interface {
	// Name returns the backend identifier.
	Name() string

	// SendSpan sends a single span to the backend.
	SendSpan(ctx context.Context, span *Span) error

	// SendBatch sends a batch of spans to the backend.
	SendBatch(ctx context.Context, batch *SpanBatch) error

	// Flush ensures all pending spans are sent.
	Flush(ctx context.Context) error

	// Close releases backend resources.
	Close() error
}
