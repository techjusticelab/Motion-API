package classifier

import (
	"context"
)

// ServiceWrapper wraps a Classifier to implement the Service interface
type ServiceWrapper struct {
	Classifier Classifier
}

// ClassifyDocument classifies a document using the wrapped classifier
func (sw *ServiceWrapper) ClassifyDocument(ctx context.Context, text string, metadata *DocumentMetadata) (*ClassificationResult, error) {
	return sw.Classifier.Classify(ctx, text, metadata)
}

// GetAvailableCategories returns all available classification categories
func (sw *ServiceWrapper) GetAvailableCategories() []string {
	if sw.Classifier != nil {
		return sw.Classifier.GetSupportedCategories()
	}
	return GetDefaultCategories()
}

// IsHealthy returns true if the classification service is healthy
func (sw *ServiceWrapper) IsHealthy() bool {
	if sw.Classifier == nil {
		return false
	}
	return sw.Classifier.IsConfigured()
}

// ValidateResult validates a classification result (required by Service interface)
func (sw *ServiceWrapper) ValidateResult(result *ClassificationResult) error {
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