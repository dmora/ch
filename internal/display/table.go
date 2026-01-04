package display

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/dmora/ch/internal/history"
	"github.com/olekukonko/tablewriter"
)

// TableOptions configures table output.
type TableOptions struct {
	Writer      io.Writer
	ShowAgent   bool // Show agent indicator
	JSON        bool // Output as JSON
	ShowIndices bool // Show message indices in search results

	// Context for headers/footers
	ProjectPath    string // Current project path (empty if global)
	IsGlobal       bool   // Showing all projects
	ProjectCount   int    // Number of projects (for global view)
	TotalAgents    int    // Total agents across all shown conversations
	CurrentProject string // Current working directory's project (for marking)
	Query          string // Search query (for search results)
}

// DefaultTableOptions returns default table options.
func DefaultTableOptions() TableOptions {
	return TableOptions{
		Writer: os.Stdout,
	}
}

// ConversationTable renders a table of conversations.
type ConversationTable struct {
	opts TableOptions
}

// NewConversationTable creates a new conversation table.
func NewConversationTable(opts TableOptions) *ConversationTable {
	if opts.Writer == nil {
		opts.Writer = os.Stdout
	}
	return &ConversationTable{opts: opts}
}

// Render renders the conversations as a table.
func (t *ConversationTable) Render(conversations []*history.ConversationMeta) error {
	if t.opts.JSON {
		return t.renderJSON(conversations)
	}
	return t.renderTable(conversations)
}

