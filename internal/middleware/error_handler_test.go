package middleware

import (
	"encoding/json"
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"github.com/stretchr/testify/assert"

	"motion-index-fiber/internal/models"
)

func TestDefaultErrorHandlerConfig(t *testing.T) {
	config := DefaultErrorHandlerConfig()

	assert.NotNil(t, config)
	assert.False(t, config.EnableStackTrace)
	assert.True(t, config.EnableLogging)
	assert.Equal(t, "error", config.LogLevel)
	assert.False(t, config.ShowInternalErrors)
}

func TestErrorHandlerMiddleware_Success(t *testing.T) {
	app := fiber.New()
	app.Use(ErrorHandlerMiddleware())
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"message": "success"})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestErrorHandlerMiddleware_FiberError(t *testing.T) {
	app := fiber.New()
	app.Use(requestid.New())
	app.Use(ErrorHandlerMiddleware())
	app.Get("/test", func(c *fiber.Ctx) error {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid request")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	var response models.APIResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	assert.NoError(t, err)
	assert.False(t, response.Success)
	assert.Equal(t, "bad_request", response.Error.Code)
	assert.Equal(t, "Invalid request", response.Error.Message)
	assert.NotEmpty(t, response.RequestID)
}

func TestErrorHandlerMiddleware_ValidationError(t *testing.T) {
	app := fiber.New()
	app.Use(requestid.New())
	app.Use(ErrorHandlerMiddleware())
	app.Get("/test", func(c *fiber.Ctx) error {
		return errors.New("validation failed: field is required")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	var response models.APIResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	assert.NoError(t, err)
	assert.False(t, response.Success)
	assert.Equal(t, "validation_error", response.Error.Code)
	assert.Equal(t, "Validation failed", response.Error.Message)
	assert.Contains(t, response.Error.Details["validation_error"], "validation failed")
}

func TestErrorHandlerMiddleware_FileUploadError(t *testing.T) {
	tests := []struct {
		name           string
		errorMsg       string
		expectedCode   string
		expectedStatus int
	}{
		{
			name:           "file too large",
			errorMsg:       "file size too large",
			expectedCode:   "file_too_large",
			expectedStatus: fiber.StatusRequestEntityTooLarge,
		},
		{
			name:           "invalid file type",
			errorMsg:       "invalid file extension",
			expectedCode:   "invalid_file_type",
			expectedStatus: fiber.StatusBadRequest,
		},
		{
			name:           "too many files",
			errorMsg:       "too many files uploaded",
			expectedCode:   "too_many_files",
			expectedStatus: fiber.StatusBadRequest,
		},
		{
			name:           "generic file error",
			errorMsg:       "file upload failed",
			expectedCode:   "file_upload_error",
			expectedStatus: fiber.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			app.Use(requestid.New())
			app.Use(ErrorHandlerMiddleware())
			app.Get("/test", func(c *fiber.Ctx) error {
				return errors.New(tt.errorMsg)
			})

			req := httptest.NewRequest("GET", "/test", nil)
			resp, err := app.Test(req)

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			var response models.APIResponse
			err = json.NewDecoder(resp.Body).Decode(&response)
			assert.NoError(t, err)
			assert.False(t, response.Success)
			assert.Equal(t, tt.expectedCode, response.Error.Code)
			assert.Equal(t, tt.errorMsg, response.Error.Message)
		})
	}
}

func TestErrorHandlerMiddleware_ProcessingError(t *testing.T) {
	tests := []struct {
		name           string
		errorMsg       string
		expectedCode   string
		expectedStatus int
	}{
		{
			name:           "processing timeout",
			errorMsg:       "processing timeout occurred",
			expectedCode:   "processing_timeout",
			expectedStatus: fiber.StatusRequestTimeout,
		},
		{
			name:           "unsupported operation",
			errorMsg:       "unsupported document format",
			expectedCode:   "unsupported_operation",
			expectedStatus: fiber.StatusBadRequest,
		},
		{
			name:           "generic processing error",
			errorMsg:       "document processing failed",
			expectedCode:   "processing_error",
			expectedStatus: fiber.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			app.Use(requestid.New())
			app.Use(ErrorHandlerMiddleware())
			app.Get("/test", func(c *fiber.Ctx) error {
				return errors.New(tt.errorMsg)
			})

			req := httptest.NewRequest("GET", "/test", nil)
			resp, err := app.Test(req)

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			var response models.APIResponse
			err = json.NewDecoder(resp.Body).Decode(&response)
			assert.NoError(t, err)
			assert.False(t, response.Success)
			assert.Equal(t, tt.expectedCode, response.Error.Code)
		})
	}
}

func TestErrorHandlerMiddleware_StorageError(t *testing.T) {
	tests := []struct {
		name           string
		errorMsg       string
		expectedCode   string
		expectedStatus int
	}{
		{
			name:           "file not found",
			errorMsg:       "file not found in storage",
			expectedCode:   "file_not_found",
			expectedStatus: fiber.StatusNotFound,
		},
		{
			name:           "access denied",
			errorMsg:       "permission denied to access storage",
			expectedCode:   "storage_access_denied",
			expectedStatus: fiber.StatusForbidden,
		},
		{
			name:           "storage full",
			errorMsg:       "storage space exceeded quota",
			expectedCode:   "storage_full",
			expectedStatus: fiber.StatusInsufficientStorage,
		},
		{
			name:           "generic storage error",
			errorMsg:       "storage operation failed",
			expectedCode:   "storage_error",
			expectedStatus: fiber.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			app.Use(requestid.New())
			app.Use(ErrorHandlerMiddleware())
			app.Get("/test", func(c *fiber.Ctx) error {
				return errors.New(tt.errorMsg)
			})

			req := httptest.NewRequest("GET", "/test", nil)
			resp, err := app.Test(req)

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			var response models.APIResponse
			err = json.NewDecoder(resp.Body).Decode(&response)
			assert.NoError(t, err)
			assert.False(t, response.Success)
			assert.Equal(t, tt.expectedCode, response.Error.Code)
		})
	}
}

func TestErrorHandlerMiddleware_GenericError(t *testing.T) {
	app := fiber.New()
	app.Use(requestid.New())
	app.Use(ErrorHandlerMiddleware())
	app.Get("/test", func(c *fiber.Ctx) error {
		return errors.New("some unexpected error")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)

	var response models.APIResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	assert.NoError(t, err)
	assert.False(t, response.Success)
	assert.Equal(t, "internal_server_error", response.Error.Code)
	assert.Equal(t, "An internal server error occurred", response.Error.Message)
}

func TestErrorHandlerMiddleware_GenericErrorWithInternalErrors(t *testing.T) {
	config := &ErrorHandlerConfig{
		ShowInternalErrors: true,
		EnableLogging:      false,
	}

	app := fiber.New()
	app.Use(requestid.New())
	app.Use(ErrorHandlerMiddleware(config))
	app.Get("/test", func(c *fiber.Ctx) error {
		return errors.New("internal error details")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)

	var response models.APIResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	assert.NoError(t, err)
	assert.False(t, response.Success)
	assert.Equal(t, "internal_server_error", response.Error.Code)
	assert.Equal(t, "internal error details", response.Error.Message)
	assert.Equal(t, "internal error details", response.Error.Details["error"])
}

func TestErrorHandlerMiddleware_Panic(t *testing.T) {
	app := fiber.New()
	app.Use(requestid.New())
	app.Use(ErrorHandlerMiddleware())
	app.Get("/test", func(c *fiber.Ctx) error {
		panic("something went wrong")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)

	var response models.APIResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	assert.NoError(t, err)
	assert.False(t, response.Success)
	assert.Equal(t, "internal_server_error", response.Error.Code)
	assert.Equal(t, "An unexpected error occurred", response.Error.Message)
}

func TestErrorHandlerMiddleware_PanicWithStackTrace(t *testing.T) {
	config := &ErrorHandlerConfig{
		EnableStackTrace: true,
		EnableLogging:    false,
	}

	app := fiber.New()
	app.Use(requestid.New())
	app.Use(ErrorHandlerMiddleware(config))
	app.Get("/test", func(c *fiber.Ctx) error {
		panic("something went wrong")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)

	var response models.APIResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	assert.NoError(t, err)
	assert.False(t, response.Success)
	assert.Equal(t, "internal_server_error", response.Error.Code)
	assert.NotEmpty(t, response.Error.Details["panic"])
	assert.Contains(t, response.Error.Details["panic"], "panic")
}

func TestRequestLoggingMiddleware(t *testing.T) {
	app := fiber.New()
	app.Use(requestid.New())
	app.Use(RequestLoggingMiddleware())
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"message": "success"})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestCORSMiddleware(t *testing.T) {
	app := fiber.New()
	app.Use(CORSMiddleware())
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"message": "success"})
	})

	// Test regular request
	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	assert.Equal(t, "*", resp.Header.Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "GET, POST, PUT, DELETE, OPTIONS", resp.Header.Get("Access-Control-Allow-Methods"))
	assert.Equal(t, "Content-Type, Authorization, X-Requested-With", resp.Header.Get("Access-Control-Allow-Headers"))
	assert.Equal(t, "86400", resp.Header.Get("Access-Control-Max-Age"))
}

