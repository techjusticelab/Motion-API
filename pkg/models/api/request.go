package api

import (
	"fmt"
	"mime/multipart"
	"motion-index-fiber/pkg/models/core"
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

// DefaultProcessOptions returns default processing options
func DefaultProcessOptions() *ProcessOptions {
	return &ProcessOptions{
		ExtractText:    true,
		ClassifyDoc:    true,
		IndexDocument:  true,
		StoreDocument:  true,
		TimeoutSeconds: 300, // 5 minutes
		RetryCount:     3,
	}
}

// Validate validates the processing options
func (po *ProcessOptions) Validate() error {
	if po.TimeoutSeconds < 1 || po.TimeoutSeconds > 300 {
		return fmt.Errorf("timeout_seconds must be between 1 and 300")
	}
	if po.RetryCount < 0 || po.RetryCount > 3 {
		return fmt.Errorf("retry_count must be between 0 and 3")
	}
	return nil
}

// ApplyDefaults applies default values to unset fields
func (po *ProcessOptions) ApplyDefaults() {
	if po.TimeoutSeconds == 0 {
		po.TimeoutSeconds = 300
	}
	if po.RetryCount == 0 {
		po.RetryCount = 3
	}
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

// RedactDocumentRequest represents a request to redact a document
type RedactDocumentRequest struct {
	DocumentID        string           `json:"document_id" validate:"omitempty"`
	PDFBase64         string           `json:"pdf_base64" validate:"omitempty"`
	CustomRedactions  []RedactionItem  `json:"custom_redactions,omitempty"`
	Redactions        []RedactionItem  `json:"redactions,omitempty"`
	Options           *RedactionOptions `json:"options,omitempty"`
}

// RedactionItem represents a single redaction to apply (service layer compatible)
type RedactionItem struct {
	ID        string    `json:"id,omitempty"`
	Page      int       `json:"page" validate:"required,min=1"`
	Text      string    `json:"text,omitempty"`
	BBox      []float64 `json:"bbox,omitempty"` // [x0, y0, x1, y1]
	Type      string    `json:"type" validate:"omitempty,oneof=text image full"`
	Citation  string    `json:"citation,omitempty"`
	Reason    string    `json:"reason,omitempty"`
	LegalCode string    `json:"legal_code,omitempty"`
	Applied   bool      `json:"applied,omitempty"`
	// Legacy fields for backward compatibility
	X      float64 `json:"x,omitempty"`
	Y      float64 `json:"y,omitempty"`
	Width  float64 `json:"width,omitempty"`
	Height float64 `json:"height,omitempty"`
}

// RedactionOptions defines options for redaction processing
type RedactionOptions struct {
	Color       string `json:"color" validate:"omitempty"`
	FillPattern string `json:"fill_pattern" validate:"omitempty,oneof=solid crosshatch"`
	Permanent   bool   `json:"permanent" validate:"omitempty"`
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

// DateRange represents a date range for queries
type DateRange = core.DateRange

// HealthCheckRequest represents a health check request
type HealthCheckRequest struct {
	Component string `json:"component" validate:"omitempty,oneof=storage search pipeline classifier extractor"`
	Deep      bool   `json:"deep" validate:"omitempty"`
}

// ClassifyDocumentRequest represents a request to classify a document
type ClassifyDocumentRequest struct {
	DocumentID string `json:"document_id" validate:"required"`
	Text       string `json:"text,omitempty"`
	Force      bool   `json:"force" validate:"omitempty"`
}

// UpdateClassificationRequest represents a request to update classification
type UpdateClassificationRequest struct {
	DocumentID       string   `json:"document_id" validate:"required"`
	DocumentType     string   `json:"document_type" validate:"omitempty"`
	Category         string   `json:"category" validate:"omitempty"`
	Subject          string   `json:"subject" validate:"omitempty"`
	Summary          string   `json:"summary" validate:"omitempty"`
	LegalTags        []string `json:"legal_tags" validate:"omitempty"`
	ManualOverride   bool     `json:"manual_override" validate:"omitempty"`
}
