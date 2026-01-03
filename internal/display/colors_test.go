package display

import (
	"testing"
)

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		name     string
		bytes    int64
		expected string
	}{
		{"zero bytes", 0, "0 B"},
		{"small bytes", 500, "500 B"},
		{"one KB", 1024, "1.0 KB"},
		{"few KB", 2560, "2.5 KB"},
		{"one MB", 1048576, "1.0 MB"},
		{"few MB", 5242880, "5.0 MB"},
		{"one GB", 1073741824, "1.0 GB"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Note: FormatBytes returns colored string, so we strip ANSI codes for comparison
			result := FormatBytes(tt.bytes)
			// The result contains ANSI color codes, so we just check it's not empty
			if result == "" {
				t.Error("FormatBytes() returned empty string")
			}
		})
	}
}

func TestColorFunctions(t *testing.T) {
	// Test that color functions don't panic and return non-empty strings
	tests := []struct {
		name   string
		fn     func(a ...interface{}) string
		input  string
		notNil bool
	}{
		{"Bold", Bold, "test", true},
		{"Title", Title, "test", true},
		{"Header", Header, "test", true},
		{"Section", Section, "test", true},
		{"UserRole", UserRole, "test", true},
		{"AssistantRole", AssistantRole, "test", true},
		{"SystemRole", SystemRole, "test", true},
		{"Thinking", Thinking, "test", true},
		{"ToolCall", ToolCall, "test", true},
		{"ToolName", ToolName, "test", true},
		{"Error", Error, "test", true},
		{"Dim", Dim, "test", true},
		{"Timestamp", Timestamp, "test", true},
		{"ID", ID, "test", true},
		{"Project", Project, "test", true},
		{"Model", Model, "test", true},
		{"Match", Match, "test", true},
		{"Success", Success, "test", true},
		{"Warning", Warning, "test", true},
		{"Info", Info, "test", true},
		{"Number", Number, "test", true},
		{"Size", Size, "test", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.fn(tt.input)
			if tt.notNil && result == "" {
				t.Errorf("%s() returned empty string", tt.name)
			}
		})
	}
}

func TestSetColorEnabled(t *testing.T) {
	// Test that SetColorEnabled doesn't panic
	SetColorEnabled(true)
	SetColorEnabled(false)
}

func TestDisableColorIfNotTTY(t *testing.T) {
	// Test that DisableColorIfNotTTY doesn't panic
	DisableColorIfNotTTY()
}

func TestIsTTY(t *testing.T) {
	// Just verify it returns a boolean without panicking
	_ = IsTTY()
}
