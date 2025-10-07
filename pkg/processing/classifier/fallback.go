package classifier

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"
)

// fallbackClassifier implements a fallback classification service that tries multiple providers
type fallbackClassifier struct {
	openai         Classifier
	claude         Classifier
	ollama         Classifier
	enableFallback bool
	retryAttempts  int
	retryDelay     time.Duration
}

// NewFallbackClassifier creates a new fallback classification service
func NewFallbackClassifier(config *FallbackConfig) (Classifier, error) {
	if config == nil {
		return nil, fmt.Errorf("fallback configuration is required")
	}

	fc := &fallbackClassifier{
		enableFallback: config.EnableFallback,
		retryAttempts:  config.RetryAttempts,
		retryDelay:     config.RetryDelay,
	}

	// Initialize OpenAI classifier (primary)
	if config.OpenAI != nil && config.OpenAI.APIKey != "" {
		openaiClassifier, err := NewOpenAIClassifier(config.OpenAI)
		if err != nil {
			log.Printf("[FALLBACK] Warning: Failed to initialize OpenAI classifier: %v", err)
		} else {
			fc.openai = openaiClassifier
			log.Printf("[FALLBACK] ✅ OpenAI classifier initialized")
		}
	}

	// Initialize Claude classifier (first fallback)
	if config.Claude != nil && config.Claude.APIKey != "" {
		claudeClassifier, err := NewClaudeClassifier(config.Claude)
		if err != nil {
			log.Printf("[FALLBACK] Warning: Failed to initialize Claude classifier: %v", err)
		} else {
			fc.claude = claudeClassifier
			log.Printf("[FALLBACK] ✅ Claude classifier initialized")
		}
	}

	// Initialize Ollama classifier (local fallback)
	if config.Ollama != nil && config.Ollama.BaseURL != "" {
		ollamaClassifier, err := NewOllamaClassifier(config.Ollama)
		if err != nil {
			log.Printf("[FALLBACK] Warning: Failed to initialize Ollama classifier: %v", err)
		} else {
			fc.ollama = ollamaClassifier
			log.Printf("[FALLBACK] ✅ Ollama classifier initialized")
		}
	}

	// Ensure at least one classifier is available
	if fc.openai == nil && fc.claude == nil && fc.ollama == nil {
		return nil, fmt.Errorf("no classifiers could be initialized")
	}

	return fc, nil
}

// Classify attempts to classify using fallback strategy: Ollama → Claude → OpenAI (cost-effective order)
func (fc *fallbackClassifier) Classify(ctx context.Context, text string, metadata *DocumentMetadata) (*ClassificationResult, error) {
	var lastErr error
	
	// Try Ollama first (cost-effective local model)
	if fc.ollama != nil {
		log.Printf("[FALLBACK] 🔄 Attempting classification with Ollama (cost-effective local model)")
		result, err := fc.ollama.Classify(ctx, text, metadata)
		if err == nil {
			log.Printf("[FALLBACK] ✅ Ollama classification successful")
			return result, nil
		}
		
		lastErr = err
		errorType := fc.categorizeError(err)
		log.Printf("[FALLBACK] ❌ Ollama classification failed: %s - %v", errorType, err)
		
		// If not enabled for fallback, return the error
		if !fc.enableFallback {
			return nil, err
		}
		
		// Don't fallback on certain error types (auth, validation, etc.)
		if !fc.shouldFallback(err) {
			log.Printf("[FALLBACK] 🚫 Error type %s not suitable for fallback", errorType)
			return nil, err
		}
		
		// Wait before trying next provider
		time.Sleep(fc.retryDelay)
	}

	// Try Claude as second fallback
	if fc.claude != nil {
		log.Printf("[FALLBACK] 🔄 Attempting classification with Claude (second fallback)")
		result, err := fc.claude.Classify(ctx, text, metadata)
		if err == nil {
			log.Printf("[FALLBACK] ✅ Claude classification successful")
			return result, nil
		}
		
		lastErr = err
		errorType := fc.categorizeError(err)
		log.Printf("[FALLBACK] ❌ Claude classification failed: %s - %v", errorType, err)
		
		// Wait before trying next provider
		time.Sleep(fc.retryDelay)
	}

	// Try OpenAI as expensive final fallback
	if fc.openai != nil {
		log.Printf("[FALLBACK] 🔄 Attempting classification with OpenAI (expensive final fallback)")
		result, err := fc.openai.Classify(ctx, text, metadata)
		if err == nil {
			log.Printf("[FALLBACK] ✅ OpenAI classification successful")
			// Mark as expensive fallback in the result
			result.Summary = "[EXPENSIVE FALLBACK] " + result.Summary
			result.Confidence = result.Confidence * 0.9 // Slight confidence reduction for expensive fallback
			return result, nil
		}
		
		lastErr = err
		errorType := fc.categorizeError(err)
		log.Printf("[FALLBACK] ❌ OpenAI classification failed: %s - %v", errorType, err)
	}

	// All providers failed
	log.Printf("[FALLBACK] ❌ All classification providers failed")
	if lastErr != nil {
		return nil, fmt.Errorf("all classification providers failed, last error: %w", lastErr)
	}
	
	return nil, fmt.Errorf("all classification providers failed")
}

