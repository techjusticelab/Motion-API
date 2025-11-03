package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
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
)

// mockProcessUseCase is a mock implementation of ProcessDocumentUseCase
type mockProcessUseCase struct {
	mock.Mock
}

func (m *mockProcessUseCase) Execute(ctx context.Context, req *dto.ProcessDocumentRequest) (*dto.ProcessDocumentResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.ProcessDocumentResponse), args.Error(1)
}

// mockBatchProcessUseCase is a mock implementation of BatchProcessDocumentUseCase
type mockBatchProcessUseCase struct {
	mock.Mock
}

func (m *mockBatchProcessUseCase) Execute(ctx context.Context, req *dto.BatchProcessDocumentRequest) (*dto.BatchProcessDocumentResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.BatchProcessDocumentResponse), args.Error(1)
}

// mockIndexUseCase is a mock implementation of IndexDocumentUseCase
type mockIndexUseCase struct {
	mock.Mock
}

func (m *mockIndexUseCase) Execute(ctx context.Context, req *dto.IndexDocumentRequest) (*dto.IndexDocumentResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.IndexDocumentResponse), args.Error(1)
}

// mockUpdateMetadataUseCase is a mock implementation of UpdateMetadataUseCase
type mockUpdateMetadataUseCase struct {
	mock.Mock
}

func (m *mockUpdateMetadataUseCase) Execute(ctx context.Context, req *dto.UpdateMetadataRequest) (*dto.UpdateMetadataResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.UpdateMetadataResponse), args.Error(1)
}

func TestNewDocumentHandler(t *testing.T) {
	// Given: Mock use cases
	mockProcess := new(mockProcessUseCase)
	mockBatch := new(mockBatchProcessUseCase)
	mockIndex := new(mockIndexUseCase)
	mockUpdate := new(mockUpdateMetadataUseCase)

	// When: Handler is created
	handler := NewDocumentHandler(mockProcess, mockBatch, mockIndex, mockUpdate)

	// Then: Handler is properly initialized
	assert.NotNil(t, handler)
	assert.Equal(t, mockProcess, handler.processUseCase)
	assert.Equal(t, mockBatch, handler.batchProcessUseCase)
	assert.Equal(t, mockIndex, handler.indexUseCase)
	assert.Equal(t, mockUpdate, handler.updateMetadataUseCase)
}

func TestDocumentHandler_ProcessDocument_Success(t *testing.T) {
	// Given: Mock process use case with successful response
	mockProcess := new(mockProcessUseCase)
	now := time.Now()
	mockProcess.On("Execute", mock.Anything, mock.AnythingOfType("*dto.ProcessDocumentRequest")).
		Return(&dto.ProcessDocumentResponse{
			DocumentID:  "doc_123",
			Stored:      true,
			StorageURL:  "https://storage.example.com/doc_123",
			Classified:  true,
			Indexed:     true,
			CreatedAt:   now,
			ProcessedAt: &now,
		}, nil)

	handler := NewDocumentHandler(mockProcess, new(mockBatchProcessUseCase), new(mockIndexUseCase), new(mockUpdateMetadataUseCase))
	app := fiber.New()
	app.Post("/documents", handler.ProcessDocument)

	// When: Request is made with file upload
	body, contentType := createMultipartForm(map[string]string{
		"case_name": "Test Case",
	}, map[string][]byte{
		"file": []byte("test document content"),
	})

	req := httptest.NewRequest("POST", "/documents", body)
	req.Header.Set("Content-Type", contentType)
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Then: Document is processed successfully
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var response presenter.Response
	respBody, _ := io.ReadAll(resp.Body)
	err = json.Unmarshal(respBody, &response)
	require.NoError(t, err)

	assert.True(t, response.Success)
	mockProcess.AssertExpectations(t)
}

