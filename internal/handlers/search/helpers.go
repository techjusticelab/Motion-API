package search

import (
	"motion-index-fiber/pkg/models"
)

// validateSearchRequest validates a search request
// This is a shared utility function used across search handlers
func validateSearchRequest(req *models.SearchRequest) error {
	if req.Size > models.MaxSearchSize {
		req.Size = models.MaxSearchSize
	}
	if req.Size <= 0 {
		req.Size = models.DefaultSearchSize
	}
	if req.From < 0 {
		req.From = 0
	}

	// Validate sort order
	if req.SortOrder != "" && req.SortOrder != "asc" && req.SortOrder != "desc" {
		req.SortOrder = "desc"
	}

	return nil
}
