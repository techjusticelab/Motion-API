package search

import (
	"motion-index-fiber/pkg/models/core"
)

// Filters represents search filters that can be applied
type Filters struct {
	DocType       []string               `json:"doc_type,omitempty"`
	Court         []string               `json:"court,omitempty"`
	Judge         []string               `json:"judge,omitempty"`
	Author        []string               `json:"author,omitempty"`
	Status        []string               `json:"status,omitempty"`
	LegalTags     []string               `json:"legal_tags,omitempty"`
	DateRange     *core.DateRange      `json:"date_range,omitempty"`
	CustomFilters map[string]interface{} `json:"custom_filters,omitempty"`
}

// SortOptions represents sorting configuration for search queries
type SortOptions struct {
	Field     string    `json:"field"`
	Order     SortOrder `json:"order"`
	Ascending bool      `json:"ascending"` // For backward compatibility with tests
}

// SortOrder represents search result ordering
type SortOrder string

const (
	SortOrderAsc  SortOrder = "asc"
	SortOrderDesc SortOrder = "desc"
)

// PaginationOptions represents pagination configuration
type PaginationOptions struct {
	Offset int `json:"offset"`
	Limit  int `json:"limit"`
}

// HighlightOptions represents highlighting configuration
type HighlightOptions struct {
	Fields []string `json:"fields"`
}

// DefaultSearchSize is the default number of results to return
const DefaultSearchSize = 20

// MaxSearchSize is the maximum number of results allowed
const MaxSearchSize = 100
