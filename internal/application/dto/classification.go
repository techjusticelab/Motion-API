package dto

import (
	"fmt"
	"time"

	"motion-index-fiber/internal/application/validation"
	"motion-index-fiber/internal/domain/document"
)

// ClassifyDocumentRequest triggers classification for an existing document.
type ClassifyDocumentRequest struct {
	DocumentID string
	Force      bool
}

// Validate ensures the request is well-formed.
func (r *ClassifyDocumentRequest) Validate() error {
	errs := validation.NewErrorSet()

	if _, err := document.NewDocumentID(r.DocumentID); err != nil {
		errs.Append("documentId", err)
	}

	return errs.Error()
}

// ClassificationResponse captures classification results.
type ClassificationResponse struct {
	DocumentID   string
	DocumentType string
	Category     string
	Confidence   float64
	LegalTags    []string
	ClassifiedBy string
	ClassifiedAt time.Time
}

// UpdateClassificationRequest allows updating classification metadata (e.g. tags).
type UpdateClassificationRequest struct {
	DocumentID string
	LegalTags  []string
}

// Validate ensures the update request is sound.
func (r *UpdateClassificationRequest) Validate() error {
	errs := validation.NewErrorSet()

	if _, err := document.NewDocumentID(r.DocumentID); err != nil {
		errs.Append("documentId", err)
	}

	if len(r.LegalTags) == 0 {
		errs.Append("legalTags", fmt.Errorf("at least one legal tag is required"))
	}

	return errs.Error()
}
