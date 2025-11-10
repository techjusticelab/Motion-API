package validation

// FileValidationRules defines validation rules for file uploads
type FileValidationRules struct {
	MaxSize           int64    `json:"max_size_bytes"`
	AllowedExtensions []string `json:"allowed_extensions"`
	AllowedMimeTypes  []string `json:"allowed_mime_types"`
	MinSize           int64    `json:"min_size_bytes"`
}

// ValidationError represents a structured validation error
type ValidationError struct {
	Field   string `json:"field"`
	Tag     string `json:"tag"`
	Value   string `json:"value"`
	Message string `json:"message"`
}

// DefaultFileValidationRules returns default file validation rules
func DefaultFileValidationRules() *FileValidationRules {
	return &FileValidationRules{
		MaxSize: 100 * 1024 * 1024, // 100MB
		AllowedExtensions: []string{
			"pdf", "doc", "docx", "txt", "rtf", "html", "htm",
		},
		AllowedMimeTypes: []string{
			"application/pdf",
			"application/msword",
			"application/vnd.openxmlformats-officedocument.wordprocessingml.document",
			"text/plain",
			"application/rtf",
			"text/html",
		},
		MinSize: 1, // 1 byte minimum
	}
}

// Validation rule constants
const (
	MaxMetadataFields      = 50
	MaxMetadataKeyLength   = 100
	MaxMetadataValueLength = 1000
	MaxSearchQueryLength   = 500
	DefaultMaxFileSize     = 100 * 1024 * 1024 // 100MB
	MinFileSize            = 1                  // 1 byte
)
