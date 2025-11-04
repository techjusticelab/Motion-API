package extractor

import (
	"bytes"
	"context"
	"errors"
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

	// PPT files may actually be PPTX archives
	if bytes.HasPrefix(content, []byte("PK")) {
		pptxExtractor := NewPPTXExtractor()
		result, err := pptxExtractor.Extract(ctx, bytes.NewReader(content), metadata)
		if err == nil && result != nil && strings.TrimSpace(result.Text) != "" {
			if result.Metadata == nil {
				result.Metadata = map[string]interface{}{}
			}
			result.Metadata["original_format"] = "ppt"
			result.Metadata["detected_archive"] = true
			return result, nil
		}
	}

	text, conversionErr := convertWithLibreOffice(ctx, content, "ppt")
	if strings.TrimSpace(text) == "" {
		decoded := decodeUTF16LE(content)
		if strings.TrimSpace(decoded) == "" {
			if conversionErr != nil {
				return nil, NewExtractionError("ppt", "failed to convert PPT to text", conversionErr)
			}
			return nil, NewExtractionError("ppt", "no text extracted from PPT document", errors.New("empty text content"))
		}
		text = decoded
	}

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
			"format":          "ppt",
			"file_size":       len(content),
			"conversion_tool": "libreoffice",
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
