package middleware

import (
	"errors"
	"log"
	"strings"

	"github.com/gofiber/fiber/v2"
	domainerrors "motion-index-fiber/internal/domain/errors"
	infraerrors "motion-index-fiber/internal/infrastructure/errors"
	"motion-index-fiber/internal/infrastructure/http/presenter"
)

// ErrorTranslator translates domain/application/infrastructure errors to HTTP responses
func ErrorTranslator() fiber.Handler {
	return func(c *fiber.Ctx) error {
		err := c.Next()
		if err == nil {
			return nil
		}

		// Log the error for debugging
		log.Printf("Error occurred: %v", err)

		// Translate error to HTTP status and response
		statusCode, code, message, details := translateError(err)

		return presenter.ErrorResponse(c, statusCode, code, message, details)
	}
}

// translateError maps errors to HTTP status codes and error codes
func translateError(err error) (statusCode int, code string, message string, details interface{}) {
	// Handle Fiber errors first (they already have status codes)
	var fiberErr *fiber.Error
	if errors.As(err, &fiberErr) {
		return fiberErr.Code, mapFiberCodeToErrorCode(fiberErr.Code), fiberErr.Message, nil
	}

	// Domain errors - Document errors
	if errors.Is(err, domainerrors.ErrDocumentNotFound) {
		return fiber.StatusNotFound, "DOCUMENT_NOT_FOUND", "Document not found", nil
	}
	if errors.Is(err, domainerrors.ErrEmptyDocumentID) {
		return fiber.StatusBadRequest, "EMPTY_DOCUMENT_ID", "Document ID cannot be empty", nil
	}
	if errors.Is(err, domainerrors.ErrEmptyFileName) {
		return fiber.StatusBadRequest, "EMPTY_FILE_NAME", "File name cannot be empty", nil
	}
	if errors.Is(err, domainerrors.ErrInvalidFileName) {
		return fiber.StatusBadRequest, "INVALID_FILE_NAME", "Invalid file name", nil
	}
	if errors.Is(err, domainerrors.ErrInvalidFilePath) {
		return fiber.StatusBadRequest, "INVALID_FILE_PATH", "Invalid file path", nil
	}
	if errors.Is(err, domainerrors.ErrEmptyFilePath) {
		return fiber.StatusBadRequest, "EMPTY_FILE_PATH", "File path cannot be empty", nil
	}
	if errors.Is(err, domainerrors.ErrInvalidS3URI) {
		return fiber.StatusBadRequest, "INVALID_S3_URI", "Invalid S3 URI", nil
	}
	if errors.Is(err, domainerrors.ErrEmptyHash) {
		return fiber.StatusBadRequest, "EMPTY_HASH", "Hash cannot be empty", nil
	}
	if errors.Is(err, domainerrors.ErrInvalidHash) {
		return fiber.StatusBadRequest, "INVALID_HASH", "Invalid hash value", nil
	}
	if errors.Is(err, domainerrors.ErrInvalidContentType) {
		return fiber.StatusBadRequest, "INVALID_CONTENT_TYPE", "Invalid content type", nil
	}
	if errors.Is(err, domainerrors.ErrInvalidFileSize) {
		return fiber.StatusBadRequest, "INVALID_FILE_SIZE", "Invalid file size", nil
	}
	if errors.Is(err, domainerrors.ErrCannotClassifyEmptyDocument) {
		return fiber.StatusBadRequest, "CANNOT_CLASSIFY_EMPTY_DOCUMENT", "Cannot classify document without content", nil
	}

	// Domain errors - Classification errors
	if errors.Is(err, domainerrors.ErrInvalidConfidence) {
		return fiber.StatusBadRequest, "INVALID_CONFIDENCE", "Confidence must be between 0.0 and 1.0", nil
	}
	if errors.Is(err, domainerrors.ErrInvalidClassification) {
		return fiber.StatusBadRequest, "INVALID_CLASSIFICATION", "Invalid classification", nil
	}
	if errors.Is(err, domainerrors.ErrClassifierNotConfigured) {
		return fiber.StatusServiceUnavailable, "CLASSIFIER_NOT_CONFIGURED", "Classifier is not configured", nil
	}
	if errors.Is(err, domainerrors.ErrUnsupportedClassification) {
		return fiber.StatusBadRequest, "UNSUPPORTED_CLASSIFICATION", "Classification type not supported", nil
	}

	// Domain errors - Legal entity errors
	if errors.Is(err, domainerrors.ErrInvalidCaseNumber) {
		return fiber.StatusBadRequest, "INVALID_CASE_NUMBER", "Invalid case number format", nil
	}
	if errors.Is(err, domainerrors.ErrInvalidPartyRole) {
		return fiber.StatusBadRequest, "INVALID_PARTY_ROLE", "Invalid party role", nil
	}
	if errors.Is(err, domainerrors.ErrInvalidBarNumber) {
		return fiber.StatusBadRequest, "INVALID_BAR_NUMBER", "Invalid bar number", nil
	}

	// Domain errors - Repository errors
	if errors.Is(err, domainerrors.ErrDuplicateDocument) {
		return fiber.StatusConflict, "DUPLICATE_DOCUMENT", "Document already exists", nil
	}
	if errors.Is(err, domainerrors.ErrConcurrencyConflict) {
		return fiber.StatusConflict, "CONCURRENCY_CONFLICT", "Document was modified by another process", nil
	}

	// Validation errors
	var validationErr *domainerrors.ValidationError
	if errors.As(err, &validationErr) {
		return fiber.StatusBadRequest, "VALIDATION_ERROR", validationErr.Error(), map[string]interface{}{
			"field": validationErr.Field,
			"value": validationErr.Value,
		}
	}

	// Infrastructure errors - Check error message patterns
	errMsg := err.Error()

	// Storage errors
	if strings.Contains(errMsg, "storage error") || strings.Contains(errMsg, "storage access denied") {
		if strings.Contains(errMsg, "access denied") || strings.Contains(errMsg, "forbidden") {
			return fiber.StatusForbidden, "STORAGE_ACCESS_DENIED", "Access to storage denied", nil
		}
		if infraerrors.IsTimeout(err) {
			return fiber.StatusGatewayTimeout, "STORAGE_TIMEOUT", "Storage operation timed out", nil
		}
		return fiber.StatusInternalServerError, "STORAGE_ERROR", "Storage operation failed", nil
	}

	// Search errors
	if strings.Contains(errMsg, "search error") || strings.Contains(errMsg, "search service") {
		if strings.Contains(errMsg, "service unavailable") || strings.Contains(errMsg, "connection refused") {
			return fiber.StatusServiceUnavailable, "SEARCH_UNAVAILABLE", "Search service is unavailable", nil
		}
		if infraerrors.IsTimeout(err) {
			return fiber.StatusGatewayTimeout, "SEARCH_TIMEOUT", "Search operation timed out", nil
		}
		return fiber.StatusInternalServerError, "SEARCH_ERROR", "Search operation failed", nil
	}

	// AI provider errors
	if strings.Contains(errMsg, "AI provider") {
		if strings.Contains(errMsg, "authentication") {
			return fiber.StatusServiceUnavailable, "AI_AUTH_FAILED", "AI provider authentication failed", nil
		}
		if infraerrors.IsRateLimit(err) {
			return fiber.StatusTooManyRequests, "AI_RATE_LIMIT", "AI provider rate limit exceeded", nil
		}
		if strings.Contains(errMsg, "quota") {
			return fiber.StatusServiceUnavailable, "AI_QUOTA_EXCEEDED", "AI provider quota exceeded", nil
		}
		if infraerrors.IsTimeout(err) {
			return fiber.StatusGatewayTimeout, "AI_TIMEOUT", "AI provider timeout", nil
		}
		return fiber.StatusInternalServerError, "AI_ERROR", "AI provider error", nil
	}

	// Generic timeout
	if infraerrors.IsTimeout(err) {
		return fiber.StatusGatewayTimeout, "TIMEOUT", "Operation timed out", nil
	}

	// Default: Internal server error
	return fiber.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred", nil
}

// mapFiberCodeToErrorCode maps Fiber HTTP status codes to error codes
func mapFiberCodeToErrorCode(statusCode int) string {
	switch statusCode {
	case fiber.StatusBadRequest:
		return "BAD_REQUEST"
	case fiber.StatusUnauthorized:
		return "UNAUTHORIZED"
	case fiber.StatusForbidden:
		return "FORBIDDEN"
	case fiber.StatusNotFound:
		return "NOT_FOUND"
	case fiber.StatusMethodNotAllowed:
		return "METHOD_NOT_ALLOWED"
	case fiber.StatusConflict:
		return "CONFLICT"
	case fiber.StatusUnprocessableEntity:
		return "UNPROCESSABLE_ENTITY"
	case fiber.StatusTooManyRequests:
		return "TOO_MANY_REQUESTS"
	case fiber.StatusInternalServerError:
		return "INTERNAL_ERROR"
	case fiber.StatusServiceUnavailable:
		return "SERVICE_UNAVAILABLE"
	case fiber.StatusGatewayTimeout:
		return "GATEWAY_TIMEOUT"
	default:
		return "UNKNOWN_ERROR"
	}
}
