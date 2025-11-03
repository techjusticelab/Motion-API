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
)

// mockClassifyUseCase is a mock implementation of ClassifyDocumentUseCase
type mockClassifyUseCase struct {
	mock.Mock
}

func (m *mockClassifyUseCase) Execute(ctx context.Context, req *dto.ClassifyDocumentRequest) (*dto.ClassificationResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.ClassificationResponse), args.Error(1)
}

func TestNewClassificationHandler(t *testing.T) {
	// Given: Mock use case
	mockUseCase := new(mockClassifyUseCase)

	// When: Handler is created
	handler := NewClassificationHandler(mockUseCase)

	// Then: Handler is properly initialized
	assert.NotNil(t, handler)
	assert.Equal(t, mockUseCase, handler.classifyUseCase)
}

func TestClassificationHandler_ClassifyDocument_Success(t *testing.T) {
	// Given: Mock use case with successful response
	mockUseCase := new(mockClassifyUseCase)
	now := time.Now()
	mockUseCase.On("Execute", mock.Anything, mock.MatchedBy(func(req *dto.ClassifyDocumentRequest) bool {
		return req.DocumentID == "doc_123" && !req.Force
	})).Return(&dto.ClassificationResponse{
		DocumentID:   "doc_123",
		DocumentType: "motion",
		Category:     "criminal",
		Confidence:   0.92,
		LegalTags:    []string{"motion to suppress", "evidence"},
		ClassifiedBy: "openai",
		ClassifiedAt: now,
	}, nil)

	handler := NewClassificationHandler(mockUseCase)
	app := fiber.New()
	app.Post("/documents/:id/classify", handler.ClassifyDocument)

	// When: Request is made
	req := httptest.NewRequest("POST", "/documents/doc_123/classify", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Then: Classification result is returned
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var response presenter.Response
	body, _ := io.ReadAll(resp.Body)
	err = json.Unmarshal(body, &response)
	require.NoError(t, err)

	assert.True(t, response.Success)
	data := response.Data.(map[string]interface{})
	assert.Equal(t, "doc_123", data["document_id"])
	assert.Equal(t, "motion", data["document_type"])
	assert.Equal(t, "criminal", data["category"])
	assert.Equal(t, 0.92, data["confidence"])

	mockUseCase.AssertExpectations(t)
}

func TestClassificationHandler_ClassifyDocument_WithForce(t *testing.T) {
	// Given: Mock use case with force=true
	mockUseCase := new(mockClassifyUseCase)
	mockUseCase.On("Execute", mock.Anything, mock.MatchedBy(func(req *dto.ClassifyDocumentRequest) bool {
		return req.DocumentID == "doc_123" && req.Force
	})).Return(&dto.ClassificationResponse{
		DocumentID: "doc_123",
	}, nil)

	handler := NewClassificationHandler(mockUseCase)
	app := fiber.New()
	app.Post("/documents/:id/classify", handler.ClassifyDocument)

	// When: Request is made with force parameter
	body := strings.NewReader(`{"force": true}`)
	req := httptest.NewRequest("POST", "/documents/doc_123/classify", body)
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Then: Classification is forced
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	mockUseCase.AssertExpectations(t)
}

func TestClassificationHandler_ClassifyDocument_MissingID(t *testing.T) {
	// Given: Handler
	handler := NewClassificationHandler(new(mockClassifyUseCase))
	app := fiber.New()
	app.Post("/documents/:id/classify", handler.ClassifyDocument)

	// When: Request is made without document ID
	req := httptest.NewRequest("POST", "/documents//classify", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Then: 404 is returned (route doesn't match)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
}

func TestClassificationHandler_ClassifyDocument_Error(t *testing.T) {
	// Given: Mock use case with error
	mockUseCase := new(mockClassifyUseCase)
	mockUseCase.On("Execute", mock.Anything, mock.AnythingOfType("*dto.ClassifyDocumentRequest")).
		Return(nil, fiber.NewError(fiber.StatusInternalServerError, "classification failed"))

	handler := NewClassificationHandler(mockUseCase)
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
	app.Post("/documents/:id/classify", handler.ClassifyDocument)

	// When: Request is made
	req := httptest.NewRequest("POST", "/documents/doc_123/classify", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Then: Error response is returned
	assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
	mockUseCase.AssertExpectations(t)
}

func TestClassificationHandler_ReclassifyDocument_Success(t *testing.T) {
	// Given: Mock use case for reclassification
	mockUseCase := new(mockClassifyUseCase)
	mockUseCase.On("Execute", mock.Anything, mock.MatchedBy(func(req *dto.ClassifyDocumentRequest) bool {
		return req.DocumentID == "doc_123" && req.Force
	})).Return(&dto.ClassificationResponse{
		DocumentID:   "doc_123",
		DocumentType: "motion",
	}, nil)

	handler := NewClassificationHandler(mockUseCase)
	app := fiber.New()
	app.Put("/documents/:id/classify", handler.ReclassifyDocument)

	// When: Request is made
	req := httptest.NewRequest("PUT", "/documents/doc_123/classify", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Then: Classification result is returned with force=true
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	mockUseCase.AssertExpectations(t)
}

func TestClassificationHandler_ReclassifyDocument_MissingID(t *testing.T) {
	// Given: Handler
	handler := NewClassificationHandler(new(mockClassifyUseCase))
	app := fiber.New()
	app.Put("/documents/:id/classify", handler.ReclassifyDocument)

	// When: Request is made without document ID
	req := httptest.NewRequest("PUT", "/documents//classify", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Then: 404 is returned (route doesn't match)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
}