func TestDocumentHandler_ProcessDocument_MissingFile(t *testing.T) {
	// Given: Handler
	handler := NewDocumentHandler(new(mockProcessUseCase), new(mockBatchProcessUseCase), new(mockIndexUseCase), new(mockUpdateMetadataUseCase))
	app := fiber.New()
	app.Post("/documents", handler.ProcessDocument)

	// When: Request is made without file
	req := httptest.NewRequest("POST", "/documents", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Then: Error response is returned
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestDocumentHandler_IndexDocument_Success(t *testing.T) {
	// Given: Mock index use case
	mockIndex := new(mockIndexUseCase)
	mockIndex.On("Execute", mock.Anything, mock.MatchedBy(func(req *dto.IndexDocumentRequest) bool {
		return req.DocumentID == "doc_123" && !req.Force
	})).Return(&dto.IndexDocumentResponse{
		DocumentID: "doc_123",
		Indexed:    true,
	}, nil)

	handler := NewDocumentHandler(new(mockProcessUseCase), new(mockBatchProcessUseCase), mockIndex, new(mockUpdateMetadataUseCase))
	app := fiber.New()
	app.Post("/documents/:id/index", handler.IndexDocument)

	// When: Request is made
	req := httptest.NewRequest("POST", "/documents/doc_123/index", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Then: Document is indexed
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var response presenter.Response
	body, _ := io.ReadAll(resp.Body)
	err = json.Unmarshal(body, &response)
	require.NoError(t, err)

	assert.True(t, response.Success)
	mockIndex.AssertExpectations(t)
}

func TestDocumentHandler_IndexDocument_WithForce(t *testing.T) {
	// Given: Mock index use case with force=true
	mockIndex := new(mockIndexUseCase)
	mockIndex.On("Execute", mock.Anything, mock.MatchedBy(func(req *dto.IndexDocumentRequest) bool {
		return req.DocumentID == "doc_123" && req.Force
	})).Return(&dto.IndexDocumentResponse{
		DocumentID: "doc_123",
		Indexed:    true,
	}, nil)

	handler := NewDocumentHandler(new(mockProcessUseCase), new(mockBatchProcessUseCase), mockIndex, new(mockUpdateMetadataUseCase))
	app := fiber.New()
	app.Post("/documents/:id/index", handler.IndexDocument)

	// When: Request is made with force parameter
	body := strings.NewReader(`{"force": true}`)
	req := httptest.NewRequest("POST", "/documents/doc_123/index", body)
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Then: Document is force indexed
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	mockIndex.AssertExpectations(t)
}

func TestDocumentHandler_UpdateMetadata_Success(t *testing.T) {
	// Given: Mock update metadata use case
	mockUpdate := new(mockUpdateMetadataUseCase)
	now := time.Now()
	mockUpdate.On("Execute", mock.Anything, mock.MatchedBy(func(req *dto.UpdateMetadataRequest) bool {
		return req.DocumentID == "doc_123" && req.Language == "en"
	})).Return(&dto.UpdateMetadataResponse{
		DocumentID:   "doc_123",
		Language:     "en",
		LegalTags:    []string{"motion", "evidence"},
		AIClassified: true,
		ProcessedAt:  &now,
	}, nil)

	handler := NewDocumentHandler(new(mockProcessUseCase), new(mockBatchProcessUseCase), new(mockIndexUseCase), mockUpdate)
	app := fiber.New()
	app.Put("/documents/:id/metadata", handler.UpdateMetadata)

	// When: Request is made
	body := strings.NewReader(`{"language": "en", "legal_tags": ["motion", "evidence"], "ai_classified": true}`)
	req := httptest.NewRequest("PUT", "/documents/doc_123/metadata", body)
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Then: Metadata is updated
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var response presenter.Response
	respBody, _ := io.ReadAll(resp.Body)
	err = json.Unmarshal(respBody, &response)
	require.NoError(t, err)

	assert.True(t, response.Success)
	mockUpdate.AssertExpectations(t)
}

func TestDocumentHandler_UpdateMetadata_MissingID(t *testing.T) {
	// Given: Handler
	handler := NewDocumentHandler(new(mockProcessUseCase), new(mockBatchProcessUseCase), new(mockIndexUseCase), new(mockUpdateMetadataUseCase))
	app := fiber.New()
	app.Put("/documents/:id/metadata", handler.UpdateMetadata)

	// When: Request is made without document ID
	req := httptest.NewRequest("PUT", "/documents//metadata", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Then: 404 is returned (route doesn't match)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
}

// Helper function to create multipart form data
func createMultipartForm(fields map[string]string, files map[string][]byte) (*bytes.Buffer, string) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add fields
	for key, value := range fields {
		_ = writer.WriteField(key, value)
	}

	// Add files
	for filename, content := range files {
		part, _ := writer.CreateFormFile("file", filename)
		part.Write(content)
	}

	writer.Close()
	return body, writer.FormDataContentType()
}
