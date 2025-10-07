package extractor

import (
	"context"
	"io"
)

// Extractor defines the interface for text extraction from documents
type Extractor interface {
	// Extract extracts text from the provided document
	Extract(ctx context.Context, reader io.Reader, metadata *DocumentMetadata) (*ExtractionResult, error)

	// SupportedFormats returns the file formats this extractor supports
	SupportedFormats() []string

	// CanExtract checks if this extractor can handle the given format
	CanExtract(format string) bool
}

// Service provides text extraction functionality
type Service interface {
	// ExtractText extracts text from a document using the appropriate extractor
	ExtractText(ctx context.Context, reader io.Reader, metadata *DocumentMetadata) (*ExtractionResult, error)

	// GetExtractor returns the appropriate extractor for the given format
	GetExtractor(format string) (Extractor, error)

	// SupportedFormats returns all supported file formats
	SupportedFormats() []string
}

// DocumentMetadata contains information about the document being processed
type DocumentMetadata struct {
	FileName   string            `json:"file_name"`
	MimeType   string            `json:"mime_type"`
	Size       int64             `json:"size"`
	Format     string            `json:"format"`
	Properties map[string]string `json:"properties,omitempty"`
}

// ExtractionResult contains the result of text extraction
type ExtractionResult struct {
	Text      string                 `json:"text"`
	PageCount int                    `json:"page_count,omitempty"`
	WordCount int                    `json:"word_count"`
	CharCount int                    `json:"char_count"`
	Language  string                 `json:"language,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Success   bool                   `json:"success"`
	Error     string                 `json:"error,omitempty"`
	Duration  int64                  `json:"duration_ms"`
}

// ExtractionError represents errors that occur during text extraction
type ExtractionError struct {
	Format  string
	Message string
	Cause   error
}

func (e *ExtractionError) Error() string {
	if e.Cause != nil {
		return e.Message + ": " + e.Cause.Error()
	}
	return e.Message
}

func (e *ExtractionError) Unwrap() error {
	return e.Cause
}

// NewExtractionError creates a new extraction error
func NewExtractionError(format, message string, cause error) *ExtractionError {
	return &ExtractionError{
		Format:  format,
		Message: message,
		Cause:   cause,
	}
}