func TestCORSMiddleware_OptionsRequest(t *testing.T) {
	app := fiber.New()
	app.Use(CORSMiddleware())
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"message": "success"})
	})

	// Test OPTIONS request (preflight)
	req := httptest.NewRequest("OPTIONS", "/test", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusNoContent, resp.StatusCode)
	assert.Equal(t, "*", resp.Header.Get("Access-Control-Allow-Origin"))
}

func TestErrorTypeDetection(t *testing.T) {
	tests := []struct {
		name          string
		errorMsg      string
		detectionFunc func(error) bool
		expected      bool
	}{
		{
			name:          "validation error detected",
			errorMsg:      "validation failed",
			detectionFunc: isValidationError,
			expected:      true,
		},
		{
			name:          "file upload error detected",
			errorMsg:      "file size too large",
			detectionFunc: isFileUploadError,
			expected:      true,
		},
		{
			name:          "processing error detected",
			errorMsg:      "document processing failed",
			detectionFunc: isProcessingError,
			expected:      true,
		},
		{
			name:          "storage error detected",
			errorMsg:      "s3 bucket access denied",
			detectionFunc: isStorageError,
			expected:      true,
		},
		{
			name:          "generic error not detected as validation",
			errorMsg:      "some random error",
			detectionFunc: isValidationError,
			expected:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := errors.New(tt.errorMsg)
			result := tt.detectionFunc(err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetErrorCodeFromStatus(t *testing.T) {
	tests := []struct {
		status       int
		expectedCode string
	}{
		{fiber.StatusBadRequest, "bad_request"},
		{fiber.StatusUnauthorized, "unauthorized"},
		{fiber.StatusForbidden, "forbidden"},
		{fiber.StatusNotFound, "not_found"},
		{fiber.StatusMethodNotAllowed, "method_not_allowed"},
		{fiber.StatusRequestTimeout, "request_timeout"},
		{fiber.StatusRequestEntityTooLarge, "request_too_large"},
		{fiber.StatusUnsupportedMediaType, "unsupported_media_type"},
		{fiber.StatusTooManyRequests, "too_many_requests"},
		{fiber.StatusInternalServerError, "internal_server_error"},
		{fiber.StatusBadGateway, "bad_gateway"},
		{fiber.StatusServiceUnavailable, "service_unavailable"},
		{fiber.StatusGatewayTimeout, "gateway_timeout"},
		{999, "unknown_error"},
	}

	for _, tt := range tests {
		t.Run(tt.expectedCode, func(t *testing.T) {
			code := getErrorCodeFromStatus(tt.status)
			assert.Equal(t, tt.expectedCode, code)
		})
	}
}
