package models

import (
	"time"

	"motion-index-fiber/pkg/processing/classifier"
)

// IndexDocumentRequest represents a request to index a single document
type IndexDocumentRequest struct {
	DocumentID           string                            `json:"document_id" validate:"required"`
	DocumentPath         string                            `json:"document_path,omitempty"`
	Text                 string                            `json:"text" validate:"required"`
	ClassificationResult *classifier.ClassificationResult `json:"classification_result" validate:"required"`
	FileName             string                            `json:"file_name,omitempty"`
	ContentType          string                            `json:"content_type,omitempty"`
	Size                 int64                             `json:"size,omitempty"`
	FileURL              string                            `json:"file_url,omitempty"`
}

// IndexDocumentResponse represents the response from indexing a document
type IndexDocumentResponse struct {
	DocumentID  string    `json:"document_id"`
	IndexID     string    `json:"index_id"`
	Success     bool      `json:"success"`
	Message     string    `json:"message,omitempty"`
	Error       string    `json:"error,omitempty"`
	IndexedAt   time.Time `json:"indexed_at"`
}