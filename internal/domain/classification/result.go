package classification

import (
	"strings"
	"time"

	"motion-index-fiber/internal/domain/document"
	domainerrors "motion-index-fiber/internal/domain/errors"
)

// Result captures the outcome of a classification request.
type Result struct {
	documentType document.DocumentType
	category     document.Category
	confidence   document.Confidence
	legalTags    []string
	classifiedAt time.Time
	provider     string
	explanation  string
}

// NewResult builds a validated classification result.
func NewResult(
	docType document.DocumentType,
	category document.Category,
	confidence document.Confidence,
	provider string,
	legalTags []string,
	explanation string,
) (*Result, error) {
	if docType.IsZero() {
		return nil, domainerrors.ErrInvalidClassification
	}
	if category.IsZero() {
		return nil, domainerrors.ErrInvalidClassification
	}
	if !confidence.IsValid() {
		return nil, domainerrors.ErrInvalidConfidence
	}

	result := &Result{
		documentType: docType,
		category:     category,
		confidence:   confidence,
		legalTags:    sanitizeTags(legalTags),
		classifiedAt: time.Now(),
		provider:     strings.TrimSpace(provider),
		explanation:  strings.TrimSpace(explanation),
	}
	return result, nil
}

// DocumentType returns the classified document type.
func (r *Result) DocumentType() document.DocumentType {
	return r.documentType
}

// Category returns the classification category.
func (r *Result) Category() document.Category {
	return r.category
}

// Confidence returns the confidence measure.
func (r *Result) Confidence() document.Confidence {
	return r.confidence
}

// LegalTags returns a copy of legal tags.
func (r *Result) LegalTags() []string {
	if len(r.legalTags) == 0 {
		return nil
	}
	copySlice := make([]string, len(r.legalTags))
	copy(copySlice, r.legalTags)
	return copySlice
}

// ClassifiedAt returns when the classification was produced.
func (r *Result) ClassifiedAt() time.Time {
	return r.classifiedAt
}

// Provider returns the classifier identifier.
func (r *Result) Provider() string {
	return r.provider
}

// Explanation returns optional classification details.
func (r *Result) Explanation() string {
	return r.explanation
}

// ToDocumentClassification converts the result into the document aggregate format.
func (r *Result) ToDocumentClassification() (*document.Classification, error) {
	return document.NewClassification(
		r.documentType,
		r.category,
		r.confidence,
		r.provider,
		r.legalTags,
	)
}

func sanitizeTags(tags []string) []string {
	if len(tags) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(tags))
	result := make([]string, 0, len(tags))
	for _, tag := range tags {
		trimmed := strings.TrimSpace(tag)
		if trimmed == "" {
			continue
		}
		key := strings.ToLower(trimmed)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, trimmed)
	}
	if len(result) == 0 {
		return nil
	}
	return result
}
