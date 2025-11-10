package search

import (
	"motion-index-fiber/pkg/models/core"
)

// SearchResult represents the response from a search query
type SearchResult struct {
	TotalHits    int64                  `json:"total_hits"`
	MaxScore     float64                `json:"max_score,omitempty"`
	Documents    []*SearchDocument      `json:"documents"`
	Aggregations map[string]interface{} `json:"aggregations,omitempty"`
	Took         int64                  `json:"took_ms"`
	TimedOut     bool                   `json:"timed_out"`
}

// SearchDocument represents a document in search results
type SearchDocument struct {
	ID         string                 `json:"id"`
	Score      float64                `json:"score,omitempty"`
	Document   map[string]interface{} `json:"document"`
	Highlights map[string][]string    `json:"highlights,omitempty"`
}

// SearchResponse represents the top-level search response
type SearchResponse struct {
	Success    bool                `json:"success"`
	Message    string              `json:"message,omitempty"`
	Data       *SearchResult       `json:"data,omitempty"`
	Error      *SearchError        `json:"error,omitempty"`
	RequestID  string              `json:"request_id,omitempty"`
	Timestamp  string              `json:"timestamp"`
	Documents  []*core.Document  `json:"documents,omitempty"` // For backward compatibility with tests
	Total      int64               `json:"total,omitempty"`     // For backward compatibility with tests
}

// SearchError represents an error in search operations
type SearchError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// NewSearchResult creates a new search result
func NewSearchResult() *SearchResult {
	return &SearchResult{
		Documents:    make([]*SearchDocument, 0),
		Aggregations: make(map[string]interface{}),
		TimedOut:     false,
	}
}
