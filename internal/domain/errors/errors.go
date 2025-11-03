package errors

import (
	stdErrors "errors"
	"fmt"
)

// Domain errors
var (
	// Document errors
	ErrDocumentNotFound            = stdErrors.New("document not found")
	ErrEmptyDocumentID             = stdErrors.New("document ID cannot be empty")
	ErrEmptyFileName               = stdErrors.New("file name cannot be empty")
	ErrInvalidFileName             = stdErrors.New("invalid file name")
	ErrInvalidFilePath             = stdErrors.New("invalid file path")
	ErrEmptyFilePath               = stdErrors.New("file path cannot be empty")
	ErrInvalidS3URI                = stdErrors.New("invalid S3 URI")
	ErrEmptyHash                   = stdErrors.New("hash cannot be empty")
	ErrInvalidHash                 = stdErrors.New("invalid hash value")
	ErrInvalidContentType          = stdErrors.New("invalid content type")
	ErrInvalidFileSize             = stdErrors.New("invalid file size")
	ErrCannotClassifyEmptyDocument = stdErrors.New("cannot classify document without content")

	// Classification errors
	ErrInvalidConfidence         = stdErrors.New("confidence must be between 0.0 and 1.0")
	ErrInvalidClassification     = stdErrors.New("invalid classification")
	ErrClassifierNotConfigured   = stdErrors.New("classifier not configured")
	ErrUnsupportedClassification = stdErrors.New("classification type not supported")

	// Legal entity errors
	ErrInvalidCaseNumber = stdErrors.New("invalid case number format")
	ErrInvalidPartyRole  = stdErrors.New("invalid party role")
	ErrInvalidBarNumber  = stdErrors.New("invalid bar number")

	// Repository errors
	ErrDuplicateDocument   = stdErrors.New("document already exists")
	ErrConcurrencyConflict = stdErrors.New("document was modified by another process")
)

// ValidationError represents a validation failure
type ValidationError struct {
	Field   string
	Message string
	Value   interface{}
}

// Error returns the formatted validation error.
func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation failed for %s: %s", e.Field, e.Message)
}

// NewValidationError creates a new validation error.
func NewValidationError(field, message string, value interface{}) *ValidationError {
	return &ValidationError{
		Field:   field,
		Message: message,
		Value:   value,
	}
}
