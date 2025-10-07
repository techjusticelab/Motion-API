package extractor

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTextExtractor_Extract(t *testing.T) {
	extractor := NewTextExtractor()
	ctx := context.Background()

	tests := []struct {
		name          string
		content       string
		expectedText  string
		expectedWords int
	}{
		{
			name:          "simple text",
			content:       "Hello world! This is a test.",
			expectedText:  "Hello world! This is a test.",
			expectedWords: 6,
		},
		{
			name:          "text with line breaks",
			content:       "Line 1\nLine 2\n\nLine 3",
			expectedText:  "Line 1\nLine 2\n\nLine 3",
			expectedWords: 6,
		},
		{
			name:          "text with extra whitespace",
			content:       "   Spaced   text   with   extra   whitespace   ",
			expectedText:  "Spaced   text   with   extra   whitespace",
			expectedWords: 5,
		},
		{
			name:          "empty content",
			content:       "",
			expectedText:  "",
			expectedWords: 0,
		},
		{
			name:          "only whitespace",
			content:       "   \n\n\t   ",
			expectedText:  "",
			expectedWords: 0,
		},
		{
			name:          "mixed line endings",
			content:       "Windows line\r\nMac line\rUnix line\n",
			expectedText:  "Windows line\nMac line\nUnix line",
			expectedWords: 6,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.content)
			metadata := &DocumentMetadata{
				FileName: "test.txt",
				Format:   "txt",
				Size:     int64(len(tt.content)),
			}

			result, err := extractor.Extract(ctx, reader, metadata)

			require.NoError(t, err)
			assert.Equal(t, tt.expectedText, result.Text)
			assert.Equal(t, tt.expectedWords, result.WordCount)
			assert.Equal(t, len(tt.expectedText), result.CharCount)
			assert.Equal(t, 1, result.PageCount)
			assert.Contains(t, result.Metadata, "encoding")
			assert.Contains(t, result.Metadata, "format")
		})
	}
}

func TestTextExtractor_SupportedFormats(t *testing.T) {
	extractor := NewTextExtractor()
	formats := extractor.SupportedFormats()

	expectedFormats := []string{"txt", "text", "log", "md", "markdown", "csv", "json", "xml", "html", "htm"}

	for _, expected := range expectedFormats {
		assert.Contains(t, formats, expected)
	}
}

func TestTextExtractor_CanExtract(t *testing.T) {
	extractor := NewTextExtractor()

	tests := []struct {
		format   string
		expected bool
	}{
		{"txt", true},
		{"TXT", true},
		{"md", true},
		{"json", true},
		{"pdf", false},
		{"docx", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			result := extractor.CanExtract(tt.format)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCleanText(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "normalize line endings",
			input:    "Line 1\r\nLine 2\rLine 3\n",
			expected: "Line 1\nLine 2\nLine 3",
		},
		{
			name:     "remove excessive blank lines",
			input:    "Line 1\n\n\n\nLine 2\n\n\nLine 3",
			expected: "Line 1\n\nLine 2\n\nLine 3",
		},
		{
			name:     "trim whitespace",
			input:    "   Line with spaces   \n  Another line  ",
			expected: "Line with spaces\nAnother line",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "only whitespace",
			input:    "   \n\n\t   ",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cleanText(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCountWords(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{"simple sentence", "Hello world test", 3},
		{"empty string", "", 0},
		{"only whitespace", "   \n\t  ", 0},
		{"punctuation", "Hello, world! How are you?", 5},
		{"multiple spaces", "Word1    Word2     Word3", 3},
		{"newlines", "Line1\nLine2\nLine3", 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := countWords(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
