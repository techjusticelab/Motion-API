package handlers

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"motion-index-fiber/internal/application/dto"
)

// Mock use cases for testing

type mockProcessUseCase struct {
	result *dto.ProcessDocumentResponse
	err    error
}

func (m *mockProcessUseCase) Execute(ctx context.Context, req *dto.ProcessDocumentRequest) (*dto.ProcessDocumentResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.result, nil
}

type mockBatchProcessUseCase struct {
	result *dto.BatchProcessDocumentResponse
	err    error
}

func (m *mockBatchProcessUseCase) Execute(ctx context.Context, req *dto.BatchProcessDocumentRequest) (*dto.BatchProcessDocumentResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.result, nil
}

type mockUpdateMetadataUseCase struct {
	result *dto.UpdateMetadataResponse
	err    error
}

func (m *mockUpdateMetadataUseCase) Execute(ctx context.Context, req *dto.UpdateMetadataRequest) (*dto.UpdateMetadataResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.result, nil
}

type mockAnalyzeRedactionsUseCase struct {
	result *dto.AnalyzeRedactionsResponse
	err    error
}

func (m *mockAnalyzeRedactionsUseCase) Execute(ctx context.Context, req *dto.AnalyzeRedactionsRequest) (*dto.AnalyzeRedactionsResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.result, nil
}

type mockApplyRedactionsUseCase struct {
	result *dto.ApplyRedactionsResponse
	err    error
}

func (m *mockApplyRedactionsUseCase) Execute(ctx context.Context, req *dto.ApplyRedactionsRequest) (*dto.ApplyRedactionsResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.result, nil
}

// Tests for ProcessDocument

