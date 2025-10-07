package redaction

import (
	"context"
	"io"
)

// Service defines the interface for redaction operations
type Service interface {
	// RedactPDF redacts a PDF document and returns the redacted PDF and metadata
	RedactPDF(ctx context.Context, pdfData io.Reader, options *Options) (*Result, error)
	
	// AnalyzePDF analyzes a PDF for potential redactions without applying them
	AnalyzePDF(ctx context.Context, pdfData io.Reader, options *Options) (*AnalysisResult, error)
	
	// ApplyCustomRedactions applies custom redactions to a PDF
	ApplyCustomRedactions(ctx context.Context, pdfData io.Reader, redactions []RedactionItem) (*Result, error)
}

// Options configures redaction behavior
type Options struct {
	UseAI            bool     `json:"use_ai"`
	CaliforniaLaws   bool     `json:"california_laws"`
	IncludePatterns  []string `json:"include_patterns,omitempty"`
	ExcludePatterns  []string `json:"exclude_patterns,omitempty"`
	ReplacementChar  string   `json:"replacement_char"`
}

// RedactionItem represents a single redaction
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

// Result represents the result of a redaction operation
type Result struct {
	RedactedPDF  []byte          `json:"-"`
	PDFBase64    string          `json:"pdf_base64"`
	Redactions   []RedactionItem `json:"redactions"`
	TotalCount   int             `json:"total_count"`
	Success      bool            `json:"success"`
	Error        string          `json:"error,omitempty"`
}

// AnalysisResult represents the result of redaction analysis
type AnalysisResult struct {
	Redactions []RedactionItem `json:"redactions"`
	TotalCount int             `json:"total_count"`
	Success    bool            `json:"success"`
	Error      string          `json:"error,omitempty"`
}

// CaliforniaLegalCode represents a California legal code with citation info
type CaliforniaLegalCode struct {
	Code        string `json:"code"`
	Description string `json:"description"`
}

// RedactionPattern represents a pattern to search for redaction
type RedactionPattern struct {
	Name     string               `json:"name"`
	Pattern  string               `json:"pattern"`
	Citation CaliforniaLegalCode  `json:"citation"`
	Reason   string               `json:"reason"`
}