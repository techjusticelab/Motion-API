//go:build !enhanced
// +build !enhanced

package extractor

import (
	"context"
	"io"
)

// ServiceType represents the type of extraction service (stub)
type ServiceType string

const (
	ServiceTypeBasic ServiceType = "basic"
	ServiceTypeAuto  ServiceType = "basic" // Maps to basic when enhanced not available
)

// ServiceConfig holds basic configuration when enhanced features are not available
type ServiceConfig struct {
	Type ServiceType
}

// EnhancedConfig stub for when enhanced features are not available
type EnhancedConfig struct{}

// OCRConfig stub for when OCR features are not available
type OCRConfig struct{}

// DefaultEnhancedConfig returns nil when enhanced features are not available
func DefaultEnhancedConfig() *EnhancedConfig {
	return &EnhancedConfig{}
}

// DefaultOCRConfig returns stub when OCR features are not available
func DefaultOCRConfig() *OCRConfig {
	return &OCRConfig{}
}

// NewExtractionService creates a basic service when enhanced features are not available
func NewExtractionService(config *ServiceConfig) Service {
	return NewService() // Always return basic service
}

// NewEnhancedService returns basic service when enhanced features are not available
func NewEnhancedService(config *EnhancedConfig) Service {
	return NewService()
}

// CreateProductionService creates a basic service for production
func CreateProductionService() Service {
	return NewService()
}

// CreateTestService creates a basic service for testing
func CreateTestService() Service {
	return NewService()
}

// GetRecommendedServiceType always returns basic when enhanced features are not available
func GetRecommendedServiceType() ServiceType {
	return ServiceTypeBasic
}

// GetAvailableCapabilities returns basic capabilities only
func GetAvailableCapabilities() map[string]interface{} {
	return map[string]interface{}{
		"basic_text":  true,
		"basic_pdf":   true,
		"docx":        true,
		"dslipak_pdf": false,
		"ocr":         false,
		"intelligent": false,
		"recommended": ServiceTypeBasic,
		"enhanced":    false,
	}
}

// Stub types and functions for missing enhanced components

// DocumentType stub
type DocumentType int

const (
	DocumentTypeUnknown DocumentType = iota
)

// DocumentAnalysis stub
type DocumentAnalysis struct{}

// DocumentAnalyzer stub
type DocumentAnalyzer struct{}

func NewDocumentAnalyzer() *DocumentAnalyzer {
	return &DocumentAnalyzer{}
}

func (a *DocumentAnalyzer) AnalyzeDocument(ctx context.Context, reader io.Reader, metadata *DocumentMetadata) (*DocumentAnalysis, error) {
	return &DocumentAnalysis{}, nil
}

// NewDslipakPDFExtractor stub
func NewDslipakPDFExtractor() Extractor {
	return &stubExtractor{name: "dslipak-pdf"}
}

// NewOCRExtractor stub
func NewOCRExtractor(config *OCRConfig) Extractor {
	return &stubExtractor{name: "ocr"}
}

// stubExtractor provides a placeholder for unavailable extractors
type stubExtractor struct {
	name string
}

func (e *stubExtractor) Extract(ctx context.Context, reader io.Reader, metadata *DocumentMetadata) (*ExtractionResult, error) {
	return nil, NewExtractionError(e.name, "enhanced features not available - rebuild with enhanced tag", nil)
}

func (e *stubExtractor) SupportedFormats() []string {
	return []string{}
}

func (e *stubExtractor) CanExtract(format string) bool {
	return false
}