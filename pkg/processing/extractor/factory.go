//go:build enhanced
// +build enhanced

package extractor

import (
	"os"
	"strings"
)

// ServiceType represents the type of extraction service to create
type ServiceType string

const (
	ServiceTypeBasic    ServiceType = "basic"    // Original service with basic extractors
	ServiceTypeEnhanced ServiceType = "enhanced" // Intelligent service with cascading fallbacks
	ServiceTypeAuto     ServiceType = "auto"     // Automatically choose based on available dependencies
)

// ServiceConfig holds configuration for creating extraction services
type ServiceConfig struct {
	Type          ServiceType
	EnableOCR     bool
	EnableDslipak bool
	Enhanced      *EnhancedConfig
}

// NewExtractionService creates a new extraction service based on configuration
// This follows UNIX philosophy: provide simple defaults but allow customization
func NewExtractionService(config *ServiceConfig) Service {
	if config == nil {
		config = &ServiceConfig{
			Type:          ServiceTypeAuto,
			EnableOCR:     true,
			EnableDslipak: true,
		}
	}

	switch config.Type {
	case ServiceTypeBasic:
		return NewService()
	case ServiceTypeEnhanced:
		enhancedConfig := config.Enhanced
		if enhancedConfig == nil {
			enhancedConfig = DefaultEnhancedConfig()
			enhancedConfig.EnableOCR = config.EnableOCR
			enhancedConfig.EnableDslipakPDF = config.EnableDslipak
		}
		return NewEnhancedService(enhancedConfig)
	case ServiceTypeAuto:
		return NewAutoService(config)
	default:
		return NewAutoService(config)
	}
}

// NewAutoService automatically chooses the best service based on available dependencies
func NewAutoService(config *ServiceConfig) Service {
	// Check for advanced dependencies
	hasOCR := isTesseractAvailable()
	hasDslipak := true // Always available if compiled

	// If we have advanced capabilities, use enhanced service
	if hasOCR || hasDslipak {
		enhancedConfig := DefaultEnhancedConfig()
		enhancedConfig.EnableOCR = hasOCR && config.EnableOCR
		enhancedConfig.EnableDslipakPDF = hasDslipak && config.EnableDslipak

		// Adjust settings based on environment
		if isLowResourceEnvironment() {
			enhancedConfig.EnableOCR = false // Disable OCR in constrained environments
			enhancedConfig.MaxRetries = 1
		}

		return NewEnhancedService(enhancedConfig)
	}

	// Fallback to basic service
	return NewService()
}

// isTesseractAvailable checks if Tesseract OCR is available on the system
func isTesseractAvailable() bool {
	// Check common installation paths
	tesseractPaths := []string{
		"/usr/bin/tesseract",
		"/usr/local/bin/tesseract",
		"/opt/homebrew/bin/tesseract", // macOS Homebrew
		"/nix/store/.*/bin/tesseract",  // Nix store (pattern)
	}

	for _, path := range tesseractPaths {
		if strings.Contains(path, "*") {
			// For Nix, we need to check if it's in PATH since store paths vary
			continue
		}
		if _, err := os.Stat(path); err == nil {
			return true
		}
	}

	// Check if tesseract is in PATH (covers Nix and other package managers)
	if path := os.Getenv("PATH"); path != "" {
		paths := strings.Split(path, ":")
		for _, dir := range paths {
			tesseractPath := dir + "/tesseract"
			if _, err := os.Stat(tesseractPath); err == nil {
				return true
			}
		}
	}

	// Basic check - if we can create an OCR extractor without errors, Tesseract is likely available
	// This will be implemented properly when the OCR extractor is complete

	return false
}

// isLowResourceEnvironment checks if we're running in a resource-constrained environment
func isLowResourceEnvironment() bool {
	// Check environment variables that might indicate constraints
	if os.Getenv("MEMORY_LIMIT") != "" || os.Getenv("CPU_LIMIT") != "" {
		return true
	}

	// Check for container environments
	if os.Getenv("KUBERNETES_SERVICE_HOST") != "" || os.Getenv("DOCKER_CONTAINER") != "" {
		return true
	}

	// Could add more sophisticated resource detection here
	return false
}

// GetRecommendedServiceType returns the recommended service type for the current environment
func GetRecommendedServiceType() ServiceType {
	if isTesseractAvailable() {
		return ServiceTypeEnhanced
	}
	return ServiceTypeBasic
}

// GetAvailableCapabilities returns information about what extraction capabilities are available
func GetAvailableCapabilities() map[string]interface{} {
	capabilities := map[string]interface{}{
		"basic_text":    true,
		"basic_pdf":     true,
		"docx":          true,
		"dslipak_pdf":   true,
		"ocr":           isTesseractAvailable(),
		"intelligent":   true,
		"recommended":   GetRecommendedServiceType(),
	}

	// Add system information
	if isTesseractAvailable() {
		capabilities["ocr_available"] = true
	}

	capabilities["environment"] = map[string]interface{}{
		"low_resource": isLowResourceEnvironment(),
		"path":         os.Getenv("PATH"),
	}

	return capabilities
}

// CreateProductionService creates a service configured for production use
func CreateProductionService() Service {
	config := &ServiceConfig{
		Type:          ServiceTypeEnhanced,
		EnableOCR:     true,
		EnableDslipak: true,
		Enhanced: &EnhancedConfig{
			EnableOCR:               isTesseractAvailable(),
			EnableDslipakPDF:       true,
			MinTextThreshold:       10,
			OCRConfidenceThreshold: 0.0,
			MaxRetries:             3,
			ExtractionTimeout:      300000000000, // 5 minutes in nanoseconds
			EnableFallbacks:        true,
			OCRConfig: &OCRConfig{
				Language:            "eng",
				DPI:                 300,
				PageSegMode:         1,
				EnableGPU:          false, // Conservative for production
				ConfidenceThreshold: 0.0,
				MaxConcurrentPages:  4,
				CleanupTemp:        true,
			},
		},
	}

	return NewExtractionService(config)
}

// CreateTestService creates a service configured for testing
func CreateTestService() Service {
	config := &ServiceConfig{
		Type:          ServiceTypeBasic, // Use basic for predictable testing
		EnableOCR:     false,            // Disable OCR for faster tests
		EnableDslipak: true,
	}

	return NewExtractionService(config)
}