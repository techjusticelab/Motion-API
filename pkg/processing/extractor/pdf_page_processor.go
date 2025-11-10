package extractor

import (
	"context"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/ledongthuc/pdf"
)

// PDFPageProcessorConfig defines configuration for page-level extraction.
type PDFPageProcessorConfig struct {
	ChunkSize        int
	ChunkCharLimit   int
	TotalCharLimit   int
	DefaultPageLimit int
	EnableDebugLog   bool
}

// DefaultPDFPageProcessorConfig provides default settings for the page processor.
func DefaultPDFPageProcessorConfig() PDFPageProcessorConfig {
	return PDFPageProcessorConfig{
		ChunkSize:        5,
		ChunkCharLimit:   500 * 1024,
		TotalCharLimit:   2 * 1024 * 1024,
		DefaultPageLimit: 0,
		EnableDebugLog:   false,
	}
}

// PDFPageProcessor handles page-level text extraction.
type PDFPageProcessor struct {
	textCleaner *PDFTextCleaner
	config      PDFPageProcessorConfig
}

// NewPDFPageProcessor creates a new page processor.
func NewPDFPageProcessor(textCleaner *PDFTextCleaner, config PDFPageProcessorConfig) *PDFPageProcessor {
	defaults := DefaultPDFPageProcessorConfig()
	if config.ChunkSize <= 0 {
		config.ChunkSize = defaults.ChunkSize
	}
	if config.ChunkCharLimit <= 0 {
		config.ChunkCharLimit = defaults.ChunkCharLimit
	}
	if config.TotalCharLimit <= 0 {
		config.TotalCharLimit = defaults.TotalCharLimit
	}
	if config.DefaultPageLimit <= 0 {
		config.DefaultPageLimit = defaults.DefaultPageLimit
	}

	if textCleaner == nil {
		textCleaner = NewPDFTextCleaner(DefaultPDFTextCleanerConfig())
	}

	return &PDFPageProcessor{
		textCleaner: textCleaner,
		config:      config,
	}
}

// ExtractAllText extracts text from the PDF reader respecting limits and context cancellation.
func (p *PDFPageProcessor) ExtractAllText(ctx context.Context, reader *pdf.Reader, metadata *DocumentMetadata) (string, int, bool, error) {
	pageCount := reader.NumPage()
	maxPages := pageCount
	pageLimitApplied := false
	charLimitApplied := false
	debugLog := p.shouldLogDebug(metadata)

	if metadata != nil && metadata.Properties != nil {
		if limitStr, exists := metadata.Properties["max_pdf_pages"]; exists {
			if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 {
				if limit < maxPages {
					maxPages = limit
					pageLimitApplied = pageCount > limit
				}
			}
		}
	}

	if !pageLimitApplied && p.config.DefaultPageLimit > 0 && p.config.DefaultPageLimit < maxPages {
		maxPages = p.config.DefaultPageLimit
		pageLimitApplied = pageCount > p.config.DefaultPageLimit
	}

	pagesToProcess := pageCount
	if maxPages < pageCount {
		pagesToProcess = maxPages
	}

	limitReached := pageLimitApplied

	if debugLog {
		log.Printf("[PDF-EXTRACT] 📖 PDF has %d pages, processing %d pages", pageCount, pagesToProcess)
		if limitReached {
			log.Printf("[PDF-EXTRACT] ⚠️ Page limit reached: processing only first %d of %d pages", pagesToProcess, pageCount)
		}
	}

	var finalText strings.Builder
	if pagesToProcess > 0 {
		finalText.Grow(minInt(p.config.TotalCharLimit, pagesToProcess*64*1024))
	}

	totalCharCount := 0
	processedPages := 0

	for chunkStart := 1; chunkStart <= pagesToProcess; chunkStart += p.config.ChunkSize {
		chunkEnd := chunkStart + p.config.ChunkSize - 1
		if chunkEnd > pagesToProcess {
			chunkEnd = pagesToProcess
		}

		if debugLog {
			log.Printf("[PDF-EXTRACT] 🔄 Processing chunk: pages %d-%d", chunkStart, chunkEnd)
		}

		chunkText, chunkPageCount, err := p.processPageChunk(ctx, reader, chunkStart, chunkEnd, debugLog)
		if err != nil {
			if ctx.Err() != nil {
				return finalText.String(), processedPages, limitReached, ctx.Err()
			}
			if debugLog {
				log.Printf("[PDF-EXTRACT] ⚠️ Error processing chunk %d-%d: %v", chunkStart, chunkEnd, err)
			}
			continue
		}

		if chunkText != "" {
			cleanedChunk := p.textCleaner.StreamCleanText(chunkText)
			if cleanedChunk != "" {
				if finalText.Len() > 0 {
					finalText.WriteString("\n\n")
				}
				finalText.WriteString(cleanedChunk)
				totalCharCount += len(cleanedChunk)
			}
		}

		processedPages += chunkPageCount

		if totalCharCount > p.config.TotalCharLimit {
			if debugLog {
				log.Printf("[PDF-EXTRACT] ⚠️ Total text limit reached (%d chars), stopping at chunk %d-%d", totalCharCount, chunkStart, chunkEnd)
			}
			limitReached = true
			charLimitApplied = true
			break
		}

		if chunkPageCount < (chunkEnd - chunkStart + 1) {
			pagesToProcess = chunkStart + chunkPageCount - 1
			limitReached = true
			break
		}
	}

	result := finalText.String()
	if metadata != nil && metadata.Properties != nil {
		metadata.Properties["pdf_extraction_limited"] = strconv.FormatBool(limitReached)
		if pageLimitApplied {
			metadata.Properties["pdf_page_limit"] = strconv.Itoa(pagesToProcess)
		}
		metadata.Properties["pdf_pages_processed"] = strconv.Itoa(processedPages)
		metadata.Properties["pdf_total_pages"] = strconv.Itoa(pageCount)
		metadata.Properties["pdf_char_limit_hit"] = strconv.FormatBool(charLimitApplied)
		if charLimitApplied {
			metadata.Properties["pdf_char_limit"] = strconv.Itoa(p.config.TotalCharLimit)
		}
	}

	if debugLog {
		if limitReached {
			log.Printf("[PDF-EXTRACT] Limited extraction: %d chars from %d pages (skipped %d pages)", len(result), processedPages, pageCount-processedPages)
		} else {
			log.Printf("[PDF-EXTRACT] Complete extraction: %d chars from %d pages", len(result), processedPages)
		}
	}

	return result, processedPages, limitReached, nil
}

