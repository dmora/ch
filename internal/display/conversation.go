package display

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/dmora/ch/internal/history"
	"github.com/dmora/ch/internal/jsonl"
)

// ConversationDisplayOptions configures conversation display.
type ConversationDisplayOptions struct {
	Writer       io.Writer
	ShowThinking bool   // Include thinking blocks
	ShowTools    bool   // Include tool calls
	JSON         bool   // Output as JSON
	Raw          bool   // Output raw JSONL
}

// DefaultConversationDisplayOptions returns default display options.
func DefaultConversationDisplayOptions() ConversationDisplayOptions {
	return ConversationDisplayOptions{
		Writer: os.Stdout,
	}
}

// ConversationDisplay renders a full conversation.
type ConversationDisplay struct {
	opts ConversationDisplayOptions
}

// NewConversationDisplay creates a new conversation display.
func NewConversationDisplay(opts ConversationDisplayOptions) *ConversationDisplay {
	if opts.Writer == nil {
		opts.Writer = os.Stdout
	}
	return &ConversationDisplay{opts: opts}
}

// Render renders the conversation.
func (d *ConversationDisplay) Render(conv *history.Conversation) error {
	if d.opts.Raw {
		return d.renderRaw(conv)
	}
	if d.opts.JSON {
		return d.renderJSON(conv)
	}
	return d.renderFormatted(conv)
}

func (d *ConversationDisplay) renderRaw(conv *history.Conversation) error {
	parser, err := jsonl.NewParser(conv.Meta.Path)
	if err != nil {
		return err
	}
	defer parser.Close()

	for {
		line, err := parser.NextRaw()
		if err != nil {
			return err
		}
		if line == nil {
			break
		}
		fmt.Fprintln(d.opts.Writer, string(line))
	}
	return nil
}

