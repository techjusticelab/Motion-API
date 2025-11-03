package dto

import (
	"fmt"
	"time"
)

// ListStorageObjectsRequest represents a request to list storage objects.
type ListStorageObjectsRequest struct {
	Prefix   string
	Limit    int
	Cursor   string
	FileType string
	MinSize  int64
	MaxSize  int64
}

// Validate validates the list storage objects request.
func (r *ListStorageObjectsRequest) Validate() error {
	if r.Limit <= 0 || r.Limit > 500 {
		r.Limit = 50
	}
	if r.MinSize < 0 {
		return fmt.Errorf("min_size cannot be negative")
	}
	if r.MaxSize < 0 && r.MaxSize != -1 {
		return fmt.Errorf("max_size must be positive or -1 for no limit")
	}
	if r.MaxSize > 0 && r.MinSize > r.MaxSize {
		return fmt.Errorf("min_size cannot be greater than max_size")
	}
	return nil
}

// ListStorageObjectsResponse represents the response from listing storage objects.
type ListStorageObjectsResponse struct {
	Documents       []StorageObjectDTO `json:"documents"`
	NextCursor      string             `json:"next_cursor"`
	HasMore         bool               `json:"has_more"`
	TotalReturned   int                `json:"total_returned"`
	TotalEstimated  int                `json:"total_estimated"`
}

// StorageObjectDTO represents a storage object in the list response.
type StorageObjectDTO struct {
	Path         string    `json:"path"`
	Size         int64     `json:"size"`
	LastModified time.Time `json:"last_modified"`
	FileType     string    `json:"file_type"`
	Filename     string    `json:"filename"`
}

// CountStorageObjectsRequest represents a request to count storage objects.
type CountStorageObjectsRequest struct {
	Prefix   string
	FileType string
	MinSize  int64
	MaxSize  int64
}

// Validate validates the count storage objects request.
func (r *CountStorageObjectsRequest) Validate() error {
	if r.MinSize < 0 {
		return fmt.Errorf("min_size cannot be negative")
	}
	if r.MaxSize < 0 && r.MaxSize != -1 {
		return fmt.Errorf("max_size must be positive or -1 for no limit")
	}
	if r.MaxSize > 0 && r.MinSize > r.MaxSize {
		return fmt.Errorf("min_size cannot be greater than max_size")
	}
	return nil
}

// CountStorageObjectsResponse represents the response from counting storage objects.
type CountStorageObjectsResponse struct {
	TotalCount     int        `json:"total_count"`
	Prefix         string     `json:"prefix"`
	AppliedFilters FilterInfo `json:"applied_filters"`
}

// FilterInfo represents applied filters in the response.
type FilterInfo struct {
	FileType string `json:"file_type"`
	MinSize  int64  `json:"min_size"`
	MaxSize  int64  `json:"max_size"`
}

// SearchStorageObjectsRequest represents a request to search storage objects by filename.
type SearchStorageObjectsRequest struct {
	NamePattern string
	Prefix      string
	Limit       int
	ExactMatch  bool
}

// Validate validates the search storage objects request.
func (r *SearchStorageObjectsRequest) Validate() error {
	if r.NamePattern == "" {
		return fmt.Errorf("name pattern is required")
	}
	if r.Limit <= 0 || r.Limit > 100 {
		r.Limit = 20
	}
	return nil
}

// SearchStorageObjectsResponse represents the response from searching storage objects.
type SearchStorageObjectsResponse struct {
	Documents     []StorageObjectDetailDTO `json:"documents"`
	TotalFound    int                      `json:"total_found"`
	SearchPattern string                   `json:"search_pattern"`
	ExactMatch    bool                     `json:"exact_match"`
	Limit         int                      `json:"limit"`
}

// StorageObjectDetailDTO represents a storage object with additional details.
type StorageObjectDetailDTO struct {
	Path         string    `json:"path"`
	Filename     string    `json:"filename"`
	Size         int64     `json:"size"`
	LastModified time.Time `json:"last_modified"`
	FileType     string    `json:"file_type"`
	DirectURL    string    `json:"direct_url"`
	SignedURL    string    `json:"signed_url"`
	APIURL       string    `json:"api_url"`
}

// GetStorageObjectURLRequest represents a request to get a storage object URL.
type GetStorageObjectURLRequest struct {
	Path         string
	UseSignedURL bool
	Expiration   time.Duration
}

// Validate validates the get storage object URL request.
func (r *GetStorageObjectURLRequest) Validate() error {
	if r.Path == "" {
		return fmt.Errorf("path is required")
	}
	if r.Expiration <= 0 {
		r.Expiration = time.Hour
	}
	if r.Expiration > 24*time.Hour {
		r.Expiration = 24 * time.Hour
	}
	return nil
}

// GetStorageObjectURLResponse represents the response from getting a storage object URL.
type GetStorageObjectURLResponse struct {
	URL         string        `json:"url"`
	Path        string        `json:"path"`
	ContentType string        `json:"content_type"`
	IsSignedURL bool          `json:"is_signed_url"`
	ExpiresIn   time.Duration `json:"expires_in,omitempty"`
}
