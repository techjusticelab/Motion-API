package storage

import (
	"context"
	"fmt"
	"strings"
	"time"

	"motion-index-fiber/internal/application/dto"
	"motion-index-fiber/pkg/storage"
)

// SearchStorageObjectsUseCase handles searching storage objects by filename.
type SearchStorageObjectsUseCase struct {
	storage storage.Service
}

// NewSearchStorageObjectsUseCase creates a new use case instance.
func NewSearchStorageObjectsUseCase(storage storage.Service) *SearchStorageObjectsUseCase {
	return &SearchStorageObjectsUseCase{storage: storage}
}

// Execute searches for storage objects matching the filename pattern.
func (uc *SearchStorageObjectsUseCase) Execute(ctx context.Context, req *dto.SearchStorageObjectsRequest) (*dto.SearchStorageObjectsResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// List all objects with prefix
	objects, err := uc.storage.List(ctx, req.Prefix)
	if err != nil {
		return nil, fmt.Errorf("failed to list objects: %w", err)
	}

	// Search and collect matches
	var matches []dto.StorageObjectDetailDTO
	namePattern := strings.ToLower(req.NamePattern)

	for _, obj := range objects {
		// Skip directories
		if isDirectory(obj.Path) {
			continue
		}

		filename := strings.ToLower(getFilename(obj.Path))

		// Check if matches
		var isMatch bool
		if req.ExactMatch {
			ext := getFileExtension(obj.Path)
			isMatch = filename == namePattern || filename == namePattern+ext
		} else {
			isMatch = strings.Contains(filename, namePattern)
		}

		if isMatch {
			// Generate URLs
			directURL := uc.storage.GetURL(obj.Path)
			signedURL, _ := uc.storage.GetSignedURL(obj.Path, time.Hour)

			// Build API URL
			apiURL := fmt.Sprintf("/api/v1/files/%s", strings.TrimPrefix(obj.Path, "documents/"))

			matches = append(matches, dto.StorageObjectDetailDTO{
				Path:         obj.Path,
				Filename:     getFilename(obj.Path),
				Size:         obj.Size,
				LastModified: obj.LastModified,
				FileType:     getFileExtension(obj.Path),
				DirectURL:    directURL,
				SignedURL:    signedURL,
				APIURL:       apiURL,
			})

			// Stop if we've reached the limit
			if len(matches) >= req.Limit {
				break
			}
		}
	}

	return &dto.SearchStorageObjectsResponse{
		Documents:     matches,
		TotalFound:    len(matches),
		SearchPattern: req.NamePattern,
		ExactMatch:    req.ExactMatch,
		Limit:         req.Limit,
	}, nil
}
