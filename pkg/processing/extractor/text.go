package extractor

import (
	"context"
	"io"
	"strings"
	"unicode/utf8"
)

// textExtractor handles plain text files
type textExtractor struct{}

// NewTextExtractor creates a new text extractor
func NewTextExtractor() Extractor {
	return &textExtractor{}
}

// Extract extracts text from plain text files
func (e *textExtractor) Extract(ctx context.Context, reader io.Reader, metadata *DocumentMetadata) (*ExtractionResult, error) {
	// Read all content
	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, NewExtractionError("txt", "failed to read text file", err)
	}

	// Convert to string and validate UTF-8
	text := string(content)
	if !utf8.ValidString(text) {
		// Try to fix invalid UTF-8
		text = strings.ToValidUTF8(text, "�")
	}

	// Clean up the text using enhanced cleaner
	cleaner := NewTextCleaner(DefaultCleaningConfig())
	text = cleaner.CleanText(text)

	// Count words and characters
	wordCount := countWords(text)
	charCount := len(text)

	return &ExtractionResult{
		Text:      text,
		WordCount: wordCount,
		CharCount: charCount,
		PageCount: 1, // Text files are considered single page
		Metadata: map[string]interface{}{
			"encoding": "utf-8",
			"format":   "plain_text",
		},
	}, nil
}

// SupportedFormats returns the formats this extractor supports
func (e *textExtractor) SupportedFormats() []string {
	return []string{"txt", "text", "log", "md", "markdown", "csv", "json", "xml", "html", "htm"}
}

// CanExtract checks if this extractor can handle the given format
func (e *textExtractor) CanExtract(format string) bool {
	format = strings.ToLower(format)
	for _, supported := range e.SupportedFormats() {
		if format == supported {
			return true
		}
	}
	return false
}

// cleanText performs basic text cleaning
func cleanText(text string) string {
	// Normalize line endings
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")

	// Remove excessive whitespace while preserving structure
	lines := strings.Split(text, "\n")
	var cleanedLines []string

	for _, line := range lines {
		// Trim whitespace from each line
		line = strings.TrimSpace(line)
		cleanedLines = append(cleanedLines, line)
	}

	// Join lines back together
	text = strings.Join(cleanedLines, "\n")

	// Remove excessive blank lines (more than 2 consecutive)
	text = strings.ReplaceAll(text, "\n\n\n\n", "\n\n")
	text = strings.ReplaceAll(text, "\n\n\n", "\n\n")

	return strings.TrimSpace(text)
}

// countWords counts the number of words in the text
func countWords(text string) int {
	if text == "" {
		return 0
	}

	// Split by whitespace and count non-empty tokens
	words := strings.Fields(text)
	return len(words)
}
