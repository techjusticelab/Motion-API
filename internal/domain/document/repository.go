package document

import (
	"context"
	"time"
)

// DocumentRepository defines persistence operations for the document aggregate.
type DocumentRepository interface {
	Save(ctx context.Context, document *Document) error
	FindByID(ctx context.Context, id DocumentID) (*Document, error)
	FindAll(ctx context.Context, filter Filter) ([]*Document, error)
	Delete(ctx context.Context, id DocumentID) error
	Exists(ctx context.Context, id DocumentID) (bool, error)
	Count(ctx context.Context, filter Filter) (int, error)
}

// Filter specifies criteria for querying documents.
type Filter struct {
	// Pagination
	Limit  int
	Offset int

	// Filters
	DocumentType  *DocumentType
	Category      *Category
	MinConfidence *Confidence
	FromDate      *time.Time
	ToDate        *time.Time

	// Search
	TextQuery string
	LegalTags []string

	// Sorting
	SortBy    string
	SortOrder SortOrder
}

// Normalise ensures filter values are sanitised.
func (f *Filter) Normalise() {
	if f == nil {
		return
	}
	f.LegalTags = sanitizeTags(f.LegalTags)
	if f.SortBy == "" {
		f.SortBy = "created_at"
	}
	if f.SortOrder == "" {
		f.SortOrder = SortDesc
	}
}

// HasPagination reports whether pagination is configured.
func (f Filter) HasPagination() bool {
	return f.Limit > 0
}

// SortOrder controls ordering direction.
type SortOrder string

const (
	// SortAsc sorts ascending.
	SortAsc SortOrder = "asc"
	// SortDesc sorts descending.
	SortDesc SortOrder = "desc"
)
