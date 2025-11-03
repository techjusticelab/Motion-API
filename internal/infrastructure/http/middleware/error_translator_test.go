package middleware

import (
	"encoding/json"
	"errors"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	domainerrors "motion-index-fiber/internal/domain/errors"
	"motion-index-fiber/internal/infrastructure/http/presenter"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestErrorTranslator_DomainErrors(t *testing.T) {
	tests := []struct {
		name           string
		err            error
		expectedStatus int
		expectedCode   string
	}{
		{
			name:           "document not found",
			err:            domainerrors.ErrDocumentNotFound,
			expectedStatus: fiber.StatusNotFound,
			expectedCode:   "DOCUMENT_NOT_FOUND",
		},
		{
			name:           "empty document ID",
			err:            domainerrors.ErrEmptyDocumentID,
			expectedStatus: fiber.StatusBadRequest,
			expectedCode:   "EMPTY_DOCUMENT_ID",
		},
		{
			name:           "invalid file name",
			err:            domainerrors.ErrInvalidFileName,
			expectedStatus: fiber.StatusBadRequest,
			expectedCode:   "INVALID_FILE_NAME",
		},
		{
			name:           "invalid confidence",
			err:            domainerrors.ErrInvalidConfidence,
			expectedStatus: fiber.StatusBadRequest,
			expectedCode:   "INVALID_CONFIDENCE",
		},
		{
			name:           "classifier not configured",
			err:            domainerrors.ErrClassifierNotConfigured,
			expectedStatus: fiber.StatusServiceUnavailable,
			expectedCode:   "CLASSIFIER_NOT_CONFIGURED",
		},
		{
			name:           "duplicate document",
			err:            domainerrors.ErrDuplicateDocument,
			expectedStatus: fiber.StatusConflict,
			expectedCode:   "DUPLICATE_DOCUMENT",
		},
		{
			name:           "concurrency conflict",
			err:            domainerrors.ErrConcurrencyConflict,
			expectedStatus: fiber.StatusConflict,
			expectedCode:   "CONCURRENCY_CONFLICT",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: App with error translator middleware
			app := fiber.New()
			app.Use(ErrorTranslator())
			app.Get("/test", func(c *fiber.Ctx) error {
				return tt.err
			})

			// When: Request is made
			req := httptest.NewRequest("GET", "/test", nil)
			resp, err := app.Test(req)
			require.NoError(t, err)

			// Then: Error is translated correctly
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			var response presenter.Response
			body, _ := io.ReadAll(resp.Body)
			err = json.Unmarshal(body, &response)
			require.NoError(t, err)

			assert.False(t, response.Success)
			assert.Equal(t, tt.expectedCode, response.Error.Code)
		})
	}
}

func TestErrorTranslator_ValidationError(t *testing.T) {
	// Given: App with error translator middleware
	app := fiber.New()
	app.Use(ErrorTranslator())
	app.Post("/test", func(c *fiber.Ctx) error {
		return domainerrors.NewValidationError("email", "required", nil)
	})

	// When: Request is made
	req := httptest.NewRequest("POST", "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Then: Validation error is translated correctly
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	var response presenter.Response
	body, _ := io.ReadAll(resp.Body)
	err = json.Unmarshal(body, &response)
	require.NoError(t, err)

	assert.False(t, response.Success)
	assert.Equal(t, "VALIDATION_ERROR", response.Error.Code)
	assert.NotNil(t, response.Error.Details)
}

func TestErrorTranslator_FiberErrors(t *testing.T) {
	tests := []struct {
		name           string
		fiberErr       *fiber.Error
		expectedStatus int
		expectedCode   string
	}{
		{
			name:           "bad request",
			fiberErr:       fiber.NewError(fiber.StatusBadRequest, "Bad request"),
			expectedStatus: fiber.StatusBadRequest,
			expectedCode:   "BAD_REQUEST",
		},
		{
			name:           "unauthorized",
			fiberErr:       fiber.NewError(fiber.StatusUnauthorized, "Unauthorized"),
			expectedStatus: fiber.StatusUnauthorized,
			expectedCode:   "UNAUTHORIZED",
		},
		{
			name:           "forbidden",
			fiberErr:       fiber.NewError(fiber.StatusForbidden, "Forbidden"),
			expectedStatus: fiber.StatusForbidden,
			expectedCode:   "FORBIDDEN",
		},
		{
			name:           "not found",
			fiberErr:       fiber.NewError(fiber.StatusNotFound, "Not found"),
			expectedStatus: fiber.StatusNotFound,
			expectedCode:   "NOT_FOUND",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: App with error translator middleware
			app := fiber.New()
			app.Use(ErrorTranslator())
			app.Get("/test", func(c *fiber.Ctx) error {
				return tt.fiberErr
			})

			// When: Request is made
			req := httptest.NewRequest("GET", "/test", nil)
			resp, err := app.Test(req)
			require.NoError(t, err)

			// Then: Fiber error is translated correctly
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			var response presenter.Response
			body, _ := io.ReadAll(resp.Body)
			err = json.Unmarshal(body, &response)
			require.NoError(t, err)

			assert.False(t, response.Success)
			assert.Equal(t, tt.expectedCode, response.Error.Code)
		})
	}
}

