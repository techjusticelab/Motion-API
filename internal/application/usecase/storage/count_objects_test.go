package storage

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"motion-index-fiber/internal/application/dto"
	"motion-index-fiber/pkg/storage"
)

func TestCountStorageObjectsUseCase_Execute(t *testing.T) {
	tests := []struct {
		name          string
		request       *dto.CountStorageObjectsRequest
		mockObjects   []*storage.StorageObject
		mockError     error
		expectError   bool
		expectedCount int
	}{
		{
			name: "count all documents",
			request: &dto.CountStorageObjectsRequest{
				Prefix:  "documents/",
				MinSize: 0,
				MaxSize: -1,
			},
			mockObjects: []*storage.StorageObject{
				{Path: "documents/file1.pdf", Size: 1024, LastModified: time.Now()},
				{Path: "documents/file2.pdf", Size: 2048, LastModified: time.Now()},
				{Path: "documents/file3.pdf", Size: 3072, LastModified: time.Now()},
			},
			expectError:   false,
			expectedCount: 3,
		},
		{
			name: "count with file type filter",
			request: &dto.CountStorageObjectsRequest{
				Prefix:   "documents/",
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
		},
		{
			name: "count with size range",
			request: &dto.CountStorageObjectsRequest{
				Prefix:  "documents/",
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
		},
		{
			name: "exclude system files",
			request: &dto.CountStorageObjectsRequest{
				Prefix:  "documents/",
				MinSize: 0,
				MaxSize: -1,
			},
			mockObjects: []*storage.StorageObject{
				{Path: "documents/file1.pdf", Size: 1024, LastModified: time.Now()},
				{Path: "documents/.DS_Store", Size: 512, LastModified: time.Now()},
				{Path: "documents/file.tmp", Size: 1024, LastModified: time.Now()},
			},
			expectError:   false,
			expectedCount: 1,
		},
		{
			name: "exclude directories and small files",
			request: &dto.CountStorageObjectsRequest{
				Prefix:  "documents/",
				MinSize: 0,
				MaxSize: -1,
			},
			mockObjects: []*storage.StorageObject{
				{Path: "documents/file1.pdf", Size: 1024, LastModified: time.Now()},
				{Path: "documents/subdir/", Size: 0, LastModified: time.Now()},
				{Path: "documents/tiny.pdf", Size: 50, LastModified: time.Now()},
			},
			expectError:   false,
			expectedCount: 1,
		},
		{
			name: "empty result",
			request: &dto.CountStorageObjectsRequest{
				Prefix:  "documents/",
				MinSize: 0,
				MaxSize: -1,
			},
			mockObjects:   []*storage.StorageObject{},
			expectError:   false,
			expectedCount: 0,
		},
		{
			name: "invalid request - negative min size",
			request: &dto.CountStorageObjectsRequest{
				Prefix:  "documents/",
				MinSize: -100,
				MaxSize: -1,
			},
			expectError: true,
		},
		{
			name: "invalid request - min greater than max",
			request: &dto.CountStorageObjectsRequest{
				Prefix:  "documents/",
				MinSize: 5000,
				MaxSize: 1000,
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStorage := newMockStorageService(tt.mockObjects, tt.mockError)
			useCase := NewCountStorageObjectsUseCase(mockStorage)

			result, err := useCase.Execute(context.Background(), tt.request)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expectedCount, result.TotalCount)
			assert.Equal(t, tt.request.Prefix, result.Prefix)
			assert.Equal(t, tt.request.FileType, result.AppliedFilters.FileType)
			assert.Equal(t, tt.request.MinSize, result.AppliedFilters.MinSize)
			assert.Equal(t, tt.request.MaxSize, result.AppliedFilters.MaxSize)
		})
	}
}
