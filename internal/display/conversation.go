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

// PaginationOptions controls message pagination for display.
type PaginationOptions struct {
	First      int // Show first N messages (0 = no limit)
	Last       int // Show last N messages (0 = no limit)
	RangeStart int // Start of range (1-based, 0 = not set)
	RangeEnd   int // End of range (1-based, 0 = not set)
	FitTokens  int // Auto-select messages to fit token budget (0 = disabled)
	AfterIndex int // Start after message N for cursor pagination (0 = start from beginning)
	Limit      int // Max messages to show with AfterIndex (0 = no limit)
}

// IsSet returns true if any pagination option is configured.
func (p PaginationOptions) IsSet() bool {
	return p.First > 0 || p.Last > 0 || p.RangeStart > 0 || p.FitTokens > 0 || p.AfterIndex > 0 || p.Limit > 0
}

// ConversationDisplayOptions configures conversation display.
type ConversationDisplayOptions struct {
	Writer        io.Writer
	ShowThinking  bool              // Include thinking blocks
	ShowTools     bool              // Include tool calls
	ShowNumbering bool              // Show message indices [N] prefix
	RoleFilter    string            // Filter by role: user, assistant, system (empty = all)
	JSON          bool              // Output as JSON
	Raw           bool              // Output raw JSONL
	AgentCount    int               // Number of agents spawned by this conversation
	Pagination    PaginationOptions // Pagination controls
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
		Index     int                    `json:"index,omitempty"` // 1-based message index
		Timestamp string                 `json:"timestamp,omitempty"`
		Role      string                 `json:"role,omitempty"`
		Model     string                 `json:"model,omitempty"`
		Text      string                 `json:"text,omitempty"`
		Thinking  string                 `json:"thinking,omitempty"`
		ToolCalls []jsonl.ToolCall       `json:"tool_calls,omitempty"`
		Raw       map[string]interface{} `json:"raw,omitempty"`
	}

	// Apply pagination filtering
	filteredMessages, hasGap := d.filterMessages(conv.Entries)

	// Build a map of filtered entries for quick lookup
	filteredSet := make(map[*jsonl.RawEntry]bool)
	for _, entry := range filteredMessages {
		filteredSet[entry] = true
	}

	var messages []jsonMessage
	msgIndex := 0

	for _, entry := range conv.Entries {
		if !entry.Type.IsMessage() {
			continue
		}
		msgIndex++

		// Skip if not in filtered set
		if d.opts.Pagination.IsSet() && !filteredSet[entry] {
			continue
		}

		jm := jsonMessage{
			Type:      string(entry.Type),
			Index:     msgIndex,
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

	// Count total messages
	totalMessages := 0
	for _, e := range conv.Entries {
		if e.Type.IsMessage() {
			totalMessages++
		}
	}

	output := struct {
		ID            string        `json:"id"`
		SessionID     string        `json:"session_id"`
		Project       string        `json:"project"`
		IsAgent       bool          `json:"is_agent"`
		TotalMessages int           `json:"total_messages"`
		ShownMessages int           `json:"shown_messages"`
		HasGap        bool          `json:"has_gap,omitempty"`
		Messages      []jsonMessage `json:"messages"`
	}{
		ID:            conv.Meta.ID,
		SessionID:     conv.Meta.SessionID,
		Project:       conv.Meta.ProjectPath,
		IsAgent:       conv.Meta.IsAgent,
		TotalMessages: totalMessages,
		ShownMessages: len(messages),
		HasGap:        hasGap,
		Messages:      messages,
	}

	encoder := json.NewEncoder(d.opts.Writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}

// filterMessages applies pagination options to filter entries.
// Only counts user/assistant/system entries as "messages".
// Returns (filtered messages, hasGap bool).
func (d *ConversationDisplay) filterMessages(entries []*jsonl.RawEntry) ([]*jsonl.RawEntry, bool) {
	// First, extract only message entries (with optional role filter)
	var messages []*jsonl.RawEntry
	for _, entry := range entries {
		if entry.Type.IsMessage() {
			// Apply role filter if specified
			if d.opts.RoleFilter != "" && string(entry.Type) != d.opts.RoleFilter {
				continue
			}
			messages = append(messages, entry)
		}
	}

	if !d.opts.Pagination.IsSet() {
		return messages, false // no gap indicator needed
	}

	totalMessages := len(messages)

	// Handle cursor pagination (--after-index, --limit)
	if d.opts.Pagination.AfterIndex > 0 || d.opts.Pagination.Limit > 0 {
		return d.applyCursorPagination(messages)
	}

	// Handle --fit-tokens
	if d.opts.Pagination.FitTokens > 0 {
		return d.fitToTokenBudget(messages, d.opts.Pagination.FitTokens)
	}

	// Handle --range X-Y
	if d.opts.Pagination.RangeStart > 0 {
		start := d.opts.Pagination.RangeStart - 1 // Convert to 0-based
		end := d.opts.Pagination.RangeEnd

		if start >= totalMessages {
			return nil, false
		}
		if end > totalMessages {
			end = totalMessages
		}

		return messages[start:end], false
	}

	// Handle --first and/or --last
	first := d.opts.Pagination.First
	last := d.opts.Pagination.Last

	// Only --first specified
	if first > 0 && last == 0 {
		if first >= totalMessages {
			return messages, false
		}
		return messages[:first], false
	}

	// Only --last specified
	if last > 0 && first == 0 {
		if last >= totalMessages {
			return messages, false
		}
		return messages[totalMessages-last:], false
	}

	// Both --first and --last specified
	if first > 0 && last > 0 {
		if first+last >= totalMessages {
			// No gap needed, show all
			return messages, false
		}

		// Return first N + last M (gap will be rendered between)
		result := make([]*jsonl.RawEntry, 0, first+last)
		result = append(result, messages[:first]...)
		result = append(result, messages[totalMessages-last:]...)
		return result, true // has gap
	}

	return messages, false
}

// applyCursorPagination implements --after-index N --limit M cursor-based iteration.
func (d *ConversationDisplay) applyCursorPagination(messages []*jsonl.RawEntry) ([]*jsonl.RawEntry, bool) {
	afterIdx := d.opts.Pagination.AfterIndex
	limit := d.opts.Pagination.Limit
	total := len(messages)

	// Start position: after message N means start at index N (0-based)
	startPos := afterIdx
	if startPos >= total {
		return nil, false
	}

	endPos := total
	if limit > 0 && startPos+limit < total {
		endPos = startPos + limit
	}

	hasGapBefore := startPos > 0
	return messages[startPos:endPos], hasGapBefore
}

// estimateTokens estimates the token count for a message.
// Uses ~4 characters per token as a rough heuristic.
func estimateTokens(entry *jsonl.RawEntry) int {
	msg, err := jsonl.ParseMessage(entry)
	if err != nil || msg == nil {
		return 0
	}
	text := jsonl.ExtractText(msg)
	return (len(text) + 3) / 4 // Ceiling division for ~4 chars/token
}

// fitToTokenBudget selects messages from the end to fit within token budget.
func (d *ConversationDisplay) fitToTokenBudget(messages []*jsonl.RawEntry, budget int) ([]*jsonl.RawEntry, bool) {
	if len(messages) == 0 {
		return messages, false
	}

	totalTokens := 0
	startIdx := len(messages)

	// Work backwards from most recent
	for i := len(messages) - 1; i >= 0; i-- {
		tokens := estimateTokens(messages[i])
		if totalTokens+tokens > budget {
			break
		}
		totalTokens += tokens
		startIdx = i
	}

	if startIdx == 0 {
		return messages, false // Everything fits
	}

	return messages[startIdx:], true // hasGap = true, we omitted earlier messages
}

// renderGapIndicator renders a visual gap indicator between first and last messages.
func (d *ConversationDisplay) renderGapIndicator(totalMessages, firstCount, lastCount int) {
	omitted := totalMessages - firstCount - lastCount
	fmt.Fprintln(d.opts.Writer)
	fmt.Fprintf(d.opts.Writer, "%s\n", Dim(strings.Repeat("·", 40)))
	fmt.Fprintf(d.opts.Writer, "%s\n",
		Dim(fmt.Sprintf("    ... %d messages omitted ...", omitted)))
	fmt.Fprintf(d.opts.Writer, "%s\n", Dim(strings.Repeat("·", 40)))
}

// renderPaginationInfo shows pagination status in output.
func (d *ConversationDisplay) renderPaginationInfo(shown, total int) {
	fmt.Fprintln(d.opts.Writer)
	fmt.Fprintf(d.opts.Writer, "%s %s\n",
		Dim("Showing:"),
		Number(fmt.Sprintf("%d of %d messages", shown, total)))
}

// renderCursorInfo shows cursor pagination status with next page hint.
func (d *ConversationDisplay) renderCursorInfo(shown, total, afterIndex int) {
	fmt.Fprintln(d.opts.Writer)
	startIdx := afterIndex + 1
	endIdx := afterIndex + shown
	fmt.Fprintf(d.opts.Writer, "%s %s\n",
		Dim("Showing:"),
		Number(fmt.Sprintf("messages %d-%d of %d", startIdx, endIdx, total)))

	if endIdx < total {
		fmt.Fprintf(d.opts.Writer, "%s %s\n",
			Dim("Next page:"),
			ID(fmt.Sprintf("--after-index %d --limit %d", endIdx, shown)))
	}
}

// renderFitTokensInfo shows auto-selected pagination info.
func (d *ConversationDisplay) renderFitTokensInfo(shown, total, budget int) {
	fmt.Fprintln(d.opts.Writer)
	fmt.Fprintf(d.opts.Writer, "%s %s (token budget: ~%d)\n",
		Dim("Auto-selected:"),
		Number(fmt.Sprintf("last %d of %d messages", shown, total)),
		budget)
}

func (d *ConversationDisplay) renderFormatted(conv *history.Conversation) error {
	// Header
	d.renderHeader(conv)

	// Apply pagination filtering
	messages, hasGap := d.filterMessages(conv.Entries)

	// Build index map for all messages to get correct indices
	// Note: index is based on original position in conversation (before role filter)
	indexMap := make(map[*jsonl.RawEntry]int)
	totalMessages := 0
	for _, e := range conv.Entries {
		if e.Type.IsMessage() {
			totalMessages++
			indexMap[e] = totalMessages
		}
	}

	// Render messages with gap if needed
	if hasGap && d.opts.Pagination.First > 0 && d.opts.Pagination.Last > 0 {
		// First/last combined mode with gap
		firstCount := d.opts.Pagination.First
		lastCount := d.opts.Pagination.Last

		// Render first N messages
		for i := 0; i < firstCount && i < len(messages); i++ {
			d.renderEntry(messages[i], indexMap[messages[i]])
		}

		// Render gap indicator
		d.renderGapIndicator(totalMessages, firstCount, lastCount)

		// Render last M messages
		for i := firstCount; i < len(messages); i++ {
			d.renderEntry(messages[i], indexMap[messages[i]])
		}
	} else if hasGap {
		// Other pagination modes with gap (cursor, fit-tokens) - show simple gap at start
		omitted := totalMessages - len(messages)
		fmt.Fprintln(d.opts.Writer)
		fmt.Fprintf(d.opts.Writer, "%s\n", Dim(fmt.Sprintf("    ... %d earlier messages omitted ...", omitted)))
		fmt.Fprintln(d.opts.Writer)

		// Render all messages
		for _, entry := range messages {
			d.renderEntry(entry, indexMap[entry])
		}
	} else {
		// No gap, render all filtered messages
		for _, entry := range messages {
			d.renderEntry(entry, indexMap[entry])
		}
	}

	// Show appropriate pagination info
	if d.opts.Pagination.AfterIndex > 0 || d.opts.Pagination.Limit > 0 {
		d.renderCursorInfo(len(messages), totalMessages, d.opts.Pagination.AfterIndex)
	} else if d.opts.Pagination.FitTokens > 0 {
		d.renderFitTokensInfo(len(messages), totalMessages, d.opts.Pagination.FitTokens)
	} else if d.opts.Pagination.IsSet() {
		d.renderPaginationInfo(len(messages), totalMessages)
	}

	// Footer with navigation hints
	d.renderFooter(conv)

	return nil
}

func (d *ConversationDisplay) renderFooter(conv *history.Conversation) {
	// Don't show footer for agents
	if conv.Meta.IsAgent {
		return
	}

	fmt.Fprintln(d.opts.Writer)
	fmt.Fprintln(d.opts.Writer, strings.Repeat("─", 60))

	shortID := history.ShortID(conv.Meta.ID)

	if d.opts.AgentCount > 0 {
		fmt.Fprintf(d.opts.Writer, "%s Run: %s\n",
			Dim(fmt.Sprintf("%d agent(s) spawned.", d.opts.AgentCount)),
			ID(fmt.Sprintf("ch agents %s", shortID)),
		)
	}

	fmt.Fprintf(d.opts.Writer, "%s %s\n", Dim("Resume:"), ID(fmt.Sprintf("ch resume %s", shortID)))
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

func (d *ConversationDisplay) renderEntry(entry *jsonl.RawEntry, index int) {
	msg, err := jsonl.ParseMessage(entry)
	if err != nil || msg == nil {
		return
	}

	// Check if there's any visible content
	hasContent := false
	for _, block := range msg.Content {
		switch block.Type {
		case jsonl.BlockTypeText:
			if block.Text != "" {
				hasContent = true
			}
		case jsonl.BlockTypeThinking:
			if d.opts.ShowThinking && block.Thinking != "" {
				hasContent = true
			}
		case jsonl.BlockTypeToolUse, jsonl.BlockTypeToolResult:
			if d.opts.ShowTools {
				hasContent = true
			}
		}
		if hasContent {
			break
		}
	}

	// Skip entries with no visible content
	if !hasContent {
		return
	}

	fmt.Fprintln(d.opts.Writer)

	// Message index prefix (if enabled)
	if d.opts.ShowNumbering && index > 0 {
		fmt.Fprintf(d.opts.Writer, "%s ", Number(fmt.Sprintf("[%d]", index)))
	}

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
func RenderAgentList(w io.Writer, agents []*history.ConversationMeta, parentID string, asJSON bool, filter string) error {
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
		if filter != "" {
			fmt.Fprintf(w, "%s\n", Dim(fmt.Sprintf("No agents of type '%s' found for this conversation", filter)))
		} else {
			fmt.Fprintln(w, Dim("No agents found for this conversation"))
		}
		return nil
	}

	fmt.Fprintf(w, "\n%s %s\n", Title("Agents for conversation"), ID(history.ShortID(parentID)))
	if filter != "" {
		fmt.Fprintf(w, "%s\n\n", Dim(fmt.Sprintf("Found %d agent(s) of type '%s'", len(agents), filter)))
	} else {
		fmt.Fprintf(w, "%s\n\n", Dim(fmt.Sprintf("Found %d agent(s)", len(agents))))
	}

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
