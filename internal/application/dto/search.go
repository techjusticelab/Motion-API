package dto

import (
	"time"

	"motion-index-fiber/internal/application/validation"
)

// SearchDocumentsRequest holds query parameters for searching documents.
type SearchDocumentsRequest struct {
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

// Validate ensures search constraints are reasonable.
func (r *SearchDocumentsRequest) Validate() error {
	errs := validation.NewErrorSet()

	if err := validation.ValidatePagination(r.Page, r.PageSize); err != nil {
		errs.Append("pagination", err)
	}

	if err := validation.ValidateConfidenceRange(r.MinConfidence); err != nil {
		errs.Append("minConfidence", err)
	}

	if err := validation.ValidateDateRange(r.FromDate, r.ToDate); err != nil {
		errs.Append("dateRange", err)
	}

	if err := validation.ValidateSort(r.SortBy, r.SortOrder); err != nil {
		errs.Append("sort", err)
	}

	return errs.Error()
}

// SearchResultsResponse summarises search hits.
type SearchResultsResponse struct {
	Documents  []DocumentSummary
	Total      int
	Page       int
	PageSize   int
	TotalPages int
}

// DocumentSummary is a condensed view of a document for listings.
type DocumentSummary struct {
	ID           string
	FileName     string
	DocumentType string
	Category     string
	Snippet      string
	Confidence   float64
	CreatedAt    time.Time
}
