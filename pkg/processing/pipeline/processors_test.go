package pipeline

import (
	"context"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"motion-index-fiber/pkg/models"
	"motion-index-fiber/pkg/processing/classifier"
	"motion-index-fiber/pkg/search"
	"motion-index-fiber/pkg/storage"
)

// MockStorageService implements storage.Service for testing
type MockStorageService struct {
	mock.Mock
}

func (m *MockStorageService) Upload(ctx context.Context, path string, content io.Reader, metadata *storage.UploadMetadata) (*storage.UploadResult, error) {
	args := m.Called(ctx, path, content, metadata)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*storage.UploadResult), args.Error(1)
}

func (m *MockStorageService) Download(ctx context.Context, path string) (io.ReadCloser, error) {
	args := m.Called(ctx, path)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(io.ReadCloser), args.Error(1)
}

func (m *MockStorageService) Delete(ctx context.Context, path string) error {
	args := m.Called(ctx, path)
	return args.Error(0)
}

func (m *MockStorageService) GetURL(path string) string {
	args := m.Called(path)
	return args.String(0)
}

func (m *MockStorageService) GetSignedURL(path string, expiration time.Duration) (string, error) {
	args := m.Called(path, expiration)
	return args.String(0), args.Error(1)
}

func (m *MockStorageService) Exists(ctx context.Context, path string) (bool, error) {
	args := m.Called(ctx, path)
	return args.Bool(0), args.Error(1)
}

func (m *MockStorageService) List(ctx context.Context, prefix string) ([]*storage.StorageObject, error) {
	args := m.Called(ctx, prefix)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*storage.StorageObject), args.Error(1)
}

func (m *MockStorageService) IsHealthy() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockStorageService) GetMetrics() map[string]interface{} {
	args := m.Called()
	return args.Get(0).(map[string]interface{})
}

// MockSearchService implements search.Service for testing
type MockSearchService struct {
	mock.Mock
}

func (m *MockSearchService) SearchDocuments(ctx context.Context, req *models.SearchRequest) (*models.SearchResult, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.SearchResult), args.Error(1)
}

func (m *MockSearchService) IndexDocument(ctx context.Context, doc *models.Document) (string, error) {
	args := m.Called(ctx, doc)
	return args.String(0), args.Error(1)
}

func (m *MockSearchService) BulkIndexDocuments(ctx context.Context, docs []*models.Document) (*models.BulkResult, error) {
	args := m.Called(ctx, docs)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.BulkResult), args.Error(1)
}

func (m *MockSearchService) UpdateDocumentMetadata(ctx context.Context, docID string, metadata map[string]interface{}) error {
	args := m.Called(ctx, docID, metadata)
	return args.Error(0)
}

func (m *MockSearchService) DeleteDocument(ctx context.Context, docID string) error {
	args := m.Called(ctx, docID)
	return args.Error(0)
}

func (m *MockSearchService) GetDocument(ctx context.Context, docID string) (*models.Document, error) {
	args := m.Called(ctx, docID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Document), args.Error(1)
}

func (m *MockSearchService) DocumentExists(ctx context.Context, docID string) (bool, error) {
	args := m.Called(ctx, docID)
	return args.Bool(0), args.Error(1)
}

func (m *MockSearchService) GetLegalTags(ctx context.Context) ([]*models.TagCount, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.TagCount), args.Error(1)
}

func (m *MockSearchService) GetDocumentTypes(ctx context.Context) ([]*models.TypeCount, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.TypeCount), args.Error(1)
}

func (m *MockSearchService) GetMetadataFieldValues(ctx context.Context, field string, prefix string, size int) ([]*models.FieldValue, error) {
	args := m.Called(ctx, field, prefix, size)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.FieldValue), args.Error(1)
}

func (m *MockSearchService) GetMetadataFieldValuesWithFilters(ctx context.Context, req *models.MetadataFieldValuesRequest) ([]*models.FieldValue, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.FieldValue), args.Error(1)
}

func (m *MockSearchService) GetDocumentStats(ctx context.Context) (*models.DocumentStats, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.DocumentStats), args.Error(1)
}

