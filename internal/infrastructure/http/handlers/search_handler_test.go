package handlers

import (
	"context"
	"encoding/json"
	"io"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"motion-index-fiber/internal/application/dto"
	"motion-index-fiber/internal/infrastructure/http/presenter"
	"motion-index-fiber/pkg/models"
)

// mockSearchUseCase is a mock implementation of SearchDocumentsUseCase
type mockSearchUseCase struct {
	mock.Mock
}

func (m *mockSearchUseCase) Execute(ctx context.Context, req *dto.SearchDocumentsRequest) (*dto.SearchResultsResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.SearchResultsResponse), args.Error(1)
}

func TestNewSearchHandler(t *testing.T) {
	// Given: Mock dependencies
	mockUseCase := new(mockSearchUseCase)
	mockService := new(mockSearchService)

	// When: Handler is created
	handler := NewSearchHandler(mockUseCase, mockService)

	// Then: Handler is properly initialized
	assert.NotNil(t, handler)
	assert.Equal(t, mockUseCase, handler.searchUseCase)
	assert.Equal(t, mockService, handler.searchService)
}

func TestSearchHandler_SearchDocuments_Success(t *testing.T) {
	// Given: Mock use case with successful response
	mockUseCase := new(mockSearchUseCase)
	now := time.Now()
	mockUseCase.On("Execute", mock.Anything, mock.AnythingOfType("*dto.SearchDocumentsRequest")).
		Return(&dto.SearchResultsResponse{
			Documents: []dto.DocumentSummary{
				{
					ID:           "doc_1",
					FileName:     "test.pdf",
					DocumentType: "motion",
					CreatedAt:    now,
				},
			},
			Total:      1,
			Page:       1,
			PageSize:   20,
			TotalPages: 1,
		}, nil)

	handler := NewSearchHandler(mockUseCase, new(mockSearchService))
	app := fiber.New()
	app.Get("/search", handler.SearchDocuments)

	// When: Request is made
	req := httptest.NewRequest("GET", "/search?q=motion&page=1&page_size=20", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Then: Search results are returned
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var response presenter.Response
	body, _ := io.ReadAll(resp.Body)
	err = json.Unmarshal(body, &response)
	require.NoError(t, err)

	assert.True(t, response.Success)
	data := response.Data.(map[string]interface{})
	documents := data["documents"].([]interface{})
	assert.Len(t, documents, 1)
	assert.Equal(t, float64(1), data["total"])

	mockUseCase.AssertExpectations(t)
}

func TestSearchHandler_SearchDocuments_Error(t *testing.T) {
	// Given: Mock use case with error
	mockUseCase := new(mockSearchUseCase)
	mockUseCase.On("Execute", mock.Anything, mock.AnythingOfType("*dto.SearchDocumentsRequest")).
		Return(nil, fiber.NewError(fiber.StatusInternalServerError, "search failed"))

	handler := NewSearchHandler(mockUseCase, new(mockSearchService))
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		err := c.Next()
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(map[string]string{
				"error": err.Error(),
			})
		}
		return nil
	})
	app.Get("/search", handler.SearchDocuments)

	// When: Request is made
	req := httptest.NewRequest("GET", "/search?q=motion", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Then: Error response is returned
	assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)

	mockUseCase.AssertExpectations(t)
}

