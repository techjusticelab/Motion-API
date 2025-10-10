package extractor

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"unicode"

	"github.com/ledongthuc/pdf"
)

// pdfExtractor handles PDF files using the ledongthuc/pdf library
type pdfExtractor struct{}

// NewPDFExtractor creates a new PDF extractor
func NewPDFExtractor() Extractor {
	return &pdfExtractor{}
}

// Extract extracts text from PDF files with fallback mechanisms
func (e *pdfExtractor) Extract(ctx context.Context, reader io.Reader, metadata *DocumentMetadata) (result *ExtractionResult, err error) {
	// Add panic recovery to prevent server crashes
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[PDF-EXTRACT] 💥 Panic recovered: %v", r)
			err = NewExtractionError("pdf", fmt.Sprintf("PDF processing panic: %v", r), nil)
			result = &ExtractionResult{
				Text:      "",
				WordCount: 0,
				CharCount: 0,
				PageCount: 0,
				Language:  "unknown",
				Metadata: map[string]interface{}{
					"format":         "pdf",
					"extraction":     "failed_panic_recovery",
					"error":          fmt.Sprintf("panic: %v", r),
					"pages_processed": 0,
				},
			}
		}
	}()
	
	// Read the PDF content
	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, NewExtractionError("pdf", "failed to read PDF file", err)
	}

	log.Printf("[PDF-EXTRACT] 📄 Processing PDF: %s, size: %d bytes", metadata.FileName, len(content))

	// Enhanced PDF validation - be more lenient
	if len(content) < 4 {
		log.Printf("[PDF-EXTRACT] ❌ PDF too small: %d bytes", len(content))
		return nil, NewExtractionError("pdf", "file too small to be a valid PDF", nil)
	}

	// Check for PDF header (be more flexible)
	header := string(content[:4])
	log.Printf("[PDF-EXTRACT] 🔍 PDF header check: %q", header)
	if header != "%PDF" {
		// Try to find PDF header within the first 1024 bytes (some files have prefixes)
		headerFound := false
		searchLimit := 1024
		if len(content) < searchLimit {
			searchLimit = len(content)
		}

		log.Printf("[PDF-EXTRACT] 🔍 Searching for PDF header in first %d bytes", searchLimit)
		for i := 0; i <= searchLimit-4; i++ {
			if string(content[i:i+4]) == "%PDF" {
				headerFound = true
				content = content[i:] // Trim prefix
				log.Printf("[PDF-EXTRACT] ✅ Found PDF header at position %d", i)
				break
			}
		}

		if !headerFound {
			log.Printf("[PDF-EXTRACT] ❌ No valid PDF header found")
			return nil, NewExtractionError("pdf", "invalid PDF file format", nil)
		}
	}

	// Try primary extraction method
	log.Printf("[PDF-EXTRACT] 🔄 Attempting primary extraction method (ledongthuc/pdf)")
	text, pageCount, err := e.extractWithPrimaryMethod(content, metadata)
	if err == nil && text != "" {
		// Success with primary method
		log.Printf("[PDF-EXTRACT] ✅ Primary method successful: %d chars, %d pages", len(text), pageCount)
		log.Printf("[PDF-EXTRACT] 🧹 Before cleaning: %d chars", len(text))
		text = e.cleanText(text)
		log.Printf("[PDF-EXTRACT] 🧹 After cleaning: %d chars", len(text))
		wordCount := countWords(text)
		charCount := len(text)
		language := e.detectLanguage(text)

		log.Printf("[PDF-EXTRACT] 🔍 About to return result: Text=%d chars, WordCount=%d, CharCount=%d",
			len(text), wordCount, charCount)

		result := &ExtractionResult{
			Text:      text,
			WordCount: wordCount,
			CharCount: charCount,
			PageCount: pageCount,
			Language:  language,
			Metadata: map[string]interface{}{
				"format":      "pdf",
				"file_size":   len(content),
				"extraction":  "ledongthuc/pdf",
				"pdf_version": e.extractPDFVersion(content),
			},
		}

		log.Printf("[PDF-EXTRACT] 🔍 Created ExtractionResult: Text field length=%d", len(result.Text))
		return result, nil
	}

	log.Printf("[PDF-EXTRACT] ⚠️ Primary method failed: err=%v, text_len=%d", err, len(text))

	// Primary method failed, try fallback methods
	log.Printf("[PDF-EXTRACT] 🔄 Attempting fallback extraction methods")
	text, pageCount, extractionMethod, fallbackErr := e.extractWithFallbackMethods(content)
	if fallbackErr != nil {
		// All methods failed
		log.Printf("[PDF-EXTRACT] ❌ All extraction methods failed: %v", fallbackErr)
		return nil, NewExtractionError("pdf", "failed to extract text with all methods", err)
	}

	// Clean up the text
	text = e.cleanText(text)

	// Count words and characters
	wordCount := countWords(text)
	charCount := len(text)

	// Detect language (basic detection)
	language := e.detectLanguage(text)

	log.Printf("[PDF-EXTRACT] 🔍 Fallback path - About to return result: Text=%d chars, WordCount=%d",
		len(text), wordCount)

	result := &ExtractionResult{
		Text:      text,
		WordCount: wordCount,
		CharCount: charCount,
		PageCount: pageCount,
		Language:  language,
		Metadata: map[string]interface{}{
			"format":      "pdf",
			"file_size":   len(content),
			"extraction":  extractionMethod,
			"pdf_version": e.extractPDFVersion(content),
		},
	}

	log.Printf("[PDF-EXTRACT] 🔍 Fallback ExtractionResult: Text field length=%d", len(result.Text))
	return result, nil
}