func (m *MockSearchService) GetAllFieldOptions(ctx context.Context) (*models.FieldOptions, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.FieldOptions), args.Error(1)
}

func (m *MockSearchService) IsHealthy() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockSearchService) Health(ctx context.Context) (*search.HealthStatus, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*search.HealthStatus), args.Error(1)
}

// Tests for Storage Processor

func TestStorageProcessor_GenerateStoragePath(t *testing.T) {
	mockStorage := new(MockStorageService)
	processor := NewStorageProcessor(mockStorage).(*storageProcessor)

	tests := []struct {
		name       string
		fileName   string
		docID      string
		wantPrefix string
		wantSuffix string
	}{
		{
			name:       "Standard PDF file",
			fileName:   "legal-document.pdf",
			docID:      "doc_123456",
			wantPrefix: "documents/",
			wantSuffix: "/doc_123456/legal-document.pdf",
		},
		{
			name:       "File with spaces",
			fileName:   "my document.pdf",
			docID:      "doc_789",
			wantPrefix: "documents/",
			wantSuffix: "/doc_789/my_document.pdf",
		},
		{
			name:       "File with special characters",
			fileName:   "doc/with/slashes.pdf",
			docID:      "doc_999",
			wantPrefix: "documents/",
			wantSuffix: "/doc_999/doc_with_slashes.pdf",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := processor.generateStoragePath(tt.fileName, tt.docID)

			assert.True(t, strings.HasPrefix(path, tt.wantPrefix),
				fmt.Sprintf("Path should start with %s, got: %s", tt.wantPrefix, path))
			assert.True(t, strings.HasSuffix(path, tt.wantSuffix),
				fmt.Sprintf("Path should end with %s, got: %s", tt.wantSuffix, path))

			// Verify path contains year/month structure
			assert.Contains(t, path, "/202", "Path should contain year component")
		})
	}
}

func TestStorageProcessor_Process(t *testing.T) {
	tests := []struct {
		name          string
		request       *ProcessRequest
		mockURL       string
		wantSuccess   bool
		wantError     bool
		validatePath  func(t *testing.T, path string)
	}{
		{
			name: "Successful storage processing",
			request: &ProcessRequest{
				ID:          "doc_12345",
				FileName:    "test-document.pdf",
				ContentType: "application/pdf",
				Size:        1024,
			},
			mockURL:     "https://bucket.nyc3.digitaloceanspaces.com/documents/2025/01/doc_12345/test-document.pdf",
			wantSuccess: true,
			wantError:   false,
			validatePath: func(t *testing.T, path string) {
				assert.Contains(t, path, "doc_12345")
				assert.Contains(t, path, "test-document.pdf")
				assert.True(t, strings.HasPrefix(path, "documents/"), "Path should start with 'documents/'")
			},
		},
		{
			name: "Storage with long filename",
			request: &ProcessRequest{
				ID:          "doc_67890",
				FileName:    "very-long-filename-with-many-characters-that-exceeds-normal-length.pdf",
				ContentType: "application/pdf",
				Size:        2048,
			},
			mockURL:     "https://bucket.nyc3.digitaloceanspaces.com/documents/2025/01/doc_67890/very-long-filename.pdf",
			wantSuccess: true,
			wantError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStorage := new(MockStorageService)
			processor := NewStorageProcessor(mockStorage)

			// Mock GetURL to return the expected URL
			mockStorage.On("GetURL", mock.AnythingOfType("string")).Return(tt.mockURL)

			result, err := processor.Process(context.Background(), tt.request)

			if tt.wantError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, result)
			require.NotNil(t, result.StorageResult)

			assert.Equal(t, tt.wantSuccess, result.StorageResult.Success)
			assert.NotEmpty(t, result.StorageResult.StoragePath)
			assert.Equal(t, tt.mockURL, result.StorageResult.URL)

			if tt.validatePath != nil {
				tt.validatePath(t, result.StorageResult.StoragePath)
			}

			mockStorage.AssertExpectations(t)
		})
	}
}

