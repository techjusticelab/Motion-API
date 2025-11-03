package errors

import (
	stdErrors "errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestErrorMessages(t *testing.T) {
	cases := []struct {
		name    string
		err     error
		message string
	}{
		{"ErrDocumentNotFound", ErrDocumentNotFound, "document not found"},
		{"ErrEmptyDocumentID", ErrEmptyDocumentID, "document ID cannot be empty"},
		{"ErrEmptyFileName", ErrEmptyFileName, "file name cannot be empty"},
		{"ErrInvalidFileName", ErrInvalidFileName, "invalid file name"},
		{"ErrInvalidFilePath", ErrInvalidFilePath, "invalid file path"},
		{"ErrEmptyFilePath", ErrEmptyFilePath, "file path cannot be empty"},
		{"ErrInvalidS3URI", ErrInvalidS3URI, "invalid S3 URI"},
		{"ErrEmptyHash", ErrEmptyHash, "hash cannot be empty"},
		{"ErrInvalidHash", ErrInvalidHash, "invalid hash value"},
		{"ErrInvalidContentType", ErrInvalidContentType, "invalid content type"},
		{"ErrInvalidFileSize", ErrInvalidFileSize, "invalid file size"},
		{"ErrCannotClassifyEmptyDocument", ErrCannotClassifyEmptyDocument, "cannot classify document without content"},
		{"ErrInvalidConfidence", ErrInvalidConfidence, "confidence must be between 0.0 and 1.0"},
		{"ErrInvalidClassification", ErrInvalidClassification, "invalid classification"},
		{"ErrClassifierNotConfigured", ErrClassifierNotConfigured, "classifier not configured"},
		{"ErrUnsupportedClassification", ErrUnsupportedClassification, "classification type not supported"},
		{"ErrInvalidCaseNumber", ErrInvalidCaseNumber, "invalid case number format"},
		{"ErrInvalidPartyRole", ErrInvalidPartyRole, "invalid party role"},
		{"ErrInvalidBarNumber", ErrInvalidBarNumber, "invalid bar number"},
		{"ErrDuplicateDocument", ErrDuplicateDocument, "document already exists"},
		{"ErrConcurrencyConflict", ErrConcurrencyConflict, "document was modified by another process"},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.message, tt.err.Error())
		})
	}
}

func TestErrorWrapping(t *testing.T) {
	wrapped := fmt.Errorf("wrapping: %w", ErrDocumentNotFound)
	assert.True(t, stdErrors.Is(wrapped, ErrDocumentNotFound))
}

func TestValidationError_Error(t *testing.T) {
	err := &ValidationError{
		Field:   "fileName",
		Message: "cannot be empty",
		Value:   "",
	}
	assert.Equal(t, "validation failed for fileName: cannot be empty", err.Error())
}

func TestNewValidationError(t *testing.T) {
	err := NewValidationError("hash", "invalid length", "abc")
	assert.Equal(t, "hash", err.Field)
	assert.Equal(t, "invalid length", err.Message)
	assert.Equal(t, "abc", err.Value)
}
