package persistence

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"motion-index-fiber/pkg/storage"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// mockStorageService is a mock implementation of storage.Service
type mockStorageService struct {
	mock.Mock
}

func (m *mockStorageService) Upload(ctx context.Context, key string, reader io.Reader, metadata *storage.UploadMetadata) (*storage.UploadResult, error) {
	args := m.Called(ctx, key, reader, metadata)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*storage.UploadResult), args.Error(1)
}

func (m *mockStorageService) Download(ctx context.Context, key string) (io.ReadCloser, error) {
	args := m.Called(ctx, key)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(io.ReadCloser), args.Error(1)
}

func (m *mockStorageService) Delete(ctx context.Context, key string) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

func (m *mockStorageService) GetURL(key string) string {
	args := m.Called(key)
	return args.String(0)
}

func (m *mockStorageService) IsHealthy() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *mockStorageService) Exists(ctx context.Context, key string) (bool, error) {
	args := m.Called(ctx, key)
	return args.Bool(0), args.Error(1)
}

func (m *mockStorageService) GetMetrics() map[string]interface{} {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(map[string]interface{})
}

func (m *mockStorageService) GetSignedURL(key string, expiresIn time.Duration) (string, error) {
	args := m.Called(key, expiresIn)
	return args.String(0), args.Error(1)
}

func (m *mockStorageService) List(ctx context.Context, prefix string) ([]*storage.StorageObject, error) {
	args := m.Called(ctx, prefix)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*storage.StorageObject), args.Error(1)
}

func TestNewStorageAdapter(t *testing.T) {
	mockService := new(mockStorageService)
	adapter := NewStorageAdapter(mockService)

	assert.NotNil(t, adapter)
	assert.Equal(t, mockService, adapter.service)
}

func TestStorageAdapter_Store_Success(t *testing.T) {
	mockService := new(mockStorageService)
	adapter := NewStorageAdapter(mockService)

	ctx := context.Background()
	path := "documents/test.pdf"
	content := strings.NewReader("test content")
	expectedURL := "https://example.com/documents/test.pdf"

	mockService.On("Upload", ctx, path, content, mock.AnythingOfType("*storage.UploadMetadata")).
		Return(&storage.UploadResult{
			Success: true,
			URL:     expectedURL,
		}, nil)

	url, err := adapter.Store(ctx, path, content)

	require.NoError(t, err)
	assert.Equal(t, expectedURL, url)
	mockService.AssertExpectations(t)
}

func TestStorageAdapter_Store_UploadError(t *testing.T) {
	mockService := new(mockStorageService)
	adapter := NewStorageAdapter(mockService)

	ctx := context.Background()
	path := "documents/test.pdf"
	content := strings.NewReader("test content")
	expectedErr := errors.New("upload failed")

	mockService.On("Upload", ctx, path, content, mock.AnythingOfType("*storage.UploadMetadata")).
		Return(nil, expectedErr)

	url, err := adapter.Store(ctx, path, content)

	require.Error(t, err)
	assert.Empty(t, url)
	assert.Contains(t, err.Error(), "storage error")
	mockService.AssertExpectations(t)
}

func TestStorageAdapter_Store_UploadNotSuccessful(t *testing.T) {
	mockService := new(mockStorageService)
	adapter := NewStorageAdapter(mockService)

	ctx := context.Background()
	path := "documents/test.pdf"
	content := strings.NewReader("test content")

	mockService.On("Upload", ctx, path, content, mock.AnythingOfType("*storage.UploadMetadata")).
		Return(&storage.UploadResult{
			Success: false,
			Error:   "insufficient quota",
		}, nil)

	url, err := adapter.Store(ctx, path, content)

	require.Error(t, err)
	assert.Empty(t, url)
	mockService.AssertExpectations(t)
}

func TestStorageAdapter_Retrieve_Success(t *testing.T) {
	mockService := new(mockStorageService)
	adapter := NewStorageAdapter(mockService)

	ctx := context.Background()
	path := "documents/test.pdf"
	expectedReader := io.NopCloser(strings.NewReader("test content"))

	mockService.On("Download", ctx, path).
		Return(expectedReader, nil)

	reader, err := adapter.Retrieve(ctx, path)

	require.NoError(t, err)
	assert.NotNil(t, reader)
	mockService.AssertExpectations(t)
}

func TestStorageAdapter_Retrieve_Error(t *testing.T) {
	mockService := new(mockStorageService)
	adapter := NewStorageAdapter(mockService)

	ctx := context.Background()
	path := "documents/nonexistent.pdf"
	expectedErr := errors.New("not found")

	mockService.On("Download", ctx, path).
		Return(nil, expectedErr)

	reader, err := adapter.Retrieve(ctx, path)

	require.Error(t, err)
	assert.Nil(t, reader)
	// Error is translated to "document not found" by error translation layer
	assert.Contains(t, err.Error(), "not found")
	mockService.AssertExpectations(t)
}

func TestStorageAdapter_Delete_Success(t *testing.T) {
	mockService := new(mockStorageService)
	adapter := NewStorageAdapter(mockService)

	ctx := context.Background()
	path := "documents/test.pdf"

	mockService.On("Delete", ctx, path).
		Return(nil)

	err := adapter.Delete(ctx, path)

	require.NoError(t, err)
	mockService.AssertExpectations(t)
}

func TestStorageAdapter_Delete_Error(t *testing.T) {
	mockService := new(mockStorageService)
	adapter := NewStorageAdapter(mockService)

	ctx := context.Background()
	path := "documents/test.pdf"
	expectedErr := errors.New("delete failed")

	mockService.On("Delete", ctx, path).
		Return(expectedErr)

	err := adapter.Delete(ctx, path)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "storage error")
	mockService.AssertExpectations(t)
}

func TestStorageAdapter_URL(t *testing.T) {
	mockService := new(mockStorageService)
	adapter := NewStorageAdapter(mockService)

	path := "documents/test.pdf"
	expectedURL := "https://example.com/documents/test.pdf"

	mockService.On("GetURL", path).
		Return(expectedURL)

	url := adapter.URL(path)

	assert.Equal(t, expectedURL, url)
	mockService.AssertExpectations(t)
}