// SupportedFormats returns the formats this extractor supports
func (e *pdfExtractor) SupportedFormats() []string {
	return []string{"pdf"}
}

// CanExtract checks if this extractor can handle the given format
func (e *pdfExtractor) CanExtract(format string) bool {
	return strings.ToLower(format) == "pdf"
}

// extractWithPrimaryMethod uses the original ledongthuc/pdf method
func (e *pdfExtractor) extractWithPrimaryMethod(content []byte, metadata *DocumentMetadata) (string, int, error) {
	// Create a reader from the content
	contentReader := bytes.NewReader(content)

	// Open PDF for reading
	log.Printf("[PDF-EXTRACT] 🔓 Opening PDF with ledongthuc/pdf library")
	pdfReader, err := pdf.NewReader(contentReader, int64(len(content)))
	if err != nil {
		log.Printf("[PDF-EXTRACT] ❌ Failed to open PDF with ledongthuc/pdf: %v", err)
		return "", 0, err
	}

	log.Printf("[PDF-EXTRACT] ✅ PDF opened successfully, extracting text from pages")
	// Extract text from pages (respecting page limit)
	text, pageCount, err := e.extractAllText(pdfReader, metadata)
	log.Printf("[PDF-EXTRACT] 📊 Primary extraction result: %d chars, %d pages, err: %v", len(text), pageCount, err)
	return text, pageCount, err
}

// extractWithFallbackMethods tries alternative extraction approaches
func (e *pdfExtractor) extractWithFallbackMethods(content []byte) (string, int, string, error) {
	log.Printf("[PDF-EXTRACT] 🔄 Starting fallback extraction methods")

	// Fallback 1: Try to extract raw text streams from PDF
	log.Printf("[PDF-EXTRACT] 🔄 Attempting fallback method 1: raw stream extraction")
	text, pageCount := e.extractRawTextStreams(content)
	log.Printf("[PDF-EXTRACT] 📊 Raw stream extraction result: %d chars, %d pages", len(text), pageCount)
	if text != "" {
		log.Printf("[PDF-EXTRACT] ✅ Raw stream extraction successful")
		return text, pageCount, "raw_stream_extraction", nil
	}

	// Fallback 2: Basic pattern matching for common text patterns
	log.Printf("[PDF-EXTRACT] 🔄 Attempting fallback method 2: pattern extraction")
	text = e.extractBasicTextPatterns(content)
	log.Printf("[PDF-EXTRACT] 📊 Pattern extraction result: %d chars", len(text))
	if text != "" {
		log.Printf("[PDF-EXTRACT] ✅ Pattern extraction successful")
		return text, 1, "pattern_extraction", nil
	}

	log.Printf("[PDF-EXTRACT] ❌ All fallback methods failed to extract text")
	return "", 0, "", fmt.Errorf("all extraction methods failed")
}

