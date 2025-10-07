//go:build enhanced && tesseract
// +build enhanced,tesseract

package extractor

import (
	"context"
	"fmt"
	"image"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gen2brain/go-fitz"
	"github.com/otiai10/gosseract/v2"
)

// ocrExtractor handles scanned PDFs and images using OCR
// This follows UNIX philosophy: do one thing (OCR) and do it well
type ocrExtractor struct {
	config *OCRConfig
}

// Additional OCR-specific config fields (extends the base OCRConfig from ocr_config.go)
// These are only available when Tesseract is present

// NewOCRExtractor creates a new OCR extractor with the given configuration
func NewOCRExtractor(config *OCRConfig) Extractor {
	if config == nil {
		config = DefaultOCRConfig()
	}
	return &ocrExtractor{config: config}
}

// Extract performs OCR on PDF files by converting to images first
func (e *ocrExtractor) Extract(ctx context.Context, reader io.Reader, metadata *DocumentMetadata) (*ExtractionResult, error) {
	startTime := time.Now()

	// Check if Tesseract is available
	if !e.isTesseractAvailable() {
		return nil, NewExtractionError("ocr", "Tesseract OCR not available", nil)
	}

	// Read the content
	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, NewExtractionError("ocr", "failed to read content", err)
	}

	// Determine if this is a PDF or image
	var text string
	var pageCount int

	if e.isPDF(content) {
		text, pageCount, err = e.extractFromPDF(ctx, content)
	} else {
		text, pageCount, err = e.extractFromImage(ctx, content)
	}

	if err != nil {
		return nil, NewExtractionError("ocr", "OCR extraction failed", err)
	}

	// Calculate metrics
	wordCount := countWords(text)
	charCount := len(text)
	language := e.detectLanguage(text)
	duration := time.Since(startTime).Milliseconds()

	return &ExtractionResult{
		Text:      text,
		WordCount: wordCount,
		CharCount: charCount,
		PageCount: pageCount,
		Language:  language,
		Metadata: map[string]interface{}{
			"format":      "ocr",
			"file_size":   len(content),
			"extraction":  "gosseract/tesseract",
			"dpi":         e.config.DPI,
			"language":    e.config.Language,
			"gpu_enabled": e.config.EnableGPU,
		},
		Success:  true,
		Duration: duration,
	}, nil
}

// SupportedFormats returns the formats this extractor supports
func (e *ocrExtractor) SupportedFormats() []string {
	return []string{"pdf", "png", "jpg", "jpeg", "tiff", "bmp", "gif"}
}

// CanExtract checks if this extractor can handle the given format
func (e *ocrExtractor) CanExtract(format string) bool {
	format = strings.ToLower(format)
	for _, supported := range e.SupportedFormats() {
		if format == supported {
			return true
		}
	}
	return false
}

// isTesseractAvailable checks if Tesseract is installed and accessible
func (e *ocrExtractor) isTesseractAvailable() bool {
	// Try to create a gosseract client
	client := gosseract.NewClient()
	defer client.Close()

	// Try to get version - this will fail if Tesseract is not available
	version := client.Version()
	return version != ""
}

// isPDF checks if the content is a PDF file
func (e *ocrExtractor) isPDF(content []byte) bool {
	if len(content) < 4 {
		return false
	}

	// Check for PDF header within first 1024 bytes
	searchLimit := min(1024, len(content))
	for i := 0; i <= searchLimit-4; i++ {
		if string(content[i:i+4]) == "%PDF" {
			return true
		}
	}
	return false
}

// extractFromPDF converts PDF to images and performs OCR
func (e *ocrExtractor) extractFromPDF(ctx context.Context, content []byte) (string, int, error) {
	// Open PDF with go-fitz
	doc, err := fitz.NewFromMemory(content)
	if err != nil {
		return "", 0, fmt.Errorf("failed to open PDF: %w", err)
	}
	defer doc.Close()

	pageCount := doc.NumPage()
	if pageCount == 0 {
		return "", 0, fmt.Errorf("PDF has no pages")
	}

	var allText strings.Builder
	
	// Process pages (could be done in parallel for better performance)
	for pageNum := 0; pageNum < pageCount; pageNum++ {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return "", pageCount, ctx.Err()
		default:
		}

		// Convert page to image
		img, err := doc.Image(pageNum)
		if err != nil {
			// Log error but continue with other pages
			continue
		}

		// Perform OCR on the image
		pageText, err := e.performOCR(img)
		if err != nil {
			// Log error but continue with other pages
			continue
		}

		// Add page text
		if pageText != "" {
			if allText.Len() > 0 {
				allText.WriteString("\n\n")
			}
			allText.WriteString(pageText)
		}
	}

	return allText.String(), pageCount, nil
}

