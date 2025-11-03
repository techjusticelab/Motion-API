package storage

import (
	"context"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"motion-index-fiber/internal/application/dto"
	"motion-index-fiber/pkg/storage"
)

func TestListStorageObjectsUseCase_Execute(t *testing.T) {
	tests := []struct {
		name           string
		request        *dto.ListStorageObjectsRequest
		mockObjects    []*storage.StorageObject
		mockError      error
		expectError    bool
		expectedCount  int
		validateResult func(*testing.T, *dto.ListStorageObjectsResponse)
	}{
		{
			name: "successful list with pagination",
			request: &dto.ListStorageObjectsRequest{
				Prefix:  "documents/",
				Limit:   2,
				MinSize: 0,
				MaxSize: -1,
			},
			mockObjects: []*storage.StorageObject{
				{Path: "documents/file1.pdf", Size: 1024, LastModified: time.Now()},
				{Path: "documents/file2.pdf", Size: 2048, LastModified: time.Now()},
				{Path: "documents/file3.pdf", Size: 3072, LastModified: time.Now()},
			},
			expectError:   false,
			expectedCount: 2,
			validateResult: func(t *testing.T, resp *dto.ListStorageObjectsResponse) {
				assert.Len(t, resp.Documents, 2)
				assert.True(t, resp.HasMore)
				assert.NotEmpty(t, resp.NextCursor)
				assert.Equal(t, 3, resp.TotalEstimated)
			},
		},
		{
			name: "filter by file type",
			request: &dto.ListStorageObjectsRequest{
				Prefix:   "documents/",
				Limit:    10,
				FileType: ".pdf",
				MinSize:  0,
				MaxSize:  -1,
			},
			mockObjects: []*storage.StorageObject{
				{Path: "documents/file1.pdf", Size: 1024, LastModified: time.Now()},
				{Path: "documents/file2.docx", Size: 2048, LastModified: time.Now()},
				{Path: "documents/file3.pdf", Size: 3072, LastModified: time.Now()},
			},
			expectError:   false,
			expectedCount: 2,
			validateResult: func(t *testing.T, resp *dto.ListStorageObjectsResponse) {
				assert.Len(t, resp.Documents, 2)
				for _, doc := range resp.Documents {
					assert.Equal(t, ".pdf", doc.FileType)
				}
			},
		},
		{
			name: "filter by size range",
			request: &dto.ListStorageObjectsRequest{
				Prefix:  "documents/",
				Limit:   10,
				MinSize: 1500,
				MaxSize: 2500,
			},
			mockObjects: []*storage.StorageObject{
				{Path: "documents/file1.pdf", Size: 1024, LastModified: time.Now()},
				{Path: "documents/file2.pdf", Size: 2048, LastModified: time.Now()},
				{Path: "documents/file3.pdf", Size: 3072, LastModified: time.Now()},
			},
			expectError:   false,
			expectedCount: 1,
			validateResult: func(t *testing.T, resp *dto.ListStorageObjectsResponse) {
				assert.Len(t, resp.Documents, 1)
				assert.Equal(t, int64(2048), resp.Documents[0].Size)
			},
		},
		{
			name: "skip system files and directories",
			request: &dto.ListStorageObjectsRequest{
				Prefix:  "documents/",
				Limit:   10,
				MinSize: 0,
				MaxSize: -1,
			},
			mockObjects: []*storage.StorageObject{
				{Path: "documents/file1.pdf", Size: 1024, LastModified: time.Now()},
				{Path: "documents/.DS_Store", Size: 512, LastModified: time.Now()},
				{Path: "documents/__MACOSX/", Size: 0, LastModified: time.Now()},
				{Path: "documents/subdir/", Size: 0, LastModified: time.Now()},
				{Path: "documents/file.tmp", Size: 1024, LastModified: time.Now()},
			},
			expectError:   false,
			expectedCount: 1,
			validateResult: func(t *testing.T, resp *dto.ListStorageObjectsResponse) {
				assert.Len(t, resp.Documents, 1)
				assert.Equal(t, "documents/file1.pdf", resp.Documents[0].Path)
			},
		},
		{
			name: "skip very small files",
			request: &dto.ListStorageObjectsRequest{
				Prefix:  "documents/",
				Limit:   10,
				MinSize: 0,
				MaxSize: -1,
			},
			mockObjects: []*storage.StorageObject{
				{Path: "documents/file1.pdf", Size: 50, LastModified: time.Now()}, // Too small
				{Path: "documents/file2.pdf", Size: 1024, LastModified: time.Now()},
			},
			expectError:   false,
			expectedCount: 1,
		},
		{
			name: "invalid request - negative min size",
			request: &dto.ListStorageObjectsRequest{
				Prefix:  "documents/",
				Limit:   10,
				MinSize: -100,
				MaxSize: -1,
			},
			expectError: true,
		},
		{
			name: "invalid request - min size greater than max size",
			request: &dto.ListStorageObjectsRequest{
				Prefix:  "documents/",
				Limit:   10,
				MinSize: 5000,
				MaxSize: 1000,
			},
			expectError: true,
		},
		{
			name: "empty result set",
			request: &dto.ListStorageObjectsRequest{
				Prefix:  "documents/",
				Limit:   10,
				MinSize: 0,
				MaxSize: -1,
			},
			mockObjects:   []*storage.StorageObject{},
			expectError:   false,
			expectedCount: 0,
			validateResult: func(t *testing.T, resp *dto.ListStorageObjectsResponse) {
				assert.Empty(t, resp.Documents)
				assert.False(t, resp.HasMore)
				assert.Empty(t, resp.NextCursor)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStorage := newMockStorageService(tt.mockObjects, tt.mockError)
			useCase := NewListStorageObjectsUseCase(mockStorage)

			result, err := useCase.Execute(context.Background(), tt.request)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expectedCount, result.TotalReturned)

			if tt.validateResult != nil {
				tt.validateResult(t, result)
			}
		})
	}
}

