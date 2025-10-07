//go:build enhanced && !tesseract
// +build enhanced,!tesseract

package extractor

import (
	"context"
	"io"
)

// ocrExtractor provides a placeholder when enhanced features are enabled but Tesseract is not available
type ocrExtractor struct {
	config *OCRConfig
}

// NewOCRExtractor creates a placeholder OCR extractor when Tesseract is not available
func NewOCRExtractor(config *OCRConfig) Extractor {
	if config == nil {
		config = DefaultOCRConfig()
	}
	return &ocrExtractor{config: config}
}

// Extract always returns an error indicating OCR is not available
func (e *ocrExtractor) Extract(ctx context.Context, reader io.Reader, metadata *DocumentMetadata) (*ExtractionResult, error) {
	return nil, NewExtractionError("ocr", "Tesseract OCR not available - install with: nix-env -iA nixpkgs.tesseract", nil)
}

// SupportedFormats returns empty slice when OCR is not available
func (e *ocrExtractor) SupportedFormats() []string {
	return []string{}
}

// CanExtract always returns false when OCR is not available
func (e *ocrExtractor) CanExtract(format string) bool {
	return false
}

// isTesseractAvailable always returns false in placeholder implementation
func (e *ocrExtractor) isTesseractAvailable() bool {
	return false
}

// GetOCRInfo returns information about OCR unavailability
func (e *ocrExtractor) GetOCRInfo() map[string]interface{} {
	return map[string]interface{}{
		"available":        false,
		"error":           "Tesseract not installed",
		"install_command": "nix-env -iA nixpkgs.tesseract",
		"language":        e.config.Language,
		"dpi":            e.config.DPI,
		"page_seg_mode":   e.config.PageSegMode,
		"enable_gpu":      e.config.EnableGPU,
		"max_concurrent":  e.config.MaxConcurrentPages,
		"processing_timeout": e.config.ProcessingTimeout.String(),
		"preprocess_images": e.config.PreprocessImages,
		"enhance_contrast":  e.config.EnhanceContrast,
		"cleanup_temp":      e.config.CleanupTemp,
		"status":            "tesseract not available - install required",
	}
}