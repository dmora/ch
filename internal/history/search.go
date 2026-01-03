package history

import (
	"bufio"
	"os"
	"strings"
	"sync"

	"github.com/dmora/ch/internal/jsonl"
)

// SearchResult represents a search match.
type SearchResult struct {
	Meta       *ConversationMeta
	MatchCount int      // Number of matches in this conversation
	Previews   []string // Preview snippets showing matches (first few)
}

// SearchOptions configures the search.
type SearchOptions struct {
	ProjectsDir   string // Base projects directory
	ProjectPath   string // Filter to specific project (empty = all)
	IncludeAgents bool   // Include agent conversations
	Limit         int    // Maximum number of results (0 = no limit)
	CaseSensitive bool   // Case-sensitive search
	Workers       int    // Number of parallel workers
}

// DefaultSearchOptions returns default search options.
func DefaultSearchOptions() SearchOptions {
	return SearchOptions{
		ProjectsDir: DefaultProjectsDir(),
		Workers:     4,
	}
}

// Search searches for a query across conversations.
func Search(query string, opts SearchOptions) ([]*SearchResult, error) {
	if opts.ProjectsDir == "" {
		opts.ProjectsDir = DefaultProjectsDir()
	}
	if opts.Workers <= 0 {
		opts.Workers = 4
	}

	// Prepare query for case-insensitive search
	searchQuery := query
	if !opts.CaseSensitive {
		searchQuery = strings.ToLower(query)
	}

	// Find all conversation files
	scanner := NewScanner(ScannerOptions{
		ProjectsDir:   opts.ProjectsDir,
		ProjectPath:   opts.ProjectPath,
		IncludeAgents: opts.IncludeAgents,
	})

	files, err := scanner.findFiles()
	if err != nil {
		return nil, err
	}

	// Search files in parallel
	var wg sync.WaitGroup
	var mu sync.Mutex
	var results []*SearchResult

	fileChan := make(chan string, len(files))
	for _, f := range files {
		fileChan <- f
	}
	close(fileChan)

	for i := 0; i < opts.Workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for path := range fileChan {
				result := searchFile(path, searchQuery, opts.CaseSensitive)
				if result != nil {
					mu.Lock()
					// Check limit
					if opts.Limit > 0 && len(results) >= opts.Limit {
						mu.Unlock()
						return
					}
					results = append(results, result)
					mu.Unlock()
				}
			}
		}()
	}

	wg.Wait()

	// Apply limit
	if opts.Limit > 0 && len(results) > opts.Limit {
		results = results[:opts.Limit]
	}

	return results, nil
}

// searchFile searches a single file for the query in message content.
func searchFile(path string, query string, caseSensitive bool) *SearchResult {
	file, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer file.Close()

	var matchCount int
	var previews []string
	const maxPreviews = 3
	const previewLen = 150

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), jsonl.MaxScannerBuffer)

	for scanner.Scan() {
		line := scanner.Bytes()

		// Parse entry to check if it's a message
		entry, err := jsonl.ParseEntry(line)
		if err != nil || !entry.Type.IsMessage() {
			continue
		}

		// Parse message and search in content
		msg, err := jsonl.ParseMessage(entry)
		if err != nil || msg == nil {
			continue
		}

		text := jsonl.ExtractText(msg)
		if text == "" {
			continue
		}

		// Search in message text
		searchText := text
		searchQuery := query
		if !caseSensitive {
			searchText = strings.ToLower(text)
			searchQuery = strings.ToLower(query)
		}

		if !strings.Contains(searchText, searchQuery) {
			continue
		}

		matchCount++

		// Extract preview if we need more
		if len(previews) < maxPreviews {
			preview := extractPreviewFromText(text, query, caseSensitive, previewLen)
			if preview != "" {
				previews = append(previews, preview)
			}
		}
	}

	if matchCount == 0 {
		return nil
	}

	// Get metadata
	meta, err := ScanConversationMeta(path)
	if err != nil {
		return nil
	}

	return &SearchResult{
		Meta:       meta,
		MatchCount: matchCount,
		Previews:   previews,
	}
}