func TestErrorTranslator_InfrastructureErrors(t *testing.T) {
	tests := []struct {
		name           string
		err            error
		expectedStatus int
		expectedCode   string
	}{
		{
			name:           "storage error",
			err:            errors.New("storage error: upload failed"),
			expectedStatus: fiber.StatusInternalServerError,
			expectedCode:   "STORAGE_ERROR",
		},
		{
			name:           "storage access denied",
			err:            errors.New("storage access denied: forbidden"),
			expectedStatus: fiber.StatusForbidden,
			expectedCode:   "STORAGE_ACCESS_DENIED",
		},
		{
			name:           "storage timeout",
			err:            errors.New("storage error: timeout"),
			expectedStatus: fiber.StatusGatewayTimeout,
			expectedCode:   "STORAGE_TIMEOUT",
		},
		{
			name:           "search error",
			err:            errors.New("search error: query failed"),
			expectedStatus: fiber.StatusInternalServerError,
			expectedCode:   "SEARCH_ERROR",
		},
		{
			name:           "search unavailable",
			err:            errors.New("search service unavailable: connection refused"),
			expectedStatus: fiber.StatusServiceUnavailable,
			expectedCode:   "SEARCH_UNAVAILABLE",
		},
		{
			name:           "AI provider authentication",
			err:            errors.New("AI provider authentication failed"),
			expectedStatus: fiber.StatusServiceUnavailable,
			expectedCode:   "AI_AUTH_FAILED",
		},
		{
			name:           "AI provider rate limit",
			err:            errors.New("AI provider rate limit exceeded"),
			expectedStatus: fiber.StatusTooManyRequests,
			expectedCode:   "AI_RATE_LIMIT",
		},
		{
			name:           "AI provider quota",
			err:            errors.New("AI provider quota exceeded"),
			expectedStatus: fiber.StatusServiceUnavailable,
			expectedCode:   "AI_QUOTA_EXCEEDED",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: App with error translator middleware
			app := fiber.New()
			app.Use(ErrorTranslator())
			app.Get("/test", func(c *fiber.Ctx) error {
				return tt.err
			})

			// When: Request is made
			req := httptest.NewRequest("GET", "/test", nil)
			resp, err := app.Test(req)
			require.NoError(t, err)

			// Then: Infrastructure error is translated correctly
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			var response presenter.Response
			body, _ := io.ReadAll(resp.Body)
			err = json.Unmarshal(body, &response)
			require.NoError(t, err)

			assert.False(t, response.Success)
			assert.Equal(t, tt.expectedCode, response.Error.Code)
		})
	}
}

func TestErrorTranslator_DefaultError(t *testing.T) {
	// Given: App with error translator middleware
	app := fiber.New()
	app.Use(ErrorTranslator())
	app.Get("/test", func(c *fiber.Ctx) error {
		return errors.New("some unknown error")
	})

	// When: Request is made
	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Then: Default error response is returned
	assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)

	var response presenter.Response
	body, _ := io.ReadAll(resp.Body)
	err = json.Unmarshal(body, &response)
	require.NoError(t, err)

	assert.False(t, response.Success)
	assert.Equal(t, "INTERNAL_ERROR", response.Error.Code)
	assert.Equal(t, "An unexpected error occurred", response.Error.Message)
}

func TestErrorTranslator_NoError(t *testing.T) {
	// Given: App with error translator middleware
	app := fiber.New()
	app.Use(ErrorTranslator())
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("success")
	})

	// When: Request is made
	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Then: Response is successful
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	assert.Equal(t, "success", string(body))
}
