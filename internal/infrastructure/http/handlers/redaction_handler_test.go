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

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"motion-index-fiber/internal/application/dto"
	"motion-index-fiber/internal/infrastructure/http/presenter"
)

// mockAnalyzeRedactionsUseCase is a mock implementation
type mockAnalyzeRedactionsUseCase struct {
	mock.Mock
}

func (m *mockAnalyzeRedactionsUseCase) Execute(ctx context.Context, req *dto.AnalyzeRedactionsRequest) (*dto.AnalyzeRedactionsResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.AnalyzeRedactionsResponse), args.Error(1)
}

// mockApplyRedactionsUseCase is a mock implementation
type mockApplyRedactionsUseCase struct {
	mock.Mock
}

func (m *mockApplyRedactionsUseCase) Execute(ctx context.Context, req *dto.ApplyRedactionsRequest) (*dto.ApplyRedactionsResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.ApplyRedactionsResponse), args.Error(1)
}

func TestNewRedactionHandler(t *testing.T) {
	// Given: Mock use cases
	mockAnalyze := new(mockAnalyzeRedactionsUseCase)
	mockApply := new(mockApplyRedactionsUseCase)

	// When: Handler is created
	handler := NewRedactionHandler(mockAnalyze, mockApply)

	// Then: Handler is properly initialized
	assert.NotNil(t, handler)
	assert.Equal(t, mockAnalyze, handler.analyzeUseCase)
	assert.Equal(t, mockApply, handler.applyUseCase)
}

func TestRedactionHandler_AnalyzeRedactions_Success(t *testing.T) {
	// Given: Mock analyze use case with successful response
	mockAnalyze := new(mockAnalyzeRedactionsUseCase)
	mockAnalyze.On("Execute", mock.Anything, mock.AnythingOfType("*dto.AnalyzeRedactionsRequest")).
		Return(&dto.AnalyzeRedactionsResponse{
			DocumentID: "doc_123",
			FileName:   "test.pdf",
			Redactions: []dto.RedactionItem{
				{
					ID:        "red_1",
					Page:      1,
					Text:      "SSN: 123-45-6789",
					Type:      "ssn",
					Citation:  "CA Penal Code 293",
					Reason:    "Personal Information",
					LegalCode: "PC293",
					Applied:   false,
				},
			},
			TotalCount: 1,
		}, nil)

	handler := NewRedactionHandler(mockAnalyze, new(mockApplyRedactionsUseCase))
	app := fiber.New()
	app.Post("/redactions/analyze", handler.AnalyzeRedactions)

	// When: Request is made with file upload
	body, contentType := createMultipartFormForRedaction(map[string]string{
		"use_ai": "true",
	}, map[string][]byte{
		"file": []byte("PDF content"),
	})

	req := httptest.NewRequest("POST", "/redactions/analyze", body)
	req.Header.Set("Content-Type", contentType)
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Then: Analysis result is returned
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var response presenter.Response
	respBody, _ := io.ReadAll(resp.Body)
	err = json.Unmarshal(respBody, &response)
	require.NoError(t, err)

	assert.True(t, response.Success)
	mockAnalyze.AssertExpectations(t)
}