func TestFilterObjects(t *testing.T) {
	tests := []struct {
		name          string
		objects       []*storage.StorageObject
		fileType      string
		minSize       int64
		maxSize       int64
		expectedCount int
	}{
		{
			name: "no filters",
			objects: []*storage.StorageObject{
				{Path: "documents/file1.pdf", Size: 1024},
				{Path: "documents/file2.pdf", Size: 2048},
			},
			fileType:      "",
			minSize:       0,
			maxSize:       -1,
			expectedCount: 2,
		},
		{
			name: "file type filter",
			objects: []*storage.StorageObject{
				{Path: "documents/file1.pdf", Size: 1024},
				{Path: "documents/file2.docx", Size: 2048},
			},
			fileType:      ".pdf",
			minSize:       0,
			maxSize:       -1,
			expectedCount: 1,
		},
		{
			name: "size range filter",
			objects: []*storage.StorageObject{
				{Path: "documents/file1.pdf", Size: 500},
				{Path: "documents/file2.pdf", Size: 1500},
				{Path: "documents/file3.pdf", Size: 2500},
			},
			fileType:      "",
			minSize:       1000,
			maxSize:       2000,
			expectedCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filterObjects(tt.objects, tt.fileType, tt.minSize, tt.maxSize)
			assert.Len(t, result, tt.expectedCount)
		})
	}
}

func TestPaginateObjects(t *testing.T) {
	objects := []*storage.StorageObject{
		{Path: "documents/file1.pdf", Size: 1024, LastModified: time.Now()},
		{Path: "documents/file2.pdf", Size: 2048, LastModified: time.Now()},
		{Path: "documents/file3.pdf", Size: 3072, LastModified: time.Now()},
		{Path: "documents/file4.pdf", Size: 4096, LastModified: time.Now()},
		{Path: "documents/file5.pdf", Size: 5120, LastModified: time.Now()},
	}

	tests := []struct {
		name           string
		cursor         string
		limit          int
		expectedCount  int
		expectedHasMore bool
	}{
		{
			name:            "first page",
			cursor:          "",
			limit:           2,
			expectedCount:   2,
			expectedHasMore: true,
		},
		{
			name:            "second page",
			cursor:          encodeCursor(2),
			limit:           2,
			expectedCount:   2,
			expectedHasMore: true,
		},
		{
			name:            "last page",
			cursor:          encodeCursor(4),
			limit:           2,
			expectedCount:   1,
			expectedHasMore: false,
		},
		{
			name:            "all items",
			cursor:          "",
			limit:           10,
			expectedCount:   5,
			expectedHasMore: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := paginateObjects(objects, tt.cursor, tt.limit)
			assert.Len(t, result.Documents, tt.expectedCount)
			assert.Equal(t, tt.expectedHasMore, result.HasMore)
		})
	}
}

// Mock storage service
type mockStorageService struct {
	objects []*storage.StorageObject
	err     error
}

func newMockStorageService(objects []*storage.StorageObject, err error) *mockStorageService {
	return &mockStorageService{objects: objects, err: err}
}

func (m *mockStorageService) List(ctx context.Context, prefix string) ([]*storage.StorageObject, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.objects, nil
}

func (m *mockStorageService) Exists(ctx context.Context, path string) (bool, error) {
	return true, nil
}

func (m *mockStorageService) GetURL(path string) string {
	return "https://example.com/" + path
}

func (m *mockStorageService) GetSignedURL(path string, expiration time.Duration) (string, error) {
	return "https://example.com/" + path + "?signed=true", nil
}

func (m *mockStorageService) Store(ctx context.Context, path string, content []byte) error {
	return nil
}

func (m *mockStorageService) Retrieve(ctx context.Context, path string) ([]byte, error) {
	return []byte{}, nil
}

func (m *mockStorageService) Delete(ctx context.Context, path string) error {
	return nil
}

func (m *mockStorageService) Download(ctx context.Context, path string) (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader("")), nil
}

func (m *mockStorageService) Upload(ctx context.Context, path string, content io.Reader, metadata *storage.UploadMetadata) (*storage.UploadResult, error) {
	return &storage.UploadResult{Path: path}, nil
}

func (m *mockStorageService) IsHealthy() bool {
	return true
}

func (m *mockStorageService) GetMetrics() map[string]interface{} {
	return map[string]interface{}{}
}
