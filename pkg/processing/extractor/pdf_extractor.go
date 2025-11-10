package extractor

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"

	"github.com/ledongthuc/pdf"
)

// pdfExtractor orchestrates PDF extraction using modular components.
type pdfExtractor struct {
	streamProcessor *PDFStreamProcessor
	textCleaner     *PDFTextCleaner
	validator       *PDFValidator
	pageProcessor   *PDFPageProcessor
	fallback        *PDFFallbackExtractor
}

// NewPDFExtractor creates a new PDF extractor instance.
func NewPDFExtractor() Extractor {
	textCleaner := NewPDFTextCleaner(DefaultPDFTextCleanerConfig())
	streamProcessor := NewPDFStreamProcessor(textCleaner, DefaultStreamProcessorConfig())
	validator := NewPDFValidator()
	pageProcessor := NewPDFPageProcessor(textCleaner, DefaultPDFPageProcessorConfig())
	fallback := NewPDFFallbackExtractor(streamProcessor, textCleaner, validator, DefaultPDFFallbackExtractorConfig())

	return &pdfExtractor{
		streamProcessor: streamProcessor,
		textCleaner:     textCleaner,
		validator:       validator,
		pageProcessor:   pageProcessor,
		fallback:        fallback,
	}
}

// SupportedFormats returns the formats this extractor supports.
func (e *pdfExtractor) SupportedFormats() []string {
	return []string{"pdf"}
}

// CanExtract checks if this extractor can handle the given format.
func (e *pdfExtractor) CanExtract(format string) bool {
	return strings.ToLower(format) == "pdf"
}

