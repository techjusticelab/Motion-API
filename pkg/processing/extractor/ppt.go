package extractor

import (
	"bytes"
	"context"
	"io"
	"strings"
)

// pptExtractor handles legacy PowerPoint (PPT) files by decoding text streams
type pptExtractor struct{}

// NewPPTExtractor creates a new PPT extractor instance
func NewPPTExtractor() Extractor {
	return &pptExtractor{}
}

// Extract extracts textual content from PPT files
func (e *pptExtractor) Extract(ctx context.Context, reader io.Reader, metadata *DocumentMetadata) (*ExtractionResult, error) {
	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, NewExtractionError("ppt", "failed to read PPT file", err)
	}

	// PPT files may be mislabeled PPTX archives
	if bytes.HasPrefix(content, []byte("PK")) {
		pptxExtractor := NewPPTXExtractor()
		return pptxExtractor.Extract(ctx, bytes.NewReader(content), metadata)
	}

	text := decodeUTF16LE(content)
	cleaner := NewTextCleaner(DefaultCleaningConfig())
	text = cleaner.CleanText(text)

	wordCount := countWords(text)
	charCount := len(text)

	return &ExtractionResult{
		Text:      text,
		WordCount: wordCount,
		CharCount: charCount,
		PageCount: 0,
		Success:   true,
		Metadata: map[string]interface{}{
			"format":    "ppt",
			"file_size": len(content),
		},
	}, nil
}

// SupportedFormats returns supported formats for PPT extractor
func (e *pptExtractor) SupportedFormats() []string {
	return []string{"ppt"}
}

// CanExtract checks if the format is supported
func (e *pptExtractor) CanExtract(format string) bool {
	format = strings.ToLower(format)
	for _, supported := range e.SupportedFormats() {
		if format == supported {
			return true
		}
	}
	return false
}
