package ai

import (
	"context"
	"fmt"

	"motion-index-fiber/internal/application/ports"
	"motion-index-fiber/internal/domain/classification"
	"motion-index-fiber/internal/domain/document"
	infraerrors "motion-index-fiber/internal/infrastructure/errors"
	"motion-index-fiber/pkg/processing/classifier"
)

// ClassificationAdapter adapts the existing pkg/processing/classifier to the application port.
type ClassificationAdapter struct {
	classifier classifier.Classifier
	provider   string
}

// NewClassificationAdapter creates a new classification adapter.
func NewClassificationAdapter(c classifier.Classifier, provider string) *ClassificationAdapter {
	return &ClassificationAdapter{
		classifier: c,
		provider:   provider,
	}
}

// Classify implements the ClassificationService port.
func (a *ClassificationAdapter) Classify(ctx context.Context, doc *document.Document) (*classification.Result, error) {
	if !a.classifier.IsConfigured() {
		return nil, fmt.Errorf("classifier %s is not configured", a.provider)
	}

	// Extract text from document
	text := doc.Text().String()
	if text == "" {
		return nil, fmt.Errorf("document has no text to classify")
	}

	// Prepare metadata
	metadata := &classifier.DocumentMetadata{
		FileName:  doc.FileName().String(),
		FileType:  doc.ContentType().String(),
		Size:      doc.Size().Bytes(),
		WordCount: len(text) / 5, // Rough estimate
	}

	// Call underlying classifier
	result, err := a.classifier.Classify(ctx, text, metadata)
	if err != nil {
		return nil, infraerrors.TranslateAIError(err)
	}

	// Convert result to domain classification result
	return convertToDomainResult(result, a.provider)
}

// SupportedDocumentTypes returns the supported document types.
func (a *ClassificationAdapter) SupportedDocumentTypes() []string {
	return a.classifier.GetSupportedCategories()
}

// ProviderName returns the name of the AI provider.
func (a *ClassificationAdapter) ProviderName() string {
	return a.provider
}

// convertToDomainResult converts pkg classifier result to domain classification result.
func convertToDomainResult(pkgResult *classifier.ClassificationResult, provider string) (*classification.Result, error) {
	if pkgResult == nil {
		return nil, fmt.Errorf("classification result is nil")
	}

	// Create document type value object
	docType, err := document.NewDocumentType(pkgResult.DocumentType)
	if err != nil {
		// Fallback to "other" if invalid
		docType, _ = document.NewDocumentType("other")
	}

	// Create category value object
	category, err := document.NewCategory(pkgResult.LegalCategory)
	if err != nil {
		// Fallback to "other" if invalid
		category, _ = document.NewCategory("other")
	}

	// Create confidence value object
	confidence, err := document.NewConfidence(pkgResult.Confidence)
	if err != nil {
		// Fallback to 0.5 if invalid
		confidence, _ = document.NewConfidence(0.5)
	}

	// Create domain result
	result, err := classification.NewResult(
		docType,
		category,
		confidence,
		provider,
		pkgResult.LegalTags,
		pkgResult.Summary,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create classification result: %w", err)
	}

	return result, nil
}

// Ensure ClassificationAdapter implements the port
var _ ports.ClassificationService = (*ClassificationAdapter)(nil)
