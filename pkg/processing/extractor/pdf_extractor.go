package extractor

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"runtime"
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
	log.Printf("[PDF-EXTRACT] Starting extraction - File: %s", metadata.FileName)

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
		// Force garbage collection after extraction to release PDF memory
		runtime.GC()
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

	log.Printf("[PDF-EXTRACT] 📄 Processing PDF: %s, size: %d bytes", metadata.FileName, len(content))

	if len(content) < 4 {
		log.Printf("[PDF-EXTRACT] ❌ PDF too small: %d bytes", len(content))
		return nil, NewExtractionError("pdf", "file too small to be a valid PDF", nil)
	}

	header := string(content[:4])
	log.Printf("[PDF-EXTRACT] 🔍 PDF header check: %q", header)
	if header != "%PDF" {
		headerFound := false
		searchLimit := 1024
		if len(content) < searchLimit {
			searchLimit = len(content)
		}

		log.Printf("[PDF-EXTRACT] 🔍 Searching for PDF header in first %d bytes", searchLimit)
		for i := 0; i <= searchLimit-4; i++ {
			if string(content[i:i+4]) == "%PDF" {
				headerFound = true
				content = content[i:]
				log.Printf("[PDF-EXTRACT] ✅ Found PDF header at position %d", i)
				break
			}
		}

		if !headerFound {
			log.Printf("[PDF-EXTRACT] ❌ No valid PDF header found")
			return nil, NewExtractionError("pdf", "invalid PDF file format", nil)
		}
	}

	log.Printf("[PDF-EXTRACT] Attempting primary extraction method")
	text, pageCount, err := e.extractWithPrimaryMethod(ctx, content, metadata)
	if err == nil && text != "" {
		log.Printf("[PDF-EXTRACT] Primary method successful: %d chars, %d pages", len(text), pageCount)
		text = e.textCleaner.Clean(text)
		log.Printf("[PDF-EXTRACT] After cleaning: %d chars", len(text))
		runtime.GC()

		if e.validator.IsGarbageText(text) {
			log.Printf("[PDF-EXTRACT] Primary extraction produced garbage text, retrying with fallback")
			err = fmt.Errorf("garbage text detected")
		} else {
			return &ExtractionResult{
				Text:      text,
				WordCount: CountWords(text),
				CharCount: len(text),
				PageCount: pageCount,
				Language:  e.detectLanguage(text),
				Metadata: map[string]interface{}{
					"format":      "pdf",
					"file_size":   contentSize,
					"extraction":  "primary",
					"pdf_version": e.extractPDFVersion(content),
				},
			}, nil
		}
	}

	if err != nil {
		log.Printf("[PDF-EXTRACT] Primary method failed: %v", err)
	}

	log.Printf("[PDF-EXTRACT] Attempting fallback extraction methods")
	fallbackText, fallbackPageCount, extractionMethod, fallbackErr := e.extractWithFallbackMethods(content)
	if fallbackErr != nil || fallbackText == "" {
		log.Printf("[PDF-EXTRACT] ❌ Fallback methods failed: %v", fallbackErr)
		return nil, NewExtractionError("pdf", "pdf extraction failed", err)
	}

	fallbackText = e.textCleaner.Clean(fallbackText)
	if e.validator.IsGarbageText(fallbackText) {
		log.Printf("[PDF-EXTRACT] Fallback extraction produced garbage text")
		return nil, NewExtractionError("pdf", "pdf extraction produced garbage text", err)
	}

	runtime.GC()

	result = &ExtractionResult{
		Text:      fallbackText,
		WordCount: CountWords(fallbackText),
		CharCount: len(fallbackText),
		PageCount: fallbackPageCount,
		Language:  e.detectLanguage(fallbackText),
		Metadata: map[string]interface{}{
			"format":      "pdf",
			"file_size":   contentSize,
			"extraction":  extractionMethod,
			"pdf_version": e.extractPDFVersion(content),
		},
	}

	log.Printf("[PDF-EXTRACT] 🔍 Fallback ExtractionResult: Text field length=%d", len(result.Text))
	return result, nil
}

func (e *pdfExtractor) extractWithPrimaryMethod(ctx context.Context, content []byte, metadata *DocumentMetadata) (string, int, error) {
	contentReader := bytes.NewReader(content)
	log.Printf("[PDF-EXTRACT] 🔓 Opening PDF with ledongthuc/pdf library")
	pdfReader, err := pdf.NewReader(contentReader, int64(len(content)))
	if err != nil {
		log.Printf("[PDF-EXTRACT] ❌ Failed to open PDF with ledongthuc/pdf: %v", err)
		return "", 0, err
	}

	log.Printf("[PDF-EXTRACT] ✅ PDF opened successfully, extracting text from pages")
	return e.pageProcessor.ExtractAllText(ctx, pdfReader, metadata)
}

func (e *pdfExtractor) extractWithFallbackMethods(content []byte) (string, int, string, error) {
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

// CountWords counts whitespace-separated words in the provided text.
func CountWords(text string) int {
	if text == "" {
		return 0
	}

	return len(strings.Fields(text))
}
