package extractor

import (
	"fmt"
	"log"
	"runtime"
	"strings"
)

// PDFFallbackExtractorConfig configures fallback extraction behaviour.
type PDFFallbackExtractorConfig struct {
	MaxContentSize    int
	MaxExtractedChars int
	MaxBytesToScan    int
	MaxLinesToScan    int
	EnableDebugLog    bool
}

// DefaultPDFFallbackExtractorConfig returns default fallback configuration.
func DefaultPDFFallbackExtractorConfig() PDFFallbackExtractorConfig {
	return PDFFallbackExtractorConfig{
		MaxContentSize:    5 * 1024 * 1024,
		MaxExtractedChars: 500 * 1024,
		MaxBytesToScan:    2 * 1024 * 1024,
		MaxLinesToScan:    10000,
		EnableDebugLog:    true,
	}
}

// PDFFallbackExtractor orchestrates alternative extraction strategies.
type PDFFallbackExtractor struct {
	streamProcessor *PDFStreamProcessor
	textCleaner     *PDFTextCleaner
	validator       *PDFValidator
	config          PDFFallbackExtractorConfig
}

// NewPDFFallbackExtractor creates a new fallback extractor instance.
func NewPDFFallbackExtractor(streamProcessor *PDFStreamProcessor, textCleaner *PDFTextCleaner, validator *PDFValidator, config PDFFallbackExtractorConfig) *PDFFallbackExtractor {
	if streamProcessor == nil {
		streamProcessor = NewPDFStreamProcessor(textCleaner, DefaultStreamProcessorConfig())
	}
	if textCleaner == nil {
		textCleaner = NewPDFTextCleaner(DefaultPDFTextCleanerConfig())
	}
	if validator == nil {
		validator = NewPDFValidator()
	}

	return &PDFFallbackExtractor{
		streamProcessor: streamProcessor,
		textCleaner:     textCleaner,
		validator:       validator,
		config:          config,
	}
}

// Extract executes the configured fallback strategies.
func (f *PDFFallbackExtractor) Extract(content []byte) (string, int, string, error) {
	if len(content) > f.config.MaxContentSize {
		return "", 0, "", fmt.Errorf("content too large for fallback extraction")
	}

	if f.validator.IsLikelyGarbagePDF(content) {
		return "", 0, "", fmt.Errorf("early garbage detection failed")
	}

	if f.config.EnableDebugLog {
		log.Printf("[PDF-EXTRACT] Attempting fallback method 1: raw stream extraction")
	}

	text, pageCount := f.streamProcessor.ExtractRawTextStreams(content)
	if text != "" {
		return text, pageCount, "raw_stream_extraction", nil
	}

	if f.config.EnableDebugLog {
		log.Printf("[PDF-EXTRACT] Attempting fallback method 2: pattern extraction")
	}

	text = f.extractBasicTextPatterns(content)
	if text != "" {
		return text, 1, "pattern_extraction", nil
	}

	return "", 0, "", fmt.Errorf("all extraction methods failed")
}

func (f *PDFFallbackExtractor) extractBasicTextPatterns(content []byte) string {
	var text strings.Builder
	text.Grow(64 * 1024)

	scanLimit := len(content)
	if scanLimit > f.config.MaxBytesToScan {
		scanLimit = f.config.MaxBytesToScan
	}

	if f.config.EnableDebugLog {
		log.Printf("[PDF-EXTRACT] 🔍 Pattern extraction: scanning %d bytes for text patterns", scanLimit)
	}

	lineStart := 0
	linesProcessed := 0
	addedLines := 0

	for i := 0; i < scanLimit; i++ {
		if content[i] == '\n' || i == scanLimit-1 {
			lineEnd := i
			if i == scanLimit-1 && content[i] != '\n' {
				lineEnd = i + 1
			}

			if lineEnd > lineStart {
				lineBytes := content[lineStart:lineEnd]
				if f.validator.IsPotentialTextBytes(lineBytes) {
					lineStr := string(lineBytes)
					cleaned := f.textCleaner.CleanExtractedText(lineStr)

					if len(cleaned) > 2 {
						if addedLines < 5 {
							log.Printf("[PDF-EXTRACT] 📝 Pattern match line %d: %q", linesProcessed+1, truncateString(cleaned, 50))
						}

						if text.Len() > 0 {
							text.WriteString(" ")
						}
						text.WriteString(cleaned)
						addedLines++
					}
				}
			}

			lineStart = i + 1
			linesProcessed++

			if text.Len() > f.config.MaxExtractedChars {
				if f.config.EnableDebugLog {
					log.Printf("[PDF-EXTRACT] ⚠️ Reached size limit (%d chars) at line %d", text.Len(), linesProcessed)
				}
				break
			}

			if linesProcessed > f.config.MaxLinesToScan {
				if f.config.EnableDebugLog {
					log.Printf("[PDF-EXTRACT] ⚠️ Line limit reached (%d lines)", f.config.MaxLinesToScan)
				}
				break
			}
		}
	}

	result := text.String()
	text.Reset()
	runtime.GC()

	if f.config.EnableDebugLog {
		log.Printf("[PDF-EXTRACT] 📊 Pattern extraction result: processed %d lines, added %d lines, total %d chars", linesProcessed, addedLines, len(result))
	}

	return result
}

func truncateString(s string, limit int) string {
	if len(s) <= limit {
		return s
	}
	return s[:limit]
}
