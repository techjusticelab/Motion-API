//go:build enhanced
// +build enhanced

package extractor

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"unicode"

	"github.com/dslipak/pdf"
)

// dslipakPDFExtractor handles PDF files using the dslipak/pdf library
// This is an enhanced version that combines rsc/pdf and ledongthuc/pdf improvements
type dslipakPDFExtractor struct{}

// NewDslipakPDFExtractor creates a new dslipak PDF extractor
func NewDslipakPDFExtractor() Extractor {
	return &dslipakPDFExtractor{}
}

// Extract extracts text from PDF files using the dslipak/pdf library
func (e *dslipakPDFExtractor) Extract(ctx context.Context, reader io.Reader, metadata *DocumentMetadata) (*ExtractionResult, error) {
	// Read the PDF content
	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, NewExtractionError("pdf", "failed to read PDF file", err)
	}

	// Validate PDF format
	if err := e.validatePDF(content); err != nil {
		return nil, NewExtractionError("pdf", "invalid PDF format", err)
	}

	// Create a reader from the content
	contentReader := bytes.NewReader(content)

	// Open PDF with dslipak/pdf
	r, err := pdf.NewReader(contentReader, int64(len(content)))
	if err != nil {
		return nil, NewExtractionError("pdf", "failed to open PDF with dslipak/pdf", err)
	}

	// Extract text using multiple methods for best results
	text, pageCount, err := e.extractTextMultiMethod(r)
	if err != nil {
		return nil, NewExtractionError("pdf", "failed to extract text", err)
	}

	// Clean and process the extracted text
	text = e.cleanText(text)
	wordCount := countWords(text)
	charCount := len(text)
	language := e.detectLanguage(text)

	return &ExtractionResult{
		Text:      text,
		WordCount: wordCount,
		CharCount: charCount,
		PageCount: pageCount,
		Language:  language,
		Metadata: map[string]interface{}{
			"format":      "pdf",
			"file_size":   len(content),
			"extraction":  "dslipak/pdf",
			"pdf_version": e.extractPDFVersion(content),
		},
		Success:  true,
		Duration: 0, // Will be set by service
	}, nil
}

// SupportedFormats returns the formats this extractor supports
func (e *dslipakPDFExtractor) SupportedFormats() []string {
	return []string{"pdf"}
}

// CanExtract checks if this extractor can handle the given format
func (e *dslipakPDFExtractor) CanExtract(format string) bool {
	return strings.ToLower(format) == "pdf"
}

// validatePDF performs basic PDF validation
func (e *dslipakPDFExtractor) validatePDF(content []byte) error {
	if len(content) < 4 {
		return fmt.Errorf("file too small to be a valid PDF")
	}

	// Check for PDF header (be flexible about position)
	headerFound := false
	searchLimit := min(1024, len(content))

	for i := 0; i <= searchLimit-4; i++ {
		if string(content[i:i+4]) == "%PDF" {
			headerFound = true
			break
		}
	}

	if !headerFound {
		return fmt.Errorf("PDF header not found")
	}

	return nil
}

// extractTextMultiMethod tries multiple extraction approaches for best results
func (e *dslipakPDFExtractor) extractTextMultiMethod(reader *pdf.Reader) (string, int, error) {
	pageCount := reader.NumPage()
	if pageCount == 0 {
		return "", 0, fmt.Errorf("PDF has no pages")
	}

	// Method 1: Try GetPlainText (fastest, basic)
	text, err := e.extractWithPlainText(reader)
	if err == nil && text != "" {
		return text, pageCount, nil
	}

	// Method 2: Try GetTextByRow (structured, better for legal documents)
	text, err = e.extractWithTextByRow(reader)
	if err == nil && text != "" {
		return text, pageCount, nil
	}

	// Method 3: Try page-by-page extraction with error recovery
	text, err = e.extractPageByPage(reader)
	if err == nil && text != "" {
		return text, pageCount, nil
	}

	// If all methods fail, return empty text but don't error
	// This allows OCR fallback to be attempted
	return "", pageCount, nil
}

