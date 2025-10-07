//go:build enhanced
// +build enhanced

package extractor

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"path/filepath"
	"strings"
	"time"
)

// enhancedService implements intelligent document analysis and cascading extraction
// Following UNIX philosophy: compose simple, focused tools into a powerful system
type enhancedService struct {
	// Core extractors - each does one thing well
	textExtractor    Extractor
	pdfExtractor     Extractor
	dslipakExtractor Extractor
	ocrExtractor     Extractor
	docxExtractor    Extractor
	
	// Document analyzer - single responsibility
	analyzer *DocumentAnalyzer
	
	// Configuration
	config *EnhancedConfig
}

// EnhancedConfig holds configuration for the enhanced extraction service
type EnhancedConfig struct {
	// Extraction preferences
	EnableOCR                bool    // Default: true
	EnableDslipakPDF        bool    // Default: true
	MinTextThreshold        int     // Minimum chars to consider extraction successful
	OCRConfidenceThreshold  float64 // Minimum OCR confidence
	
	// Performance settings
	MaxRetries              int           // Default: 3
	ExtractionTimeout       time.Duration // Default: 5 minutes
	EnableFallbacks         bool          // Default: true
	
	// OCR settings
	OCRConfig *OCRConfig
}

// DefaultEnhancedConfig returns sensible defaults
func DefaultEnhancedConfig() *EnhancedConfig {
	return &EnhancedConfig{
		EnableOCR:               true,
		EnableDslipakPDF:       true,
		MinTextThreshold:       10,
		OCRConfidenceThreshold: 0.0,
		MaxRetries:             3,
		ExtractionTimeout:      5 * time.Minute,
		EnableFallbacks:        true,
		OCRConfig:              DefaultOCRConfig(),
	}
}

// NewEnhancedService creates a new enhanced extraction service
func NewEnhancedService(config *EnhancedConfig) Service {
	if config == nil {
		config = DefaultEnhancedConfig()
	}

	service := &enhancedService{
		analyzer: NewDocumentAnalyzer(),
		config:   config,
	}

	// Initialize extractors following UNIX philosophy - single purpose tools
	service.initializeExtractors()

	return service
}

// initializeExtractors sets up all available extractors
func (s *enhancedService) initializeExtractors() {
	// Core text extractors
	s.textExtractor = NewTextExtractor()
	s.pdfExtractor = NewPDFExtractor() // Original ledongthuc/pdf
	s.docxExtractor = NewDOCXExtractor()
	
	// Enhanced extractors
	if s.config.EnableDslipakPDF {
		s.dslipakExtractor = NewDslipakPDFExtractor()
	}
	
	if s.config.EnableOCR {
		s.ocrExtractor = NewOCRExtractor(s.config.OCRConfig)
	}
}

// ExtractText implements intelligent extraction with cascading fallbacks
func (s *enhancedService) ExtractText(ctx context.Context, reader io.Reader, metadata *DocumentMetadata) (*ExtractionResult, error) {
	startTime := time.Now()
	
	// Add timeout to context
	ctx, cancel := context.WithTimeout(ctx, s.config.ExtractionTimeout)
	defer cancel()

	// Read content once and reuse (UNIX: avoid repeated work)
	content, err := io.ReadAll(reader)
	if err != nil {
		return s.createErrorResult(startTime, "failed to read content", err), err
	}

	// Determine format if not provided
	if metadata.Format == "" {
		metadata.Format = s.detectFormat(metadata.FileName, metadata.MimeType)
	}

	// Analyze document to determine optimal strategy
	analysis, err := s.analyzer.AnalyzeDocument(ctx, bytes.NewReader(content), metadata)
	if err != nil {
		// If analysis fails, use simple format-based extraction
		return s.extractWithSimpleStrategy(ctx, content, metadata, startTime)
	}

	// Use intelligent extraction based on analysis
	return s.extractWithIntelligentStrategy(ctx, content, metadata, analysis, startTime)
}