// extractRawTextStreams attempts to extract text from PDF streams
func (e *pdfExtractor) extractRawTextStreams(content []byte) (string, int) {
	var extractedText strings.Builder
	contentStr := string(content)
	log.Printf("[PDF-EXTRACT] 🔍 Searching for PDF streams in %d byte content", len(contentStr))

	// Look for text streams in PDF
	streamRegex := regexp.MustCompile(`stream\s*(.*?)\s*endstream`)
	matches := streamRegex.FindAllStringSubmatch(contentStr, -1)
	log.Printf("[PDF-EXTRACT] 📊 Found %d PDF streams", len(matches))

	pageCount := 0
	for i, match := range matches {
		if len(match) > 1 {
			streamContent := match[1]
			log.Printf("[PDF-EXTRACT] 🔄 Processing stream %d (length: %d)", i+1, len(streamContent))

			// Look for text commands in the stream
			text := e.extractTextFromStream(streamContent)
			log.Printf("[PDF-EXTRACT] 📝 Stream %d extracted text: %d chars", i+1, len(text))
			if text != "" {
				if extractedText.Len() > 0 {
					extractedText.WriteString("\n\n")
				}
				extractedText.WriteString(text)
				pageCount++
			}
		}
	}

	finalResult := extractedText.String()
	log.Printf("[PDF-EXTRACT] 📊 Raw stream extraction complete: %d chars from %d streams", len(finalResult), pageCount)
	return finalResult, pageCount
}

// extractTextFromStream extracts readable text from a PDF stream
func (e *pdfExtractor) extractTextFromStream(stream string) string {
	var text strings.Builder

	// Look for text showing commands like (text) Tj, [text] TJ, etc.
	textPatterns := []*regexp.Regexp{
		regexp.MustCompile(`\((.*?)\)\s*[Tt][jJ]`), // (text) Tj
		regexp.MustCompile(`\[(.*?)\]\s*[Tt][jJ]`), // [text] TJ
		regexp.MustCompile(`\((.*?)\)\s*[Tt][dD]`), // (text) Td
	}

	for _, pattern := range textPatterns {
		matches := pattern.FindAllStringSubmatch(stream, -1)
		for _, match := range matches {
			if len(match) > 1 {
				textContent := match[1]
				// Clean and add text
				cleaned := e.cleanExtractedText(textContent)
				if cleaned != "" {
					if text.Len() > 0 {
						text.WriteString(" ")
					}
					text.WriteString(cleaned)
				}
			}
		}
	}

	return text.String()
}

