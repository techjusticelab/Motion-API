package models

import (
	"mime/multipart"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProcessOptions_Validate(t *testing.T) {
	tests := []struct {
		name     string
		opts     *ProcessOptions
		expected *ProcessOptions
	}{
		{
			name: "nil options",
			opts: nil,
		},
		{
			name: "valid options",
			opts: &ProcessOptions{
				ExtractText:    true,
				ClassifyDoc:    true,
				IndexDocument:  true,
				StoreDocument:  true,
				TimeoutSeconds: 60,
				RetryCount:     2,
			},
			expected: &ProcessOptions{
				ExtractText:    true,
				ClassifyDoc:    true,
				IndexDocument:  true,
				StoreDocument:  true,
				TimeoutSeconds: 60,
				RetryCount:     2,
			},
		},
		{
			name: "invalid timeout - too low",
			opts: &ProcessOptions{
				TimeoutSeconds: 0,
				RetryCount:     1,
			},
			expected: &ProcessOptions{
				TimeoutSeconds: 120, // Should be corrected to default
				RetryCount:     1,
			},
		},
		{
			name: "invalid timeout - too high",
			opts: &ProcessOptions{
				TimeoutSeconds: 500,
				RetryCount:     1,
			},
			expected: &ProcessOptions{
				TimeoutSeconds: 120, // Should be corrected to default
				RetryCount:     1,
			},
		},
		{
			name: "invalid retry count - too high",
			opts: &ProcessOptions{
				TimeoutSeconds: 60,
				RetryCount:     10,
			},
			expected: &ProcessOptions{
				TimeoutSeconds: 60,
				RetryCount:     1, // Should be corrected to default
			},
		},
		{
			name: "invalid retry count - negative",
			opts: &ProcessOptions{
				TimeoutSeconds: 60,
				RetryCount:     -1,
			},
			expected: &ProcessOptions{
				TimeoutSeconds: 60,
				RetryCount:     1, // Should be corrected to default
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.opts.Validate()
			assert.NoError(t, err)

			if tt.expected != nil {
				assert.Equal(t, tt.expected.TimeoutSeconds, tt.opts.TimeoutSeconds)
				assert.Equal(t, tt.expected.RetryCount, tt.opts.RetryCount)
			}
		})
	}
}

func TestProcessOptions_ApplyDefaults(t *testing.T) {
	t.Run("nil options", func(t *testing.T) {
		var opts *ProcessOptions
		opts.ApplyDefaults() // Should not panic
	})

	t.Run("zero values", func(t *testing.T) {
		opts := &ProcessOptions{}
		opts.ApplyDefaults()

		assert.Equal(t, 120, opts.TimeoutSeconds)
		assert.Equal(t, 1, opts.RetryCount)
	})

	t.Run("partial values", func(t *testing.T) {
		opts := &ProcessOptions{
			TimeoutSeconds: 60,
			// RetryCount is 0, should be set to default
		}
		opts.ApplyDefaults()

		assert.Equal(t, 60, opts.TimeoutSeconds) // Should not change
		assert.Equal(t, 1, opts.RetryCount)      // Should be set to default
	})
}

func TestDefaultProcessOptions(t *testing.T) {
	opts := DefaultProcessOptions()

	assert.NotNil(t, opts)
	assert.True(t, opts.ExtractText)
	assert.True(t, opts.ClassifyDoc)
	assert.True(t, opts.IndexDocument)
	assert.True(t, opts.StoreDocument)
	assert.Equal(t, 120, opts.TimeoutSeconds)
	assert.Equal(t, 1, opts.RetryCount)
}

func TestNewSuccessResponse(t *testing.T) {
	data := map[string]interface{}{"key": "value"}
	message := "Operation successful"

	response := NewSuccessResponse(data, message)

	assert.True(t, response.Success)
	assert.Equal(t, message, response.Message)
	assert.Equal(t, data, response.Data)
	assert.Nil(t, response.Error)
	assert.False(t, response.Timestamp.IsZero())
}

func TestNewErrorResponse(t *testing.T) {
	code := "test_error"
	message := "Test error message"
	details := map[string]interface{}{"field": "value"}

	response := NewErrorResponse(code, message, details)

	assert.False(t, response.Success)
	assert.Empty(t, response.Message)
	assert.Nil(t, response.Data)
	assert.NotNil(t, response.Error)
	assert.Equal(t, code, response.Error.Code)
	assert.Equal(t, message, response.Error.Message)
	assert.Equal(t, details, response.Error.Details)
	assert.False(t, response.Timestamp.IsZero())
}

func TestNewValidationErrorResponse(t *testing.T) {
	field := "email"
	message := "Invalid email format"

	response := NewValidationErrorResponse(field, message)

	assert.False(t, response.Success)
	assert.NotNil(t, response.Error)
	assert.Equal(t, "validation_error", response.Error.Code)
	assert.Equal(t, message, response.Error.Message)
	assert.Equal(t, field, response.Error.Field)
	assert.False(t, response.Timestamp.IsZero())
}

func TestFormatValidationErrors(t *testing.T) {
	// This test would require setting up a validator with actual validation errors
	// For now, test the basic structure
	t.Run("non-validator error", func(t *testing.T) {
		err := assert.AnError
		errors := FormatValidationErrors(err)
		assert.Empty(t, errors)
	})
}

func TestSanitizeInput(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "clean input",
			input:    "Hello World",
			expected: "Hello World",
		},
		{
			name:     "input with null bytes",
			input:    "Hello\x00World",
			expected: "HelloWorld",
		},
		{
			name:     "input with whitespace",
			input:    "  Hello World  ",
			expected: "Hello World",
		},
		{
			name:     "input with script tag",
			input:    "Hello <script>alert('xss')</script> World",
			expected: "Hello  World",
		},
		{
			name:     "input with javascript",
			input:    "Hello javascript:alert('xss') World",
			expected: "Hello  World",
		},
		{
			name:     "input with iframe",
			input:    "Hello <iframe src='evil.com'></iframe> World",
			expected: "Hello  World",
		},
		{
			name:     "mixed case dangerous content",
			input:    "Hello <SCRIPT>alert('xss')</SCRIPT> World",
			expected: "Hello <SCRIPT>alert('xss')</SCRIPT> World", // Only lowercase is caught
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeInput(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidateSearchQuery(t *testing.T) {
	tests := []struct {
		name      string
		query     string
		expectErr bool
		expected  string
	}{
		{
			name:      "valid query",
			query:     "search terms",
			expectErr: false,
			expected:  "search terms",
		},
		{
			name:      "empty query",
			query:     "",
			expectErr: true,
		},
		{
			name:      "query too long",
			query:     strings.Repeat("a", 501),
			expectErr: true,
		},
		{
			name:      "query with dangerous content",
			query:     "search <script>alert('xss')</script> terms",
			expectErr: false,
			expected:  "search  terms",
		},
		{
			name:      "query becomes empty after sanitization",
			query:     "<script></script>",
			expectErr: true,
		},
		{
			name:      "query with whitespace",
			query:     "  search terms  ",
			expectErr: false,
			expected:  "search terms",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ValidateSearchQuery(tt.query)

			if tt.expectErr {
				assert.Error(t, err)
				assert.Empty(t, result)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestValidateMetadata(t *testing.T) {
	tests := []struct {
		name      string
		metadata  map[string]string
		expectErr bool
	}{
		{
			name:      "nil metadata",
			metadata:  nil,
			expectErr: false,
		},
		{
			name: "valid metadata",
			metadata: map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
			expectErr: false,
		},
		{
			name:      "too many fields",
			metadata:  generateLargeMetadata(51),
			expectErr: true,
		},
		{
			name: "empty key",
			metadata: map[string]string{
				"": "value",
			},
			expectErr: true,
		},
		{
			name: "key too long",
			metadata: map[string]string{
				strings.Repeat("a", 101): "value",
			},
			expectErr: true,
		},
		{
			name: "value too long",
			metadata: map[string]string{
				"key": strings.Repeat("a", 1001),
			},
			expectErr: true,
		},
		{
			name: "dangerous key",
			metadata: map[string]string{
				"key<script>": "value",
			},
			expectErr: true,
		},
		{
			name: "dangerous value",
			metadata: map[string]string{
				"key": "value<script>alert('xss')</script>",
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMetadata(tt.metadata)

			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDefaultFileValidationRules(t *testing.T) {
	rules := DefaultFileValidationRules()

	assert.NotNil(t, rules)
	assert.Equal(t, int64(100*1024*1024), rules.MaxSize)
	assert.Equal(t, int64(1), rules.MinSize)
	assert.Contains(t, rules.AllowedExtensions, "pdf")
	assert.Contains(t, rules.AllowedExtensions, "docx")
	assert.Contains(t, rules.AllowedMimeTypes, "application/pdf")
}

func TestValidateFile(t *testing.T) {
	rules := DefaultFileValidationRules()

	tests := []struct {
		name      string
		file      *multipart.FileHeader
		expectErr bool
	}{
		{
			name:      "nil file",
			file:      nil,
			expectErr: true,
		},
		{
			name: "valid file",
			file: &multipart.FileHeader{
				Filename: "document.pdf",
				Size:     1024,
			},
			expectErr: false,
		},
		{
			name: "file too small",
			file: &multipart.FileHeader{
				Filename: "document.pdf",
				Size:     0,
			},
			expectErr: true,
		},
		{
			name: "file too large",
			file: &multipart.FileHeader{
				Filename: "document.pdf",
				Size:     rules.MaxSize + 1,
			},
			expectErr: true,
		},
		{
			name: "invalid extension",
			file: &multipart.FileHeader{
				Filename: "document.exe",
				Size:     1024,
			},
			expectErr: true,
		},
		{
			name: "valid docx file",
			file: &multipart.FileHeader{
				Filename: "document.docx",
				Size:     2048,
			},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateFile(tt.file, rules)

			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateFiles(t *testing.T) {
	rules := DefaultFileValidationRules()
	maxFiles := 5

	tests := []struct {
		name      string
		files     []*multipart.FileHeader
		expectErr bool
	}{
		{
			name:      "no files",
			files:     []*multipart.FileHeader{},
			expectErr: true,
		},
		{
			name: "valid files",
			files: []*multipart.FileHeader{
				{Filename: "doc1.pdf", Size: 1024},
				{Filename: "doc2.docx", Size: 2048},
			},
			expectErr: false,
		},
		{
			name: "too many files",
			files: []*multipart.FileHeader{
				{Filename: "doc1.pdf", Size: 1024},
				{Filename: "doc2.pdf", Size: 1024},
				{Filename: "doc3.pdf", Size: 1024},
				{Filename: "doc4.pdf", Size: 1024},
				{Filename: "doc5.pdf", Size: 1024},
				{Filename: "doc6.pdf", Size: 1024}, // Exceeds maxFiles
			},
			expectErr: true,
		},
		{
			name: "invalid file in batch",
			files: []*multipart.FileHeader{
				{Filename: "doc1.pdf", Size: 1024},
				{Filename: "doc2.exe", Size: 1024}, // Invalid extension
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateFiles(tt.files, rules, maxFiles)

			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Helper function to generate large metadata for testing
func generateLargeMetadata(size int) map[string]string {
	metadata := make(map[string]string)
	for i := 0; i < size; i++ {
		key := "key" + string(rune(i))
		value := "value" + string(rune(i))
		metadata[key] = value
	}
	return metadata
}
