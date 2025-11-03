package storage

import (
	"context"
	"fmt"

	"motion-index-fiber/internal/application/dto"
	"motion-index-fiber/pkg/storage"
)

// CountStorageObjectsUseCase handles counting storage objects with filters.
type CountStorageObjectsUseCase struct {
	storage storage.Service
}

// NewCountStorageObjectsUseCase creates a new use case instance.
func NewCountStorageObjectsUseCase(storage storage.Service) *CountStorageObjectsUseCase {
	return &CountStorageObjectsUseCase{storage: storage}
}

// Execute counts storage objects with the provided filters.
func (uc *CountStorageObjectsUseCase) Execute(ctx context.Context, req *dto.CountStorageObjectsRequest) (*dto.CountStorageObjectsResponse, error) {
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

	return &dto.CountStorageObjectsResponse{
		TotalCount: len(filtered),
		Prefix:     req.Prefix,
		AppliedFilters: dto.FilterInfo{
			FileType: req.FileType,
			MinSize:  req.MinSize,
			MaxSize:  req.MaxSize,
		},
	}, nil
}