// GetSupportedCategories returns the categories supported by any available classifier
func (fc *fallbackClassifier) GetSupportedCategories() []string {
	if fc.openai != nil {
		return fc.openai.GetSupportedCategories()
	}
	if fc.claude != nil {
		return fc.claude.GetSupportedCategories()
	}
	if fc.ollama != nil {
		return fc.ollama.GetSupportedCategories()
	}
	return GetDefaultCategories()
}

// IsConfigured returns true if at least one classifier is configured
func (fc *fallbackClassifier) IsConfigured() bool {
	return (fc.openai != nil && fc.openai.IsConfigured()) ||
		   (fc.claude != nil && fc.claude.IsConfigured()) ||
		   (fc.ollama != nil && fc.ollama.IsConfigured())
}

// categorizeError categorizes errors for better logging and fallback decisions
func (fc *fallbackClassifier) categorizeError(err error) string {
	if err == nil {
		return "unknown"
	}
	
	errStr := strings.ToLower(err.Error())
	
	// Quota and rate limit errors (good candidates for fallback)
	if strings.Contains(errStr, "quota") || strings.Contains(errStr, "billing") {
		return "QUOTA_EXCEEDED"
	}
	if strings.Contains(errStr, "rate limit") || strings.Contains(errStr, "status 429") {
		return "RATE_LIMIT"
	}
	if strings.Contains(errStr, "insufficient_quota") {
		return "INSUFFICIENT_QUOTA"
	}
	
	// Auth errors (not good for fallback)
	if strings.Contains(errStr, "status 401") || strings.Contains(errStr, "unauthorized") {
		return "API_AUTH_ERROR"
	}
	
	// Server errors (good for fallback)
	if strings.Contains(errStr, "status 5") || strings.Contains(errStr, "server error") {
		return "API_SERVER_ERROR"
	}
	
	// Network errors (good for fallback)
	if strings.Contains(errStr, "timeout") || strings.Contains(errStr, "context deadline exceeded") {
		return "TIMEOUT"
	}
	if strings.Contains(errStr, "connection") || strings.Contains(errStr, "network") {
		return "NETWORK_ERROR"
	}
	
	// Bad request errors (not good for fallback)
	if strings.Contains(errStr, "status 400") || strings.Contains(errStr, "bad request") {
		return "API_BAD_REQUEST"
	}
	
	// JSON/parsing errors (could indicate provider-specific issue)
	if strings.Contains(errStr, "json") || strings.Contains(errStr, "unmarshal") {
		return "RESPONSE_PARSE_ERROR"
	}
	
	return "UNKNOWN_ERROR"
}

// shouldFallback determines if an error type is suitable for fallback
func (fc *fallbackClassifier) shouldFallback(err error) bool {
	errorType := fc.categorizeError(err)
	
	// These errors are good candidates for fallback
	switch errorType {
	case "QUOTA_EXCEEDED", "RATE_LIMIT", "INSUFFICIENT_QUOTA":
		return true
	case "API_SERVER_ERROR", "TIMEOUT", "NETWORK_ERROR":
		return true
	case "RESPONSE_PARSE_ERROR": // Provider-specific parsing issue
		return true
	}
	
	// These errors are not good for fallback (will likely fail on other providers too)
	switch errorType {
	case "API_AUTH_ERROR", "API_BAD_REQUEST":
		return false
	}
	
	// For unknown errors, allow fallback
	return true
}

// NewFallbackService creates a new fallback classification service from config
func NewFallbackService(aiConfig interface{}) (Service, error) {
	// This would be called from the handlers with the AI config
	// For now, return a simple implementation
	return nil, fmt.Errorf("NewFallbackService not yet implemented - use NewFallbackClassifier directly")
}