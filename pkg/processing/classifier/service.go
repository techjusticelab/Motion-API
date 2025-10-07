package classifier

import (
	"context"
	"fmt"
	"time"
)

// service implements the Service interface
type service struct {
	classifier Classifier
	config     *Config
}

// Config holds configuration for the classification service
type Config struct {
	Provider   string        `json:"provider"` // "openai", "claude", "ollama", "fallback", "mock", etc.
	APIKey     string        `json:"api_key"`
	Model      string        `json:"model"`
	MaxRetries int           `json:"max_retries"`
	Timeout    time.Duration `json:"timeout"`
}

// ClaudeConfig holds configuration for Claude API
type ClaudeConfig struct {
	APIKey  string `json:"api_key"`
	Model   string `json:"model"`
	BaseURL string `json:"base_url"`
}

// OllamaConfig holds configuration for Ollama local models
type OllamaConfig struct {
	BaseURL string        `json:"base_url"`
	Model   string        `json:"model"`
	Timeout time.Duration `json:"timeout"`
}

// FallbackConfig holds configuration for fallback classification service
type FallbackConfig struct {
	OpenAI         *Config       `json:"openai"`
	Claude         *ClaudeConfig `json:"claude"`
	Ollama         *OllamaConfig `json:"ollama"`
	EnableFallback bool          `json:"enable_fallback"`
	RetryAttempts  int           `json:"retry_attempts"`
	RetryDelay     time.Duration `json:"retry_delay"`
}

// NewService creates a new classification service
func NewService(config *Config) (Service, error) {
	if config == nil {
		return nil, fmt.Errorf("configuration is required")
	}

	// Create classifier based on provider
	var classifier Classifier
	var err error

	switch config.Provider {
	case "openai":
		classifier, err = NewOpenAIClassifier(config)
	case "claude":
		claudeConfig := &ClaudeConfig{
			APIKey:  config.APIKey,
			Model:   config.Model,
			BaseURL: "https://api.anthropic.com",
		}
		classifier, err = NewClaudeClassifier(claudeConfig)
	case "ollama":
		ollamaConfig := &OllamaConfig{
			BaseURL: "http://localhost:11434",
			Model:   config.Model,
			Timeout: config.Timeout,
		}
		classifier, err = NewOllamaClassifier(ollamaConfig)
	case "mock":
		classifier = NewMockClassifier()
	default:
		return nil, fmt.Errorf("unsupported classification provider: %s", config.Provider)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create classifier: %w", err)
	}

	return &service{
		classifier: classifier,
		config:     config,
	}, nil
}

// ClassifyDocument classifies a document and returns the results
func (s *service) ClassifyDocument(ctx context.Context, text string, metadata *DocumentMetadata) (*ClassificationResult, error) {
	startTime := time.Now()

	// Handle empty text gracefully - create default classification
	if text == "" {
		return &ClassificationResult{
			DocumentType:   DocumentTypeOther,
			LegalCategory:  LegalCategoryCivil,
			Confidence:     0.1, // Low confidence for empty text
			Keywords:       []string{"unextracted", "empty"},
			Summary:        "Document text could not be extracted (may be scanned image or corrupted)",
			Success:        true,
			ProcessingTime: time.Since(startTime).Milliseconds(),
		}, nil
	}

	// Classify using the configured classifier
	result, err := s.classifier.Classify(ctx, text, metadata)
	if err != nil {
		return &ClassificationResult{
			Success:        false,
			Error:          err.Error(),
			ProcessingTime: time.Since(startTime).Milliseconds(),
		}, err
	}

	// Set processing time and success flag
	result.ProcessingTime = time.Since(startTime).Milliseconds()
	result.Success = true

	// Validate result
	if err := s.validateResult(result); err != nil {
		result.Success = false
		result.Error = err.Error()
		return result, err
	}

	return result, nil
}

// GetAvailableCategories returns all available classification categories
func (s *service) GetAvailableCategories() []string {
	if s.classifier != nil {
		return s.classifier.GetSupportedCategories()
	}
	return GetDefaultCategories()
}

// IsHealthy returns true if the classification service is healthy
func (s *service) IsHealthy() bool {
	if s.classifier == nil {
		return false
	}

	// Check if classifier is configured
	return s.classifier.IsConfigured()
}

// validateResult performs basic validation on classification results
func (s *service) validateResult(result *ClassificationResult) error {
	if result == nil {
		return NewClassificationError("validation", "nil result returned from classifier", nil)
	}

	// Validate confidence score
	if result.Confidence < 0 || result.Confidence > 1 {
		return NewClassificationError("validation", "confidence score must be between 0 and 1", nil)
	}

	// Validate document type
	if result.DocumentType == "" {
		return NewClassificationError("validation", "document type is required", nil)
	}

	// Validate legal category
	if result.LegalCategory == "" {
		return NewClassificationError("validation", "legal category is required", nil)
	}

	return nil
}

// SetClassifier allows changing the classifier implementation
func (s *service) SetClassifier(classifier Classifier) {
	s.classifier = classifier
}

// GetConfig returns the service configuration
func (s *service) GetConfig() *Config {
	return s.config
}

// ValidateResult exposes result validation for testing
func (s *service) ValidateResult(result *ClassificationResult) error {
	return s.validateResult(result)
}
