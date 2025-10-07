package models

import (
	"mime/multipart"

	"motion-index-fiber/pkg/models"
)

// ProcessDocumentRequest represents a document processing request
type ProcessDocumentRequest struct {
	File        *multipart.FileHeader `form:"file" validate:"required"`
	Category    string                `form:"category" validate:"omitempty,oneof=motion order contract brief memo other"`
	Description string                `form:"description" validate:"omitempty,max=500"`
	CaseName    string                `form:"case_name" validate:"omitempty,max=200"`
	CaseNumber  string                `form:"case_number" validate:"omitempty,max=50"`
	Author      string                `form:"author" validate:"omitempty,max=100"`
	Judge       string                `form:"judge" validate:"omitempty,max=100"`
	Court       string                `form:"court" validate:"omitempty,max=200"`
	LegalTags   []string              `form:"legal_tags" validate:"omitempty,dive,max=50"`
	Options     *ProcessOptions       `json:"options,omitempty"`
}

// ProcessOptions defines processing options
type ProcessOptions struct {
	ExtractText    bool `json:"extract_text" validate:"omitempty"`
	ClassifyDoc    bool `json:"classify_document" validate:"omitempty"`
	IndexDocument  bool `json:"index_document" validate:"omitempty"`
	StoreDocument  bool `json:"store_document" validate:"omitempty"`
	TimeoutSeconds int  `json:"timeout_seconds" validate:"omitempty,min=1,max=300"`
	RetryCount     int  `json:"retry_count" validate:"omitempty,min=0,max=3"`
}

// BatchProcessRequest represents a batch document processing request
type BatchProcessRequest struct {
	Files       []*multipart.FileHeader `form:"files" validate:"required,min=1,max=10"`
	Category    string                  `form:"category" validate:"omitempty,oneof=motion order contract brief memo other"`
	Description string                  `form:"description" validate:"omitempty,max=500"`
	CaseName    string                  `form:"case_name" validate:"omitempty,max=200"`
	CaseNumber  string                  `form:"case_number" validate:"omitempty,max=50"`
	Options     *ProcessOptions         `json:"options,omitempty"`
}

// Re-export SearchRequest from pkg/models for consistency
type SearchDocumentsRequest = models.SearchRequest

// UpdateMetadataRequest represents a request to update document metadata
type UpdateMetadataRequest struct {
	DocumentID string            `json:"document_id" validate:"required"`
	Metadata   map[string]string `json:"metadata" validate:"required"`
	CaseName   string            `json:"case_name" validate:"omitempty,max=200"`
	CaseNumber string            `json:"case_number" validate:"omitempty,max=50"`
	Author     string            `json:"author" validate:"omitempty,max=100"`
	Judge      string            `json:"judge" validate:"omitempty,max=100"`
	Court      string            `json:"court" validate:"omitempty,max=200"`
	LegalTags  []string          `json:"legal_tags" validate:"omitempty,dive,max=50"`
	Status     string            `json:"status" validate:"omitempty,oneof=draft review approved published archived"`
}

// DeleteDocumentRequest represents a request to delete a document
type DeleteDocumentRequest struct {
	DocumentID string `json:"document_id" validate:"required"`
	Reason     string `json:"reason" validate:"omitempty,max=200"`
}

// AnalyzeRedactionsRequest represents a request to analyze document redactions
type AnalyzeRedactionsRequest struct {
	File        *multipart.FileHeader `form:"file" validate:"required"`
	Sensitivity string                `form:"sensitivity" validate:"omitempty,oneof=low medium high"`
}

// BulkUploadRequest represents a bulk upload request
type BulkUploadRequest struct {
	Files       []*multipart.FileHeader `form:"files" validate:"required,min=1,max=50"`
	Category    string                  `form:"category" validate:"omitempty,oneof=motion order contract brief memo other"`
	CaseName    string                  `form:"case_name" validate:"omitempty,max=200"`
	CaseNumber  string                  `form:"case_number" validate:"omitempty,max=50"`
	AutoProcess bool                    `form:"auto_process" validate:"omitempty"`
}

// GetDocumentRequest represents a request to retrieve a document
type GetDocumentRequest struct {
	DocumentID string `json:"document_id" validate:"required"`
	Format     string `json:"format" validate:"omitempty,oneof=json full metadata"`
}

// GetDocumentStatsRequest represents a request for document statistics
type GetDocumentStatsRequest struct {
	DateRange   *DateRange `json:"date_range,omitempty"`
	GroupBy     string     `json:"group_by" validate:"omitempty,oneof=date category court judge author"`
	Granularity string     `json:"granularity" validate:"omitempty,oneof=day week month year"`
}

// Re-export DateRange from pkg/models
type DateRange = models.DateRange

// HealthCheckRequest represents a health check request
type HealthCheckRequest struct {
	Component string `json:"component" validate:"omitempty,oneof=storage search pipeline classifier extractor"`
	Deep      bool   `json:"deep" validate:"omitempty"`
}

// Default values for process options
func DefaultProcessOptions() *ProcessOptions {
	return &ProcessOptions{
		ExtractText:    true,
		ClassifyDoc:    true,
		IndexDocument:  true,
		StoreDocument:  true,
		TimeoutSeconds: 120,
		RetryCount:     1,
	}
}

// Validate checks if ProcessOptions are valid
func (opts *ProcessOptions) Validate() error {
	if opts == nil {
		return nil
	}

	if opts.TimeoutSeconds < 1 || opts.TimeoutSeconds > 300 {
		opts.TimeoutSeconds = 120
	}

	if opts.RetryCount < 0 || opts.RetryCount > 3 {
		opts.RetryCount = 1
	}

	return nil
}

// RedactDocumentRequest represents a request to create a redacted version of a document
type RedactDocumentRequest struct {
	DocumentID       string                 `json:"document_id,omitempty"`
	ApplyRedactions  bool                   `json:"apply_redactions"`
	CustomRedactions []RedactionItem        `json:"custom_redactions,omitempty"`
	Options          *RedactionOptions      `json:"options,omitempty"`
	// For file upload redaction (alternative to document_id)
	PDFBase64        string                 `json:"pdf_base64,omitempty"`
}

// RedactionOptions configures redaction behavior
type RedactionOptions struct {
	UseAI            bool     `json:"use_ai"`
	CaliforniaLaws   bool     `json:"california_laws"`
	IncludePatterns  []string `json:"include_patterns,omitempty"`
	ExcludePatterns  []string `json:"exclude_patterns,omitempty"`
	ReplacementChar  string   `json:"replacement_char"`
}

// RedactionItem represents a single redaction to apply
type RedactionItem struct {
	ID          string    `json:"id"`
	Page        int       `json:"page"`
	Text        string    `json:"text"`
	BBox        []float64 `json:"bbox"` // [x0, y0, x1, y1]
	Type        string    `json:"type"`
	Citation    string    `json:"citation,omitempty"`
	Reason      string    `json:"reason,omitempty"`
	LegalCode   string    `json:"legal_code,omitempty"`
	Applied     bool      `json:"applied"`
}

// ApplyDefaults applies default values to process options
func (opts *ProcessOptions) ApplyDefaults() {
	if opts == nil {
		return
	}

	if opts.TimeoutSeconds == 0 {
		opts.TimeoutSeconds = 120
	}

	if opts.RetryCount == 0 {
		opts.RetryCount = 1
	}
}
