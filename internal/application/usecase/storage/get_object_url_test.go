package storage

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"motion-index-fiber/internal/application/dto"
	"motion-index-fiber/pkg/storage"
)

func TestGetStorageObjectURLUseCase_Execute(t *testing.T) {
	tests := []struct {
		name              string
		request           *dto.GetStorageObjectURLRequest
		mockObjects       []*storage.StorageObject
		existsResult      bool
		existsError       error
		expectError       bool
		expectedURLPrefix string
		validateResponse  func(*testing.T, *dto.GetStorageObjectURLResponse)
	}{
		{
			name: "get public URL for existing document",
			request: &dto.GetStorageObjectURLRequest{
				Path:         "documents/test.pdf",
				UseSignedURL: false,
				Expiration:   time.Hour,
			},
			existsResult:      true,
			expectError:       false,
			expectedURLPrefix: "https://example.com/documents/test.pdf",
			validateResponse: func(t *testing.T, resp *dto.GetStorageObjectURLResponse) {
				assert.Equal(t, "documents/test.pdf", resp.Path)
				assert.Equal(t, "application/pdf", resp.ContentType)
				assert.False(t, resp.IsSignedURL)
			},
		},
		{
			name: "get signed URL for existing document",
			request: &dto.GetStorageObjectURLRequest{
				Path:         "documents/test.pdf",
				UseSignedURL: true,
				Expiration:   time.Hour,
			},
			existsResult:      true,
			expectError:       false,
			expectedURLPrefix: "https://example.com/documents/test.pdf?signed=true",
			validateResponse: func(t *testing.T, resp *dto.GetStorageObjectURLResponse) {
				assert.True(t, resp.IsSignedURL)
				assert.Equal(t, time.Hour, resp.ExpiresIn)
			},
		},
		{
			name: "auto-add documents prefix",
			request: &dto.GetStorageObjectURLRequest{
				Path:         "test.pdf",
				UseSignedURL: false,
				Expiration:   time.Hour,
			},
			existsResult: true,
			expectError:  false,
			validateResponse: func(t *testing.T, resp *dto.GetStorageObjectURLResponse) {
				assert.Equal(t, "documents/test.pdf", resp.Path)
			},
		},
		{
			name: "document not found",
			request: &dto.GetStorageObjectURLRequest{
				Path:         "documents/nonexistent.pdf",
				UseSignedURL: false,
				Expiration:   time.Hour,
			},
			existsResult: false,
			mockObjects:  []*storage.StorageObject{},
			expectError:  true,
		},
		{
			name: "missing path",
			request: &dto.GetStorageObjectURLRequest{
				Path:         "",
				UseSignedURL: false,
				Expiration:   time.Hour,
			},
			expectError: true,
		},
		{
			name: "content type detection for different extensions",
			request: &dto.GetStorageObjectURLRequest{
				Path:         "documents/test.docx",
				UseSignedURL: false,
				Expiration:   time.Hour,
			},
			existsResult: true,
			expectError:  false,
			validateResponse: func(t *testing.T, resp *dto.GetStorageObjectURLResponse) {
				assert.Equal(t, "application/vnd.openxmlformats-officedocument.wordprocessingml.document", resp.ContentType)
			},
		},
		{
			name: "default expiration applied",
			request: &dto.GetStorageObjectURLRequest{
				Path:         "documents/test.pdf",
				UseSignedURL: true,
				Expiration:   0, // Should default to 1 hour
			},
			existsResult: true,
			expectError:  false,
			validateResponse: func(t *testing.T, resp *dto.GetStorageObjectURLResponse) {
				assert.Equal(t, time.Hour, resp.ExpiresIn)
			},
		},
		{
			name: "cap expiration at 24 hours",
			request: &dto.GetStorageObjectURLRequest{
				Path:         "documents/test.pdf",
				UseSignedURL: true,
				Expiration:   48 * time.Hour, // Should cap at 24h
			},
			existsResult: true,
			expectError:  false,
			validateResponse: func(t *testing.T, resp *dto.GetStorageObjectURLResponse) {
				assert.Equal(t, 24*time.Hour, resp.ExpiresIn)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStorage := newMockStorageServiceWithExists(tt.mockObjects, nil, tt.existsResult, tt.existsError)
			useCase := NewGetStorageObjectURLUseCase(mockStorage)

			result, err := useCase.Execute(context.Background(), tt.request)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.NotEmpty(t, result.URL)

			if tt.expectedURLPrefix != "" {
				assert.Contains(t, result.URL, tt.expectedURLPrefix)
			}

			if tt.validateResponse != nil {
				tt.validateResponse(t, result)
			}
		})
	}
}

func TestResolveDocumentPath(t *testing.T) {
	tests := []struct {
		name          string
		requestedPath string
		mockObjects   []*storage.StorageObject
		expectFound   bool
		expectedPath  string
	}{
		{
			name:          "exact match found",
			requestedPath: "documents/motion.pdf",
			mockObjects: []*storage.StorageObject{
				{Path: "documents/motion.pdf", Size: 1024},
			},
			expectFound:  true,
			expectedPath: "documents/motion.pdf",
		},
		{
			name:          "case insensitive match",
			requestedPath: "documents/MOTION.pdf",
			mockObjects: []*storage.StorageObject{
				{Path: "documents/motion.pdf", Size: 1024},
			},
			expectFound:  true,
			expectedPath: "documents/motion.pdf",
		},
		{
			name:          "resolve by document ID prefix",
			requestedPath: "documents/doc_123_motion.pdf",
			mockObjects: []*storage.StorageObject{
				{Path: "documents/doc_123/actual_motion.pdf", Size: 1024},
			},
			expectFound:  true,
			expectedPath: "documents/doc_123/actual_motion.pdf",
		},
		{
			name:          "no match found",
			requestedPath: "documents/nonexistent.pdf",
			mockObjects:   []*storage.StorageObject{},
			expectFound:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStorage := newMockStorageServiceWithExists(tt.mockObjects, nil, false, nil)
			useCase := NewGetStorageObjectURLUseCase(mockStorage)

			path, found := useCase.resolveDocumentPath(context.Background(), tt.requestedPath)

			assert.Equal(t, tt.expectFound, found)
			if tt.expectFound {
				assert.Equal(t, tt.expectedPath, path)
			}
		})
	}
}

// Enhanced mock storage with exists capability
type mockStorageServiceWithExists struct {
	*mockStorageService
	existsResult bool
	existsError  error
}

func newMockStorageServiceWithExists(objects []*storage.StorageObject, err error, existsResult bool, existsError error) *mockStorageServiceWithExists {
	return &mockStorageServiceWithExists{
		mockStorageService: newMockStorageService(objects, err),
		existsResult:       existsResult,
		existsError:        existsError,
	}
}

func (m *mockStorageServiceWithExists) Exists(ctx context.Context, path string) (bool, error) {
	if m.existsError != nil {
		return false, m.existsError
	}
	return m.existsResult, nil
}

func (m *mockStorageServiceWithExists) GetSignedURL(path string, expiration time.Duration) (string, error) {
	if expiration > 24*time.Hour {
		expiration = 24 * time.Hour
	}
	return fmt.Sprintf("https://example.com/%s?signed=true&exp=%s", path, expiration), nil
}
