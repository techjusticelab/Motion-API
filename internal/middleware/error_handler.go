package middleware

import (
	"errors"
	"log"
	"runtime/debug"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"motion-index-fiber/internal/models"
)

// ErrorHandlerConfig defines configuration for error handling middleware
type ErrorHandlerConfig struct {
	EnableStackTrace   bool
	EnableLogging      bool
	LogLevel           string
	ShowInternalErrors bool
}

// DefaultErrorHandlerConfig returns default error handler configuration
func DefaultErrorHandlerConfig() *ErrorHandlerConfig {
	return &ErrorHandlerConfig{
		EnableStackTrace:   false, // Disable in production
		EnableLogging:      true,
		LogLevel:           "error",
		ShowInternalErrors: false, // Hide internal errors in production
	}
}

// ErrorHandlerMiddleware creates middleware for centralized error handling
func ErrorHandlerMiddleware(config ...*ErrorHandlerConfig) fiber.Handler {
	cfg := DefaultErrorHandlerConfig()
	if len(config) > 0 && config[0] != nil {
		cfg = config[0]
	}

	return func(c *fiber.Ctx) error {
		// Handle panic recovery
		defer func() {
			if r := recover(); r != nil {
				if cfg.EnableLogging {
					log.Printf("[PANIC] %v\n%s", r, debug.Stack())
				}

				response := models.NewErrorResponse(
					"internal_server_error",
					"An unexpected error occurred",
					nil,
				)

				if requestID := c.Locals(requestid.ConfigDefault.ContextKey); requestID != nil {
					response.RequestID = requestID.(string)
				}

				if cfg.EnableStackTrace {
					response.Error.Details = map[string]interface{}{
						"panic": string(debug.Stack()),
					}
				}

				c.Status(fiber.StatusInternalServerError).JSON(response)
			}
		}()

		// Continue to next handler
		err := c.Next()
		if err != nil {
			return handleError(c, err, cfg)
		}

		return nil
	}
}

// handleError processes different types of errors and returns appropriate responses
func handleError(c *fiber.Ctx, err error, config *ErrorHandlerConfig) error {
	// Get request ID for tracking
	requestID := ""
	if id := c.Locals(requestid.ConfigDefault.ContextKey); id != nil {
		requestID = id.(string)
	}

	// Log error if logging is enabled
	if config.EnableLogging {
		log.Printf("[ERROR] [%s] %s %s - %v", requestID, c.Method(), c.Path(), err)
	}

	// Handle Fiber errors
	var fiberErr *fiber.Error
	if errors.As(err, &fiberErr) {
		return handleFiberError(c, fiberErr, requestID, config)
	}

	// Handle validation errors
	if isValidationError(err) {
		return handleValidationError(c, err, requestID)
	}

	// Handle file upload errors
	if isFileUploadError(err) {
		return handleFileUploadError(c, err, requestID)
	}

	// Handle processing pipeline errors
	if isProcessingError(err) {
		return handleProcessingError(c, err, requestID)
	}

	// Handle storage errors
	if isStorageError(err) {
		return handleStorageError(c, err, requestID)
	}

	// Handle generic errors
	return handleGenericError(c, err, requestID, config)
}

// handleFiberError handles Fiber framework errors
func handleFiberError(c *fiber.Ctx, fiberErr *fiber.Error, requestID string, config *ErrorHandlerConfig) error {
	response := &models.APIResponse{
		Success:   false,
		RequestID: requestID,
		Timestamp: time.Now(),
		Error: &models.APIError{
			Code:    getErrorCodeFromStatus(fiberErr.Code),
			Message: fiberErr.Message,
		},
	}

	if config.ShowInternalErrors && fiberErr.Code >= 500 {
		response.Error.Details = map[string]interface{}{
			"internal_error": fiberErr.Error(),
		}
	}

	return c.Status(fiberErr.Code).JSON(response)
}

// handleValidationError handles validation errors
func handleValidationError(c *fiber.Ctx, err error, requestID string) error {
	response := &models.APIResponse{
		Success:   false,
		RequestID: requestID,
		Timestamp: time.Now(),
		Error: &models.APIError{
			Code:    "validation_error",
			Message: "Validation failed",
			Details: map[string]interface{}{
				"validation_error": err.Error(),
			},
		},
	}

	return c.Status(fiber.StatusBadRequest).JSON(response)
}

// handleFileUploadError handles file upload specific errors
func handleFileUploadError(c *fiber.Ctx, err error, requestID string) error {
	status := fiber.StatusBadRequest
	code := "file_upload_error"

	// Determine specific error type
	errMsg := err.Error()
	if strings.Contains(errMsg, "too large") || strings.Contains(errMsg, "size") {
		code = "file_too_large"
		status = fiber.StatusRequestEntityTooLarge
	} else if strings.Contains(errMsg, "extension") || strings.Contains(errMsg, "type") {
		code = "invalid_file_type"
	} else if strings.Contains(errMsg, "too many") {
		code = "too_many_files"
	}

	response := &models.APIResponse{
		Success:   false,
		RequestID: requestID,
		Timestamp: time.Now(),
		Error: &models.APIError{
			Code:    code,
			Message: err.Error(),
		},
	}

	return c.Status(status).JSON(response)
}