func TestSearchHandler_GetLegalTags_Success(t *testing.T) {
	// Given: Mock service with tags
	mockService := new(mockSearchService)
	mockService.On("GetLegalTags", mock.Anything).
		Return([]*models.TagCount{
			{Tag: "criminal", Count: 10},
			{Tag: "civil", Count: 5},
		}, nil)

	handler := NewSearchHandler(new(mockSearchUseCase), mockService)
	app := fiber.New()
	app.Get("/legal-tags", handler.GetLegalTags)

	// When: Request is made
	req := httptest.NewRequest("GET", "/legal-tags", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Then: Tags are returned
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var response presenter.Response
	body, _ := io.ReadAll(resp.Body)
	err = json.Unmarshal(body, &response)
	require.NoError(t, err)

	assert.True(t, response.Success)
	data := response.Data.(map[string]interface{})
	tags := data["tags"].([]interface{})
	assert.Len(t, tags, 2)

	mockService.AssertExpectations(t)
}

func TestSearchHandler_GetDocumentTypes_Success(t *testing.T) {
	// Given: Mock service with types
	mockService := new(mockSearchService)
	mockService.On("GetDocumentTypes", mock.Anything).
		Return([]*models.TypeCount{
			{Type: "motion", Count: 15},
			{Type: "brief", Count: 8},
		}, nil)

	handler := NewSearchHandler(new(mockSearchUseCase), mockService)
	app := fiber.New()
	app.Get("/document-types", handler.GetDocumentTypes)

	// When: Request is made
	req := httptest.NewRequest("GET", "/document-types", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Then: Types are returned
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var response presenter.Response
	body, _ := io.ReadAll(resp.Body)
	err = json.Unmarshal(body, &response)
	require.NoError(t, err)

	assert.True(t, response.Success)
	data := response.Data.(map[string]interface{})
	types := data["types"].([]interface{})
	assert.Len(t, types, 2)

	mockService.AssertExpectations(t)
}

func TestSearchHandler_GetDocumentStats_Success(t *testing.T) {
	// Given: Mock service with stats
	mockService := new(mockSearchService)
	mockService.On("GetDocumentStats", mock.Anything).
		Return(&models.DocumentStats{
			TotalDocuments: 100,
			IndexSize:      "1.0 MB",
		}, nil)

	handler := NewSearchHandler(new(mockSearchUseCase), mockService)
	app := fiber.New()
	app.Get("/document-stats", handler.GetDocumentStats)

	// When: Request is made
	req := httptest.NewRequest("GET", "/document-stats", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Then: Stats are returned
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var response presenter.Response
	body, _ := io.ReadAll(resp.Body)
	err = json.Unmarshal(body, &response)
	require.NoError(t, err)

	assert.True(t, response.Success)

	mockService.AssertExpectations(t)
}

func TestSearchHandler_GetFieldOptions_Success(t *testing.T) {
	// Given: Mock service with field options
	mockService := new(mockSearchService)
	mockService.On("GetAllFieldOptions", mock.Anything).
		Return(&models.FieldOptions{
			DocTypes: []*models.FieldValue{
				{Value: "motion", Count: 10},
				{Value: "brief", Count: 5},
			},
			LegalTags: []*models.FieldValue{
				{Value: "criminal", Count: 15},
				{Value: "civil", Count: 8},
			},
		}, nil)

	handler := NewSearchHandler(new(mockSearchUseCase), mockService)
	app := fiber.New()
	app.Get("/field-options", handler.GetFieldOptions)

	// When: Request is made
	req := httptest.NewRequest("GET", "/field-options", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Then: Field options are returned
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var response presenter.Response
	body, _ := io.ReadAll(resp.Body)
	err = json.Unmarshal(body, &response)
	require.NoError(t, err)

	assert.True(t, response.Success)

	mockService.AssertExpectations(t)
}

func TestSearchHandler_GetMetadataFieldValues_Success(t *testing.T) {
	// Given: Mock service with field values
	mockService := new(mockSearchService)
	mockService.On("GetMetadataFieldValues", mock.Anything, "case_name", "", 50).
		Return([]*models.FieldValue{
			{Value: "State v. Doe", Count: 5},
		}, nil)

	handler := NewSearchHandler(new(mockSearchUseCase), mockService)
	app := fiber.New()
	app.Get("/metadata-fields/:field", handler.GetMetadataFieldValues)

	// When: Request is made
	req := httptest.NewRequest("GET", "/metadata-fields/case_name", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Then: Field values are returned
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var response presenter.Response
	body, _ := io.ReadAll(resp.Body)
	err = json.Unmarshal(body, &response)
	require.NoError(t, err)

	assert.True(t, response.Success)
	data := response.Data.(map[string]interface{})
	assert.Equal(t, "case_name", data["field"])

	mockService.AssertExpectations(t)
}

func TestSearchHandler_GetMetadataFieldValues_MissingField(t *testing.T) {
	// Given: Handler
	handler := NewSearchHandler(new(mockSearchUseCase), new(mockSearchService))
	app := fiber.New()
	app.Get("/metadata-fields/:field", handler.GetMetadataFieldValues)

	// When: Request is made without field
	req := httptest.NewRequest("GET", "/metadata-fields/", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Then: Error is returned
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
}

func TestSearchHandler_PostMetadataFieldValues_Success(t *testing.T) {
	// Given: Mock service with field values
	mockService := new(mockSearchService)
	mockService.On("GetMetadataFieldValuesWithFilters", mock.Anything, mock.AnythingOfType("*models.MetadataFieldValuesRequest")).
		Return([]*models.FieldValue{
			{Value: "criminal", Count: 10},
		}, nil)

	handler := NewSearchHandler(new(mockSearchUseCase), mockService)
	app := fiber.New()
	app.Post("/metadata-field-values", handler.PostMetadataFieldValues)

	// When: Request is made
	body := strings.NewReader(`{"field": "legal_tags", "size": 10}`)
	req := httptest.NewRequest("POST", "/metadata-field-values", body)
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Then: Field values are returned
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var response presenter.Response
	respBody, _ := io.ReadAll(resp.Body)
	err = json.Unmarshal(respBody, &response)
	require.NoError(t, err)

	assert.True(t, response.Success)
	data := response.Data.(map[string]interface{})
	assert.Equal(t, "legal_tags", data["field"])

	mockService.AssertExpectations(t)
}

func TestSearchHandler_GetDocument_Success(t *testing.T) {
	// Given: Mock service with document
	mockService := new(mockSearchService)
	mockService.On("GetDocument", mock.Anything, "doc_123").
		Return(&models.Document{
			ID:       "doc_123",
			FileName: "test.pdf",
		}, nil)

	handler := NewSearchHandler(new(mockSearchUseCase), mockService)
	app := fiber.New()
	app.Get("/documents/:id", handler.GetDocument)

	// When: Request is made
	req := httptest.NewRequest("GET", "/documents/doc_123", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Then: Document is returned
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var response presenter.Response
	body, _ := io.ReadAll(resp.Body)
	err = json.Unmarshal(body, &response)
	require.NoError(t, err)

	assert.True(t, response.Success)

	mockService.AssertExpectations(t)
}

func TestSearchHandler_DeleteDocument_Success(t *testing.T) {
	// Given: Mock service
	mockService := new(mockSearchService)
	mockService.On("DeleteDocument", mock.Anything, "doc_123").
		Return(nil)

	handler := NewSearchHandler(new(mockSearchUseCase), mockService)
	app := fiber.New()
	app.Delete("/documents/:id", handler.DeleteDocument)

	// When: Request is made
	req := httptest.NewRequest("DELETE", "/documents/doc_123", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Then: Success response is returned
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var response presenter.Response
	body, _ := io.ReadAll(resp.Body)
	err = json.Unmarshal(body, &response)
	require.NoError(t, err)

	assert.True(t, response.Success)
	data := response.Data.(map[string]interface{})
	assert.Equal(t, "Document deleted successfully", data["message"])

	mockService.AssertExpectations(t)
}
