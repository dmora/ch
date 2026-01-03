package backend

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/dmora/ch/internal/sync"
	"github.com/fatih/color"
)

// ConsoleConfig configures the console backend.
type ConsoleConfig struct {
	Writer  io.Writer
	Verbose bool
	Format  string // "text" or "json"
	NoColor bool
}

// DefaultConsoleConfig returns default console configuration.
func DefaultConsoleConfig() ConsoleConfig {
	return ConsoleConfig{
		Writer:  os.Stdout,
		Verbose: false,
		Format:  "text",
	}
}

// ConsoleBackend outputs spans to the console for testing.
type ConsoleBackend struct {
	config ConsoleConfig
	stats  Stats
}

// NewConsoleBackend creates a new console backend.
func NewConsoleBackend(config ConsoleConfig) *ConsoleBackend {
	if config.Writer == nil {
		config.Writer = os.Stdout
	}
	return &ConsoleBackend{
		config: config,
	}
}

// Name returns "console".
func (c *ConsoleBackend) Name() string {
	return "console"
}

// SendSpan outputs a span to the console.
func (c *ConsoleBackend) SendSpan(ctx context.Context, span *sync.Span) error {
	if c.config.Format == "json" {
		return c.sendJSON(span)
	}
	return c.sendText(span)
}

// sendJSON outputs span as JSON.
func (c *ConsoleBackend) sendJSON(span *sync.Span) error {
	data, err := json.MarshalIndent(span, "", "  ")
	if err != nil {
		c.stats.SpansFailed++
		return err
	}
	fmt.Fprintln(c.config.Writer, string(data))
	c.stats.SpansSent++
	c.stats.BytesSent += int64(len(data))
	return nil
}

// sendText outputs span as formatted text.
func (c *ConsoleBackend) sendText(span *sync.Span) error {
	w := c.config.Writer

	// Color functions (disabled if NoColor)
	dim := color.New(color.Faint).SprintFunc()
	bold := color.New(color.Bold).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()

	if c.config.NoColor {
		dim = fmt.Sprint
		bold = fmt.Sprint
		cyan = fmt.Sprint
		green = fmt.Sprint
		yellow = fmt.Sprint
	}

	// Header line
	kindColor := cyan
	switch span.Kind {
	case sync.SpanKindGeneration:
		kindColor = green
	case sync.SpanKindTrace:
		kindColor = yellow
	}

	fmt.Fprintf(w, "%s %s %s\n",
		dim("[SYNC]"),
		kindColor(string(span.Kind)),
		bold(span.Name),
	)

	// IDs
	traceID := span.TraceID
	if len(traceID) > 8 {
		traceID = traceID[:8]
	}
	spanID := span.ID
	if len(spanID) > 8 {
		spanID = spanID[:8]
	}
	fmt.Fprintf(w, "  %s: %s  %s: %s\n",
		dim("trace"), traceID,
		dim("span"), spanID,
	)

	// Time
	fmt.Fprintf(w, "  %s: %s\n",
		dim("time"), span.StartTime.Format(time.RFC3339),
	)

	// Verbose output
	if c.config.Verbose {
		// Input/Output
		if span.Input != "" {
			fmt.Fprintf(w, "  %s: %s\n", dim("input"), truncate(span.Input, 200))
		}
		if span.Output != "" {
			fmt.Fprintf(w, "  %s: %s\n", dim("output"), truncate(span.Output, 200))
		}

		// Model
		if span.Model != "" {
			fmt.Fprintf(w, "  %s: %s\n", dim("model"), span.Model)
		}

		// Source
		fmt.Fprintf(w, "  %s: %s:%d\n",
			dim("source"), span.SourceFile, span.SourceLine)
	}

	fmt.Fprintln(w) // Blank line between spans

	c.stats.SpansSent++
	return nil
}

// SendBatch outputs a batch of spans.
func (c *ConsoleBackend) SendBatch(ctx context.Context, batch *sync.SpanBatch) error {
	if c.config.Format == "json" {
		data, err := json.MarshalIndent(batch, "", "  ")
		if err != nil {
			return err
		}
		fmt.Fprintln(c.config.Writer, string(data))
		c.stats.SpansSent += len(batch.Spans)
		c.stats.BytesSent += int64(len(data))
		return nil
	}

	// Text format: header + spans
	dim := color.New(color.Faint).SprintFunc()
	bold := color.New(color.Bold).SprintFunc()
	if c.config.NoColor {
		dim = fmt.Sprint
		bold = fmt.Sprint
	}

	traceID := batch.TraceID
	if len(traceID) > 8 {
		traceID = traceID[:8]
	}

	fmt.Fprintf(c.config.Writer, "%s %s %s (%d spans)\n",
		dim("[BATCH]"),
		bold(traceID),
		dim(batch.Project),
		len(batch.Spans),
	)
	fmt.Fprintln(c.config.Writer)

	for _, span := range batch.Spans {
		if err := c.SendSpan(ctx, span); err != nil {
			return err
		}
	}
	return nil
}

// Flush is a no-op for console backend.
func (c *ConsoleBackend) Flush(ctx context.Context) error {
	return nil
}

// Close is a no-op for console backend.
func (c *ConsoleBackend) Close() error {
	return nil
}

// Stats returns backend statistics.
func (c *ConsoleBackend) Stats() Stats {
	return c.stats
}

// truncate shortens a string for display.
func truncate(s string, maxLen int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\t", " ")
	if len(s) > maxLen {
		return s[:maxLen-3] + "..."
	}
	return s
}
