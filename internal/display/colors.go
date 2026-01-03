// Package display provides utilities for formatting output to the terminal.
package display

import (
	"os"

	"github.com/fatih/color"
	"github.com/mattn/go-isatty"
)

// IsTTY returns true if stdout is a terminal.
func IsTTY() bool {
	return isatty.IsTerminal(os.Stdout.Fd()) || isatty.IsCygwinTerminal(os.Stdout.Fd())
}

// SetColorEnabled enables or disables color output.
func SetColorEnabled(enabled bool) {
	color.NoColor = !enabled
}

// DisableColorIfNotTTY disables color output if stdout is not a terminal.
func DisableColorIfNotTTY() {
	if !IsTTY() {
		color.NoColor = true
	}
}

// Color shortcuts
var (
	// Headers and titles
	Bold    = color.New(color.Bold).SprintFunc()
	Title   = color.New(color.Bold, color.FgCyan).SprintFunc()
	Header  = color.New(color.Bold, color.FgWhite).SprintFunc()
	Section = color.New(color.Bold, color.FgBlue).SprintFunc()

	// Roles
	UserRole      = color.New(color.Bold, color.FgGreen).SprintFunc()
	AssistantRole = color.New(color.Bold, color.FgBlue).SprintFunc()
	SystemRole    = color.New(color.Bold, color.FgYellow).SprintFunc()

	// Content types
	Thinking = color.New(color.Italic, color.FgMagenta).SprintFunc()
	ToolCall = color.New(color.FgCyan).SprintFunc()
	ToolName = color.New(color.Bold, color.FgCyan).SprintFunc()
	Error    = color.New(color.FgRed).SprintFunc()

	// Metadata
	Dim       = color.New(color.Faint).SprintFunc()
	Timestamp = color.New(color.Faint).SprintFunc()
	ID        = color.New(color.FgYellow).SprintFunc()
	Project   = color.New(color.FgMagenta).SprintFunc()
	Model     = color.New(color.Faint, color.FgCyan).SprintFunc()

	// Search
	Match = color.New(color.Bold, color.FgYellow).SprintFunc()

	// Status
	Success = color.New(color.FgGreen).SprintFunc()
	Warning = color.New(color.FgYellow).SprintFunc()
	Info    = color.New(color.FgCyan).SprintFunc()

	// Numbers
	Number = color.New(color.FgYellow).SprintFunc()
	Size   = color.New(color.FgCyan).SprintFunc()
)

// FormatBytes formats a byte count as human-readable string.
func FormatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return color.New(color.FgCyan).Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return color.New(color.FgCyan).Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