// Extract extracts text from PDF files with fallback mechanisms.
func (e *pdfExtractor) Extract(ctx context.Context, reader io.Reader, metadata *DocumentMetadata) (result *ExtractionResult, err error) {
	if metadata == nil {
		metadata = &DocumentMetadata{}
	}

	if metadata.Properties == nil {
		metadata.Properties = make(map[string]string)
	}

	log.Printf("[PDF-EXTRACT] Starting extraction - File: %s", metadata.FileName)
	debugLog := e.shouldLogDebug(metadata)

	defer func() {
		if r := recover(); r != nil {
			log.Printf("[PDF-EXTRACT] Panic recovered: %v", r)
			err = NewExtractionError("pdf", fmt.Sprintf("PDF processing panic: %v", r), nil)
			result = &ExtractionResult{
				Text:      "",
				WordCount: 0,
				CharCount: 0,
				PageCount: 0,
				Language:  "unknown",
				Metadata: map[string]interface{}{
					"format":     "pdf",
					"extraction": "failed_panic_recovery",
					"error":      fmt.Sprintf("panic: %v", r),
				},
			}
		}
	}()

	select {
	case <-ctx.Done():
		return nil, NewExtractionError("pdf", "context cancelled before reading PDF", ctx.Err())
	default:
	}

	// Read PDF content - this will be freed after extraction
	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, NewExtractionError("pdf", "failed to read PDF file", err)
	}

	// Store content size before processing
	contentSize := len(content)

	select {
	case <-ctx.Done():
		return nil, NewExtractionError("pdf", "context cancelled after reading PDF", ctx.Err())
	default:
	}

	if debugLog {
		log.Printf("[PDF-EXTRACT] 📄 Processing PDF: %s, size: %d bytes", metadata.FileName, len(content))
	}

	if len(content) < 4 {
		log.Printf("[PDF-EXTRACT] ❌ PDF too small: %d bytes", len(content))
		return nil, NewExtractionError("pdf", "file too small to be a valid PDF", nil)
	}

	header := string(content[:4])
	if debugLog {
		log.Printf("[PDF-EXTRACT] 🔍 PDF header check: %q", header)
	}
	if header != "%PDF" {
		headerFound := false
		searchLimit := 1024
		if len(content) < searchLimit {
			searchLimit = len(content)
		}

		if debugLog {
			log.Printf("[PDF-EXTRACT] 🔍 Searching for PDF header in first %d bytes", searchLimit)
		}
		for i := 0; i <= searchLimit-4; i++ {
			if string(content[i:i+4]) == "%PDF" {
				headerFound = true
				content = content[i:]
				if debugLog {
					log.Printf("[PDF-EXTRACT] ✅ Found PDF header at position %d", i)
				}
				break
			}
		}

		if !headerFound {
			log.Printf("[PDF-EXTRACT] ❌ No valid PDF header found")
			return nil, NewExtractionError("pdf", "invalid PDF file format", nil)
		}
	}

	if debugLog {
		log.Printf("[PDF-EXTRACT] Attempting primary extraction method")
	}
	text, pageCount, limited, err := e.extractWithPrimaryMethod(ctx, content, metadata)
	if err == nil && text != "" {
		if debugLog {
			log.Printf("[PDF-EXTRACT] Primary method successful: %d chars, %d pages", len(text), pageCount)
		}
		text = e.textCleaner.Clean(text)
		if debugLog {
			log.Printf("[PDF-EXTRACT] After cleaning: %d chars", len(text))
		}

		if e.validator.IsGarbageText(text) {
			log.Printf("[PDF-EXTRACT] Primary extraction produced garbage text, retrying with fallback")
			err = fmt.Errorf("garbage text detected")
		} else {
			metadataMap := e.buildResultMetadata(contentSize, "primary", e.extractPDFVersion(content), limited, metadata)
			return &ExtractionResult{
				Text:      text,
				WordCount: CountWords(text),
				CharCount: len(text),
				PageCount: pageCount,
				Language:  e.detectLanguage(text),
				Metadata:  metadataMap,
				Success:   true,
			}, nil
		}
	}

	if err != nil {
		log.Printf("[PDF-EXTRACT] Primary method failed: %v", err)
	}

	if debugLog {
		log.Printf("[PDF-EXTRACT] Attempting fallback extraction methods")
	}
	fallbackText, fallbackPageCount, extractionMethod, fallbackNeedsDecompression, fallbackErr := e.extractWithFallbackMethods(content)
	if fallbackErr != nil || fallbackText == "" {
		log.Printf("[PDF-EXTRACT] ❌ Fallback methods failed: %v", fallbackErr)
		errMessage := "pdf extraction failed"
		if fallbackNeedsDecompression {
			errMessage = errMessage + ": compressed streams detected"
		}
		return nil, NewExtractionError("pdf", errMessage, err)
	}

	fallbackText = e.textCleaner.Clean(fallbackText)
	if e.validator.IsGarbageText(fallbackText) {
		log.Printf("[PDF-EXTRACT] Fallback extraction produced garbage text")
		return nil, NewExtractionError("pdf", "pdf extraction produced garbage text", err)
	}

	extractionLimited := false
	if metadata != nil && metadata.Properties != nil {
		extractionLimited = strings.EqualFold(metadata.Properties["pdf_extraction_limited"], "true")
	}

	if metadata != nil && metadata.Properties != nil {
		metadata.Properties["pdf_fallback_requires_decompression"] = strconv.FormatBool(fallbackNeedsDecompression)
	}

	metadataMap := e.buildResultMetadata(contentSize, extractionMethod, e.extractPDFVersion(content), extractionLimited, metadata)

	result = &ExtractionResult{
		Text:      fallbackText,
		WordCount: CountWords(fallbackText),
		CharCount: len(fallbackText),
		PageCount: fallbackPageCount,
		Language:  e.detectLanguage(fallbackText),
		Metadata:  metadataMap,
		Success:   true,
	}

	if debugLog {
		log.Printf("[PDF-EXTRACT] 🔍 Fallback ExtractionResult: Text field length=%d", len(result.Text))
	}
	return result, nil
}

