package search

import (
	"motion-index-fiber/pkg/models/core"
)

// SearchRequest represents a search query with legal-specific filters
type SearchRequest struct {
	Query             string             `json:"query,omitempty" validate:"max=1000"`
	DocType           string             `json:"doc_type,omitempty"`
	CaseNumber        string             `json:"case_number,omitempty"`
	CaseName          string             `json:"case_name,omitempty"`
	Judge             []string           `json:"judge,omitempty"`
	Court             []string           `json:"court,omitempty"`
	Author            string             `json:"author,omitempty"`
	Status            string             `json:"status,omitempty"`
	LegalTags         []string           `json:"legal_tags,omitempty"`
	LegalTagsMatchAll bool               `json:"legal_tags_match_all"`
	DateRange         *core.DateRange  `json:"date_range,omitempty"`
	Size              int                `json:"size" validate:"min=1,max=100"`
	From              int                `json:"from" validate:"min=0"`
	SortBy            string             `json:"sort_by,omitempty"`
	SortOrder         string             `json:"sort_order,omitempty"`
	IncludeHighlights bool               `json:"include_highlights"`
	FuzzySearch       bool               `json:"fuzzy_search"`
	Filters           interface{}        `json:"filters,omitempty"` // Can be *Filters or map[string]interface{}
	Sort              *SortOptions       `json:"sort,omitempty"`
	Pagination        *PaginationOptions `json:"pagination,omitempty"`
	Highlight         *HighlightOptions  `json:"highlight,omitempty"`
	Limit             int                `json:"limit,omitempty"` // For backward compatibility with tests
}

// ValidateSearchRequest validates a search request and applies defaults
func ValidateSearchRequest(req *SearchRequest) error {
	if req.Size > MaxSearchSize {
		req.Size = MaxSearchSize
	}
	if req.Size <= 0 {
		req.Size = DefaultSearchSize
	}
	if req.From < 0 {
		req.From = 0
	}

	// Validate sort order
	if req.SortOrder != "" && req.SortOrder != "asc" && req.SortOrder != "desc" {
		req.SortOrder = "desc"
	}

	// Handle backward compatibility with Limit field
	if req.Limit > 0 && req.Size == 0 {
		req.Size = req.Limit
		if req.Size > MaxSearchSize {
			req.Size = MaxSearchSize
		}
	}

	return nil
}

// ApplyDefaults applies default values to a search request
func (sr *SearchRequest) ApplyDefaults() {
	if sr.Size <= 0 {
		sr.Size = DefaultSearchSize
	}
	if sr.From < 0 {
		sr.From = 0
	}
	if sr.SortOrder == "" {
		sr.SortOrder = "desc"
	}
	if sr.SortBy == "" {
		sr.SortBy = "relevance"
	}
}

// GetEffectiveSize returns the effective size for the search request
func (sr *SearchRequest) GetEffectiveSize() int {
	if sr.Size > 0 {
		return sr.Size
	}
	if sr.Limit > 0 {
		return sr.Limit
	}
	return DefaultSearchSize
}

// GetEffectiveFrom returns the effective from offset for the search request
func (sr *SearchRequest) GetEffectiveFrom() int {
	if sr.From < 0 {
		return 0
	}
	return sr.From
}

// HasFilters returns true if the search request has any filters applied
func (sr *SearchRequest) HasFilters() bool {
	return sr.DocType != "" ||
		sr.CaseNumber != "" ||
		sr.CaseName != "" ||
		len(sr.Judge) > 0 ||
		len(sr.Court) > 0 ||
		sr.Author != "" ||
		sr.Status != "" ||
		len(sr.LegalTags) > 0 ||
		sr.DateRange != nil && !sr.DateRange.IsEmpty()
}

// GetFilterCount returns the number of active filters
func (sr *SearchRequest) GetFilterCount() int {
	count := 0
	if sr.DocType != "" {
		count++
	}
	if sr.CaseNumber != "" {
		count++
	}
	if sr.CaseName != "" {
		count++
	}
	if len(sr.Judge) > 0 {
		count++
	}
	if len(sr.Court) > 0 {
		count++
	}
	if sr.Author != "" {
		count++
	}
	if sr.Status != "" {
		count++
	}
	if len(sr.LegalTags) > 0 {
		count++
	}
	if sr.DateRange != nil && !sr.DateRange.IsEmpty() {
		count++
	}
	return count
}

// NewSearchRequest creates a new search request with defaults
func NewSearchRequest() *SearchRequest {
	req := &SearchRequest{
		Size:              DefaultSearchSize,
		From:              0,
		SortBy:            "relevance",
		SortOrder:         "desc",
		IncludeHighlights: true,
		FuzzySearch:       false,
		LegalTagsMatchAll: false,
	}
	return req
}

// MetadataFieldValuesRequest represents a request for metadata field values with custom filters
type MetadataFieldValuesRequest struct {
	Field         string                 `json:"field" validate:"required"`
	Prefix        string                 `json:"prefix,omitempty"`
	Size          int                    `json:"size,omitempty" validate:"min=1,max=1000"`
	Filters       map[string]interface{} `json:"filters,omitempty"`
	ExcludeValues []string               `json:"exclude_values,omitempty"`
}
