package ports

import (
	"context"
	"io"
	"time"

	"motion-index-fiber/internal/domain/classification"
	"motion-index-fiber/internal/domain/document"
)

// ClassificationService provides AI classification capabilities.
type ClassificationService interface {
	Classify(ctx context.Context, doc *document.Document) (*classification.Result, error)
	SupportedDocumentTypes() []string
	ProviderName() string
}

// StorageService abstracts storage of document binaries.
type StorageService interface {
	Store(ctx context.Context, path string, content io.Reader) (string, error)
	Retrieve(ctx context.Context, path string) (io.ReadCloser, error)
	Delete(ctx context.Context, path string) error
	URL(path string) string
}

// SearchService abstracts indexing and search interactions.
type SearchService interface {
	Index(ctx context.Context, doc *document.Document) error
	Update(ctx context.Context, id string, fields map[string]interface{}) error
	Delete(ctx context.Context, id string) error
	Search(ctx context.Context, query SearchQuery) (*SearchResults, error)
}

// EventBus dispatches domain events emitted by aggregates.
type EventBus interface {
	Publish(ctx context.Context, events ...document.DomainEvent) error
}

// RedactionService provides PDF redaction capabilities.
type RedactionService interface {
	Analyze(ctx context.Context, reader io.Reader, options *RedactionOptions) (*RedactionAnalysis, error)
	Redact(ctx context.Context, reader io.Reader, options *RedactionOptions) (*RedactionResult, error)
	ApplyCustom(ctx context.Context, reader io.Reader, options *RedactionOptions, redactions []RedactionItem) (*RedactionResult, error)
}

// SearchQuery captures search parameters.
type SearchQuery struct {
	Query         string
	DocumentType  string
	Category      string
	LegalTags     []string
	FromDate      *time.Time
	ToDate        *time.Time
	MinConfidence float64
	Page          int
	PageSize      int
	SortBy        string
	SortOrder     string
}

// SearchResults summarizes paginated search results.
type SearchResults struct {
	Total    int
	Page     int
	PageSize int
	Hits     []SearchHit
}

// SearchHit is a single search summary entry.
type SearchHit struct {
	ID           string
	FileName     string
	DocumentType string
	Category     string
	Snippet      string
	Confidence   float64
	CreatedAt    time.Time
}

// RedactionOptions map to redaction service configuration.
type RedactionOptions struct {
	UseAI           bool
	CaliforniaLaws  bool
	IncludePatterns []string
	ExcludePatterns []string
	ReplacementChar string
}

// RedactionItem captures a redaction region.
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

// RedactionAnalysis summarises a redaction analysis run.
type RedactionAnalysis struct {
	Redactions []RedactionItem
	TotalCount int
}

// RedactionResult contains applied redactions and optional output.
type RedactionResult struct {
	Redactions      []RedactionItem
	TotalCount      int
	PDFBytes        []byte
	PDFBase64       string
	GeneratedFileID string
	Success         bool
	Message         string
}