func TestProcessingHandlerRefactored_ProcessDocument_Success(t *testing.T) {
	// Given: A handler with successful use case
	app := fiber.New()
	mockProcess := &mockProcessUseCase{
		result: &dto.ProcessDocumentResponse{
			DocumentID:  "doc_123",
			Stored:      true,
			StorageURL:  "https://storage.example.com/doc_123.pdf",
			Classified:  true,
			Indexed:     true,
			CreatedAt:   time.Now(),
			ProcessedAt: timePtr(time.Now()),
		},
	}

	handler := &ProcessingHandlerRefactored{
		processUC: mockProcess,
	}

	app.Post("/api/documents/process", handler.ProcessDocument)

	// When: Valid document is uploaded
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "test.pdf")
	part.Write([]byte("test content"))
	writer.Close()

	req := httptest.NewRequest("POST", "/api/documents/process", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Then: Response is successful
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestProcessingHandlerRefactored_ProcessDocument_MissingFile(t *testing.T) {
	// Given: A handler
	app := fiber.New()
	handler := &ProcessingHandlerRefactored{
		processUC: &mockProcessUseCase{},
	}

	app.Post("/api/documents/process", handler.ProcessDocument)

	// When: No file is provided
	req := httptest.NewRequest("POST", "/api/documents/process", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Then: Response is 400 Bad Request
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// Tests for BatchProcessDocuments

func TestProcessingHandlerRefactored_BatchProcess_Success(t *testing.T) {
	// Given: A handler with successful batch use case
	app := fiber.New()
	mockBatch := &mockBatchProcessUseCase{
		result: &dto.BatchProcessDocumentResponse{
			Total:     2,
			Succeeded: 2,
			Failed:    0,
			Errors:    nil,
		},
	}

	handler := &ProcessingHandlerRefactored{
		batchUC: mockBatch,
	}

	app.Post("/api/documents/batch", handler.BatchProcessDocuments)

	// When: Multiple files are uploaded
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part1, _ := writer.CreateFormFile("files", "test1.pdf")
	part1.Write([]byte("content 1"))
	part2, _ := writer.CreateFormFile("files", "test2.pdf")
	part2.Write([]byte("content 2"))
	writer.Close()

	req := httptest.NewRequest("POST", "/api/documents/batch", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Then: Response is successful
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestProcessingHandlerRefactored_BatchProcess_NoFiles(t *testing.T) {
	// Given: A handler
	app := fiber.New()
	handler := &ProcessingHandlerRefactored{
		batchUC: &mockBatchProcessUseCase{},
	}

	app.Post("/api/documents/batch", handler.BatchProcessDocuments)

	// When: No files are provided
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	writer.Close()

	req := httptest.NewRequest("POST", "/api/documents/batch", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Then: Response is 400 Bad Request
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// Tests for UpdateMetadata

func TestProcessingHandlerRefactored_UpdateMetadata_Success(t *testing.T) {
	// Given: A handler with successful metadata use case
	app := fiber.New()
	mockUpdate := &mockUpdateMetadataUseCase{
		result: &dto.UpdateMetadataResponse{
			DocumentID:   "doc_123",
			Language:     "en",
			LegalTags:    []string{"motion", "suppress"},
			AIClassified: true,
		},
	}

	handler := &ProcessingHandlerRefactored{
		updateMetadataUC: mockUpdate,
	}

	app.Put("/api/documents/:id/metadata", handler.UpdateMetadata)

	// When: Valid metadata is provided
	body := `{"language":"en","legal_tags":["motion"],"ai_classified":true}`
	req := httptest.NewRequest("PUT", "/api/documents/doc_123/metadata", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Then: Response is successful
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestProcessingHandlerRefactored_UpdateMetadata_InvalidJSON(t *testing.T) {
	// Given: A handler
	app := fiber.New()
	handler := &ProcessingHandlerRefactored{
		updateMetadataUC: &mockUpdateMetadataUseCase{},
	}

	app.Put("/api/documents/:id/metadata", handler.UpdateMetadata)

	// When: Invalid JSON is provided
	req := httptest.NewRequest("PUT", "/api/documents/doc_123/metadata", strings.NewReader("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Then: Response is 400 Bad Request
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// Tests for AnalyzeRedactions

func TestProcessingHandlerRefactored_AnalyzeRedactions_WithFile(t *testing.T) {
	// Given: A handler with successful analysis use case
	app := fiber.New()
	mockAnalyze := &mockAnalyzeRedactionsUseCase{
		result: &dto.AnalyzeRedactionsResponse{
			DocumentID: "doc_123",
			FileName:   "test.pdf",
			Redactions: []dto.RedactionItem{
				{
					ID:   "redact_1",
					Page: 1,
					Text: "sensitive info",
					Type: "ssn",
				},
			},
			TotalCount: 1,
		},
	}

	handler := &ProcessingHandlerRefactored{
		analyzeRedactionUC: mockAnalyze,
	}

	app.Post("/api/documents/redactions/analyze", handler.AnalyzeRedactions)

	// When: File is provided for analysis
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "test.pdf")
	part.Write([]byte("test content"))
	writer.Close()

	req := httptest.NewRequest("POST", "/api/documents/redactions/analyze", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Then: Response is successful
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestProcessingHandlerRefactored_AnalyzeRedactions_WithJSON(t *testing.T) {
	// Given: A handler with successful analysis use case
	app := fiber.New()
	mockAnalyze := &mockAnalyzeRedactionsUseCase{
		result: &dto.AnalyzeRedactionsResponse{
			DocumentID: "doc_123",
			TotalCount: 0,
		},
	}

	handler := &ProcessingHandlerRefactored{
		analyzeRedactionUC: mockAnalyze,
	}

	app.Post("/api/documents/redactions/analyze", handler.AnalyzeRedactions)

	// When: JSON request with document ID
	body := `{"document_id":"doc_123"}`
	req := httptest.NewRequest("POST", "/api/documents/redactions/analyze", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Then: Response is successful
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// Tests for ApplyRedactions

func TestProcessingHandlerRefactored_ApplyRedactions_Success(t *testing.T) {
	// Given: A handler with successful apply use case
	app := fiber.New()
	mockApply := &mockApplyRedactionsUseCase{
		result: &dto.ApplyRedactionsResponse{
			DocumentID:      "doc_123",
			FileName:        "test.pdf",
			TotalRedactions: 1,
			PDFBase64:       "base64content",
		},
	}

	handler := &ProcessingHandlerRefactored{
		applyRedactionUC: mockApply,
	}

	app.Post("/api/documents/redactions/apply", handler.ApplyRedactions)

	// When: Valid redaction request
	body := `{"document_id":"doc_123","custom_redactions":[]}`
	req := httptest.NewRequest("POST", "/api/documents/redactions/apply", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Then: Response is successful
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestProcessingHandlerRefactored_ApplyRedactions_InvalidJSON(t *testing.T) {
	// Given: A handler
	app := fiber.New()
	handler := &ProcessingHandlerRefactored{
		applyRedactionUC: &mockApplyRedactionsUseCase{},
	}

	app.Post("/api/documents/redactions/apply", handler.ApplyRedactions)

	// When: Invalid JSON
	req := httptest.NewRequest("POST", "/api/documents/redactions/apply", strings.NewReader("invalid"))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Then: Response is 400 Bad Request
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// Integration test

func TestProcessingHandlerRefactored_Integration(t *testing.T) {
	// Given: A fully configured handler
	app := fiber.New()

	handler := &ProcessingHandlerRefactored{
		processUC: &mockProcessUseCase{
			result: &dto.ProcessDocumentResponse{
				DocumentID: "doc_123",
				Stored:     true,
				Classified: true,
				Indexed:    true,
				CreatedAt:  time.Now(),
			},
		},
		batchUC: &mockBatchProcessUseCase{
			result: &dto.BatchProcessDocumentResponse{
				Total:     1,
				Succeeded: 1,
				Failed:    0,
			},
		},
		updateMetadataUC: &mockUpdateMetadataUseCase{
			result: &dto.UpdateMetadataResponse{
				DocumentID: "doc_123",
				Language:   "en",
			},
		},
	}

	// Register routes
	app.Post("/api/documents/process", handler.ProcessDocument)
	app.Post("/api/documents/batch", handler.BatchProcessDocuments)
	app.Put("/api/documents/:id/metadata", handler.UpdateMetadata)

	t.Run("process document", func(t *testing.T) {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		part, _ := writer.CreateFormFile("file", "test.pdf")
		part.Write([]byte("content"))
		writer.Close()

		req := httptest.NewRequest("POST", "/api/documents/process", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("batch process", func(t *testing.T) {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		part, _ := writer.CreateFormFile("files", "test.pdf")
		part.Write([]byte("content"))
		writer.Close()

		req := httptest.NewRequest("POST", "/api/documents/batch", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("update metadata", func(t *testing.T) {
		body := `{"language":"en","legal_tags":[],"ai_classified":false}`
		req := httptest.NewRequest("PUT", "/api/documents/doc_123/metadata", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}

// Helper functions

func timePtr(t time.Time) *time.Time {
	return &t
}