func (e *pdfExtractor) extractWithPrimaryMethod(ctx context.Context, content []byte, metadata *DocumentMetadata) (string, int, bool, error) {
	contentReader := bytes.NewReader(content)
	debugLog := e.shouldLogDebug(metadata)
	if debugLog {
		log.Printf("[PDF-EXTRACT] 🔓 Opening PDF with ledongthuc/pdf library")
	}
	pdfReader, err := pdf.NewReader(contentReader, int64(len(content)))
	if err != nil {
		log.Printf("[PDF-EXTRACT] ❌ Failed to open PDF with ledongthuc/pdf: %v", err)
		return "", 0, false, err
	}

	if debugLog {
		log.Printf("[PDF-EXTRACT] ✅ PDF opened successfully, extracting text from pages")
	}
	return e.pageProcessor.ExtractAllText(ctx, pdfReader, metadata)
}

func (e *pdfExtractor) extractWithFallbackMethods(content []byte) (string, int, string, bool, error) {
	return e.fallback.Extract(content)
}

func (e *pdfExtractor) extractPDFVersion(content []byte) string {
	if len(content) < 8 {
		return "unknown"
	}

	header := string(content[:8])
	if strings.HasPrefix(header, "%PDF-") {
		return header[5:]
	}

	return "unknown"
}

func (e *pdfExtractor) detectLanguage(text string) string {
	englishWords := []string{
		"the", "and", "of", "to", "a", "in", "for", "is", "on", "that",
		"by", "this", "with", "from", "they", "we", "say", "her", "she",
		"or", "an", "will", "my", "one", "all", "would", "there", "their",
	}

	words := strings.Fields(strings.ToLower(text))
	if len(words) == 0 {
		return "unknown"
	}

	englishCount := 0
	totalWords := len(words)
	maxWords := 100

	if totalWords > maxWords {
		words = words[:maxWords]
		totalWords = maxWords
	}

	for _, word := range words {
		for _, englishWord := range englishWords {
			if word == englishWord {
				englishCount++
				break
			}
		}
	}

	if float64(englishCount)/float64(totalWords) > 0.2 {
		return "en"
	}

	return "unknown"
}

func (e *pdfExtractor) buildResultMetadata(contentSize int, extractionMethod, pdfVersion string, extractionLimited bool, metadata *DocumentMetadata) map[string]interface{} {
	result := map[string]interface{}{
		"format":             "pdf",
		"file_size":          contentSize,
		"extraction":         extractionMethod,
		"pdf_version":        pdfVersion,
		"extraction_limited": extractionLimited,
	}

	if metadata != nil && metadata.Properties != nil {
		if value := metadata.Properties["pdf_pages_processed"]; value != "" {
			if processed, err := strconv.Atoi(value); err == nil {
				result["pages_processed"] = processed
			}
		}
		if value := metadata.Properties["pdf_total_pages"]; value != "" {
			if total, err := strconv.Atoi(value); err == nil {
				result["total_pages"] = total
			}
		}
		if value := metadata.Properties["pdf_page_limit"]; value != "" {
			if limit, err := strconv.Atoi(value); err == nil {
				result["page_limit"] = limit
			}
		}
		if value := metadata.Properties["pdf_char_limit_hit"]; value != "" {
			result["char_limit_hit"] = strings.EqualFold(value, "true")
		}
		if value := metadata.Properties["pdf_char_limit"]; value != "" {
			if limit, err := strconv.Atoi(value); err == nil {
				result["char_limit"] = limit
			}
		}
		if value := metadata.Properties["pdf_fallback_requires_decompression"]; value != "" {
			result["fallback_requires_decompression"] = strings.EqualFold(value, "true")
		}
	}

	return result
}

func (e *pdfExtractor) shouldLogDebug(metadata *DocumentMetadata) bool {
	if metadata != nil && metadata.Properties != nil {
		if value, exists := metadata.Properties["pdf_debug"]; exists {
			return strings.EqualFold(value, "true")
		}
	}
	return false
}

// CountWords counts whitespace-separated words in the provided text.
func CountWords(text string) int {
	if text == "" {
		return 0
	}

	return len(strings.Fields(text))
}