// extractWithIntelligentStrategy uses document analysis to choose the best extraction approach
func (s *enhancedService) extractWithIntelligentStrategy(ctx context.Context, content []byte, metadata *DocumentMetadata, analysis *DocumentAnalysis, startTime time.Time) (*ExtractionResult, error) {
	var result *ExtractionResult
	var err error
	
	// Try recommended method first
	switch analysis.RecommendedMethod {
	case "text":
		result, err = s.tryTextExtraction(ctx, content, metadata, analysis)
	case "ocr":
		result, err = s.tryOCRExtraction(ctx, content, metadata)
	default:
		result, err = s.tryTextExtraction(ctx, content, metadata, analysis)
	}
	
	// If primary method succeeded with sufficient text, return it
	if err == nil && result != nil && s.isExtractionSuccessful(result) {
		result.Duration = time.Since(startTime).Milliseconds()
		result.Metadata["analysis"] = analysis.GetRecommendedStrategy()
		return result, nil
	}
	
	// Try fallback methods if enabled
	if s.config.EnableFallbacks {
		for _, fallbackMethod := range analysis.Fallbacks {
			switch fallbackMethod {
			case "dslipak":
				if s.config.EnableDslipakPDF && s.dslipakExtractor != nil {
					result, err = s.tryDslipakExtraction(ctx, content, metadata)
				}
			case "ocr":
				if s.config.EnableOCR && s.ocrExtractor != nil {
					result, err = s.tryOCRExtraction(ctx, content, metadata)
				}
			case "text":
				result, err = s.tryOriginalPDFExtraction(ctx, content, metadata)
			}
			
			// If fallback succeeded, return it
			if err == nil && result != nil && s.isExtractionSuccessful(result) {
				result.Duration = time.Since(startTime).Milliseconds()
				result.Metadata["analysis"] = analysis.GetRecommendedStrategy()
				result.Metadata["used_fallback"] = fallbackMethod
				return result, nil
			}
		}
	}
	
	// If all methods failed, return the best result we got or an error
	if result != nil {
		result.Duration = time.Since(startTime).Milliseconds()
		result.Metadata["analysis"] = analysis.GetRecommendedStrategy()
		result.Metadata["extraction_incomplete"] = true
		return result, nil
	}
	
	return s.createErrorResult(startTime, "all extraction methods failed", err), err
}

// extractWithSimpleStrategy falls back to original logic when analysis fails
func (s *enhancedService) extractWithSimpleStrategy(ctx context.Context, content []byte, metadata *DocumentMetadata, startTime time.Time) (*ExtractionResult, error) {
	// Use original service logic as fallback
	extractor, err := s.getExtractorForFormat(metadata.Format)
	if err != nil {
		return s.createErrorResult(startTime, "no extractor available", err), err
	}
	
	result, err := extractor.Extract(ctx, bytes.NewReader(content), metadata)
	if err != nil {
		return s.createErrorResult(startTime, "extraction failed", err), err
	}
	
	result.Duration = time.Since(startTime).Milliseconds()
	result.Metadata["strategy"] = "simple"
	return result, nil
}

// tryTextExtraction attempts text-based extraction with PDF-specific logic
func (s *enhancedService) tryTextExtraction(ctx context.Context, content []byte, metadata *DocumentMetadata, analysis *DocumentAnalysis) (*ExtractionResult, error) {
	if metadata.Format == "pdf" {
		// For PDFs, try dslipak first if available, then original
		if s.config.EnableDslipakPDF && s.dslipakExtractor != nil {
			result, err := s.tryDslipakExtraction(ctx, content, metadata)
			if err == nil && s.isExtractionSuccessful(result) {
				return result, nil
			}
		}
		
		// Fallback to original PDF extractor
		return s.tryOriginalPDFExtraction(ctx, content, metadata)
	}
	
	// For other formats, use appropriate extractor
	extractor, err := s.getExtractorForFormat(metadata.Format)
	if err != nil {
		return nil, err
	}
	
	return extractor.Extract(ctx, bytes.NewReader(content), metadata)
}

// tryDslipakExtraction attempts extraction with dslipak/pdf library
func (s *enhancedService) tryDslipakExtraction(ctx context.Context, content []byte, metadata *DocumentMetadata) (*ExtractionResult, error) {
	if s.dslipakExtractor == nil {
		return nil, fmt.Errorf("dslipak extractor not available")
	}
	
	return s.dslipakExtractor.Extract(ctx, bytes.NewReader(content), metadata)
}

// tryOriginalPDFExtraction attempts extraction with original ledongthuc/pdf library
func (s *enhancedService) tryOriginalPDFExtraction(ctx context.Context, content []byte, metadata *DocumentMetadata) (*ExtractionResult, error) {
	return s.pdfExtractor.Extract(ctx, bytes.NewReader(content), metadata)
}

