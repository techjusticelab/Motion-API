package handlers

import (
	"context"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"motion-index-fiber/internal/application/dto"
	"motion-index-fiber/pkg/models"
	"motion-index-fiber/pkg/search"
)

// Mock search use case and service

type mockSearchUseCase struct {
	result *dto.SearchResultsResponse
	err    error
}

func (m *mockSearchUseCase) Execute(ctx context.Context, req *dto.SearchDocumentsRequest) (*dto.SearchResultsResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.result, nil
}

type mockSearchService struct {
	document *models.Document
	err      error
}

func (m *mockSearchService) GetDocument(ctx context.Context, id string) (*models.Document, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.document, nil
}

func (m *mockSearchService) DeleteDocument(ctx context.Context, id string) error {
	return m.err
}

// Stub methods for interface compliance
func (m *mockSearchService) SearchDocuments(ctx context.Context, req *models.SearchRequest) (*models.SearchResult, error) {
	return nil, nil
}
func (m *mockSearchService) IndexDocument(ctx context.Context, doc *models.Document) (string, error) {
	return "", nil
}
func (m *mockSearchService) UpdateDocumentMetadata(ctx context.Context, docID string, metadata map[string]interface{}) error {
	return nil
}
func (m *mockSearchService) GetLegalTags(ctx context.Context) ([]*models.TagCount, error) {
	return nil, nil
}
func (m *mockSearchService) GetDocumentTypes(ctx context.Context) ([]*models.TypeCount, error) {
	return nil, nil
}
func (m *mockSearchService) GetDocumentStats(ctx context.Context) (*models.DocumentStats, error) {
	return nil, nil
}
func (m *mockSearchService) GetAllFieldOptions(ctx context.Context) (*models.FieldOptions, error) {
	return &models.FieldOptions{}, nil
}
func (m *mockSearchService) GetMetadataFieldValues(ctx context.Context, field string, prefix string, size int) ([]*models.FieldValue, error) {
	return nil, nil
}
func (m *mockSearchService) GetMetadataFieldValuesWithFilters(ctx context.Context, req *models.MetadataFieldValuesRequest) ([]*models.FieldValue, error) {
	return nil, nil
}
func (m *mockSearchService) BulkIndexDocuments(ctx context.Context, docs []*models.Document) (*models.BulkResult, error) {
	return &models.BulkResult{}, nil
}
func (m *mockSearchService) DocumentExists(ctx context.Context, id string) (bool, error) {
	return true, nil
}
func (m *mockSearchService) IsHealthy() bool {
	return true
}
func (m *mockSearchService) Health(ctx context.Context) (*search.HealthStatus, error) {
	return &search.HealthStatus{}, nil
}

// Ensure interface compliance
var _ search.Service = (*mockSearchService)(nil)

// Tests for SearchDocuments

