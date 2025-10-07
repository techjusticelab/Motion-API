package models

import "time"

// APIResponse represents a standard API response wrapper
type APIResponse struct {
	Success   bool        `json:"success"`
	Message   string      `json:"message,omitempty"`
	Data      interface{} `json:"data,omitempty"`
	Error     *APIError   `json:"error,omitempty"`
	RequestID string      `json:"request_id,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
}

// APIError represents a structured API error
type APIError struct {
	Code    string                 `json:"code"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
	Field   string                 `json:"field,omitempty"`
}

// NewSuccessResponse creates a successful API response
func NewSuccessResponse(data interface{}, message string) *APIResponse {
	return &APIResponse{
		Success:   true,
		Message:   message,
		Data:      data,
		Timestamp: time.Now(),
	}
}

// NewErrorResponse creates an error API response
func NewErrorResponse(code, message string, details map[string]interface{}) *APIResponse {
	return &APIResponse{
		Success: false,
		Error: &APIError{
			Code:    code,
			Message: message,
			Details: details,
		},
		Timestamp: time.Now(),
	}
}

// NewValidationErrorResponse creates a validation error response
func NewValidationErrorResponse(field, message string) *APIResponse {
	return &APIResponse{
		Success: false,
		Error: &APIError{
			Code:    "validation_error",
			Message: message,
			Field:   field,
		},
		Timestamp: time.Now(),
	}
}

// SetRequestID sets the request ID for tracing
func (r *APIResponse) SetRequestID(requestID string) *APIResponse {
	r.RequestID = requestID
	return r
}

// IsError returns true if the response represents an error
func (r *APIResponse) IsError() bool {
	return !r.Success || r.Error != nil
}

// GetErrorCode returns the error code if this is an error response
func (r *APIResponse) GetErrorCode() string {
	if r.Error != nil {
		return r.Error.Code
	}
	return ""
}

// GetErrorMessage returns the error message if this is an error response
func (r *APIResponse) GetErrorMessage() string {
	if r.Error != nil {
		return r.Error.Message
	}
	return ""
}

// WithData sets the data field and returns the response for chaining
func (r *APIResponse) WithData(data interface{}) *APIResponse {
	r.Data = data
	return r
}

// WithMessage sets the message field and returns the response for chaining
func (r *APIResponse) WithMessage(message string) *APIResponse {
	r.Message = message
	return r
}

// Common error codes
const (
	ErrorCodeValidation     = "validation_error"
	ErrorCodeAuthentication = "authentication_error"
	ErrorCodeAuthorization  = "authorization_error"
	ErrorCodeNotFound       = "not_found"
	ErrorCodeConflict       = "conflict"
	ErrorCodeRateLimit      = "rate_limit_exceeded"
	ErrorCodeInternal       = "internal_error"
	ErrorCodeBadRequest     = "bad_request"
	ErrorCodeTimeout        = "timeout"
	ErrorCodeServiceUnavailable = "service_unavailable"
)

// Common error messages
const (
	MessageValidationFailed = "Request validation failed"
	MessageNotFound        = "Resource not found"
	MessageUnauthorized    = "Authentication required"
	MessageForbidden       = "Access denied"
	MessageInternalError   = "Internal server error"
	MessageBadRequest      = "Invalid request"
	MessageTimeout         = "Request timeout"
	MessageRateLimit       = "Rate limit exceeded"
)