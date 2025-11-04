package extractor

import (
	"context"
	"fmt"
	"io"
	"log"
	"path/filepath"
	"strings"
	"time"
)

// service implements the Service interface
type service struct {
	extractors map[string]Extractor
}

// NewService creates a new text extraction service
func NewService() Service {
	s := &service{
		extractors: make(map[string]Extractor),
	}

	// Register default extractors
	s.registerDefaultExtractors()

	return s
}

// registerDefaultExtractors registers all built-in extractors
func (s *service) registerDefaultExtractors() {
	// Register text extractor
	textExtractor := NewTextExtractor()
	for _, format := range textExtractor.SupportedFormats() {
		s.extractors[format] = textExtractor
	}

	// Register PDF extractor (original ledongthuc/pdf)
	pdfExtractor := NewPDFExtractor()
	for _, format := range pdfExtractor.SupportedFormats() {
		s.extractors[format] = pdfExtractor
	}

	// Register DOCX extractor
	docxExtractor := NewDOCXExtractor()
	for _, format := range docxExtractor.SupportedFormats() {
		s.extractors[format] = docxExtractor
	}

	// Register DOC/RTF extractor
	docExtractor := NewDOCExtractor()
	for _, format := range docExtractor.SupportedFormats() {
		s.extractors[format] = docExtractor
	}

	// Register PPT extractor
	pptExtractor := NewPPTExtractor()
	for _, format := range pptExtractor.SupportedFormats() {
		s.extractors[format] = pptExtractor
	}

	// Register PPTX extractor
	pptxExtractor := NewPPTXExtractor()
	for _, format := range pptxExtractor.SupportedFormats() {
		s.extractors[format] = pptxExtractor
	}
}

// ExtractText extracts text from a document using the appropriate extractor
func (s *service) ExtractText(ctx context.Context, reader io.Reader, metadata *DocumentMetadata) (*ExtractionResult, error) {
	startTime := time.Now()

	// Check if context is already cancelled
	select {
	case <-ctx.Done():
		return &ExtractionResult{
			Success:  false,
			Error:    "context cancelled before extraction",
			Duration: time.Since(startTime).Milliseconds(),
		}, ctx.Err()
	default:
	}

	// Determine format if not provided
	if metadata.Format == "" {
		metadata.Format = s.detectFormat(metadata.FileName, metadata.MimeType)
	}

	// Get appropriate extractor
	extractor, err := s.GetExtractor(metadata.Format)
	if err != nil {
		return &ExtractionResult{
			Success:  false,
			Error:    err.Error(),
			Duration: time.Since(startTime).Milliseconds(),
		}, err
	}

	// Extract text
	result, err := extractor.Extract(ctx, reader, metadata)
	if err != nil {
		log.Printf("[EXTRACTOR-SERVICE] ❌ Extraction failed for %s: %v", metadata.Format, err)
		return &ExtractionResult{
			Success:  false,
			Error:    err.Error(),
			Duration: time.Since(startTime).Milliseconds(),
		}, err
	}

	log.Printf("[EXTRACTOR-SERVICE] 📊 Extraction result for %s: %d chars, %d words, %d pages",
		metadata.Format, len(result.Text), result.WordCount, result.PageCount)

	// Set duration
	result.Duration = time.Since(startTime).Milliseconds()
	result.Success = true

	return result, nil
}

// GetExtractor returns the appropriate extractor for the given format
func (s *service) GetExtractor(format string) (Extractor, error) {
	format = strings.ToLower(format)

	extractor, exists := s.extractors[format]
	if !exists {
		return nil, NewExtractionError(format, fmt.Sprintf("no extractor available for format: %s", format), nil)
	}

	return extractor, nil
}

// SupportedFormats returns all supported file formats
func (s *service) SupportedFormats() []string {
	formats := make([]string, 0, len(s.extractors))
	seen := make(map[string]bool)

	for format := range s.extractors {
		if !seen[format] {
			formats = append(formats, format)
			seen[format] = true
		}
	}

	return formats
}

// detectFormat attempts to detect the file format from filename and mime type
func (s *service) detectFormat(fileName, mimeType string) string {
	// Try to detect from file extension first
	if fileName != "" {
		ext := strings.ToLower(filepath.Ext(fileName))
		if ext != "" {
			ext = strings.TrimPrefix(ext, ".")
			if _, exists := s.extractors[ext]; exists {
				return ext
			}
		}
	}

	// Try to detect from MIME type
	if mimeType != "" {
		switch mimeType {
		case "application/pdf":
			return "pdf"
		case "application/vnd.openxmlformats-officedocument.wordprocessingml.document":
			return "docx"
		case "application/msword":
			return "doc"
		case "text/plain":
			return "txt"
		case "application/rtf":
			return "rtf"
		}
	}

	// Default fallback
	return "txt"
}

// RegisterExtractor allows registering custom extractors
func (s *service) RegisterExtractor(format string, extractor Extractor) {
	s.extractors[strings.ToLower(format)] = extractor
}
