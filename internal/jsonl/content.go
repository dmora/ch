package jsonl

import (
	"encoding/json"
	"strings"
)

// ExtractText extracts all text content from a message.
func ExtractText(msg *Message) string {
	if msg == nil {
		return ""
	}

	var texts []string
	for _, block := range msg.Content {
		if block.Type == BlockTypeText && block.Text != "" {
			texts = append(texts, block.Text)
		}
	}
	return strings.Join(texts, "\n")
}

// ExtractThinking extracts all thinking content from a message.
func ExtractThinking(msg *Message) string {
	if msg == nil {
		return ""
	}

	var thoughts []string
	for _, block := range msg.Content {
		if block.Type == BlockTypeThinking && block.Thinking != "" {
			thoughts = append(thoughts, block.Thinking)
		}
	}
	return strings.Join(thoughts, "\n")
}

// ExtractToolCalls extracts all tool call names from a message.
func ExtractToolCalls(msg *Message) []string {
	if msg == nil {
		return nil
	}

	var tools []string
	for _, block := range msg.Content {
		if block.Type == BlockTypeToolUse && block.Name != "" {
			tools = append(tools, block.Name)
		}
	}
	return tools
}

// ExtractPreview extracts a preview string from raw message JSON.
// It limits the result to maxLen characters.
func ExtractPreview(rawMessage json.RawMessage, maxLen int) string {
	if rawMessage == nil {
		return ""
	}

	var msg Message
	if err := json.Unmarshal(rawMessage, &msg); err != nil {
		return ""
	}

	text := ExtractText(&msg)
	if len(text) == 0 {
		return ""
	}

	// Trim whitespace and normalize
	text = strings.TrimSpace(text)
	text = strings.ReplaceAll(text, "\n", " ")
	text = strings.ReplaceAll(text, "\t", " ")

	// Collapse multiple spaces
	for strings.Contains(text, "  ") {
		text = strings.ReplaceAll(text, "  ", " ")
	}

	if len(text) > maxLen {
		text = text[:maxLen-3] + "..."
	}
	return text
}

// HasToolCalls returns true if the message contains tool calls.
func HasToolCalls(msg *Message) bool {
	if msg == nil {
		return false
	}
	for _, block := range msg.Content {
		if block.Type == BlockTypeToolUse {
			return true
		}
	}
	return false
}

// HasThinking returns true if the message contains thinking blocks.
func HasThinking(msg *Message) bool {
	if msg == nil {
		return false
	}
	for _, block := range msg.Content {
		if block.Type == BlockTypeThinking {
			return true
		}
	}
	return false
}

// ToolCall represents a parsed tool call.
type ToolCall struct {
	ID    string
	Name  string
	Input map[string]interface{}
}

// ExtractToolCallDetails extracts detailed tool call information from a message.
func ExtractToolCallDetails(msg *Message) []ToolCall {
	if msg == nil {
		return nil
	}

	var calls []ToolCall
	for _, block := range msg.Content {
		if block.Type == BlockTypeToolUse {
			call := ToolCall{
				ID:   block.ID,
				Name: block.Name,
			}
			if block.Input != nil {
				json.Unmarshal(block.Input, &call.Input)
			}
			calls = append(calls, call)
		}
	}
	return calls
}

// ToolResult represents a tool result.
type ToolResult struct {
	ToolUseID string
	Content   string
	IsError   bool
}

// ExtractToolResults extracts tool results from a message.
func ExtractToolResults(msg *Message) []ToolResult {
	if msg == nil {
		return nil
	}

	var results []ToolResult
	for _, block := range msg.Content {
		if block.Type == BlockTypeToolResult {
			result := ToolResult{
				ToolUseID: block.ToolUseID,
				IsError:   block.IsError,
			}
			// Content can be a string or array
			if block.Content != nil {
				var str string
				if err := json.Unmarshal(block.Content, &str); err == nil {
					result.Content = str
				} else {
					// Try as array of content blocks
					var blocks []ContentBlock
					if err := json.Unmarshal(block.Content, &blocks); err == nil {
						var texts []string
						for _, b := range blocks {
							if b.Text != "" {
								texts = append(texts, b.Text)
							}
						}
						result.Content = strings.Join(texts, "\n")
					}
				}
			}
			results = append(results, result)
		}
	}
	return results
}
