package persistence

import (
	"context"
	"io"

	"motion-index-fiber/internal/application/ports"
	infraerrors "motion-index-fiber/internal/infrastructure/errors"
	"motion-index-fiber/pkg/storage"
)

// StorageAdapter adapts the existing pkg/storage to the application port.
type StorageAdapter struct {
	service storage.Service
}

// NewStorageAdapter creates a new storage adapter.
func NewStorageAdapter(service storage.Service) *StorageAdapter {
	return &StorageAdapter{
		service: service,
	}
}

// Store implements the StorageService port.
func (a *StorageAdapter) Store(ctx context.Context, path string, content io.Reader) (string, error) {
	metadata := &storage.UploadMetadata{
		ContentType: "application/octet-stream", // Default, can be enhanced
	}

	result, err := a.service.Upload(ctx, path, content, metadata)
	if err != nil {
		return "", infraerrors.TranslateStorageError(err)
	}

	if !result.Success {
		return "", infraerrors.TranslateStorageError(
			storage.NewStorageError("upload_failed", result.Error, path, nil),
		)
	}

	return result.URL, nil
}

// Retrieve implements the StorageService port.
func (a *StorageAdapter) Retrieve(ctx context.Context, path string) (io.ReadCloser, error) {
	reader, err := a.service.Download(ctx, path)
	if err != nil {
		return nil, infraerrors.TranslateStorageError(err)
	}

	return reader, nil
}

// Delete implements the StorageService port.
func (a *StorageAdapter) Delete(ctx context.Context, path string) error {
	err := a.service.Delete(ctx, path)
	if err != nil {
		return infraerrors.TranslateStorageError(err)
	}

	return nil
}

// URL implements the StorageService port.
func (a *StorageAdapter) URL(path string) string {
	return a.service.GetURL(path)
}

// Ensure StorageAdapter implements the port
var _ ports.StorageService = (*StorageAdapter)(nil)
