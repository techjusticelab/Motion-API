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

func TestSearchStorageObjectsUseCase_Execute(t *testing.T) {
	tests := []struct {
		name          string
		request       *dto.SearchStorageObjectsRequest
		mockObjects   []*storage.StorageObject
		mockError     error
		expectError   bool
		expectedCount int
		validateURLs  bool
	}{
		{
			name: "search by partial filename",
			request: &dto.SearchStorageObjectsRequest{
				NamePattern: "motion",
				Prefix:      "documents/",
				Limit:       20,
				ExactMatch:  false,
			},
			mockObjects: []*storage.StorageObject{
				{Path: "documents/motion_to_suppress.pdf", Size: 1024, LastModified: time.Now()},
				{Path: "documents/other_document.pdf", Size: 2048, LastModified: time.Now()},
				{Path: "documents/motion_for_discovery.pdf", Size: 3072, LastModified: time.Now()},
			},
			expectError:   false,
			expectedCount: 2,
			validateURLs:  true,
		},
		{
			name: "exact match search",
			request: &dto.SearchStorageObjectsRequest{
				NamePattern: "motion_to_suppress.pdf",
				Prefix:      "documents/",
				Limit:       20,
				ExactMatch:  true,
			},
			mockObjects: []*storage.StorageObject{
				{Path: "documents/motion_to_suppress.pdf", Size: 1024, LastModified: time.Now()},
				{Path: "documents/motion_to_suppress_evidence.pdf", Size: 2048, LastModified: time.Now()},
			},
			expectError:   false,
			expectedCount: 1,
		},
		{
			name: "case insensitive search",
			request: &dto.SearchStorageObjectsRequest{
				NamePattern: "MOTION",
				Prefix:      "documents/",
				Limit:       20,
				ExactMatch:  false,
			},
			mockObjects: []*storage.StorageObject{
				{Path: "documents/motion_to_suppress.pdf", Size: 1024, LastModified: time.Now()},
				{Path: "documents/Motion_Brief.pdf", Size: 2048, LastModified: time.Now()},
			},
			expectError:   false,
			expectedCount: 2,
		},
		{
			name: "respect limit",
			request: &dto.SearchStorageObjectsRequest{
				NamePattern: "file",
				Prefix:      "documents/",
				Limit:       2,
				ExactMatch:  false,
			},
			mockObjects: []*storage.StorageObject{
				{Path: "documents/file1.pdf", Size: 1024, LastModified: time.Now()},
				{Path: "documents/file2.pdf", Size: 2048, LastModified: time.Now()},
				{Path: "documents/file3.pdf", Size: 3072, LastModified: time.Now()},
			},
			expectError:   false,
			expectedCount: 2,
		},
		{
			name: "skip directories",
			request: &dto.SearchStorageObjectsRequest{
				NamePattern: "file",
				Prefix:      "documents/",
				Limit:       20,
				ExactMatch:  false,
			},
			mockObjects: []*storage.StorageObject{
				{Path: "documents/file1.pdf", Size: 1024, LastModified: time.Now()},
				{Path: "documents/files/", Size: 0, LastModified: time.Now()},
			},
			expectError:   false,
			expectedCount: 1,
		},
		{
			name: "no matches",
			request: &dto.SearchStorageObjectsRequest{
				NamePattern: "nonexistent",
				Prefix:      "documents/",
				Limit:       20,
				ExactMatch:  false,
			},
			mockObjects: []*storage.StorageObject{
				{Path: "documents/file1.pdf", Size: 1024, LastModified: time.Now()},
			},
			expectError:   false,
			expectedCount: 0,
		},
		{
			name: "missing name pattern",
			request: &dto.SearchStorageObjectsRequest{
				NamePattern: "",
				Prefix:      "documents/",
				Limit:       20,
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStorage := newMockStorageService(tt.mockObjects, tt.mockError)
			useCase := NewSearchStorageObjectsUseCase(mockStorage)

			result, err := useCase.Execute(context.Background(), tt.request)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Len(t, result.Documents, tt.expectedCount)
			assert.Equal(t, tt.expectedCount, result.TotalFound)
			assert.Equal(t, tt.request.NamePattern, result.SearchPattern)
			assert.Equal(t, tt.request.ExactMatch, result.ExactMatch)

			if tt.validateURLs && len(result.Documents) > 0 {
				for _, doc := range result.Documents {
					assert.NotEmpty(t, doc.DirectURL, "DirectURL should not be empty")
					assert.NotEmpty(t, doc.SignedURL, "SignedURL should not be empty")
					assert.NotEmpty(t, doc.APIURL, "APIURL should not be empty")
					assert.Contains(t, doc.DirectURL, doc.Path)
					assert.Contains(t, doc.SignedURL, "signed=true")
				}
			}
		})
	}
}
