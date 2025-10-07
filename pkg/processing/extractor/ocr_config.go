//go:build enhanced
// +build enhanced

package extractor

import (
	"fmt"
	"time"
)

// OCRConfig holds configuration for OCR operations
type OCRConfig struct {
	// Basic OCR settings
	Language            string  // Default: "eng"
	DPI                 int     // Default: 300
	PageSegMode         int     // Default: 1 (automatic page segmentation)
	ConfidenceThreshold float64 // Default: 0.0 (accept all)
	
	// Performance settings
	EnableGPU          bool // Default: false (CPU only)
	MaxConcurrentPages int  // Default: 4
	ProcessingTimeout  time.Duration // Default: 5 minutes
	
	// Quality settings
	PreprocessImages   bool // Default: true
	EnhanceContrast    bool // Default: true
	CleanupTemp        bool // Default: true
	
	// Advanced settings
	CustomConfigPath   string // Optional custom Tesseract config
	WhitelistChars     string // Optional character whitelist
	BlacklistChars     string // Optional character blacklist
}

// DefaultOCRConfig returns sensible defaults for OCR configuration
func DefaultOCRConfig() *OCRConfig {
	return &OCRConfig{
		Language:            "eng",
		DPI:                 300,
		PageSegMode:         1,
		ConfidenceThreshold: 0.0,
		EnableGPU:          false,
		MaxConcurrentPages:  4,
		ProcessingTimeout:  5 * time.Minute,
		PreprocessImages:   true,
		EnhanceContrast:    true,
		CleanupTemp:        true,
	}
}

// Validate checks if the OCR configuration is valid
func (c *OCRConfig) Validate() error {
	if c.DPI < 72 || c.DPI > 600 {
		c.DPI = 300 // Reset to default
	}
	
	if c.PageSegMode < 0 || c.PageSegMode > 13 {
		c.PageSegMode = 1 // Reset to default
	}
	
	if c.MaxConcurrentPages < 1 {
		c.MaxConcurrentPages = 1
	}
	
	if c.MaxConcurrentPages > 16 {
		c.MaxConcurrentPages = 16 // Reasonable upper limit
	}
	
	if c.ProcessingTimeout <= 0 {
		c.ProcessingTimeout = 5 * time.Minute
	}
	
	return nil
}

// GetTesseractParams returns Tesseract parameters based on configuration
func (c *OCRConfig) GetTesseractParams() map[string]string {
	params := make(map[string]string)
	
	// Set page segmentation mode
	params["psm"] = fmt.Sprintf("%d", c.PageSegMode)
	
	// Set OCR engine mode (3 = default, based on what is available)
	params["oem"] = "3"
	
	// Custom config file if specified
	if c.CustomConfigPath != "" {
		params["config"] = c.CustomConfigPath
	}
	
	// Character restrictions
	if c.WhitelistChars != "" {
		params["whitelist"] = c.WhitelistChars
	}
	
	if c.BlacklistChars != "" {
		params["blacklist"] = c.BlacklistChars
	}
	
	return params
}