// extractBasicTextPatterns looks for readable text patterns in the PDF
func (e *pdfExtractor) extractBasicTextPatterns(content []byte) string {
	contentStr := string(content)
	var text strings.Builder
	log.Printf("[PDF-EXTRACT] 🔍 Pattern extraction: scanning %d lines", len(strings.Split(contentStr, "\n")))

	// Look for patterns that might contain readable text
	// This is a very basic approach but can work for simple PDFs
	lines := strings.Split(contentStr, "\n")

	potentialTextLines := 0
	addedLines := 0
	for i, line := range lines {
		// Skip obvious binary or control lines
		if e.isPotentialTextLine(line) {
			potentialTextLines++
			cleaned := e.cleanExtractedText(line)
			if len(cleaned) > 2 { // Only add lines with substantial content
				if addedLines < 5 { // Log first few matches for debugging
					log.Printf("[PDF-EXTRACT] 📝 Pattern match line %d: %q", i+1, cleaned[:min(50, len(cleaned))])
				}
				if text.Len() > 0 {
					text.WriteString(" ")
				}
				text.WriteString(cleaned)
				addedLines++
			}
		}
	}

	result := text.String()
	log.Printf("[PDF-EXTRACT] 📊 Pattern extraction result: found %d potential text lines, added %d lines, total %d chars",
		potentialTextLines, addedLines, len(result))
	return result
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// isPotentialTextLine checks if a line might contain readable text
func (e *pdfExtractor) isPotentialTextLine(line string) bool {
	line = strings.TrimSpace(line)

	// Skip empty lines
	if len(line) == 0 {
		return false
	}

	// Skip lines that are mostly non-printable
	printableCount := 0
	for _, r := range line {
		if unicode.IsPrint(r) {
			printableCount++
		}
	}

	return float64(printableCount)/float64(len(line)) > 0.7
}

// cleanExtractedText cleans up extracted text
func (e *pdfExtractor) cleanExtractedText(text string) string {
	// Remove PDF escape sequences
	text = strings.ReplaceAll(text, "\\n", " ")
	text = strings.ReplaceAll(text, "\\r", " ")
	text = strings.ReplaceAll(text, "\\t", " ")
	text = strings.ReplaceAll(text, "\\(", "(")
	text = strings.ReplaceAll(text, "\\)", ")")
	text = strings.ReplaceAll(text, "\\\\", "\\")

	// Remove non-printable characters
	var cleaned strings.Builder
	for _, r := range text {
		if unicode.IsPrint(r) || r == ' ' {
			cleaned.WriteRune(r)
		}
	}

	return strings.TrimSpace(cleaned.String())
}

// extractAllText extracts text from all pages of the PDF
func (e *pdfExtractor) extractAllText(reader *pdf.Reader, metadata *DocumentMetadata) (string, int, error) {
	var allText strings.Builder
	pageCount := reader.NumPage()
	
	// Get page limit from metadata properties, default to 25
	maxPages := 25
	if metadata != nil && metadata.Properties != nil {
		if limitStr, exists := metadata.Properties["max_pdf_pages"]; exists {
			if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 {
				maxPages = limit
			}
		}
	}
	
	// Determine actual pages to process
	pagesToProcess := pageCount
	limitReached := false
	if pageCount > maxPages {
		pagesToProcess = maxPages
		limitReached = true
	}
	
	log.Printf("[PDF-EXTRACT] 📖 PDF has %d pages, processing %d pages (limit: %d)", 
		pageCount, pagesToProcess, maxPages)
	
	if limitReached {
		log.Printf("[PDF-EXTRACT] ⚠️ Page limit reached: processing only first %d of %d pages", 
			maxPages, pageCount)
	}

	for pageNum := 1; pageNum <= pagesToProcess; pageNum++ {
		// Check memory usage every few pages to prevent crashes
		if pageNum%5 == 0 {
			var memStats runtime.MemStats
			runtime.ReadMemStats(&memStats)
			// If we're using more than 200MB for this extraction, abort
			if memStats.Alloc > 200*1024*1024 {
				log.Printf("[PDF-EXTRACT] ⚠️ Memory limit reached at page %d (%d MB), aborting extraction", 
					pageNum, memStats.Alloc/(1024*1024))
				limitReached = true
				pagesToProcess = pageNum - 1
				break
			}
		}
		
		page := reader.Page(pageNum)
		if page.V.IsNull() {
			log.Printf("[PDF-EXTRACT] ⚠️ Page %d is null, skipping", pageNum)
			continue
		}

		// Extract text content from the page
		pageText, err := page.GetPlainText(nil)
		if err != nil {
			log.Printf("[PDF-EXTRACT] ❌ Error extracting text from page %d: %v", pageNum, err)
			continue
		}

		if pageText == "" {
			log.Printf("[PDF-EXTRACT] ⚠️ Page %d has no text content", pageNum)
			continue
		}

		// Check if we have excessive text accumulation
		if allText.Len() > 10*1024*1024 { // 10MB of text is excessive
			log.Printf("[PDF-EXTRACT] ⚠️ Text length limit reached at page %d (%d chars), aborting extraction", 
				pageNum, allText.Len())
			limitReached = true
			pagesToProcess = pageNum
			break
		}

		// Add page text with page separator
		if allText.Len() > 0 {
			allText.WriteString("\n\n")
		}
		allText.WriteString(pageText)
	}

	finalText := allText.String()
	
	// Add metadata about page limiting
	resultPageCount := pagesToProcess
	if limitReached {
		log.Printf("[PDF-EXTRACT] 📊 Limited extraction result: %d chars from %d pages (limit reached, skipped %d pages)", 
			len(finalText), pagesToProcess, pageCount-pagesToProcess)
	} else {
		log.Printf("[PDF-EXTRACT] 📊 Total extraction result: %d chars from %d pages", len(finalText), pageCount)
	}
	
	return finalText, resultPageCount, nil
}

// cleanText performs comprehensive text cleaning using the enhanced TextCleaner
func (e *pdfExtractor) cleanText(text string) string {
	// Create text cleaner with default configuration
	cleaner := NewTextCleaner(DefaultCleaningConfig())
	
	// Apply enhanced cleaning
	log.Printf("[PDF-EXTRACT] 🧹 Before enhanced cleaning: %d chars", len(text))
	text = cleaner.CleanText(text)
	log.Printf("[PDF-EXTRACT] 🧹 After enhanced cleaning: %d chars", len(text))

	// Apply existing PDF-specific artifact removal (kept for compatibility)
	text = e.removePDFArtifacts(text)

	// Final text normalization
	text = e.finalTextNormalization(text)

	return text
}

// finalTextNormalization performs basic text normalization
func (e *pdfExtractor) finalTextNormalization(text string) string {
	// Replace multiple whitespaces with single space
	re := regexp.MustCompile(`\s+`)
	text = re.ReplaceAllString(text, " ")

	// Remove non-printable characters except newlines and tabs
	var cleaned strings.Builder
	for _, r := range text {
		if unicode.IsPrint(r) || r == '\n' || r == '\t' {
			cleaned.WriteRune(r)
		}
	}

	text = cleaned.String()

	// Normalize line breaks
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")

	// Remove excessive line breaks
	text = regexp.MustCompile(`\n{3,}`).ReplaceAllString(text, "\n\n")

	// Trim leading and trailing whitespace
	text = strings.TrimSpace(text)

	return text
}

// removePDFArtifacts removes common PDF rendering artifacts
func (e *pdfExtractor) removePDFArtifacts(text string) string {
	// Remove common PDF control sequences (use exact matches only)
	artifacts := []string{
		"endobj", "stream", "endstream", // Removed "obj" as it matches normal words
		"xref", "trailer", "startxref",
		"%%EOF", "%%Page:",
	}

	lines := strings.Split(text, "\n")
	var cleanedLines []string

	log.Printf("[PDF-ARTIFACTS] Processing %d lines for artifact removal", len(lines))
	removedCount := 0

	for i, line := range lines {
		line = strings.TrimSpace(line)

		// Skip empty lines
		if len(line) == 0 {
			cleanedLines = append(cleanedLines, line)
			continue
		}

		// Check if line contains PDF artifacts
		isArtifact := false
		artifactReason := ""

		// Check for exact artifact lines or lines that start with artifacts
		lineLower := strings.ToLower(strings.TrimSpace(line))
		for _, artifact := range artifacts {
			artifactLower := strings.ToLower(artifact)
			// Match if line exactly equals artifact or starts with artifact followed by space/end
			if lineLower == artifactLower || strings.HasPrefix(lineLower, artifactLower+" ") || strings.HasPrefix(lineLower, artifactLower) && len(lineLower) <= len(artifactLower)+5 {
				isArtifact = true
				artifactReason = fmt.Sprintf("matches artifact '%s'", artifact)
				break
			}
		}

		// Skip lines that are just numbers (likely page numbers or references)
		if !isArtifact && e.isNumericLine(line) {
			isArtifact = true
			artifactReason = "numeric line"
		}

		// Skip very short lines with only special characters
		if !isArtifact && len(line) < 3 && !e.hasAlphanumeric(line) {
			isArtifact = true
			artifactReason = "short non-alphanumeric"
		}

		// Check for repeated character patterns (dots, dashes, etc.)
		if !isArtifact && e.isRepeatedCharacterLine(line) {
			isArtifact = true
			artifactReason = "repeated character pattern"
		}

		// Check for form field lines with dots
		if !isArtifact && e.isFormFieldLine(line) {
			isArtifact = true
			artifactReason = "form field line"
		}

		// Check for header/footer patterns
		if !isArtifact && e.isHeaderFooterLine(line) {
			isArtifact = true
			artifactReason = "header/footer pattern"
		}

		if isArtifact {
			removedCount++
			if removedCount <= 10 { // Log first 10 removals
				log.Printf("[PDF-ARTIFACTS] Removing line %d (%s): %q", i+1, artifactReason, line[:min(50, len(line))])
			}
		} else {
			// Clean table of contents artifacts but keep the line
			cleanedLine := e.cleanTableOfContentsArtifacts(line)
			cleanedLines = append(cleanedLines, cleanedLine)
		}
	}

	result := strings.Join(cleanedLines, "\n")
	log.Printf("[PDF-ARTIFACTS] Artifact removal complete: %d lines removed, %d lines kept, result length: %d chars",
		removedCount, len(cleanedLines), len(result))
	return result
}

// isNumericLine checks if a line contains only numbers and common separators
func (e *pdfExtractor) isNumericLine(line string) bool {
	if len(line) == 0 {
		return false
	}

	for _, r := range line {
		if !unicode.IsDigit(r) && r != '.' && r != ',' && r != '-' && r != ' ' {
			return false
		}
	}
	return true
}

// hasAlphanumeric checks if a string contains alphanumeric characters
func (e *pdfExtractor) hasAlphanumeric(s string) bool {
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return true
		}
	}
	return false
}