// extractPreviewFromText extracts a preview snippet from text around the match.
func extractPreviewFromText(text, query string, caseSensitive bool, maxLen int) string {
	searchText := text
	searchQuery := query
	if !caseSensitive {
		searchText = strings.ToLower(text)
		searchQuery = strings.ToLower(query)
	}

	// Find match position
	idx := strings.Index(searchText, searchQuery)
	if idx < 0 {
		return ""
	}

	// Extract context around match
	start := idx - 50
	if start < 0 {
		start = 0
	}
	end := idx + len(query) + 50
	if end > len(text) {
		end = len(text)
	}

	preview := text[start:end]
	preview = strings.ReplaceAll(preview, "\n", " ")
	preview = strings.ReplaceAll(preview, "\t", " ")

	if start > 0 {
		preview = "..." + preview
	}
	if end < len(text) {
		preview = preview + "..."
	}

	if len(preview) > maxLen {
		preview = preview[:maxLen-3] + "..."
	}

	return preview
}

// extractSearchPreview extracts a preview snippet around the match.
func extractSearchPreview(line []byte, query string, caseSensitive bool, maxLen int) string {
	// Parse the entry to get message content
	entry, err := jsonl.ParseEntry(line)
	if err != nil || entry.Message == nil {
		return ""
	}

	msg, err := jsonl.ParseMessage(entry)
	if err != nil {
		return ""
	}

	text := jsonl.ExtractText(msg)
	if text == "" {
		return ""
	}

	searchText := text
	searchQuery := query
	if !caseSensitive {
		searchText = strings.ToLower(text)
		searchQuery = strings.ToLower(query)
	}

	// Find match position
	idx := strings.Index(searchText, searchQuery)
	if idx < 0 {
		return ""
	}

	// Extract context around match
	start := idx - 50
	if start < 0 {
		start = 0
	}
	end := idx + len(query) + 50
	if end > len(text) {
		end = len(text)
	}

	preview := text[start:end]
	preview = strings.ReplaceAll(preview, "\n", " ")
	preview = strings.ReplaceAll(preview, "\t", " ")

	if start > 0 {
		preview = "..." + preview
	}
	if end < len(text) {
		preview = preview + "..."
	}

	if len(preview) > maxLen {
		preview = preview[:maxLen-3] + "..."
	}

	return preview
}

// QuickSearch does a fast search that only checks if a file contains the query.
// It doesn't extract previews or count matches.
func QuickSearch(query string, opts SearchOptions) ([]*ConversationMeta, error) {
	if opts.ProjectsDir == "" {
		opts.ProjectsDir = DefaultProjectsDir()
	}
	if opts.Workers <= 0 {
		opts.Workers = 4
	}

	searchQuery := query
	if !opts.CaseSensitive {
		searchQuery = strings.ToLower(query)
	}

	scanner := NewScanner(ScannerOptions{
		ProjectsDir:   opts.ProjectsDir,
		ProjectPath:   opts.ProjectPath,
		IncludeAgents: opts.IncludeAgents,
	})

	files, err := scanner.findFiles()
	if err != nil {
		return nil, err
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	var results []*ConversationMeta

	fileChan := make(chan string, len(files))
	for _, f := range files {
		fileChan <- f
	}
	close(fileChan)

	for i := 0; i < opts.Workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for path := range fileChan {
				if quickSearchFile(path, searchQuery, opts.CaseSensitive) {
					meta, err := ScanConversationMeta(path)
					if err != nil {
						continue
					}
					mu.Lock()
					if opts.Limit > 0 && len(results) >= opts.Limit {
						mu.Unlock()
						return
					}
					results = append(results, meta)
					mu.Unlock()
				}
			}
		}()
	}

	wg.Wait()

	if opts.Limit > 0 && len(results) > opts.Limit {
		results = results[:opts.Limit]
	}

	return results, nil
}

// quickSearchFile checks if a file contains the query in message content.
func quickSearchFile(path string, query string, caseSensitive bool) bool {
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), jsonl.MaxScannerBuffer)

	for scanner.Scan() {
		line := scanner.Bytes()

		// Parse entry to check if it's a message
		entry, err := jsonl.ParseEntry(line)
		if err != nil || !entry.Type.IsMessage() {
			continue
		}

		// Parse message and search in content
		msg, err := jsonl.ParseMessage(entry)
		if err != nil || msg == nil {
			continue
		}

		text := jsonl.ExtractText(msg)
		if text == "" {
			continue
		}

		// Search in message text
		searchText := text
		searchQuery := query
		if !caseSensitive {
			searchText = strings.ToLower(text)
			searchQuery = strings.ToLower(query)
		}

		if strings.Contains(searchText, searchQuery) {
			return true
		}
	}
	return false
}
