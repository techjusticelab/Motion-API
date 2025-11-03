package persistence

import (
	"context"
	"errors"
	"testing"
	"time"

	"motion-index-fiber/internal/application/ports"
	"motion-index-fiber/internal/domain/document"
	"motion-index-fiber/pkg/models"
	"motion-index-fiber/pkg/search"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// mockSearchService is a mock implementation of search.Service
type mockSearchService struct {
	mock.Mock
}

func (m *mockSearchService) IndexDocument(ctx context.Context, doc *models.Document) (string, error) {
	args := m.Called(ctx, doc)
	return args.String(0), args.Error(1)
}

func (m *mockSearchService) BulkIndexDocuments(ctx context.Context, docs []*models.Document) (*models.BulkResult, error) {
	args := m.Called(ctx, docs)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.BulkResult), args.Error(1)
}

func (m *mockSearchService) UpdateDocumentMetadata(ctx context.Context, docID string, metadata map[string]interface{}) error {
	args := m.Called(ctx, docID, metadata)
	return args.Error(0)
}

func (m *mockSearchService) DeleteDocument(ctx context.Context, docID string) error {
	args := m.Called(ctx, docID)
	return args.Error(0)
}

func (m *mockSearchService) GetDocument(ctx context.Context, docID string) (*models.Document, error) {
	args := m.Called(ctx, docID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Document), args.Error(1)
}

func (m *mockSearchService) DocumentExists(ctx context.Context, docID string) (bool, error) {
	args := m.Called(ctx, docID)
	return args.Bool(0), args.Error(1)
}

func (m *mockSearchService) SearchDocuments(ctx context.Context, req *models.SearchRequest) (*models.SearchResult, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.SearchResult), args.Error(1)
}

func (m *mockSearchService) GetLegalTags(ctx context.Context) ([]*models.TagCount, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.TagCount), args.Error(1)
}

func (m *mockSearchService) GetDocumentTypes(ctx context.Context) ([]*models.TypeCount, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.TypeCount), args.Error(1)
}

func (m *mockSearchService) GetMetadataFieldValues(ctx context.Context, field string, prefix string, size int) ([]*models.FieldValue, error) {
	args := m.Called(ctx, field, prefix, size)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.FieldValue), args.Error(1)
}

func (m *mockSearchService) GetMetadataFieldValuesWithFilters(ctx context.Context, req *models.MetadataFieldValuesRequest) ([]*models.FieldValue, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.FieldValue), args.Error(1)
}

func (m *mockSearchService) GetDocumentStats(ctx context.Context) (*models.DocumentStats, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.DocumentStats), args.Error(1)
}

func (m *mockSearchService) GetAllFieldOptions(ctx context.Context) (*models.FieldOptions, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.FieldOptions), args.Error(1)
}

func (m *mockSearchService) IsHealthy() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *mockSearchService) Health(ctx context.Context) (*search.HealthStatus, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*search.HealthStatus), args.Error(1)
}

func TestNewSearchAdapter(t *testing.T) {
	mockService := new(mockSearchService)
	adapter := NewSearchAdapter(mockService)

	assert.NotNil(t, adapter)
	assert.Equal(t, mockService, adapter.service)
}

func TestSearchAdapter_Index_Success(t *testing.T) {
	mockService := new(mockSearchService)
	adapter := NewSearchAdapter(mockService)

	ctx := context.Background()

	// Create domain document
	doc, err := document.NewDocument(
		document.WithID("doc_123"),
		document.WithFileName("test.pdf"),
		document.WithStoragePath("documents/test.pdf"),
		document.WithContentType("application/pdf"),
		document.WithHash("a1b2c3d4e5f60123456789abcdef0123456789abcdef0123456789abcdef0123", "SHA256"),
		document.WithFileSize(1024),
	)
	require.NoError(t, err)

	mockService.On("IndexDocument", ctx, mock.AnythingOfType("*models.Document")).
		Return("doc_123", nil)

	err = adapter.Index(ctx, doc)

	require.NoError(t, err)
	mockService.AssertExpectations(t)
}