// isRepeatedCharacterLine detects lines that are mostly repeated characters (dots, dashes, underscores)
func (e *pdfExtractor) isRepeatedCharacterLine(line string) bool {
	if len(line) < 5 {
		return false
	}
	
	// Common repeated characters used in forms and layouts
	repeatedChars := map[rune]int{
		'.': 0, '-': 0, '_': 0, '=': 0, '*': 0, '+': 0, '~': 0,
	}
	
	totalChars := 0
	repeatedCount := 0
	
	for _, r := range line {
		if r == ' ' || r == '\t' { // Skip whitespace
			continue
		}
		totalChars++
		if _, exists := repeatedChars[r]; exists {
			repeatedChars[r]++
			repeatedCount++
		}
	}
	
	// If more than 70% of non-whitespace characters are repeated chars
	if totalChars > 0 && float64(repeatedCount)/float64(totalChars) > 0.7 {
		return true
	}
	
	// Special case: check for patterns like ".-.-.-" or "_._._.
	for char := range repeatedChars {
		pattern := string(char)
		// Count occurrences of the character
		charCount := strings.Count(line, pattern)
		if charCount >= 5 && len(line) > 10 {
			// Remove all whitespace and check density
			noSpace := strings.ReplaceAll(line, " ", "")
			noSpace = strings.ReplaceAll(noSpace, "\t", "")
			if float64(charCount)/float64(len(noSpace)) > 0.5 {
				return true
			}
		}
	}
	
	return false
}