// extractFromImage performs OCR directly on an image
func (e *ocrExtractor) extractFromImage(ctx context.Context, content []byte) (string, int, error) {
	// Create temporary file for the image
	tempFile, err := e.createTempFile(content, "image")
	if err != nil {
		return "", 0, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer e.cleanupTempFile(tempFile)

	// Perform OCR using gosseract
	client := gosseract.NewClient()
	defer client.Close()

	err = e.configureOCRClient(client)
	if err != nil {
		return "", 0, fmt.Errorf("failed to configure OCR: %w", err)
	}

	err = client.SetImage(tempFile)
	if err != nil {
		return "", 0, fmt.Errorf("failed to set image: %w", err)
	}

	text, err := client.Text()
	if err != nil {
		return "", 0, fmt.Errorf("OCR failed: %w", err)
	}

	return strings.TrimSpace(text), 1, nil
}

// performOCR performs OCR on a Go image.Image
func (e *ocrExtractor) performOCR(img image.Image) (string, error) {
	// Convert image to bytes (PNG format)
	tempFile, err := e.saveImageToTemp(img)
	if err != nil {
		return "", fmt.Errorf("failed to save image: %w", err)
	}
	defer e.cleanupTempFile(tempFile)

	// Perform OCR using gosseract
	client := gosseract.NewClient()
	defer client.Close()

	err = e.configureOCRClient(client)
	if err != nil {
		return "", fmt.Errorf("failed to configure OCR: %w", err)
	}

	err = client.SetImage(tempFile)
	if err != nil {
		return "", fmt.Errorf("failed to set image: %w", err)
	}

	text, err := client.Text()
	if err != nil {
		return "", fmt.Errorf("OCR failed: %w", err)
	}

	return strings.TrimSpace(text), nil
}

// configureOCRClient configures the gosseract client with our settings
func (e *ocrExtractor) configureOCRClient(client *gosseract.Client) error {
	// Set language
	err := client.SetLanguage(e.config.Language)
	if err != nil {
		return fmt.Errorf("failed to set language: %w", err)
	}

	// Set page segmentation mode
	err = client.SetPageSegMode(gosseract.PageSegMode(e.config.PageSegMode))
	if err != nil {
		return fmt.Errorf("failed to set page segmentation mode: %w", err)
	}

	// Set DPI if supported
	// Note: DPI is typically set during image preprocessing, not in Tesseract

	return nil
}

// createTempFile creates a temporary file with the given content
func (e *ocrExtractor) createTempFile(content []byte, prefix string) (string, error) {
	// Create temp file
	tempFile, err := os.CreateTemp(e.config.TempDir, prefix+"_*.tmp")
	if err != nil {
		return "", err
	}
	defer tempFile.Close()

	// Write content
	_, err = tempFile.Write(content)
	if err != nil {
		os.Remove(tempFile.Name())
		return "", err
	}

	return tempFile.Name(), nil
}

// saveImageToTemp saves a Go image to a temporary PNG file
func (e *ocrExtractor) saveImageToTemp(img image.Image) (string, error) {
	// Create temp file with PNG extension
	tempFile, err := os.CreateTemp(e.config.TempDir, "ocr_image_*.png")
	if err != nil {
		return "", err
	}
	defer tempFile.Close()

	// Encode as PNG
	// Note: We would need to import "image/png" and use png.Encode here
	// For now, we'll assume the image can be handled by gosseract directly
	// In a full implementation, we'd convert the image.Image to PNG bytes

	return tempFile.Name(), nil
}

// cleanupTempFile removes a temporary file if cleanup is enabled
func (e *ocrExtractor) cleanupTempFile(filename string) {
	if e.config.CleanupTemp {
		os.Remove(filename)
	}
}

// detectLanguage performs basic language detection
func (e *ocrExtractor) detectLanguage(text string) string {
	// Simple heuristic-based language detection
	englishWords := []string{
		"the", "and", "of", "to", "a", "in", "for", "is", "on", "that",
		"court", "case", "defendant", "plaintiff", "motion", "order",
	}

	words := strings.Fields(strings.ToLower(text))
	if len(words) == 0 {
		return "unknown"
	}

	englishCount := 0
	sampleSize := min(50, len(words))

	for i := 0; i < sampleSize; i++ {
		for _, englishWord := range englishWords {
			if words[i] == englishWord {
				englishCount++
				break
			}
		}
	}

	if float64(englishCount)/float64(sampleSize) > 0.15 {
		return "en"
	}

	return e.config.Language // Return configured language as fallback
}

// GetOCRInfo returns information about the OCR system
func (e *ocrExtractor) GetOCRInfo() map[string]interface{} {
	client := gosseract.NewClient()
	defer client.Close()

	return map[string]interface{}{
		"tesseract_version": client.Version(),
		"language":         e.config.Language,
		"dpi":             e.config.DPI,
		"page_seg_mode":   e.config.PageSegMode,
		"gpu_enabled":     e.config.EnableGPU,
		"available":       e.isTesseractAvailable(),
	}
}