package presenter

import (
	"time"

	"motion-index-fiber/internal/application/dto"
)

// ClassificationResponse represents classification results in the HTTP response
type ClassificationResponse struct {
	DocumentID   string    `json:"document_id"`
	DocumentType string    `json:"document_type,omitempty"`
	Category     string    `json:"category,omitempty"`
	Confidence   float64   `json:"confidence,omitempty"`
	LegalTags    []string  `json:"legal_tags,omitempty"`
	ClassifiedBy string    `json:"classified_by,omitempty"`
	ClassifiedAt time.Time `json:"classified_at,omitempty"`
}

// PresentClassificationResult converts ClassificationResponse DTO to presentation format
func PresentClassificationResult(dto *dto.ClassificationResponse) *ClassificationResponse {
	if dto == nil {
		return nil
	}

	return &ClassificationResponse{
		DocumentID:   dto.DocumentID,
		DocumentType: dto.DocumentType,
		Category:     dto.Category,
		Confidence:   dto.Confidence,
		LegalTags:    dto.LegalTags,
		ClassifiedBy: dto.ClassifiedBy,
		ClassifiedAt: dto.ClassifiedAt,
	}
}