func TestSearchAdapter_Index_Error(t *testing.T) {
	mockService := new(mockSearchService)
	adapter := NewSearchAdapter(mockService)

	ctx := context.Background()

	doc, err := document.NewDocument(
		document.WithID("doc_123"),
		document.WithFileName("test.pdf"),
		document.WithStoragePath("documents/test.pdf"),
		document.WithContentType("application/pdf"),
		document.WithHash("a1b2c3d4e5f60123456789abcdef0123456789abcdef0123456789abcdef0123", "SHA256"),
		document.WithFileSize(1024),
	)
	require.NoError(t, err)

	expectedErr := errors.New("indexing failed")
	mockService.On("IndexDocument", ctx, mock.AnythingOfType("*models.Document")).
		Return("", expectedErr)

	err = adapter.Index(ctx, doc)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "search error")
	mockService.AssertExpectations(t)
}

func TestSearchAdapter_Update_Success(t *testing.T) {
	mockService := new(mockSearchService)
	adapter := NewSearchAdapter(mockService)

	ctx := context.Background()
	docID := "doc_123"
	fields := map[string]interface{}{
		"status": "processed",
	}

	mockService.On("UpdateDocumentMetadata", ctx, docID, fields).
		Return(nil)

	err := adapter.Update(ctx, docID, fields)

	require.NoError(t, err)
	mockService.AssertExpectations(t)
}

func TestSearchAdapter_Update_Error(t *testing.T) {
	mockService := new(mockSearchService)
	adapter := NewSearchAdapter(mockService)

	ctx := context.Background()
	docID := "doc_123"
	fields := map[string]interface{}{
		"status": "processed",
	}

	expectedErr := errors.New("update failed")
	mockService.On("UpdateDocumentMetadata", ctx, docID, fields).
		Return(expectedErr)

	err := adapter.Update(ctx, docID, fields)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "search error")
	mockService.AssertExpectations(t)
}

func TestSearchAdapter_Delete_Success(t *testing.T) {
	mockService := new(mockSearchService)
	adapter := NewSearchAdapter(mockService)

	ctx := context.Background()
	docID := "doc_123"

	mockService.On("DeleteDocument", ctx, docID).
		Return(nil)

	err := adapter.Delete(ctx, docID)

	require.NoError(t, err)
	mockService.AssertExpectations(t)
}

func TestSearchAdapter_Delete_Error(t *testing.T) {
	mockService := new(mockSearchService)
	adapter := NewSearchAdapter(mockService)

	ctx := context.Background()
	docID := "doc_123"

	expectedErr := errors.New("delete failed")
	mockService.On("DeleteDocument", ctx, docID).
		Return(expectedErr)

	err := adapter.Delete(ctx, docID)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "search error")
	mockService.AssertExpectations(t)
}

func TestSearchAdapter_Search_Success(t *testing.T) {
	mockService := new(mockSearchService)
	adapter := NewSearchAdapter(mockService)

	ctx := context.Background()
	query := ports.SearchQuery{
		Query:         "motion",
		DocumentType:  "motion",
		Category:      "filing",
		LegalTags:     []string{"criminal"},
		MinConfidence: 0.8,
		Page:          1,
		PageSize:      20,
		SortBy:        "created_at",
		SortOrder:     "desc",
	}

	mockResult := &models.SearchResult{
		TotalHits: 10,
		MaxScore:  1.0,
		Documents: []*models.SearchDocument{
			{
				ID:    "doc_123",
				Score: 0.95,
				Document: map[string]interface{}{
					"file_name":  "motion.pdf",
					"created_at": time.Now().Format(time.RFC3339),
					"metadata": map[string]interface{}{
						"document_type":  "motion",
						"legal_category": "filing",
						"confidence":     0.95,
					},
				},
			},
		},
	}

	mockService.On("SearchDocuments", ctx, mock.AnythingOfType("*models.SearchRequest")).
		Return(mockResult, nil)

	result, err := adapter.Search(ctx, query)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 10, result.Total)
	assert.Equal(t, 1, result.Page)
	assert.Equal(t, 20, result.PageSize)
	assert.Len(t, result.Hits, 1)
	mockService.AssertExpectations(t)
}

func TestSearchAdapter_Search_Error(t *testing.T) {
	mockService := new(mockSearchService)
	adapter := NewSearchAdapter(mockService)

	ctx := context.Background()
	query := ports.SearchQuery{
		Query:    "motion",
		Page:     1,
		PageSize: 20,
	}

	expectedErr := errors.New("search failed")
	mockService.On("SearchDocuments", ctx, mock.AnythingOfType("*models.SearchRequest")).
		Return(nil, expectedErr)

	result, err := adapter.Search(ctx, query)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "search error")
	mockService.AssertExpectations(t)
}