func TestSearchHandlerRefactored_SearchDocuments_Success(t *testing.T) {
	// Given: A handler with successful search use case
	app := fiber.New()
	mockSearch := &mockSearchUseCase{
		result: &dto.SearchResultsResponse{
			Documents: []dto.DocumentSummary{
				{
					ID:           "doc_123",
					FileName:     "test.pdf",
					DocumentType: "motion",
					Category:     "filing",
					Snippet:      "test snippet",
					Confidence:   0.95,
					CreatedAt:    time.Now(),
				},
			},
			Total:      1,
			Page:       1,
			PageSize:   10,
			TotalPages: 1,
		},
	}

	handler := NewSearchHandlerRefactored(mockSearch, &mockSearchService{})
	app.Post("/search", handler.SearchDocuments)

	// When: Valid search request is made
	body := `{"query":"motion","page_size":10}`
	req := httptest.NewRequest("POST", "/search", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Then: Response is successful
	assert.Equal(t, 200, resp.StatusCode)
}

func TestSearchHandlerRefactored_SearchDocuments_WithQueryParams(t *testing.T) {
	// Given: A handler
	app := fiber.New()
	mockSearch := &mockSearchUseCase{
		result: &dto.SearchResultsResponse{
			Documents:  []dto.DocumentSummary{},
			Total:      0,
			Page:       0,
			PageSize:   10,
			TotalPages: 0,
		},
	}

	handler := NewSearchHandlerRefactored(mockSearch, &mockSearchService{})
	app.Post("/search", handler.SearchDocuments)

	// When: Search using query parameters
	req := httptest.NewRequest("POST", "/search?q=motion&size=10", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Then: Response is successful
	assert.Equal(t, 200, resp.StatusCode)
}

func TestSearchHandlerRefactored_SearchDocuments_Error(t *testing.T) {
	// Given: A handler with error
	app := fiber.New()
	mockSearch := &mockSearchUseCase{
		err: errors.New("search service unavailable"),
	}

	handler := NewSearchHandlerRefactored(mockSearch, &mockSearchService{})
	app.Post("/search", handler.SearchDocuments)

	// When: Search fails
	body := `{"query":"motion"}`
	req := httptest.NewRequest("POST", "/search", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Then: Response is 500 Internal Server Error
	assert.Equal(t, 500, resp.StatusCode)
}

// Tests for GetDocument

func TestSearchHandlerRefactored_GetDocument_Success(t *testing.T) {
	// Given: A handler with document
	app := fiber.New()
	mockService := &mockSearchService{
		document: &models.Document{
			ID:       "doc_123",
			FileName: "test.pdf",
			Text:     "test content",
		},
	}

	handler := NewSearchHandlerRefactored(&mockSearchUseCase{}, mockService)
	app.Get("/documents/:id", handler.GetDocument)

	// When: Valid document ID is requested
	req := httptest.NewRequest("GET", "/documents/doc_123", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Then: Response is successful
	assert.Equal(t, 200, resp.StatusCode)
}

func TestSearchHandlerRefactored_GetDocument_NotFound(t *testing.T) {
	// Given: A handler with document not found error
	app := fiber.New()
	mockService := &mockSearchService{
		err: errors.New("document not found"),
	}

	handler := NewSearchHandlerRefactored(&mockSearchUseCase{}, mockService)
	app.Get("/documents/:id", handler.GetDocument)

	// When: Document doesn't exist
	req := httptest.NewRequest("GET", "/documents/nonexistent", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Then: Response is 404 Not Found
	assert.Equal(t, 404, resp.StatusCode)
}

func TestSearchHandlerRefactored_GetDocument_MissingID(t *testing.T) {
	// Given: A handler
	app := fiber.New()
	handler := NewSearchHandlerRefactored(&mockSearchUseCase{}, &mockSearchService{})
	app.Get("/documents/:id", handler.GetDocument)

	// When: Empty ID is provided
	req := httptest.NewRequest("GET", "/documents/", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Then: Response is 404 (route not found) or 400
	assert.True(t, resp.StatusCode >= 400)
}

// Tests for DeleteDocument

func TestSearchHandlerRefactored_DeleteDocument_Success(t *testing.T) {
	// Given: A handler with successful deletion
	app := fiber.New()
	mockService := &mockSearchService{}

	handler := NewSearchHandlerRefactored(&mockSearchUseCase{}, mockService)
	app.Delete("/documents/:id", handler.DeleteDocument)

	// When: Valid document ID is deleted
	req := httptest.NewRequest("DELETE", "/documents/doc_123", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Then: Response is successful
	assert.Equal(t, 200, resp.StatusCode)
}

func TestSearchHandlerRefactored_DeleteDocument_Error(t *testing.T) {
	// Given: A handler with deletion error
	app := fiber.New()
	mockService := &mockSearchService{
		err: errors.New("failed to delete"),
	}

	handler := NewSearchHandlerRefactored(&mockSearchUseCase{}, mockService)
	app.Delete("/documents/:id", handler.DeleteDocument)

	// When: Deletion fails
	req := httptest.NewRequest("DELETE", "/documents/doc_123", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Then: Response is 500 Internal Server Error
	assert.Equal(t, 500, resp.StatusCode)
}

func TestSearchHandlerRefactored_DeleteDocument_MissingID(t *testing.T) {
	// Given: A handler
	app := fiber.New()
	handler := NewSearchHandlerRefactored(&mockSearchUseCase{}, &mockSearchService{})
	app.Delete("/documents/:id", handler.DeleteDocument)

	// When: Empty ID is provided
	req := httptest.NewRequest("DELETE", "/documents/", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Then: Response is 404 (route not found)
	assert.True(t, resp.StatusCode >= 400)
}

// Integration test

func TestSearchHandlerRefactored_Integration(t *testing.T) {
	// Given: A fully configured handler
	app := fiber.New()

	mockSearch := &mockSearchUseCase{
		result: &dto.SearchResultsResponse{
			Documents:  []dto.DocumentSummary{},
			Total:      0,
			Page:       1,
			PageSize:   10,
			TotalPages: 0,
		},
	}

	mockService := &mockSearchService{
		document: &models.Document{
			ID:       "doc_123",
			FileName: "test.pdf",
		},
	}

	handler := NewSearchHandlerRefactored(mockSearch, mockService)

	// Register routes
	app.Post("/search", handler.SearchDocuments)
	app.Get("/documents/:id", handler.GetDocument)
	app.Delete("/documents/:id", handler.DeleteDocument)

	t.Run("search documents", func(t *testing.T) {
		body := `{"query":"motion"}`
		req := httptest.NewRequest("POST", "/search", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
	})

	t.Run("get document", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/documents/doc_123", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
	})

	t.Run("delete document", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/documents/doc_123", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
	})
}
