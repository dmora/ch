package text

import "strings"

// Contains checks if text contains query with case sensitivity option.
func Contains(text, query string, caseSensitive bool) bool {
	if caseSensitive {
		return strings.Contains(text, query)
	}
	return strings.Contains(strings.ToLower(text), strings.ToLower(query))
}

// FindIndex finds the index of query in text with case sensitivity option.
// Returns -1 if not found.
func FindIndex(text, query string, caseSensitive bool) int {
	if caseSensitive {
		return strings.Index(text, query)
	}
	return strings.Index(strings.ToLower(text), strings.ToLower(query))
}

// ExtractPreview extracts a preview snippet around a match.
// Returns empty string if query not found.
func ExtractPreview(text, query string, caseSensitive bool, maxLen int) string {
	idx := FindIndex(text, query, caseSensitive)
	if idx < 0 {
		return ""
	}

	// Extract context around match (50 chars before/after)
	const contextChars = 50
	start := idx - contextChars
	if start < 0 {
		start = 0
	}
	end := idx + len(query) + contextChars
	if end > len(text) {
		end = len(text)
	}

	preview := text[start:end]
	preview = normalizeWhitespace(preview)

	// Add ellipsis for truncated content
	if start > 0 {
		preview = "..." + preview
	}
	if end < len(text) {
		preview = preview + "..."
	}

	// Cap to max length
	if len(preview) > maxLen {
		preview = preview[:maxLen-3] + "..."
	}

	return preview
}

// normalizeWhitespace replaces newlines and tabs with spaces.
func normalizeWhitespace(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\t", " ")
	return s
}

// NormalizeQuery normalizes a query for case-insensitive search.
func NormalizeQuery(query string, caseSensitive bool) string {
	if caseSensitive {
		return query
	}
	return strings.ToLower(query)
}
