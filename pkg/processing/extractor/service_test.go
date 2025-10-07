package extractor

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewService(t *testing.T) {
	service := NewService()
	assert.NotNil(t, service)

	// Check that default extractors are registered
	formats := service.SupportedFormats()
	assert.Contains(t, formats, "txt")
	assert.Contains(t, formats, "pdf")
	assert.Contains(t, formats, "docx")
}

func TestService_SupportedFormats(t *testing.T) {
	service := NewService()
	formats := service.SupportedFormats()

	// Should contain at least the basic formats
	expectedFormats := []string{"txt", "pdf", "docx"}
	for _, format := range expectedFormats {
		assert.Contains(t, formats, format)
	}
}

func TestService_GetExtractor(t *testing.T) {
	service := NewService()

	tests := []struct {
		name        string
		format      string
		expectError bool
	}{
		{"text format", "txt", false},
		{"PDF format", "pdf", false},
		{"DOCX format", "docx", false},
		{"case insensitive", "TXT", false},
		{"unsupported format", "xyz", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			extractor, err := service.GetExtractor(tt.format)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, extractor)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, extractor)
			}
		})
	}
}

func TestService_ExtractText(t *testing.T) {
	service := NewService()
	ctx := context.Background()

	tests := []struct {
		name     string
		content  string
		metadata *DocumentMetadata
		wantErr  bool
	}{
		{
			name:    "text file extraction",
			content: "This is a test document with some text content.",
			metadata: &DocumentMetadata{
				FileName: "test.txt",
				Format:   "txt",
				Size:     46,
			},
			wantErr: false,
		},
		{
			name:    "auto-detect format",
			content: "Auto-detected test content.",
			metadata: &DocumentMetadata{
				FileName: "document.txt",
				Size:     27,
			},
			wantErr: false,
		},
		{
			name:    "unsupported format",
			content: "Some content",
			metadata: &DocumentMetadata{
				FileName: "file.xyz",
				Format:   "xyz",
				Size:     12,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.content)
			result, err := service.ExtractText(ctx, reader, tt.metadata)

			if tt.wantErr {
				assert.Error(t, err)
				assert.False(t, result.Success)
			} else {
				assert.NoError(t, err)
				assert.True(t, result.Success)
				assert.Contains(t, result.Text, "test")
				assert.GreaterOrEqual(t, result.WordCount, 0)
				assert.GreaterOrEqual(t, result.CharCount, 0)
				assert.GreaterOrEqual(t, result.Duration, int64(0))
			}
		})
	}
}

func TestService_DetectFormat(t *testing.T) {
	service := NewService().(*service)

	tests := []struct {
		name     string
		fileName string
		mimeType string
		expected string
	}{
		{"PDF by extension", "document.pdf", "", "pdf"},
		{"DOCX by extension", "file.docx", "", "docx"},
		{"Text by extension", "readme.txt", "", "txt"},
		{"PDF by MIME type", "document", "application/pdf", "pdf"},
		{"DOCX by MIME type", "file", "application/vnd.openxmlformats-officedocument.wordprocessingml.document", "docx"},
		{"Text by MIME type", "content", "text/plain", "txt"},
		{"Case insensitive", "FILE.PDF", "", "pdf"},
		{"Default fallback", "unknown", "unknown/type", "txt"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.detectFormat(tt.fileName, tt.mimeType)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractionError(t *testing.T) {
	baseErr := assert.AnError
	extractionErr := NewExtractionError("pdf", "extraction failed", baseErr)

	assert.Equal(t, "pdf", extractionErr.Format)
	assert.Equal(t, "extraction failed", extractionErr.Message)
	assert.Equal(t, baseErr, extractionErr.Cause)
	assert.Contains(t, extractionErr.Error(), "extraction failed")
	assert.Contains(t, extractionErr.Error(), baseErr.Error())
	assert.Equal(t, baseErr, extractionErr.Unwrap())
}

func TestExtractionError_WithoutCause(t *testing.T) {
	extractionErr := NewExtractionError("txt", "simple error", nil)

	assert.Equal(t, "simple error", extractionErr.Error())
	assert.Nil(t, extractionErr.Unwrap())
}
