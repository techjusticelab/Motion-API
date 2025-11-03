package presenter

import (
	"time"

	"motion-index-fiber/internal/application/dto"
)

// DocumentResponse represents a processed document in the HTTP response
type DocumentResponse struct {
	ID           string    `json:"id"`
	Stored       bool      `json:"stored"`
	StorageURL   string    `json:"storage_url,omitempty"`
	Classified   bool      `json:"classified"`
	Indexed      bool      `json:"indexed"`
	CreatedAt    string    `json:"created_at"`
	ProcessedAt  string    `json:"processed_at,omitempty"`
	EventsQueued int       `json:"events_queued,omitempty"`
}

// BatchDocumentResponse represents a batch processing result in the HTTP response
type BatchDocumentResponse struct {
	Total     int               `json:"total"`
	Succeeded int               `json:"succeeded"`
	Failed    int               `json:"failed"`
	Errors    map[string]string `json:"errors,omitempty"`
}

// IndexResponse represents an indexing result in the HTTP response
type IndexResponse struct {
	DocumentID string `json:"document_id"`
	Indexed    bool   `json:"indexed"`
	Message    string `json:"message,omitempty"`
}

// MetadataResponse represents updated metadata in the HTTP response
type MetadataResponse struct {
	DocumentID   string    `json:"document_id"`
	Language     string    `json:"language"`
	LegalTags    []string  `json:"legal_tags"`
	AIClassified bool      `json:"ai_classified"`
	ProcessedAt  string    `json:"processed_at,omitempty"`
}

// PresentDocument converts ProcessDocumentResponse DTO to presentation format
func PresentDocument(dto *dto.ProcessDocumentResponse) *DocumentResponse {
	if dto == nil {
		return nil
	}

	resp := &DocumentResponse{
		ID:           dto.DocumentID,
		Stored:       dto.Stored,
		StorageURL:   dto.StorageURL,
		Classified:   dto.Classified,
		Indexed:      dto.Indexed,
		CreatedAt:    dto.CreatedAt.Format(time.RFC3339),
		EventsQueued: dto.EventsQueued,
	}

	if dto.ProcessedAt != nil {
		resp.ProcessedAt = dto.ProcessedAt.Format(time.RFC3339)
	}

	return resp
}

// PresentBatchDocument converts BatchProcessDocumentResponse DTO to presentation format
func PresentBatchDocument(dto *dto.BatchProcessDocumentResponse) *BatchDocumentResponse {
	if dto == nil {
		return nil
	}

	return &BatchDocumentResponse{
		Total:     dto.Total,
		Succeeded: dto.Succeeded,
		Failed:    dto.Failed,
		Errors:    dto.Errors,
	}
}

// PresentIndex converts IndexDocumentResponse DTO to presentation format
func PresentIndex(dto *dto.IndexDocumentResponse) *IndexResponse {
	if dto == nil {
		return nil
	}

	resp := &IndexResponse{
		DocumentID: dto.DocumentID,
		Indexed:    dto.Indexed,
	}

	if dto.Indexed {
		resp.Message = "Document indexed successfully"
	} else {
		resp.Message = "Document was not indexed"
	}

	return resp
}

// PresentMetadata converts UpdateMetadataResponse DTO to presentation format
func PresentMetadata(dto *dto.UpdateMetadataResponse) *MetadataResponse {
	if dto == nil {
		return nil
	}

	resp := &MetadataResponse{
		DocumentID:   dto.DocumentID,
		Language:     dto.Language,
		LegalTags:    dto.LegalTags,
		AIClassified: dto.AIClassified,
	}

	if dto.ProcessedAt != nil {
		resp.ProcessedAt = dto.ProcessedAt.Format(time.RFC3339)
	}

	return resp
}
