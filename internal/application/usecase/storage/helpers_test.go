package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsDirectory(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{"directory path", "documents/folder/", true},
		{"file path", "documents/file.pdf", false},
		{"root directory", "documents/", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isDirectory(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsSystemFile(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{"DS_Store file", "documents/.DS_Store", true},
		{"MACOSX file", "documents/__MACOSX/file.pdf", true},
		{"tmp file", "documents/file.tmp", true},
		{"log file", "documents/file.log", true},
		{"regular PDF", "documents/file.pdf", false},
		{"regular DOCX", "documents/document.docx", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isSystemFile(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMatchesFileType(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		fileType string
		expected bool
	}{
		{"PDF matches", "documents/file.pdf", ".pdf", true},
		{"DOCX matches", "documents/file.docx", ".docx", true},
		{"case insensitive PDF", "documents/file.PDF", ".pdf", true},
		{"PDF doesn't match DOCX", "documents/file.pdf", ".docx", false},
		{"extension only", "documents/file.pdf", "pdf", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matchesFileType(tt.path, tt.fileType)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetFileExtension(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{"PDF file", "documents/file.pdf", ".pdf"},
		{"DOCX file", "documents/file.docx", ".docx"},
		{"uppercase extension", "documents/file.PDF", ".pdf"},
		{"no extension", "documents/file", ""},
		{"multiple dots", "documents/file.backup.pdf", ".pdf"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getFileExtension(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetFilename(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{"simple path", "documents/file.pdf", "file.pdf"},
		{"nested path", "documents/subfolder/file.pdf", "file.pdf"},
		{"root path", "file.pdf", "file.pdf"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getFilename(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEncodeCursor(t *testing.T) {
	tests := []struct {
		name     string
		index    int
		notEmpty bool
	}{
		{"index 0", 0, true},
		{"index 10", 10, true},
		{"index 100", 100, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cursor := encodeCursor(tt.index)
			if tt.notEmpty {
				assert.NotEmpty(t, cursor)
			}
		})
	}
}

func TestDecodeCursor(t *testing.T) {
	tests := []struct {
		name          string
		index         int
		expectSuccess bool
	}{
		{"valid cursor with index 0", 0, true},
		{"valid cursor with index 10", 10, true},
		{"valid cursor with index 100", 100, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// First encode
			cursor := encodeCursor(tt.index)

			// Then decode
			decoded, err := decodeCursor(cursor)

			if tt.expectSuccess {
				assert.NoError(t, err)
				assert.Equal(t, tt.index, decoded)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestDecodeCursor_Invalid(t *testing.T) {
	tests := []struct {
		name   string
		cursor string
	}{
		{"empty cursor", ""},
		{"invalid base64", "not-valid-base64!!!"},
		{"invalid JSON", "YWJjZGVm"}, // "abcdef" in base64
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := decodeCursor(tt.cursor)
			assert.Error(t, err)
		})
	}
}

func TestGetContentTypeFromExtension(t *testing.T) {
	tests := []struct {
		ext      string
		expected string
	}{
		{".pdf", "application/pdf"},
		{".docx", "application/vnd.openxmlformats-officedocument.wordprocessingml.document"},
		{".doc", "application/msword"},
		{".txt", "text/plain"},
		{".rtf", "application/rtf"},
		{".json", "application/json"},
		{".xml", "application/xml"},
		{".html", "text/html"},
		{".htm", "text/html"},
		{".jpg", "image/jpeg"},
		{".jpeg", "image/jpeg"},
		{".png", "image/png"},
		{".gif", "image/gif"},
		{".webp", "image/webp"},
		{".bmp", "image/bmp"},
		{".tiff", "image/tiff"},
		{".tif", "image/tiff"},
		{".unknown", "application/octet-stream"},
		{"", "application/octet-stream"},
	}

	for _, tt := range tests {
		t.Run("extension "+tt.ext, func(t *testing.T) {
			result := getContentTypeFromExtension(tt.ext)
			assert.Equal(t, tt.expected, result)
		})
	}
}
