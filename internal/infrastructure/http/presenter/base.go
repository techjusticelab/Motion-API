package presenter

import (
	"time"

	"github.com/gofiber/fiber/v2"
)

// Response represents a standard API response
type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *Error      `json:"error,omitempty"`
	Meta    *Meta       `json:"meta,omitempty"`
}

// Error represents an error response
type Error struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

// Meta represents response metadata
type Meta struct {
	RequestID string `json:"request_id,omitempty"`
	Timestamp int64  `json:"timestamp"`
	Version   string `json:"version"`
}

// Success creates a success response
func Success(c *fiber.Ctx, data interface{}) error {
	return c.JSON(&Response{
		Success: true,
		Data:    data,
		Meta:    buildMeta(c),
	})
}

// SuccessWithStatus creates a success response with custom status code
func SuccessWithStatus(c *fiber.Ctx, statusCode int, data interface{}) error {
	return c.Status(statusCode).JSON(&Response{
		Success: true,
		Data:    data,
		Meta:    buildMeta(c),
	})
}

// ErrorResponse creates an error response
func ErrorResponse(c *fiber.Ctx, statusCode int, code, message string, details interface{}) error {
	return c.Status(statusCode).JSON(&Response{
		Success: false,
		Error: &Error{
			Code:    code,
			Message: message,
			Details: details,
		},
		Meta: buildMeta(c),
	})
}

// BadRequest creates a 400 Bad Request error response
func BadRequest(c *fiber.Ctx, message string, details interface{}) error {
	return ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", message, details)
}

// NotFound creates a 404 Not Found error response
func NotFound(c *fiber.Ctx, message string) error {
	return ErrorResponse(c, fiber.StatusNotFound, "NOT_FOUND", message, nil)
}

// InternalError creates a 500 Internal Server Error response
func InternalError(c *fiber.Ctx, message string) error {
	return ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", message, nil)
}

// ValidationError creates a 400 Validation Error response
func ValidationError(c *fiber.Ctx, details interface{}) error {
	return ErrorResponse(c, fiber.StatusBadRequest, "VALIDATION_ERROR", "Request validation failed", details)
}

// buildMeta constructs response metadata
func buildMeta(c *fiber.Ctx) *Meta {
	requestID := ""

	// Try to get request ID from locals (set by request ID middleware)
	if id := c.Locals("requestid"); id != nil {
		if strID, ok := id.(string); ok {
			requestID = strID
		}
	}

	// Fallback to X-Request-ID header if not in locals
	if requestID == "" {
		requestID = c.Get("X-Request-ID", "")
	}

	return &Meta{
		RequestID: requestID,
		Timestamp: time.Now().Unix(),
		Version:   "1.0.0",
	}
}
