package storage

import (
	"context"
	"fmt"

	"motion-index-fiber/internal/application/dto"
	"motion-index-fiber/pkg/storage"
)

// ListStorageObjectsUseCase handles listing storage objects with filters and pagination.
type ListStorageObjectsUseCase struct {
	storage storage.Service
}

// NewListStorageObjectsUseCase creates a new use case instance.
func NewListStorageObjectsUseCase(storage storage.Service) *ListStorageObjectsUseCase {
	return &ListStorageObjectsUseCase{storage: storage}
}

// Execute lists storage objects with the provided filters and pagination.
func (uc *ListStorageObjectsUseCase) Execute(ctx context.Context, req *dto.ListStorageObjectsRequest) (*dto.ListStorageObjectsResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// List all objects with prefix
	objects, err := uc.storage.List(ctx, req.Prefix)
	if err != nil {
		return nil, fmt.Errorf("failed to list objects: %w", err)
	}

	// Apply filters
	filtered := filterObjects(objects, req.FileType, req.MinSize, req.MaxSize)

	// Apply pagination
	paginated := paginateObjects(filtered, req.Cursor, req.Limit)

	return paginated, nil
}

// filterObjects applies file type and size filters to storage objects.
func filterObjects(objects []*storage.StorageObject, fileType string, minSize, maxSize int64) []*storage.StorageObject {
	var filtered []*storage.StorageObject

	for _, obj := range objects {
		// Skip directories
		if isDirectory(obj.Path) {
			continue
		}

		// Skip very small files (likely empty or corrupt)
		if obj.Size < 100 {
			continue
		}

		// Skip system files
		if isSystemFile(obj.Path) {
			continue
		}

		// Apply file type filter
		if fileType != "" && !matchesFileType(obj.Path, fileType) {
			continue
		}

		// Apply size filters
		if obj.Size < minSize {
			continue
		}
		if maxSize > 0 && obj.Size > maxSize {
			continue
		}

		filtered = append(filtered, obj)
	}

	return filtered
}

// paginateObjects implements cursor-based pagination.
func paginateObjects(objects []*storage.StorageObject, cursor string, limit int) *dto.ListStorageObjectsResponse {
	startIndex := 0

	// Decode cursor if provided
	if cursor != "" {
		if idx, err := decodeCursor(cursor); err == nil {
			startIndex = idx
		}
	}

	// Calculate end index
	endIndex := startIndex + limit
	if endIndex > len(objects) {
		endIndex = len(objects)
	}

	// Get page of objects
	var paginatedObjects []*storage.StorageObject
	if startIndex < len(objects) {
		paginatedObjects = objects[startIndex:endIndex]
	}

	// Generate next cursor
	var nextCursor string
	hasMore := endIndex < len(objects)
	if hasMore {
		nextCursor = encodeCursor(endIndex)
	}

	// Convert to DTOs
	documents := make([]dto.StorageObjectDTO, len(paginatedObjects))
	for i, obj := range paginatedObjects {
		documents[i] = dto.StorageObjectDTO{
			Path:         obj.Path,
			Size:         obj.Size,
			LastModified: obj.LastModified,
			FileType:     getFileExtension(obj.Path),
			Filename:     getFilename(obj.Path),
		}
	}

	return &dto.ListStorageObjectsResponse{
		Documents:      documents,
		NextCursor:     nextCursor,
		HasMore:        hasMore,
		TotalReturned:  len(documents),
		TotalEstimated: len(objects),
	}
}