func (d *ConversationDisplay) renderJSON(conv *history.Conversation) error {
	type jsonMessage struct {
		Type      string                 `json:"type"`
		Timestamp string                 `json:"timestamp,omitempty"`
		Role      string                 `json:"role,omitempty"`
		Model     string                 `json:"model,omitempty"`
		Text      string                 `json:"text,omitempty"`
		Thinking  string                 `json:"thinking,omitempty"`
		ToolCalls []jsonl.ToolCall       `json:"tool_calls,omitempty"`
		Raw       map[string]interface{} `json:"raw,omitempty"`
	}

	var messages []jsonMessage

	for _, entry := range conv.Entries {
		if !entry.Type.IsMessage() {
			continue
		}

		jm := jsonMessage{
			Type:      string(entry.Type),
			Timestamp: entry.Timestamp,
		}

		if entry.Message != nil {
			msg, _ := jsonl.ParseMessage(entry)
			if msg != nil {
				jm.Role = msg.Role
				jm.Model = msg.Model
				jm.Text = jsonl.ExtractText(msg)
				if d.opts.ShowThinking {
					jm.Thinking = jsonl.ExtractThinking(msg)
				}
				if d.opts.ShowTools {
					jm.ToolCalls = jsonl.ExtractToolCallDetails(msg)
				}
			}
		}

		messages = append(messages, jm)
	}

	output := struct {
		ID        string        `json:"id"`
		SessionID string        `json:"session_id"`
		Project   string        `json:"project"`
		IsAgent   bool          `json:"is_agent"`
		Messages  []jsonMessage `json:"messages"`
	}{
		ID:        conv.Meta.ID,
		SessionID: conv.Meta.SessionID,
		Project:   conv.Meta.ProjectPath,
		IsAgent:   conv.Meta.IsAgent,
		Messages:  messages,
	}

	encoder := json.NewEncoder(d.opts.Writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}

func (d *ConversationDisplay) renderFormatted(conv *history.Conversation) error {
	// Header
	d.renderHeader(conv)

	// Messages
	for _, entry := range conv.Entries {
		if !entry.Type.IsMessage() {
			continue
		}
		d.renderEntry(entry)
	}

	return nil
}

func (d *ConversationDisplay) renderHeader(conv *history.Conversation) {
	fmt.Fprintln(d.opts.Writer)

	// Title
	id := conv.Meta.ID
	if conv.Meta.IsAgent {
		fmt.Fprintf(d.opts.Writer, "%s %s\n", Title("Agent Conversation"), ID(id))
		if conv.Meta.ParentSessionID != "" {
			fmt.Fprintf(d.opts.Writer, "%s %s\n", Dim("Parent:"), ID(history.ShortID(conv.Meta.ParentSessionID)))
		}
	} else {
		fmt.Fprintf(d.opts.Writer, "%s %s\n", Title("Conversation"), ID(id))
	}

	// Metadata
	fmt.Fprintf(d.opts.Writer, "%s %s\n", Dim("Project:"), Project(conv.Meta.ProjectPath))
	fmt.Fprintf(d.opts.Writer, "%s %s\n", Dim("Time:"), Timestamp(conv.Meta.Timestamp.Format(time.RFC3339)))
	fmt.Fprintf(d.opts.Writer, "%s %s\n", Dim("Messages:"), Number(fmt.Sprintf("%d", conv.Meta.MessageCount)))
	if conv.Meta.Model != "" {
		fmt.Fprintf(d.opts.Writer, "%s %s\n", Dim("Model:"), Model(conv.Meta.Model))
	}

	fmt.Fprintln(d.opts.Writer)
	fmt.Fprintln(d.opts.Writer, strings.Repeat("─", 60))
}

func (d *ConversationDisplay) renderEntry(entry *jsonl.RawEntry) {
	msg, err := jsonl.ParseMessage(entry)
	if err != nil || msg == nil {
		return
	}

	fmt.Fprintln(d.opts.Writer)

	// Role header
	switch entry.Type {
	case jsonl.EntryTypeUser:
		fmt.Fprintf(d.opts.Writer, "%s", UserRole("User"))
	case jsonl.EntryTypeAssistant:
		fmt.Fprintf(d.opts.Writer, "%s", AssistantRole("Assistant"))
	case jsonl.EntryTypeSystem:
		fmt.Fprintf(d.opts.Writer, "%s", SystemRole("System"))
	}

	if entry.Timestamp != "" {
		t, _ := time.Parse(time.RFC3339, entry.Timestamp)
		if !t.IsZero() {
			fmt.Fprintf(d.opts.Writer, "  %s", Timestamp(t.Format("15:04:05")))
		}
	}
	fmt.Fprintln(d.opts.Writer)

	// Content blocks
	for _, block := range msg.Content {
		d.renderBlock(&block)
	}
}

func (d *ConversationDisplay) renderBlock(block *jsonl.ContentBlock) {
	switch block.Type {
	case jsonl.BlockTypeText:
		if block.Text != "" {
			fmt.Fprintln(d.opts.Writer, block.Text)
		}

	case jsonl.BlockTypeThinking:
		if d.opts.ShowThinking && block.Thinking != "" {
			fmt.Fprintf(d.opts.Writer, "\n%s\n", Section("Thinking:"))
			// Indent thinking content
			lines := strings.Split(block.Thinking, "\n")
			for _, line := range lines {
				fmt.Fprintln(d.opts.Writer, Thinking("  "+line))
			}
		}

	case jsonl.BlockTypeToolUse:
		if d.opts.ShowTools {
			fmt.Fprintf(d.opts.Writer, "\n%s %s\n", ToolCall("Tool:"), ToolName(block.Name))
			if block.Input != nil {
				var input map[string]interface{}
				if json.Unmarshal(block.Input, &input) == nil {
					// Show abbreviated input
					for k, v := range input {
						val := fmt.Sprintf("%v", v)
						if len(val) > 100 {
							val = val[:100] + "..."
						}
						fmt.Fprintf(d.opts.Writer, "  %s: %s\n", Dim(k), val)
					}
				}
			}
		}

	case jsonl.BlockTypeToolResult:
		if d.opts.ShowTools {
			status := Success("OK")
			if block.IsError {
				status = Error("ERROR")
			}
			fmt.Fprintf(d.opts.Writer, "%s %s\n", ToolCall("Result:"), status)
			if block.Content != nil {
				var content string
				if json.Unmarshal(block.Content, &content) == nil {
					if len(content) > 500 {
						content = content[:500] + "..."
					}
					fmt.Fprintln(d.opts.Writer, Dim(content))
				}
			}
		}
	}
}

// RenderAgentList renders a list of agents for a conversation.
func RenderAgentList(w io.Writer, agents []*history.ConversationMeta, parentID string, asJSON bool) error {
	if asJSON {
		type jsonAgent struct {
			ID        string `json:"id"`
			Timestamp string `json:"timestamp"`
			Messages  int    `json:"messages"`
			Preview   string `json:"preview"`
		}

		output := make([]jsonAgent, len(agents))
		for i, a := range agents {
			output[i] = jsonAgent{
				ID:        a.ID,
				Timestamp: a.Timestamp.Format(time.RFC3339),
				Messages:  a.MessageCount,
				Preview:   a.Preview,
			}
		}

		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")
		return encoder.Encode(output)
	}

	if len(agents) == 0 {
		fmt.Fprintln(w, Dim("No agents found for this conversation"))
		return nil
	}

	fmt.Fprintf(w, "\n%s %s\n", Title("Agents for conversation"), ID(history.ShortID(parentID)))
	fmt.Fprintf(w, "%s\n\n", Dim(fmt.Sprintf("Found %d agent(s)", len(agents))))

	for i, a := range agents {
		fmt.Fprintf(w, "%d. %s  %s  %s\n",
			i+1,
			ID("agent-"+a.ID),
			Timestamp(a.Timestamp.Format("15:04:05")),
			Dim(fmt.Sprintf("(%d messages)", a.MessageCount)),
		)
		if a.Preview != "" {
			preview := truncateString(a.Preview, 70)
			fmt.Fprintf(w, "   %s\n", preview)
		}
	}

	return nil
}

// RenderStats renders usage statistics.
func RenderStats(w io.Writer, stats *Stats, asJSON bool) error {
	if asJSON {
		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")
		return encoder.Encode(stats)
	}

	fmt.Fprintln(w)
	fmt.Fprintln(w, Title("Claude Code Usage Statistics"))
	fmt.Fprintln(w)

	fmt.Fprintf(w, "  %s %s\n", Dim("Projects:"), Number(fmt.Sprintf("%d", stats.ProjectCount)))
	fmt.Fprintf(w, "  %s %s\n", Dim("Conversations:"), Number(fmt.Sprintf("%d", stats.ConversationCount)))
	fmt.Fprintf(w, "  %s %s\n", Dim("Agents:"), Number(fmt.Sprintf("%d", stats.AgentCount)))
	fmt.Fprintf(w, "  %s %s\n", Dim("Total Messages:"), Number(fmt.Sprintf("%d", stats.TotalMessages)))
	fmt.Fprintf(w, "  %s %s\n", Dim("Total Size:"), FormatBytes(stats.TotalSize))

	if stats.OldestConversation != "" {
		fmt.Fprintf(w, "  %s %s\n", Dim("Oldest:"), Timestamp(stats.OldestConversation))
	}
	if stats.NewestConversation != "" {
		fmt.Fprintf(w, "  %s %s\n", Dim("Newest:"), Timestamp(stats.NewestConversation))
	}

	fmt.Fprintln(w)
	return nil
}

// Stats represents usage statistics.
type Stats struct {
	ProjectCount       int    `json:"project_count"`
	ConversationCount  int    `json:"conversation_count"`
	AgentCount         int    `json:"agent_count"`
	TotalMessages      int    `json:"total_messages"`
	TotalSize          int64  `json:"total_size"`
	OldestConversation string `json:"oldest_conversation,omitempty"`
	NewestConversation string `json:"newest_conversation,omitempty"`
}