// Tests for Indexing Processor with S3 URI extraction

func TestIndexingProcessor_S3URIExtraction(t *testing.T) {
	tests := []struct {
		name            string
		storageURL      string
		storagePath     string
		expectedS3URI   string
		expectedBucket  string
	}{
		{
			name:           "Extract from standard DigitalOcean Spaces URL",
			storageURL:     "https://motion-index-docs.nyc3.digitaloceanspaces.com/documents/2025/01/doc_123/file.pdf",
			storagePath:    "documents/2025/01/doc_123/file.pdf",
			expectedS3URI:  "s3://motion-index-docs/documents/2025/01/doc_123/file.pdf",
			expectedBucket: "motion-index-docs",
		},
		{
			name:           "Extract from different region",
			storageURL:     "https://my-bucket.sfo2.digitaloceanspaces.com/documents/test.pdf",
			storagePath:    "documents/test.pdf",
			expectedS3URI:  "s3://my-bucket/documents/test.pdf",
			expectedBucket: "my-bucket",
		},
		{
			name:           "Extract from bucket with hyphens",
			storageURL:     "https://my-test-bucket.nyc3.digitaloceanspaces.com/documents/file.pdf",
			storagePath:    "documents/file.pdf",
			expectedS3URI:  "s3://my-test-bucket/documents/file.pdf",
			expectedBucket: "my-test-bucket",
		},
		{
			name:           "Path with nested directories",
			storageURL:     "https://docs-bucket.nyc3.digitaloceanspaces.com/documents/2025/01/15/doc_456/legal.pdf",
			storagePath:    "documents/2025/01/15/doc_456/legal.pdf",
			expectedS3URI:  "s3://docs-bucket/documents/2025/01/15/doc_456/legal.pdf",
			expectedBucket: "docs-bucket",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSearch := new(MockSearchService)
			processor := NewIndexingProcessor(mockSearch).(*indexingProcessor)

			mockSearch.On("IndexDocument", mock.Anything, mock.AnythingOfType("*models.Document")).
				Return("doc_123", nil)

			req := &ProcessRequest{
				ID:       "doc_123",
				FileName: "test.pdf",
				Metadata: map[string]string{
					"extracted_text": "Sample text",
					"storage_path":   tt.storagePath,
					"storage_url":    tt.storageURL,
				},
			}

			result, err := processor.ProcessWithFullResult(context.Background(), req, nil)

			require.NoError(t, err)
			require.NotNil(t, result)
			require.NotNil(t, result.Document)

			// Verify S3 URI was properly extracted
			assert.Equal(t, tt.expectedS3URI, result.Document.S3URI,
				"S3 URI should match expected format")

			// Verify FilePath is set correctly
			assert.Equal(t, tt.storagePath, result.Document.FilePath,
				"FilePath should match storage path")

			// Verify FileURL is set
			assert.Equal(t, tt.storageURL, result.Document.FileURL,
				"FileURL should match storage URL")

			mockSearch.AssertExpectations(t)
		})
	}
}

func TestIndexingProcessor_S3URIFallback(t *testing.T) {
	t.Run("Fallback when URL parsing fails", func(t *testing.T) {
		mockSearch := new(MockSearchService)
		processor := NewIndexingProcessor(mockSearch).(*indexingProcessor)

		mockSearch.On("IndexDocument", mock.Anything, mock.AnythingOfType("*models.Document")).
			Return("doc_123", nil)

		req := &ProcessRequest{
			ID:       "doc_123",
			FileName: "test.pdf",
			Metadata: map[string]string{
				"extracted_text": "Sample text",
				"storage_path":   "documents/test.pdf",
				"storage_url":    "https://invalid-format-url.com/test.pdf", // Invalid format
			},
		}

		result, err := processor.ProcessWithFullResult(context.Background(), req, nil)

		require.NoError(t, err)
		require.NotNil(t, result)
		require.NotNil(t, result.Document)

		// Should fallback to unknown-bucket
		assert.Contains(t, result.Document.S3URI, "s3://unknown-bucket/")
		assert.Contains(t, result.Document.S3URI, "documents/test.pdf")

		mockSearch.AssertExpectations(t)
	})
}