// isFormFieldLine detects form field placeholder lines with mixed dots and text
func (e *pdfExtractor) isFormFieldLine(line string) bool {
	if len(line) < 10 {
		return false
	}
	
	// Count dots and letters
	dotCount := strings.Count(line, ".")
	letterCount := 0
	for _, r := range line {
		if unicode.IsLetter(r) {
			letterCount++
		}
	}
	
	// Form fields typically have format like "Name: ........................."
	// or "Address .............................. Phone ................"
	if dotCount >= 5 && letterCount > 0 && letterCount < 20 {
		// Check for typical form patterns
		formPatterns := []string{
			"name", "address", "date", "phone", "signature", "title",
			"city", "state", "zip", "email", "age", "sex", "occupation",
		}
		
		lineLower := strings.ToLower(line)
		for _, pattern := range formPatterns {
			if strings.Contains(lineLower, pattern) && dotCount > letterCount {
				return true
			}
		}
		
		// Generic check: if line has colon followed by lots of dots
		if strings.Contains(line, ":") && dotCount > 10 {
			return true
		}
	}
	
	return false
}

// isHeaderFooterLine detects common header/footer patterns
func (e *pdfExtractor) isHeaderFooterLine(line string) bool {
	if len(line) == 0 {
		return false
	}
	
	lineLower := strings.ToLower(strings.TrimSpace(line))
	
	// Common header/footer patterns
	headerFooterPatterns := []string{
		"page ", "of ", "continued", "confidential", "draft",
		"proprietary", "exhibit ", "attachment ", "schedule ",
		"case no", "docket", "filed", "clerk", "court",
	}
	
	// Check for simple page numbering
	if regexp.MustCompile(`^page\s+\d+`).MatchString(lineLower) {
		return true
	}
	if regexp.MustCompile(`^\d+\s+of\s+\d+$`).MatchString(lineLower) {
		return true
	}
	if regexp.MustCompile(`^-\s*\d+\s*-$`).MatchString(line) {
		return true
	}
	
	// Check for date stamps (MM/DD/YYYY or Month DD, YYYY format)
	if regexp.MustCompile(`\d{1,2}[\/\-]\d{1,2}[\/\-]\d{4}`).MatchString(line) && len(line) < 30 {
		return true
	}
	
	// Check for document ID patterns
	if regexp.MustCompile(`^[A-Z0-9\-]{8,}$`).MatchString(strings.ReplaceAll(line, " ", "")) {
		return true
	}
	
	// Check for patterns in short lines
	if len(line) < 50 {
		for _, pattern := range headerFooterPatterns {
			if strings.Contains(lineLower, pattern) {
				return true
			}
		}
	}
	
	return false
}