// tryOCRExtraction attempts OCR-based extraction
func (s *enhancedService) tryOCRExtraction(ctx context.Context, content []byte, metadata *DocumentMetadata) (*ExtractionResult, error) {
	if s.ocrExtractor == nil {
		return nil, fmt.Errorf("OCR extractor not available")
	}
	
	return s.ocrExtractor.Extract(ctx, bytes.NewReader(content), metadata)
}

// isExtractionSuccessful determines if an extraction result is considered successful
func (s *enhancedService) isExtractionSuccessful(result *ExtractionResult) bool {
	if result == nil || !result.Success {
		return false
	}
	
	// Check minimum text threshold
	if len(strings.TrimSpace(result.Text)) < s.config.MinTextThreshold {
		return false
	}
	
	// Additional quality checks could be added here
	return true
}

// getExtractorForFormat returns the appropriate extractor for a format (original logic)
func (s *enhancedService) getExtractorForFormat(format string) (Extractor, error) {
	format = strings.ToLower(format)
	
	switch format {
	case "pdf":
		return s.pdfExtractor, nil
	case "txt", "text":
		return s.textExtractor, nil
	case "docx":
		return s.docxExtractor, nil
	default:
		return nil, fmt.Errorf("no extractor available for format: %s", format)
	}
}

// createErrorResult creates a standardized error result
func (s *enhancedService) createErrorResult(startTime time.Time, message string, err error) *ExtractionResult {
	errorMsg := message
	if err != nil {
		errorMsg = fmt.Sprintf("%s: %v", message, err)
	}
	
	return &ExtractionResult{
		Success:  false,
		Error:    errorMsg,
		Duration: time.Since(startTime).Milliseconds(),
		Metadata: map[string]interface{}{
			"service": "enhanced",
		},
	}
}

// GetExtractor returns an extractor for the given format (compatibility)
func (s *enhancedService) GetExtractor(format string) (Extractor, error) {
	return s.getExtractorForFormat(format)
}

// SupportedFormats returns all supported formats
func (s *enhancedService) SupportedFormats() []string {
	formats := []string{"pdf", "txt", "docx"}
	
	if s.config.EnableOCR && s.ocrExtractor != nil {
		formats = append(formats, "png", "jpg", "jpeg", "tiff", "bmp", "gif")
	}
	
	return formats
}

// detectFormat attempts to detect the file format from filename and mime type
func (s *enhancedService) detectFormat(fileName, mimeType string) string {
	// Try to detect from file extension first
	if fileName != "" {
		ext := strings.ToLower(filepath.Ext(fileName))
		if ext != "" {
			ext = strings.TrimPrefix(ext, ".")
			// Check if we support this format
			for _, supported := range s.SupportedFormats() {
				if ext == supported {
					return ext
				}
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
		
		// Check for image MIME types if OCR is enabled
		if s.config.EnableOCR && strings.HasPrefix(mimeType, "image/") {
			imagePart := strings.TrimPrefix(mimeType, "image/")
			switch imagePart {
			case "png", "jpeg", "jpg", "tiff", "bmp", "gif":
				return imagePart
			}
		}
	}

	// Default fallback
	return "txt"
}

// GetSystemInfo returns information about available extractors and their capabilities
func (s *enhancedService) GetSystemInfo() map[string]interface{} {
	info := map[string]interface{}{
		"service_type":        "enhanced",
		"supported_formats":   s.SupportedFormats(),
		"dslipak_enabled":     s.config.EnableDslipakPDF,
		"ocr_enabled":         s.config.EnableOCR,
		"fallbacks_enabled":   s.config.EnableFallbacks,
		"min_text_threshold":  s.config.MinTextThreshold,
		"extraction_timeout":  s.config.ExtractionTimeout.String(),
	}
	
	// Add OCR info if available
	if s.config.EnableOCR && s.ocrExtractor != nil {
		if ocrExt, ok := s.ocrExtractor.(*ocrExtractor); ok {
			info["ocr_info"] = ocrExt.GetOCRInfo()
		}
	}
	
	return info
}

// RegisterExtractor allows registering custom extractors (compatibility)
func (s *enhancedService) RegisterExtractor(format string, extractor Extractor) {
	// For enhanced service, we could extend this to integrate with our intelligent routing
	log.Printf("Custom extractor registration not fully implemented in enhanced service for format: %s", format)
}

// Ensure enhancedService implements the Service interface
var _ Service = (*enhancedService)(nil)