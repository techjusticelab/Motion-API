package handlers

import (
	"context"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"motion-index-fiber/internal/application/dto"
)

// Mock indexing use case

type mockIndexDocumentUseCase struct {
	result *dto.IndexDocumentResponse
	err    error
}

func (m *mockIndexDocumentUseCase) Execute(ctx context.Context, req *dto.IndexDocumentRequest) (*dto.IndexDocumentResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.result, nil
}

// Tests for IndexDocument

func TestIndexingHandlerRefactored_IndexDocument_Success(t *testing.T) {
	// Given: A handler with successful indexing use case
	app := fiber.New()
	mockIndex := &mockIndexDocumentUseCase{
		result: &dto.IndexDocumentResponse{
			DocumentID: "doc_123",
			Indexed:    true,
		},
	}

	handler := NewIndexingHandlerRefactored(mockIndex)
	app.Post("/api/v1/index/document", handler.IndexDocument)

	// When: Valid indexing request is made
	body := `{"document_id":"doc_123","force":false}`
	req := httptest.NewRequest("POST", "/api/v1/index/document", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Then: Response is successful
	assert.Equal(t, 200, resp.StatusCode)
}

func TestIndexingHandlerRefactored_IndexDocument_InvalidJSON(t *testing.T) {
	// Given: A handler
	app := fiber.New()
	handler := NewIndexingHandlerRefactored(&mockIndexDocumentUseCase{})
	app.Post("/api/v1/index/document", handler.IndexDocument)

	// When: Invalid JSON is provided
	req := httptest.NewRequest("POST", "/api/v1/index/document", strings.NewReader("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Then: Response is 400 Bad Request
	assert.Equal(t, 400, resp.StatusCode)
}

func TestIndexingHandlerRefactored_IndexDocument_NotFound(t *testing.T) {
	// Given: A handler with document not found error
	app := fiber.New()
	mockIndex := &mockIndexDocumentUseCase{
		err: errors.New("document not found"),
	}

	handler := NewIndexingHandlerRefactored(mockIndex)
	app.Post("/api/v1/index/document", handler.IndexDocument)

	// When: Document doesn't exist
	body := `{"document_id":"nonexistent","force":false}`
	req := httptest.NewRequest("POST", "/api/v1/index/document", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Then: Response is 404 Not Found
	assert.Equal(t, 404, resp.StatusCode)
}

func TestIndexingHandlerRefactored_IndexDocument_InternalError(t *testing.T) {
	// Given: A handler with internal error
	app := fiber.New()
	mockIndex := &mockIndexDocumentUseCase{
		err: errors.New("search service unavailable"),
	}

	handler := NewIndexingHandlerRefactored(mockIndex)
	app.Post("/api/v1/index/document", handler.IndexDocument)

	// When: Use case fails
	body := `{"document_id":"doc_123","force":false}`
	req := httptest.NewRequest("POST", "/api/v1/index/document", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Then: Response is 500 Internal Server Error
	assert.Equal(t, 500, resp.StatusCode)
}

func TestIndexingHandlerRefactored_IndexDocument_WithForceFlag(t *testing.T) {
	// Given: A handler with successful use case
	app := fiber.New()
	mockIndex := &mockIndexDocumentUseCase{
		result: &dto.IndexDocumentResponse{
			DocumentID: "doc_123",
			Indexed:    true,
		},
	}

	handler := NewIndexingHandlerRefactored(mockIndex)
	app.Post("/api/v1/index/document", handler.IndexDocument)

	// When: Force reindex is requested
	body := `{"document_id":"doc_123","force":true}`
	req := httptest.NewRequest("POST", "/api/v1/index/document", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Then: Response is successful
	assert.Equal(t, 200, resp.StatusCode)
}

func TestIndexingHandlerRefactored_IndexDocument_EmptyDocumentID(t *testing.T) {
	// Given: A handler that returns validation error for empty ID
	app := fiber.New()
	mockIndex := &mockIndexDocumentUseCase{
		err: errors.New("documentId: document ID cannot be empty"),
	}
	handler := NewIndexingHandlerRefactored(mockIndex)
	app.Post("/api/v1/index/document", handler.IndexDocument)

	// When: Empty document ID is provided
	body := `{"document_id":"","force":false}`
	req := httptest.NewRequest("POST", "/api/v1/index/document", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Then: Response is 500 Internal Server Error (validation happens in use case)
	assert.Equal(t, 500, resp.StatusCode)
}

// Test isNotFoundError helper

func TestIsNotFoundError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "document not found",
			err:      errors.New("document not found"),
			expected: true,
		},
		{
			name:     "document with that ID does not exist",
			err:      errors.New("document with that ID does not exist"),
			expected: true,
		},
		{
			name:     "other error",
			err:      errors.New("some other error"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isNotFoundError(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Integration test

func TestIndexingHandlerRefactored_Integration(t *testing.T) {
	// Given: A fully configured handler
	app := fiber.New()

	handler := NewIndexingHandlerRefactored(&mockIndexDocumentUseCase{
		result: &dto.IndexDocumentResponse{
			DocumentID: "doc_123",
			Indexed:    true,
		},
	})

	app.Post("/api/v1/index/document", handler.IndexDocument)

	// When: Valid request is made
	body := `{"document_id":"doc_123","force":false}`
	req := httptest.NewRequest("POST", "/api/v1/index/document", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)

	// Then: Response is successful
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}