// extractWithPlainText uses the GetPlainText method
func (e *dslipakPDFExtractor) extractWithPlainText(reader *pdf.Reader) (string, error) {
	var buf bytes.Buffer
	plainText, err := reader.GetPlainText()
	if err != nil {
		return "", err
	}

	_, err = buf.ReadFrom(plainText)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

// extractWithTextByRow uses the GetTextByRow method for structured extraction
func (e *dslipakPDFExtractor) extractWithTextByRow(reader *pdf.Reader) (string, error) {
	var allText strings.Builder
	pageCount := reader.NumPage()

	for pageIndex := 1; pageIndex <= pageCount; pageIndex++ {
		page := reader.Page(pageIndex)
		if page.V.IsNull() {
			continue
		}

		rows, err := page.GetTextByRow()
		if err != nil {
			continue
		}

		for _, row := range rows {
			for _, word := range row.Content {
				if word.S != "" {
					if allText.Len() > 0 {
						allText.WriteString(" ")
					}
					allText.WriteString(word.S)
				}
			}
			// Add line break after each row
			allText.WriteString("\n")
		}

		// Add page break
		if pageIndex < pageCount {
			allText.WriteString("\n\n")
		}
	}

	return allText.String(), nil
}

// extractPageByPage processes each page individually with error recovery
func (e *dslipakPDFExtractor) extractPageByPage(reader *pdf.Reader) (string, error) {
	var allText strings.Builder
	pageCount := reader.NumPage()
	successCount := 0

	for pageIndex := 1; pageIndex <= pageCount; pageIndex++ {
		pageText, err := e.extractSinglePage(reader, pageIndex)
		if err != nil {
			// Log error but continue with other pages
			continue
		}

		if pageText != "" {
			if allText.Len() > 0 {
				allText.WriteString("\n\n")
			}
			allText.WriteString(pageText)
			successCount++
		}
	}

	if successCount == 0 {
		return "", fmt.Errorf("no text could be extracted from any page")
	}

	return allText.String(), nil
}

// extractSinglePage extracts text from a single page
func (e *dslipakPDFExtractor) extractSinglePage(reader *pdf.Reader, pageIndex int) (string, error) {
	page := reader.Page(pageIndex)
	if page.V.IsNull() {
		return "", fmt.Errorf("page %d is null", pageIndex)
	}

	// Try GetTextByRow first (more structured)
	rows, err := page.GetTextByRow()
	if err == nil && len(rows) > 0 {
		var pageText strings.Builder
		for _, row := range rows {
			for _, word := range row.Content {
				if word.S != "" {
					if pageText.Len() > 0 {
						pageText.WriteString(" ")
					}
					pageText.WriteString(word.S)
				}
			}
			pageText.WriteString("\n")
		}
		return pageText.String(), nil
	}

	// Fallback: try other page methods if available
	return "", fmt.Errorf("could not extract text from page %d", pageIndex)
}

// cleanText performs comprehensive text cleaning
func (e *dslipakPDFExtractor) cleanText(text string) string {
	// Remove excessive whitespace
	text = strings.TrimSpace(text)
	
	// Normalize line breaks
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")

	// Remove excessive line breaks (more than 2 consecutive)
	lines := strings.Split(text, "\n")
	var cleanedLines []string
	consecutiveEmpty := 0

	for _, line := range lines {
		line = strings.TrimSpace(line)
		
		if line == "" {
			consecutiveEmpty++
			if consecutiveEmpty <= 2 {
				cleanedLines = append(cleanedLines, line)
			}
		} else {
			consecutiveEmpty = 0
			cleanedLines = append(cleanedLines, line)
		}
	}

	text = strings.Join(cleanedLines, "\n")

	// Remove non-printable characters (except newlines and tabs)
	var cleaned strings.Builder
	for _, r := range text {
		if unicode.IsPrint(r) || r == '\n' || r == '\t' {
			cleaned.WriteRune(r)
		}
	}

	return strings.TrimSpace(cleaned.String())
}

// extractPDFVersion extracts the PDF version from the file header
func (e *dslipakPDFExtractor) extractPDFVersion(content []byte) string {
	if len(content) < 8 {
		return "unknown"
	}

	// Look for PDF version in first 1024 bytes
	searchLimit := min(1024, len(content))
	contentStr := string(content[:searchLimit])

	for i := 0; i <= len(contentStr)-8; i++ {
		if strings.HasPrefix(contentStr[i:], "%PDF-") {
			if i+8 <= len(contentStr) {
				return contentStr[i+5 : i+8]
			}
		}
	}

	return "unknown"
}

// detectLanguage performs basic language detection
func (e *dslipakPDFExtractor) detectLanguage(text string) string {
	// Simple heuristic-based language detection for English
	englishWords := []string{
		"the", "and", "of", "to", "a", "in", "for", "is", "on", "that",
		"by", "this", "with", "from", "they", "we", "say", "her", "she",
		"or", "an", "will", "my", "one", "all", "would", "there", "their",
		// Legal terms
		"court", "case", "defendant", "plaintiff", "motion", "order",
	}

	words := strings.Fields(strings.ToLower(text))
	if len(words) == 0 {
		return "unknown"
	}

	englishCount := 0
	sampleSize := min(100, len(words)) // Sample first 100 words

	for i := 0; i < sampleSize; i++ {
		for _, englishWord := range englishWords {
			if words[i] == englishWord {
				englishCount++
				break
			}
		}
	}

	// If more than 20% of words are common English words, assume English
	if float64(englishCount)/float64(sampleSize) > 0.2 {
		return "en"
	}

	return "unknown"
}

// Helper function to get minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}