func (p *PDFPageProcessor) processPageChunk(ctx context.Context, reader *pdf.Reader, startPage, endPage int, debugLog bool) (string, int, error) {
	var chunkText strings.Builder
	chunkText.Grow(128 * 1024)
	processedPages := 0

	for pageNum := startPage; pageNum <= endPage; pageNum++ {
		select {
		case <-ctx.Done():
			return chunkText.String(), processedPages, ctx.Err()
		default:
		}

		page := reader.Page(pageNum)
		if page.V.IsNull() {
			if debugLog {
				log.Printf("[PDF-EXTRACT] Page %d is null, skipping", pageNum)
			}
			continue
		}

		type pageResult struct {
			text string
			err  error
		}

		resultChan := make(chan pageResult, 1)
		go func() {
			pageText, err := page.GetPlainText(nil)
			resultChan <- pageResult{text: pageText, err: err}
		}()

		var pageText string
		select {
		case result := <-resultChan:
			if result.err != nil {
				if debugLog {
					log.Printf("[PDF-EXTRACT] Error extracting text from page %d: %v", pageNum, result.err)
				}
				continue
			}
			pageText = result.text
		case <-time.After(10 * time.Second):
			if debugLog {
				log.Printf("[PDF-EXTRACT] Timeout extracting page %d (10s), skipping", pageNum)
			}
			continue
		case <-ctx.Done():
			return chunkText.String(), processedPages, ctx.Err()
		}

		if pageText == "" {
			if debugLog {
				log.Printf("[PDF-EXTRACT] Page %d has no text content", pageNum)
			}
			processedPages++
			continue
		}

		if chunkText.Len() > 0 {
			chunkText.WriteString("\n\n")
		}

		cleanedPageText := p.textCleaner.BasicPageCleaning(pageText)
		chunkText.WriteString(cleanedPageText)
		processedPages++

		if chunkText.Len() > p.config.ChunkCharLimit {
			if debugLog {
				log.Printf("[PDF-EXTRACT] ⚠️ Chunk size limit reached at page %d (%d chars), stopping chunk", pageNum, chunkText.Len())
			}
			break
		}
	}

	return chunkText.String(), processedPages, nil
}

func (p *PDFPageProcessor) shouldLogDebug(metadata *DocumentMetadata) bool {
	if metadata != nil && metadata.Properties != nil {
		if value, exists := metadata.Properties["pdf_debug"]; exists {
			return strings.EqualFold(value, "true")
		}
	}
	return p.config.EnableDebugLog
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