// handleProcessingError handles document processing errors
func handleProcessingError(c *fiber.Ctx, err error, requestID string) error {
	status := fiber.StatusInternalServerError
	code := "processing_error"

	errMsg := err.Error()
	if strings.Contains(errMsg, "timeout") {
		code = "processing_timeout"
		status = fiber.StatusRequestTimeout
	} else if strings.Contains(errMsg, "unsupported") {
		code = "unsupported_operation"
		status = fiber.StatusBadRequest
	}

	response := &models.APIResponse{
		Success:   false,
		RequestID: requestID,
		Timestamp: time.Now(),
		Error: &models.APIError{
			Code:    code,
			Message: err.Error(),
		},
	}

	return c.Status(status).JSON(response)
}

// handleStorageError handles storage-related errors
func handleStorageError(c *fiber.Ctx, err error, requestID string) error {
	status := fiber.StatusInternalServerError
	code := "storage_error"

	errMsg := err.Error()
	if strings.Contains(errMsg, "not found") {
		code = "file_not_found"
		status = fiber.StatusNotFound
	} else if strings.Contains(errMsg, "permission") || strings.Contains(errMsg, "access") {
		code = "storage_access_denied"
		status = fiber.StatusForbidden
	} else if strings.Contains(errMsg, "space") || strings.Contains(errMsg, "quota") {
		code = "storage_full"
		status = fiber.StatusInsufficientStorage
	}

	response := &models.APIResponse{
		Success:   false,
		RequestID: requestID,
		Timestamp: time.Now(),
		Error: &models.APIError{
			Code:    code,
			Message: err.Error(),
		},
	}

	return c.Status(status).JSON(response)
}

// handleGenericError handles all other errors
func handleGenericError(c *fiber.Ctx, err error, requestID string, config *ErrorHandlerConfig) error {
	message := "An internal server error occurred"
	if config.ShowInternalErrors {
		message = err.Error()
	}

	response := &models.APIResponse{
		Success:   false,
		RequestID: requestID,
		Timestamp: time.Now(),
		Error: &models.APIError{
			Code:    "internal_server_error",
			Message: message,
		},
	}

	if config.ShowInternalErrors {
		response.Error.Details = map[string]interface{}{
			"error": err.Error(),
		}
	}

	return c.Status(fiber.StatusInternalServerError).JSON(response)
}

// Error type detection functions

func isValidationError(err error) bool {
	errMsg := strings.ToLower(err.Error())
	validationKeywords := []string{"validation", "invalid", "required", "format", "length"}

	for _, keyword := range validationKeywords {
		if strings.Contains(errMsg, keyword) {
			return true
		}
	}
	return false
}

func isFileUploadError(err error) bool {
	errMsg := strings.ToLower(err.Error())
	fileKeywords := []string{"file", "upload", "extension", "size", "mime", "multipart"}

	for _, keyword := range fileKeywords {
		if strings.Contains(errMsg, keyword) {
			return true
		}
	}
	return false
}

func isProcessingError(err error) bool {
	errMsg := strings.ToLower(err.Error())
	processingKeywords := []string{"processing", "extraction", "classification", "timeout", "pipeline"}

	for _, keyword := range processingKeywords {
		if strings.Contains(errMsg, keyword) {
			return true
		}
	}
	return false
}

func isStorageError(err error) bool {
	errMsg := strings.ToLower(err.Error())
	storageKeywords := []string{"storage", "s3", "bucket", "space", "disk", "write", "read"}

	for _, keyword := range storageKeywords {
		if strings.Contains(errMsg, keyword) {
			return true
		}
	}
	return false
}

// getErrorCodeFromStatus maps HTTP status codes to error codes
func getErrorCodeFromStatus(status int) string {
	switch status {
	case fiber.StatusBadRequest:
		return "bad_request"
	case fiber.StatusUnauthorized:
		return "unauthorized"
	case fiber.StatusForbidden:
		return "forbidden"
	case fiber.StatusNotFound:
		return "not_found"
	case fiber.StatusMethodNotAllowed:
		return "method_not_allowed"
	case fiber.StatusRequestTimeout:
		return "request_timeout"
	case fiber.StatusRequestEntityTooLarge:
		return "request_too_large"
	case fiber.StatusUnsupportedMediaType:
		return "unsupported_media_type"
	case fiber.StatusTooManyRequests:
		return "too_many_requests"
	case fiber.StatusInternalServerError:
		return "internal_server_error"
	case fiber.StatusBadGateway:
		return "bad_gateway"
	case fiber.StatusServiceUnavailable:
		return "service_unavailable"
	case fiber.StatusGatewayTimeout:
		return "gateway_timeout"
	default:
		return "unknown_error"
	}
}

// RequestLoggingMiddleware logs all requests
func RequestLoggingMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()

		// Continue to next handler
		err := c.Next()

		// Log request details
		duration := time.Since(start)
		requestID := ""
		if id := c.Locals(requestid.ConfigDefault.ContextKey); id != nil {
			requestID = id.(string)
		}

		log.Printf("[REQUEST] [%s] %s %s %d - %v - %s",
			requestID,
			c.Method(),
			c.Path(),
			c.Response().StatusCode(),
			duration,
			c.IP(),
		)

		return err
	}
}

// CORSMiddleware handles Cross-Origin Resource Sharing
func CORSMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Set CORS headers
		c.Set("Access-Control-Allow-Origin", "*")
		c.Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
		c.Set("Access-Control-Max-Age", "86400")

		// Handle preflight requests
		if c.Method() == "OPTIONS" {
			return c.SendStatus(fiber.StatusNoContent)
		}

		return c.Next()
	}
}