func (t *ConversationTable) renderJSON(conversations []*history.ConversationMeta) error {
	type jsonConversation struct {
		ID         string `json:"id"`
		SessionID  string `json:"session_id,omitempty"`
		Project    string `json:"project"`
		Timestamp  string `json:"timestamp"`
		Preview    string `json:"preview"`
		Messages   int    `json:"messages"`
		IsAgent    bool   `json:"is_agent,omitempty"`
		AgentCount int    `json:"agent_count,omitempty"`
		Model      string `json:"model,omitempty"`
		FileSize   int64  `json:"file_size"`
		Path       string `json:"path"`
	}

	output := make([]jsonConversation, len(conversations))
	for i, c := range conversations {
		output[i] = jsonConversation{
			ID:         c.ID,
			SessionID:  c.SessionID,
			Project:    c.ProjectPath,
			Timestamp:  c.Timestamp.Format(time.RFC3339),
			Preview:    c.Preview,
			Messages:   c.MessageCount,
			IsAgent:    c.IsAgent,
			AgentCount: c.AgentCount,
			Model:      c.Model,
			FileSize:   c.FileSize,
			Path:       c.Path,
		}
	}

	encoder := json.NewEncoder(t.opts.Writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}

func (t *ConversationTable) renderTable(conversations []*history.ConversationMeta) error {
	if len(conversations) == 0 {
		fmt.Fprintln(t.opts.Writer, Dim("No conversations found"))
		return nil
	}

	// Context header
	t.renderContextHeader(len(conversations))

	table := tablewriter.NewWriter(t.opts.Writer)
	table.SetHeader([]string{"ID", "Time", "Messages", "Preview"})
	table.SetBorder(false)
	table.SetHeaderLine(false)
	table.SetColumnSeparator("")
	table.SetTablePadding("  ")
	table.SetNoWhiteSpace(true)
	table.SetAutoWrapText(false)

	for _, c := range conversations {
		id := history.ShortID(c.ID)
		if c.IsAgent {
			id = Dim("agent-") + id
		} else if c.AgentCount > 0 {
			id = id + Dim(fmt.Sprintf(" [+%d]", c.AgentCount))
		}

		timestamp := formatRelativeTime(c.Timestamp)
		messages := fmt.Sprintf("%d", c.MessageCount)
		preview := truncateString(c.Preview, 60)

		table.Append([]string{id, timestamp, messages, preview})
	}

	table.Render()

	// Footer hint
	t.renderFooterHint(conversations)

	return nil
}

// renderContextHeader prints context about what's being displayed.
func (t *ConversationTable) renderContextHeader(count int) {
	if t.opts.IsGlobal {
		if t.opts.ProjectCount > 0 {
			fmt.Fprintf(t.opts.Writer, "%s\n\n", Dim(fmt.Sprintf("Showing %d conversations across %d projects", count, t.opts.ProjectCount)))
		} else {
			fmt.Fprintf(t.opts.Writer, "%s\n\n", Dim(fmt.Sprintf("Showing %d conversations (all projects)", count)))
		}
	} else if t.opts.ProjectPath != "" {
		fmt.Fprintf(t.opts.Writer, "%s %s\n\n", Dim("Project:"), Project(t.opts.ProjectPath))
	}
}

// renderFooterHint prints helpful hints about the output.
func (t *ConversationTable) renderFooterHint(conversations []*history.ConversationMeta) {
	// Check if any conversations have agents
	hasAgents := false
	for _, c := range conversations {
		if c.AgentCount > 0 {
			hasAgents = true
			break
		}
	}

	if hasAgents {
		fmt.Fprintf(t.opts.Writer, "\n%s\n", Dim("[+N] = spawned agents. Use 'ch agents <id>' to list, 'ch resume <id>' to continue."))
	}
}

// formatRelativeTime formats a time as a relative string (e.g., "2h ago").
func formatRelativeTime(t time.Time) string {
	now := time.Now()
	diff := now.Sub(t)

	switch {
	case diff < time.Minute:
		return Dim("just now")
	case diff < time.Hour:
		mins := int(diff.Minutes())
		return Dim(fmt.Sprintf("%dm ago", mins))
	case diff < 24*time.Hour:
		hours := int(diff.Hours())
		return Dim(fmt.Sprintf("%dh ago", hours))
	case diff < 7*24*time.Hour:
		days := int(diff.Hours() / 24)
		return Dim(fmt.Sprintf("%dd ago", days))
	default:
		return Dim(t.Format("Jan 2"))
	}
}

// truncateString truncates a string to maxLen characters.
func truncateString(s string, maxLen int) string {
	// Remove newlines
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\t", " ")

	// Collapse multiple spaces
	for strings.Contains(s, "  ") {
		s = strings.ReplaceAll(s, "  ", " ")
	}

	s = strings.TrimSpace(s)

	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// ProjectTable renders a table of projects.
type ProjectTable struct {
	opts TableOptions
}

// NewProjectTable creates a new project table.
func NewProjectTable(opts TableOptions) *ProjectTable {
	if opts.Writer == nil {
		opts.Writer = os.Stdout
	}
	return &ProjectTable{opts: opts}
}

// Render renders the projects as a table.
func (t *ProjectTable) Render(projects []*history.Project) error {
	if t.opts.JSON {
		return t.renderJSON(projects)
	}
	return t.renderTable(projects)
}

func (t *ProjectTable) renderJSON(projects []*history.Project) error {
	type jsonProject struct {
		Name          string `json:"name"`
		Path          string `json:"path"`
		Conversations int    `json:"conversations"`
		Agents        int    `json:"agents"`
		TotalSize     int64  `json:"total_size"`
	}

	output := make([]jsonProject, len(projects))
	for i, p := range projects {
		output[i] = jsonProject{
			Name:          p.Name,
			Path:          p.Path,
			Conversations: p.ConversationCount,
			Agents:        p.AgentCount,
			TotalSize:     p.TotalSize,
		}
	}

	encoder := json.NewEncoder(t.opts.Writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}

func (t *ProjectTable) renderTable(projects []*history.Project) error {
	if len(projects) == 0 {
		fmt.Fprintln(t.opts.Writer, Dim("No projects found"))
		return nil
	}

	table := tablewriter.NewWriter(t.opts.Writer)
	table.SetHeader([]string{"Project", "Conversations", "Agents", "Size"})
	table.SetBorder(false)
	table.SetHeaderLine(false)
	table.SetColumnSeparator("")
	table.SetTablePadding("  ")
	table.SetNoWhiteSpace(true)
	table.SetAutoWrapText(false)

	// Calculate totals
	var totalConvs, totalAgents int
	var totalSize int64
	for _, p := range projects {
		totalConvs += p.ConversationCount
		totalAgents += p.AgentCount
		totalSize += p.TotalSize
	}

	for _, p := range projects {
		path := truncateString(p.Path, 50)
		// Mark current project
		if t.opts.CurrentProject != "" && p.Path == t.opts.CurrentProject {
			path = path + " " + Match("*")
		}
		convs := fmt.Sprintf("%d", p.ConversationCount)
		agents := fmt.Sprintf("%d", p.AgentCount)
		size := FormatBytes(p.TotalSize)

		table.Append([]string{path, convs, agents, size})
	}

	table.Render()

	// Totals footer
	fmt.Fprintf(t.opts.Writer, "\n%s  %s  %s  %s\n",
		Dim("TOTAL"),
		Number(fmt.Sprintf("%d", totalConvs)),
		Number(fmt.Sprintf("%d", totalAgents)),
		FormatBytes(totalSize),
	)

	// Legend for current project marker
	if t.opts.CurrentProject != "" {
		fmt.Fprintf(t.opts.Writer, "\n%s\n", Dim("* = current project"))
	}

	return nil
}

// SearchResultTable renders search results.
type SearchResultTable struct {
	opts TableOptions
}

// NewSearchResultTable creates a new search result table.
func NewSearchResultTable(opts TableOptions) *SearchResultTable {
	if opts.Writer == nil {
		opts.Writer = os.Stdout
	}
	return &SearchResultTable{opts: opts}
}

// Render renders search results.
func (t *SearchResultTable) Render(results []*history.SearchResult) error {
	if t.opts.JSON {
		return t.renderJSON(results)
	}
	return t.renderTable(results)
}

func (t *SearchResultTable) renderJSON(results []*history.SearchResult) error {
	type jsonResult struct {
		ID             string   `json:"id"`
		Project        string   `json:"project"`
		MatchCount     int      `json:"match_count"`
		MessageIndices []int    `json:"message_indices,omitempty"`
		Previews       []string `json:"previews"`
		Path           string   `json:"path"`
	}

	output := make([]jsonResult, len(results))
	for i, r := range results {
		output[i] = jsonResult{
			ID:             r.Meta.ID,
			Project:        r.Meta.ProjectPath,
			MatchCount:     r.MatchCount,
			MessageIndices: r.MessageIndices,
			Previews:       r.Previews,
			Path:           r.Meta.Path,
		}
	}

	encoder := json.NewEncoder(t.opts.Writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}

func (t *SearchResultTable) renderTable(results []*history.SearchResult) error {
	if len(results) == 0 {
		fmt.Fprintln(t.opts.Writer, Dim("No matches found"))
		return nil
	}

	// Count total matches
	totalMatches := 0
	for _, r := range results {
		totalMatches += r.MatchCount
	}

	// Summary header
	fmt.Fprintf(t.opts.Writer, "%s\n\n", Dim(fmt.Sprintf("Found %d matches in %d conversations", totalMatches, len(results))))

	for i, r := range results {
		if i > 0 {
			fmt.Fprintln(t.opts.Writer)
		}

		// Header
		id := history.ShortID(r.Meta.ID)
		if r.Meta.IsAgent {
			id = "agent-" + id
		}
		fmt.Fprintf(t.opts.Writer, "%s  %s  %s\n",
			ID(id),
			Dim(r.Meta.ProjectPath),
			Match(fmt.Sprintf("[%d matches]", r.MatchCount)),
		)

		// Show message indices if enabled
		if t.opts.ShowIndices && len(r.MessageIndices) > 0 {
			fmt.Fprintf(t.opts.Writer, "  %s %s\n",
				Dim("Messages:"),
				formatMessageIndices(r.MessageIndices))
		}

		// Previews
		for _, preview := range r.Previews {
			fmt.Fprintf(t.opts.Writer, "  %s\n", preview)
		}
	}

	return nil
}

// formatMessageIndices formats a list of message indices for display.
func formatMessageIndices(indices []int) string {
	if len(indices) == 0 {
		return ""
	}
	if len(indices) <= 5 {
		parts := make([]string, len(indices))
		for i, idx := range indices {
			parts[i] = fmt.Sprintf("%d", idx)
		}
		return "[" + strings.Join(parts, ", ") + "]"
	}
	// Truncate for many matches
	parts := make([]string, 5)
	for i := 0; i < 5; i++ {
		parts[i] = fmt.Sprintf("%d", indices[i])
	}
	return fmt.Sprintf("[%s, ... +%d more]", strings.Join(parts, ", "), len(indices)-5)
}