// cleanTableOfContentsArtifacts removes TOC dots while preserving meaningful content
func (e *pdfExtractor) cleanTableOfContentsArtifacts(line string) string {
	// Pattern for TOC lines: "Chapter Title ................... Page 42"
	// Keep the title and page number, remove the dots
	
	// Check if line has pattern: text + dots + number
	if strings.Count(line, ".") < 5 {
		return line
	}
	
	// Use regex to find: (text)(dots)(optional spaces)(number)
	tocPattern := regexp.MustCompile(`^(.+?)(\.{5,})(\s*)(\d+)?\s*$`)
	matches := tocPattern.FindStringSubmatch(line)
	
	if len(matches) >= 3 {
		title := strings.TrimSpace(matches[1])
		pageNum := ""
		if len(matches) >= 5 && matches[4] != "" {
			pageNum = matches[4]
		}
		
		// Only keep if title has substantial content
		if len(title) > 3 && e.hasAlphanumeric(title) {
			if pageNum != "" {
				return fmt.Sprintf("%s (page %s)", title, pageNum)
			}
			return title
		}
	}
	
	return line
}

// extractPDFVersion extracts the PDF version from the file header
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

// detectLanguage performs basic language detection
func (e *pdfExtractor) detectLanguage(text string) string {
	// Simple heuristic-based language detection
	// Count common English words
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
	maxWords := 100 // Sample first 100 words for efficiency

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

	// If more than 20% of words are common English words, assume English
	if float64(englishCount)/float64(totalWords) > 0.2 {
		return "en"
	}

	return "unknown"
}