func TestRedactionHandler_AnalyzeRedactions_MissingFile(t *testing.T) {
	// Given: Handler
	handler := NewRedactionHandler(new(mockAnalyzeRedactionsUseCase), new(mockApplyRedactionsUseCase))
	app := fiber.New()
	app.Post("/redactions/analyze", handler.AnalyzeRedactions)

	// When: Request is made without file
	req := httptest.NewRequest("POST", "/redactions/analyze", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Then: Error response is returned
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestRedactionHandler_ApplyRedactions_FromFile_Success(t *testing.T) {
	// Given: Mock apply use case with successful response
	mockApply := new(mockApplyRedactionsUseCase)
	mockApply.On("Execute", mock.Anything, mock.AnythingOfType("*dto.ApplyRedactionsRequest")).
		Return(&dto.ApplyRedactionsResponse{
			DocumentID: "doc_123",
			FileName:   "redacted_test.pdf",
			Redactions: []dto.RedactionItem{
				{
					ID:      "red_1",
					Page:    1,
					Type:    "ssn",
					Applied: true,
				},
			},
			TotalRedactions: 1,
			PDFBase64:       "base64encodedpdf",
		}, nil)

	handler := NewRedactionHandler(new(mockAnalyzeRedactionsUseCase), mockApply)
	app := fiber.New()
	app.Post("/redactions/apply", handler.ApplyRedactions)

	// When: Request is made with file upload
	body, contentType := createMultipartFormForRedaction(map[string]string{
		"use_ai":        "true",
		"return_base64": "true",
	}, map[string][]byte{
		"file": []byte("PDF content"),
	})

	req := httptest.NewRequest("POST", "/redactions/apply", body)
	req.Header.Set("Content-Type", contentType)
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Then: Redaction result is returned
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var response presenter.Response
	respBody, _ := io.ReadAll(resp.Body)
	err = json.Unmarshal(respBody, &response)
	require.NoError(t, err)

	assert.True(t, response.Success)
	mockApply.AssertExpectations(t)
}

func TestRedactionHandler_ApplyRedactions_FromJSON_Success(t *testing.T) {
	// Given: Mock apply use case
	mockApply := new(mockApplyRedactionsUseCase)
	mockApply.On("Execute", mock.Anything, mock.MatchedBy(func(req *dto.ApplyRedactionsRequest) bool {
		return req.DocumentID == "doc_123" && req.ReturnBase64
	})).Return(&dto.ApplyRedactionsResponse{
		DocumentID:      "doc_123",
		TotalRedactions: 2,
		Redactions: []dto.RedactionItem{
			{ID: "red_1", Applied: true},
			{ID: "red_2", Applied: true},
		},
	}, nil)

	handler := NewRedactionHandler(new(mockAnalyzeRedactionsUseCase), mockApply)
	app := fiber.New()
	app.Post("/redactions/apply", handler.ApplyRedactions)

	// When: Request is made with JSON
	body := strings.NewReader(`{
		"DocumentID": "doc_123",
		"ReturnBase64": true,
		"CustomRedactions": [
			{"ID": "red_1", "Page": 1, "Type": "ssn"},
			{"ID": "red_2", "Page": 2, "Type": "address"}
		]
	}`)
	req := httptest.NewRequest("POST", "/redactions/apply", body)
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Then: Redaction result is returned
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	mockApply.AssertExpectations(t)
}

func TestRedactionHandler_ApplyRedactions_MissingFile(t *testing.T) {
	// Given: Handler
	handler := NewRedactionHandler(new(mockAnalyzeRedactionsUseCase), new(mockApplyRedactionsUseCase))
	app := fiber.New()
	app.Post("/redactions/apply", handler.ApplyRedactions)

	// When: Request is made without file (multipart)
	req := httptest.NewRequest("POST", "/redactions/apply", nil)
	req.Header.Set("Content-Type", "multipart/form-data")
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Then: Error response is returned
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestRedactionHandler_ApplyRedactions_InvalidJSON(t *testing.T) {
	// Given: Handler
	handler := NewRedactionHandler(new(mockAnalyzeRedactionsUseCase), new(mockApplyRedactionsUseCase))
	app := fiber.New()
	app.Post("/redactions/apply", handler.ApplyRedactions)

	// When: Request is made with invalid JSON
	body := strings.NewReader(`{invalid json}`)
	req := httptest.NewRequest("POST", "/redactions/apply", body)
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Then: Error response is returned
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestRedactionHandler_ApplyRedactions_Error(t *testing.T) {
	// Given: Mock apply use case with error
	mockApply := new(mockApplyRedactionsUseCase)
	mockApply.On("Execute", mock.Anything, mock.AnythingOfType("*dto.ApplyRedactionsRequest")).
		Return(nil, fiber.NewError(fiber.StatusInternalServerError, "redaction failed"))

	handler := NewRedactionHandler(new(mockAnalyzeRedactionsUseCase), mockApply)
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
	app.Post("/redactions/apply", handler.ApplyRedactions)

	// When: Request is made
	body := strings.NewReader(`{"DocumentID": "doc_123"}`)
	req := httptest.NewRequest("POST", "/redactions/apply", body)
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Then: Error response is returned
	assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
	mockApply.AssertExpectations(t)
}

// Helper function to create multipart form data (reused from document_handler_test.go)
func createMultipartFormForRedaction(fields map[string]string, files map[string][]byte) (*bytes.Buffer, string) {
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