func TestIndexingProcessor_WithFullClassificationResult(t *testing.T) {
	t.Run("Full classification with storage metadata", func(t *testing.T) {
		mockSearch := new(MockSearchService)
		processor := NewIndexingProcessor(mockSearch).(*indexingProcessor)

		filingDate := "2025-01-15"
		eventDate := "2025-01-10"

		classificationResult := &classifier.ClassificationResult{
			DocumentType:  "motion_to_suppress",
			LegalCategory: "Criminal Law",
			Subject:       "Motion to Suppress Evidence",
			Summary:       "Motion to suppress illegally obtained evidence",
			Confidence:    0.95,
			FilingDate:    &filingDate,
			EventDate:     &eventDate,
			LegalTags:     []string{"evidence", "suppression", "4th-amendment"},
		}

		fullResult := &ProcessResult{
			ClassificationResult: classificationResult,
		}

		storagePath := "documents/2025/01/doc_123/motion.pdf"
		storageURL := "https://motion-docs.nyc3.digitaloceanspaces.com/documents/2025/01/doc_123/motion.pdf"

		mockSearch.On("IndexDocument", mock.Anything, mock.MatchedBy(func(doc *models.Document) bool {
			// Verify all fields are properly set
			return doc.FilePath == storagePath &&
				doc.S3URI == "s3://motion-docs/documents/2025/01/doc_123/motion.pdf" &&
				doc.FileURL == storageURL &&
				doc.DocType == "motion_to_suppress" &&
				doc.Category == "Criminal Law" &&
				doc.Metadata.Subject == "Motion to Suppress Evidence" &&
				doc.Metadata.Confidence == 0.95 &&
				len(doc.Metadata.LegalTags) == 3
		})).Return("doc_123", nil)

		req := &ProcessRequest{
			ID:       "doc_123",
			FileName: "motion.pdf",
			Metadata: map[string]string{
				"extracted_text": "Motion to suppress evidence...",
				"storage_path":   storagePath,
				"storage_url":    storageURL,
			},
		}

		result, err := processor.ProcessWithFullResult(context.Background(), req, fullResult)

		require.NoError(t, err)
		require.NotNil(t, result)
		require.NotNil(t, result.Document)

		// Verify storage fields
		assert.Equal(t, storagePath, result.Document.FilePath)
		assert.Equal(t, "s3://motion-docs/documents/2025/01/doc_123/motion.pdf", result.Document.S3URI)
		assert.Equal(t, storageURL, result.Document.FileURL)

		// Verify classification fields
		assert.Equal(t, "motion_to_suppress", result.Document.DocType)
		assert.Equal(t, "Criminal Law", result.Document.Category)
		assert.Equal(t, "Motion to Suppress Evidence", result.Document.Metadata.Subject)

		mockSearch.AssertExpectations(t)
	})
}

// Benchmark tests

func BenchmarkStorageProcessor_GeneratePath(b *testing.B) {
	mockStorage := new(MockStorageService)
	processor := NewStorageProcessor(mockStorage).(*storageProcessor)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = processor.generateStoragePath("test-document.pdf", fmt.Sprintf("doc_%d", i))
	}
}

func BenchmarkIndexingProcessor_S3URIExtraction(b *testing.B) {
	mockSearch := new(MockSearchService)
	processor := NewIndexingProcessor(mockSearch).(*indexingProcessor)

	mockSearch.On("IndexDocument", mock.Anything, mock.AnythingOfType("*models.Document")).
		Return("doc_123", nil)

	req := &ProcessRequest{
		ID:       "doc_123",
		FileName: "test.pdf",
		Metadata: map[string]string{
			"extracted_text": "Sample text",
			"storage_path":   "documents/2025/01/doc_123/test.pdf",
			"storage_url":    "https://bucket.nyc3.digitaloceanspaces.com/documents/2025/01/doc_123/test.pdf",
		},
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = processor.ProcessWithFullResult(ctx, req, nil)
	}
}
