package storage

import (
	"context"
	"io"
	"time"
)

// Service defines the interface for document storage operations
type Service interface {
	// Upload uploads a document to storage
	Upload(ctx context.Context, path string, content io.Reader, metadata *UploadMetadata) (*UploadResult, error)

	// Download downloads a document from storage
	Download(ctx context.Context, path string) (io.ReadCloser, error)

	// Delete deletes a document from storage
	Delete(ctx context.Context, path string) error

	// GetURL returns a public URL for the document
	GetURL(path string) string

	// GetSignedURL returns a signed URL for temporary access
	GetSignedURL(path string, expiration time.Duration) (string, error)

	// Exists checks if a document exists in storage
	Exists(ctx context.Context, path string) (bool, error)

	// List lists documents in a directory
	List(ctx context.Context, prefix string) ([]*StorageObject, error)

	// IsHealthy returns true if the storage service is healthy
	IsHealthy() bool

	// GetMetrics returns storage-specific metrics
	GetMetrics() map[string]interface{}
}

// UploadMetadata contains metadata for document uploads
type UploadMetadata struct {
	ContentType     string            `json:"content_type"`
	Size            int64             `json:"size"`
	FileName        string            `json:"file_name"`
	Tags            map[string]string `json:"tags,omitempty"`
	CacheControl    string            `json:"cache_control,omitempty"`
	ContentEncoding string            `json:"content_encoding,omitempty"`
}

// UploadResult contains the result of a document upload
type UploadResult struct {
	Path       string    `json:"path"`
	URL        string    `json:"url"`
	Size       int64     `json:"size"`
	ETag       string    `json:"etag,omitempty"`
	Success    bool      `json:"success"`
	Error      string    `json:"error,omitempty"`
	UploadedAt time.Time `json:"uploaded_at"`
}

// StorageObject represents an object in storage
type StorageObject struct {
	Path         string            `json:"path"`
	Size         int64             `json:"size"`
	LastModified time.Time         `json:"last_modified"`
	ETag         string            `json:"etag,omitempty"`
	ContentType  string            `json:"content_type,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

// FileMetadata represents metadata for a file in storage
type FileMetadata struct {
	FileName    string            `json:"file_name"`
	ContentType string            `json:"content_type"`
	Size        int64             `json:"size"`
	Hash        string            `json:"hash,omitempty"`
	Tags        map[string]string `json:"tags,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

// SpacesConfig represents configuration for DigitalOcean Spaces
type SpacesConfig struct {
	AccessKey string `json:"access_key"`
	SecretKey string `json:"secret_key"`
	Bucket    string `json:"bucket"`
	Region    string `json:"region"`
	CDNDomain string `json:"cdn_domain,omitempty"`
	Endpoint  string `json:"endpoint,omitempty"`
}

// StorageError represents errors that occur during storage operations
type StorageError struct {
	Type    string
	Message string
	Path    string
	Cause   error
}

func (e *StorageError) Error() string {
	if e.Cause != nil {
		return e.Message + ": " + e.Cause.Error()
	}
	return e.Message
}

func (e *StorageError) Unwrap() error {
	return e.Cause
}

// NewStorageError creates a new storage error
func NewStorageError(errorType, message, path string, cause error) *StorageError {
	return &StorageError{
		Type:    errorType,
		Message: message,
		Path:    path,
		Cause:   cause,
	}
}
