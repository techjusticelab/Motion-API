package dto

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"motion-index-fiber/internal/application/validation"
	"motion-index-fiber/internal/domain/document"
)

// ProcessDocumentRequest captures the inputs required to ingest a new document.
type ProcessDocumentRequest struct {
	ID            string
	FileName      string
	Content       io.Reader
	ContentSize   int64
	ContentType   string
	StoragePath   string
	Text          string
	HashValue     string
	HashAlgorithm string
	StoreBinary   bool
	Classify      bool
	Index         bool
	Metadata      map[string]string
	ProcessedAt   *time.Time
}

// Validate ensures the request contains the required information.
func (r *ProcessDocumentRequest) Validate() error {
	errs := validation.NewErrorSet()

	if _, err := document.NewDocumentID(r.ID); err != nil {
		errs.Append("id", err)
	}

	if err := validation.ValidateFileName(r.FileName); err != nil {
		errs.Append("fileName", err)
	}

	if r.StoreBinary {
		if r.Content == nil {
			errs.Append("content", fmt.Errorf("content reader is required when StoreBinary is true"))
		}
		if r.ContentSize <= 0 {
			errs.Append("contentSize", fmt.Errorf("content size must be greater than zero"))
		}
		if err := validation.ValidateStoragePath(r.StoragePath); err != nil {
			errs.Append("storagePath", err)
		}
	}

	if _, err := document.NewContentType(r.ContentType); err != nil {
		errs.Append("contentType", err)
	}
	if _, err := document.NewHash(r.HashValue, r.HashAlgorithm); err != nil {
		errs.Append("hash", err)
	}

	if r.Classify {
		if strings.TrimSpace(r.Text) == "" {
			errs.Append("text", fmt.Errorf("text is required when classification is enabled"))
		}
	}

	return errs.Error()
}

// ProcessDocumentResponse returns the outcome of processing a document.
type ProcessDocumentResponse struct {
	DocumentID   string
	Stored       bool
	StorageURL   string
	Classified   bool
	Indexed      bool
	CreatedAt    time.Time
	ProcessedAt  *time.Time
	EventsQueued int
}

// BatchProcessDocumentRequest represents a request to process multiple documents.
type BatchProcessDocumentRequest struct {
	Documents []*ProcessDocumentRequest
}

// Validate ensures at least one document is included and each is valid.
func (r *BatchProcessDocumentRequest) Validate() error {
	errs := validation.NewErrorSet()

	if len(r.Documents) == 0 {
		errs.Append("documents", fmt.Errorf("at least one document is required"))
	}

	for i, req := range r.Documents {
		if req == nil {
			errs.Append(fmt.Sprintf("documents[%d]", i), fmt.Errorf("request cannot be nil"))
			continue
		}
		if err := req.Validate(); err != nil {
			errs.Append(fmt.Sprintf("documents[%d]", i), err)
		}
	}

	return errs.Error()
}

// BatchProcessDocumentResponse summarises the batch operation.
type BatchProcessDocumentResponse struct {
	Total     int
	Succeeded int
	Failed    int
	Errors    map[string]string
}

// IndexDocumentRequest captures the intent to index an existing document.
type IndexDocumentRequest struct {
	DocumentID string
	Force      bool
}

// Validate ensures the request is well-formed.
func (r *IndexDocumentRequest) Validate() error {
	errs := validation.NewErrorSet()

	if _, err := document.NewDocumentID(r.DocumentID); err != nil {
		errs.Append("documentId", err)
	}

	return errs.Error()
}

// IndexDocumentResponse reports the indexing outcome.
type IndexDocumentResponse struct {
	DocumentID string
	Indexed    bool
}

// UpdateMetadataRequest updates document metadata attributes.
type UpdateMetadataRequest struct {
	DocumentID   string
	Language     string
	LegalTags    []string
	AIClassified bool
	ProcessedAt  *time.Time
}

// Validate ensures metadata updates satisfy minimum requirements.
func (r *UpdateMetadataRequest) Validate() error {
	errs := validation.NewErrorSet()

	if _, err := document.NewDocumentID(r.DocumentID); err != nil {
		errs.Append("documentId", err)
	}

	if strings.TrimSpace(r.Language) == "" {
		errs.Append("language", errors.New("language is required"))
	}

	if r.ProcessedAt != nil && r.ProcessedAt.IsZero() {
		errs.Append("processedAt", errors.New("processedAt cannot be zero"))
	}

	return errs.Error()
}

// UpdateMetadataResponse summarises updated metadata values.
type UpdateMetadataResponse struct {
	DocumentID   string
	Language     string
	LegalTags    []string
	AIClassified bool
	ProcessedAt  *time.Time
}
