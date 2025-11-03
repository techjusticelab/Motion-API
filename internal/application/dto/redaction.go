package dto

import (
	"encoding/base64"
	"errors"
	"strings"

	"motion-index-fiber/internal/application/validation"
)

// RedactionOptions configures how redaction analysis and application run.
type RedactionOptions struct {
	UseAI           bool
	CaliforniaLaws  bool
	IncludePatterns []string
	ExcludePatterns []string
	ReplacementChar string
}

// Normalize ensures defaults where callers omitted values.
func (o *RedactionOptions) Normalize() *RedactionOptions {
	if o == nil {
		return &RedactionOptions{
			CaliforniaLaws:  true,
			ReplacementChar: "■",
		}
	}
	if strings.TrimSpace(o.ReplacementChar) == "" {
		o.ReplacementChar = "■"
	}
	return o
}

// RedactionItem represents a region to redact.
type RedactionItem struct {
	ID        string
	Page      int
	Text      string
	BBox      []float64
	Type      string
	Citation  string
	Reason    string
	LegalCode string
	Applied   bool
}

// AnalyzeRedactionsRequest requests redaction analysis.
type AnalyzeRedactionsRequest struct {
	DocumentID string
	PDFBase64  string
	Options    *RedactionOptions
}

// Validate enforces request invariants.
func (r *AnalyzeRedactionsRequest) Validate() error {
	errs := validation.NewErrorSet()

	if r.DocumentID == "" && strings.TrimSpace(r.PDFBase64) == "" {
		errs.Append("documentId", errors.New("documentId or pdfBase64 is required"))
		errs.Append("pdfBase64", errors.New("documentId or pdfBase64 is required"))
	}

	if strings.TrimSpace(r.PDFBase64) != "" {
		if _, err := base64.StdEncoding.DecodeString(r.PDFBase64); err != nil {
			errs.Append("pdfBase64", errors.New("pdfBase64 must be valid base64"))
		}
	}

	return errs.Error()
}

// AnalyzeRedactionsResponse captures analysis outcome.
type AnalyzeRedactionsResponse struct {
	DocumentID string
	FileName   string
	Redactions []RedactionItem
	TotalCount int
}

// ApplyRedactionsRequest applies redactions and optionally returns a redacted document.
type ApplyRedactionsRequest struct {
	DocumentID       string
	PDFBase64        string
	Options          *RedactionOptions
	CustomRedactions []RedactionItem
	ReturnBase64     bool
}

// Validate enforces request invariants.
func (r *ApplyRedactionsRequest) Validate() error {
	errs := validation.NewErrorSet()

	if r.DocumentID == "" && strings.TrimSpace(r.PDFBase64) == "" {
		errs.Append("documentId", errors.New("documentId or pdfBase64 is required"))
		errs.Append("pdfBase64", errors.New("documentId or pdfBase64 is required"))
	}

	if strings.TrimSpace(r.PDFBase64) != "" {
		if _, err := base64.StdEncoding.DecodeString(r.PDFBase64); err != nil {
			errs.Append("pdfBase64", errors.New("pdfBase64 must be valid base64"))
		}
	}

	return errs.Error()
}

// ApplyRedactionsResponse summarises the redaction operation.
type ApplyRedactionsResponse struct {
	DocumentID      string
	FileName        string
	Redactions      []RedactionItem
	TotalRedactions int
	PDFBase64       string
}
