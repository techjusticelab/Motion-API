package storage

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateStorageKey(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		category string
		want     func(result string) bool
	}{
		{
			name:     "with category",
			filename: "test document.pdf",
			category: "motions",
			want: func(result string) bool {
				return strings.HasPrefix(result, "documents/motions/") &&
					strings.Contains(result, "test_document.pdf")
			},
		},
		{
			name:     "without category",
			filename: "test.txt",
			category: "",
			want: func(result string) bool {
				return strings.HasPrefix(result, "documents/uncategorized/") &&
					strings.Contains(result, "test.txt")
			},
		},
		{
			name:     "filename with path",
			filename: "/path/to/file.pdf",
			category: "contracts",
			want: func(result string) bool {
				return strings.HasPrefix(result, "documents/contracts/") &&
					strings.Contains(result, "file.pdf") &&
					!strings.Contains(result, "/path/to/")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateStorageKey(tt.filename, tt.category)
			assert.True(t, tt.want(result), "Result: %s", result)
		})
	}
}

func TestGetContentTypeFromFilename(t *testing.T) {
	tests := []struct {
		filename string
		want     string
	}{
		{"document.pdf", "application/pdf"},
		{"document.PDF", "application/pdf"},
		{"document.docx", "application/vnd.openxmlformats-officedocument.wordprocessingml.document"},
		{"document.doc", "application/msword"},
		{"document.txt", "text/plain; charset=utf-8"}, // mime package adds charset
		{"document.rtf", "application/rtf"},
		{"document.html", "text/html; charset=utf-8"}, // mime package adds charset
		{"document.htm", "text/html; charset=utf-8"},  // mime package adds charset
		{"document.xml", "text/xml; charset=utf-8"},   // mime package returns text/xml not application/xml
		{"document.json", "application/json"},
		{"document.unknown", "application/octet-stream"},
		{"document", "application/octet-stream"},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			result := GetContentTypeFromFilename(tt.filename)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestValidateFileName(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		wantErr  bool
		errType  string
	}{
		{"valid filename", "document.pdf", false, ""},
		{"empty filename", "", true, "invalid_filename"},
		{"contains double dots", "../document.pdf", true, "invalid_filename"},
		{"contains slash", "path/document.pdf", true, "invalid_filename"},
		{"contains backslash", "path\\document.pdf", true, "invalid_filename"},
		{"contains colon", "C:document.pdf", true, "invalid_filename"},
		{"contains asterisk", "doc*.pdf", true, "invalid_filename"},
		{"contains question mark", "doc?.pdf", true, "invalid_filename"},
		{"contains quotes", "doc\"ument.pdf", true, "invalid_filename"},
		{"contains less than", "doc<ument.pdf", true, "invalid_filename"},
		{"contains greater than", "doc>ument.pdf", true, "invalid_filename"},
		{"contains pipe", "doc|ument.pdf", true, "invalid_filename"},
		{"very long filename", strings.Repeat("a", 256) + ".pdf", true, "invalid_filename"},
		{"max length filename", strings.Repeat("a", 251) + ".pdf", false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateFileName(tt.filename)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errType != "" {
					storageErr, ok := err.(*StorageError)
					assert.True(t, ok)
					assert.Equal(t, tt.errType, storageErr.Type)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateFileSize(t *testing.T) {
	tests := []struct {
		name    string
		size    int64
		maxSize int64
		wantErr bool
		errType string
	}{
		{"valid size", 1024, 2048, false, ""},
		{"zero size", 0, 2048, true, "invalid_size"},
		{"negative size", -1, 2048, true, "invalid_size"},
		{"exceeds max size", 3072, 2048, true, "file_too_large"},
		{"equals max size", 2048, 2048, false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateFileSize(tt.size, tt.maxSize)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errType != "" {
					storageErr, ok := err.(*StorageError)
					assert.True(t, ok)
					assert.Equal(t, tt.errType, storageErr.Type)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCalculateHash(t *testing.T) {
	content := "test content"
	reader := strings.NewReader(content)

	hash, err := CalculateHash(reader)
	assert.NoError(t, err)
	assert.NotEmpty(t, hash)
	assert.Equal(t, "9473fdd0d880a43c21b7778d34872157", hash) // MD5 of "test content"
}

func TestSanitizeStoragePath(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{"normal path", "documents/test/file.pdf", "documents/test/file.pdf"},
		{"leading slash", "/documents/test/file.pdf", "documents/test/file.pdf"},
		{"trailing slash", "documents/test/file.pdf/", "documents/test/file.pdf"},
		{"multiple slashes", "documents//test//file.pdf", "documents/test/file.pdf"},
		{"no documents prefix", "test/file.pdf", "documents/test/file.pdf"},
		{"temp path", "temp/file.pdf", "temp/file.pdf"},
		{"uploads path", "uploads/file.pdf", "uploads/file.pdf"},
		{"just filename", "file.pdf", "documents/file.pdf"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeStoragePath(tt.path)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestGetFileExtension(t *testing.T) {
	tests := []struct {
		filename string
		want     string
	}{
		{"document.pdf", ".pdf"},
		{"document.PDF", ".pdf"},
		{"document.docx", ".docx"},
		{"document", ""},
		{"document.tar.gz", ".gz"},
		{".hidden", ".hidden"}, // filepath.Ext(".hidden") returns ".hidden"
		{"document.", "."},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			result := GetFileExtension(tt.filename)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestIsLegalDocumentType(t *testing.T) {
	tests := []struct {
		contentType string
		want        bool
	}{
		{"application/pdf", true},
		{"application/vnd.openxmlformats-officedocument.wordprocessingml.document", true},
		{"application/msword", true},
		{"text/plain", true},
		{"text/plain; charset=utf-8", true}, // With charset
		{"application/rtf", true},
		{"text/html", true},
		{"text/html; charset=utf-8", true}, // With charset
		{"application/xml", true},
		{"text/xml", true},
		{"text/xml; charset=utf-8", true}, // With charset
		{"image/jpeg", false},
		{"video/mp4", false},
		{"application/octet-stream", false},
		{"application/unknown", false},
	}

	for _, tt := range tests {
		t.Run(tt.contentType, func(t *testing.T) {
			result := IsLegalDocumentType(tt.contentType)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestGenerateUniqueKey(t *testing.T) {
	filename := "test document.pdf"

	t.Run("without hash", func(t *testing.T) {
		key := GenerateUniqueKey(filename, false)
		assert.Contains(t, key, "test_document.pdf")
		assert.Regexp(t, `^\d{8}-\d{6}-test_document\.pdf$`, key)
	})

	t.Run("with hash", func(t *testing.T) {
		key := GenerateUniqueKey(filename, true)
		assert.Contains(t, key, "test_document.pdf")
		assert.Regexp(t, `^\d{8}-\d{6}-[a-f0-9]{8}-test_document\.pdf$`, key)
	})

	t.Run("filename with spaces", func(t *testing.T) {
		key := GenerateUniqueKey("my test file.pdf", false)
		assert.Contains(t, key, "my_test_file.pdf")
		assert.NotContains(t, key, " ")
	})
}

func TestParseStorageKey(t *testing.T) {
	tests := []struct {
		name string
		key  string
		want StorageKeyInfo
	}{
		{
			name: "valid key",
			key:  "documents/motions/2023/01/01/file.pdf",
			want: StorageKeyInfo{
				Category: "motions",
				Date:     "2023",
				Filename: "01/01/file.pdf",
				FullPath: "documents/motions/2023/01/01/file.pdf",
				IsValid:  true,
			},
		},
		{
			name: "simple valid key",
			key:  "documents/contracts/2023/contract.pdf",
			want: StorageKeyInfo{
				Category: "contracts",
				Date:     "2023",
				Filename: "contract.pdf",
				FullPath: "documents/contracts/2023/contract.pdf",
				IsValid:  true,
			},
		},
		{
			name: "invalid key - no documents prefix",
			key:  "files/motions/2023/file.pdf",
			want: StorageKeyInfo{
				FullPath: "files/motions/2023/file.pdf",
				IsValid:  false,
			},
		},
		{
			name: "invalid key - too few parts",
			key:  "documents/motions",
			want: StorageKeyInfo{
				FullPath: "documents/motions",
				IsValid:  false,
			},
		},
		{
			name: "filename with subdirectories",
			key:  "documents/legal/2023/subdir/another/file.pdf",
			want: StorageKeyInfo{
				Category: "legal",
				Date:     "2023",
				Filename: "subdir/another/file.pdf",
				FullPath: "documents/legal/2023/subdir/another/file.pdf",
				IsValid:  true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseStorageKey(tt.key)
			assert.Equal(t, tt.want, *result)
		})
	}
}

func TestFileTypeFromExtension(t *testing.T) {
	tests := []struct {
		filename string
		want     string
	}{
		{"document.pdf", "PDF Document"},
		{"document.PDF", "PDF Document"},
		{"document.docx", "Word Document (DOCX)"},
		{"document.doc", "Word Document (DOC)"},
		{"document.txt", "Text Document"},
		{"document.rtf", "Rich Text Document"},
		{"document.html", "HTML Document"},
		{"document.htm", "HTML Document"},
		{"document.xml", "XML Document"},
		{"document.json", "JSON Document"},
		{"document.unknown", "Unknown Document"},
		{"document", "Unknown Document"},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			result := FileTypeFromExtension(tt.filename)
			assert.Equal(t, tt.want, result)
		})
	}
}
