package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/dmora/ch/internal/display"
	"github.com/dmora/ch/internal/history"
	"github.com/dmora/ch/internal/jsonl"
	"github.com/spf13/cobra"
)

var statsCmd = &cobra.Command{
	Use:     "stats",
	Short:   "Show usage statistics",
	Long:    `Show aggregate usage statistics across all Claude Code projects.`,
	Aliases: []string{"stat", "st"},
	RunE:    runStats,
}

var (
	statsJSON   bool
	statsTokens string
)

func init() {
	statsCmd.Flags().BoolVar(&statsJSON, "json", false, "Output as JSON")
	statsCmd.Flags().StringVar(&statsTokens, "tokens", "", "Estimate token count for a conversation ID")
}

func runStats(cmd *cobra.Command, args []string) error {
	// Handle --tokens flag
	if statsTokens != "" {
		return runTokenEstimate(statsTokens)
	}

	projects, err := history.ListProjects(cfg.ProjectsDir)
	if err != nil {
		return err
	}

	stats := &display.Stats{
		ProjectCount: len(projects),
	}

	var oldest, newest time.Time

	for _, p := range projects {
		stats.ConversationCount += p.ConversationCount
		stats.AgentCount += p.AgentCount
		stats.TotalSize += p.TotalSize
	}

	// Scan for message counts and timestamps
	scanner := history.NewScanner(history.ScannerOptions{
		ProjectsDir:   cfg.ProjectsDir,
		IncludeAgents: true,
	})

	conversations, err := scanner.ScanAll()
	if err == nil {
		for _, c := range conversations {
			stats.TotalMessages += c.MessageCount
			if oldest.IsZero() || c.Timestamp.Before(oldest) {
				oldest = c.Timestamp
			}
			if newest.IsZero() || c.Timestamp.After(newest) {
				newest = c.Timestamp
			}
		}
	}

	if !oldest.IsZero() {
		stats.OldestConversation = oldest.Format("2006-01-02 15:04")
	}
	if !newest.IsZero() {
		stats.NewestConversation = newest.Format("2006-01-02 15:04")
	}

	return display.RenderStats(os.Stdout, stats, statsJSON)
}

// runTokenEstimate estimates token count for a conversation.
// Uses heuristic: ~4 characters per token (industry standard approximation).
func runTokenEstimate(id string) error {
	path, err := findConversationFile(id)
	if err != nil {
		return err
	}

	conv, err := history.LoadConversation(path)
	if err != nil {
		return fmt.Errorf("loading conversation: %w", err)
	}

	// Count characters in all message content
	var totalChars int
	var messageCount int

	for _, entry := range conv.Entries {
		if !entry.Type.IsMessage() {
			continue
		}
		messageCount++

		msg, err := jsonl.ParseMessage(entry)
		if err != nil || msg == nil {
			continue
		}

		// Count text content
		text := jsonl.ExtractText(msg)
		totalChars += len(text)

		// Count thinking content
		thinking := jsonl.ExtractThinking(msg)
		totalChars += len(thinking)

		// Count tool call inputs/outputs (rough estimate)
		for _, block := range msg.Content {
			if block.Type == jsonl.BlockTypeToolUse && block.Input != nil {
				totalChars += len(block.Input)
			}
			if block.Type == jsonl.BlockTypeToolResult && block.Content != nil {
				totalChars += len(block.Content)
			}
		}
	}

	// Token estimation: ~4 chars per token
	estimatedTokens := totalChars / 4

	if statsJSON {
		output := struct {
			ID              string `json:"id"`
			Messages        int    `json:"messages"`
			TotalCharacters int    `json:"total_characters"`
			EstimatedTokens int    `json:"estimated_tokens"`
			FileSize        int64  `json:"file_size"`
		}{
			ID:              conv.Meta.ID,
			Messages:        messageCount,
			TotalCharacters: totalChars,
			EstimatedTokens: estimatedTokens,
			FileSize:        conv.Meta.FileSize,
		}
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(output)
	}

	fmt.Println()
	fmt.Printf("%s %s\n", display.Title("Token Estimate"), display.ID(conv.Meta.ID))
	fmt.Printf("%s %s\n", display.Dim("Messages:"), display.Number(fmt.Sprintf("%d", messageCount)))
	fmt.Printf("%s %s\n", display.Dim("Characters:"), display.Number(fmt.Sprintf("%d", totalChars)))
	fmt.Printf("%s %s\n", display.Dim("Est. Tokens:"), display.Number(fmt.Sprintf("~%d", estimatedTokens)))
	fmt.Printf("%s %s\n", display.Dim("File Size:"), display.FormatBytes(conv.Meta.FileSize))
	fmt.Println()
	fmt.Println(display.Dim("Note: Token estimate uses ~4 chars/token heuristic"))

	return nil
}
