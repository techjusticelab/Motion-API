package models

import (
	"fmt"
	"mime/multipart"
	"strings"

	"github.com/go-playground/validator/v10"
)

var validate *validator.Validate

func init() {
	validate = validator.New()

	// Register custom validators
	validate.RegisterValidation("file_extension", validateFileExtension)
	validate.RegisterValidation("file_size", validateFileSize)
	validate.RegisterValidation("legal_category", validateLegalCategory)
}

// GetValidator returns the singleton validator instance
func GetValidator() *validator.Validate {
	return validate
}

// ValidateStruct validates a struct using the configured validator
func ValidateStruct(s interface{}) error {
	return validate.Struct(s)
}

// ValidationError represents a structured validation error
type ValidationError struct {
	Field   string `json:"field"`
	Tag     string `json:"tag"`
	Value   string `json:"value"`
	Message string `json:"message"`
}

// FormatValidationErrors converts validator errors to structured format
func FormatValidationErrors(err error) []*ValidationError {
	var validationErrors []*ValidationError

	if validatorErrors, ok := err.(validator.ValidationErrors); ok {
		for _, fieldError := range validatorErrors {
			validationError := &ValidationError{
				Field:   fieldError.Field(),
				Tag:     fieldError.Tag(),
				Value:   fieldError.Param(),
				Message: getErrorMessage(fieldError),
			}
			validationErrors = append(validationErrors, validationError)
		}
	}

	return validationErrors
}

// getErrorMessage returns a human-readable error message for a validation error
func getErrorMessage(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return fmt.Sprintf("%s is required", fe.Field())
	case "min":
		return fmt.Sprintf("%s must be at least %s characters", fe.Field(), fe.Param())
	case "max":
		return fmt.Sprintf("%s must be at most %s characters", fe.Field(), fe.Param())
	case "email":
		return fmt.Sprintf("%s must be a valid email address", fe.Field())
	case "oneof":
		return fmt.Sprintf("%s must be one of: %s", fe.Field(), fe.Param())
	case "file_extension":
		return fmt.Sprintf("%s must have a valid file extension: %s", fe.Field(), fe.Param())
	case "file_size":
		return fmt.Sprintf("%s file size exceeds maximum allowed size", fe.Field())
	case "legal_category":
		return fmt.Sprintf("%s must be a valid legal document category", fe.Field())
	case "gtfield":
		return fmt.Sprintf("%s must be greater than %s", fe.Field(), fe.Param())
	case "ltfield":
		return fmt.Sprintf("%s must be less than %s", fe.Field(), fe.Param())
	default:
		return fmt.Sprintf("%s is invalid", fe.Field())
	}
}

// Custom validation functions

// validateFileExtension validates that a file has an allowed extension
func validateFileExtension(fl validator.FieldLevel) bool {
	file, ok := fl.Field().Interface().(*multipart.FileHeader)
	if !ok {
		return false
	}

	allowedExtensions := strings.Split(fl.Param(), "|")
	filename := strings.ToLower(file.Filename)

	for _, ext := range allowedExtensions {
		if strings.HasSuffix(filename, "."+ext) {
			return true
		}
	}

	return false
}

// validateFileSize validates that a file is within size limits
func validateFileSize(fl validator.FieldLevel) bool {
	file, ok := fl.Field().Interface().(*multipart.FileHeader)
	if !ok {
		return false
	}

	// Default max size is 100MB if not specified
	maxSize := int64(100 * 1024 * 1024)

	if fl.Param() != "" {
		// Custom max size could be implemented here
		// For now, use default
	}

	return file.Size <= maxSize
}

// validateLegalCategory validates legal document categories
func validateLegalCategory(fl validator.FieldLevel) bool {
	category := fl.Field().String()

	validCategories := map[string]bool{
		"motion":    true,
		"order":     true,
		"contract":  true,
		"brief":     true,
		"memo":      true,
		"pleading":  true,
		"discovery": true,
		"exhibit":   true,
		"judgment":  true,
		"other":     true,
	}

	return validCategories[strings.ToLower(category)]
}

// FileValidationRules defines validation rules for file uploads
type FileValidationRules struct {
	MaxSize           int64    `json:"max_size_bytes"`
	AllowedExtensions []string `json:"allowed_extensions"`
	AllowedMimeTypes  []string `json:"allowed_mime_types"`
	MinSize           int64    `json:"min_size_bytes"`
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

// ValidateFile validates a file against the given rules
func ValidateFile(file *multipart.FileHeader, rules *FileValidationRules) error {
	if file == nil {
		return fmt.Errorf("file is required")
	}

	// Check file size
	if file.Size < rules.MinSize {
		return fmt.Errorf("file size %d bytes is below minimum %d bytes", file.Size, rules.MinSize)
	}

	if file.Size > rules.MaxSize {
		return fmt.Errorf("file size %d bytes exceeds maximum %d bytes", file.Size, rules.MaxSize)
	}

	// Check file extension
	filename := strings.ToLower(file.Filename)
	validExtension := false
	for _, ext := range rules.AllowedExtensions {
		if strings.HasSuffix(filename, "."+ext) {
			validExtension = true
			break
		}
	}

	if !validExtension {
		return fmt.Errorf("file extension not allowed. Allowed extensions: %s",
			strings.Join(rules.AllowedExtensions, ", "))
	}

	return nil
}

// ValidateFiles validates multiple files
func ValidateFiles(files []*multipart.FileHeader, rules *FileValidationRules, maxFiles int) error {
	if len(files) == 0 {
		return fmt.Errorf("at least one file is required")
	}

	if len(files) > maxFiles {
		return fmt.Errorf("too many files: %d, maximum allowed: %d", len(files), maxFiles)
	}

	for i, file := range files {
		if err := ValidateFile(file, rules); err != nil {
			return fmt.Errorf("file %d (%s): %w", i+1, file.Filename, err)
		}
	}

	return nil
}

// SanitizeInput sanitizes user input to prevent injection attacks
func SanitizeInput(input string) string {
	// Remove null bytes
	input = strings.ReplaceAll(input, "\x00", "")

	// Trim whitespace
	input = strings.TrimSpace(input)

	// Remove complete script blocks (case sensitive, only lowercase)
	input = removeScriptBlocks(input)

	// Remove complete iframe blocks (case sensitive, only lowercase)
	input = removeIframeBlocks(input)

	// Remove javascript: protocols and everything after them until whitespace
	input = removeJavaScriptProtocol(input)

	// Remove other dangerous patterns
	dangerousPatterns := []string{
		"vbscript:", "onload=", "onerror=", "<object", "</object>", "<embed", "</embed>",
	}

	for _, pattern := range dangerousPatterns {
		input = strings.ReplaceAll(input, pattern, "")
	}

	return input
}

// removeScriptBlocks removes complete <script>...</script> blocks (case sensitive, lowercase only)
func removeScriptBlocks(input string) string {
	for {
		start := strings.Index(input, "<script")
		if start == -1 {
			break
		}

		// Find the end of the opening tag
		tagEnd := strings.Index(input[start:], ">")
		if tagEnd == -1 {
			// Malformed tag, just remove what we found
			input = input[:start] + input[start+7:] // Remove "<script"
			continue
		}
		tagEnd += start + 1 // Absolute position after ">"

		// Find the closing </script> tag
		end := strings.Index(input[tagEnd:], "</script>")
		if end == -1 {
			// No closing tag, remove from start position to end
			input = input[:start]
			break
		}
		end += tagEnd + 9 // Absolute position after "</script>"

		// Remove the entire script block
		input = input[:start] + input[end:]
	}
	return input
}

// removeIframeBlocks removes complete <iframe>...</iframe> blocks (case sensitive, lowercase only)
func removeIframeBlocks(input string) string {
	for {
		start := strings.Index(input, "<iframe")
		if start == -1 {
			break
		}

		// Find the end of the opening tag
		tagEnd := strings.Index(input[start:], ">")
		if tagEnd == -1 {
			// Malformed tag, just remove what we found
			input = input[:start] + input[start+7:] // Remove "<iframe"
			continue
		}
		tagEnd += start + 1 // Absolute position after ">"

		// Find the closing </iframe> tag
		end := strings.Index(input[tagEnd:], "</iframe>")
		if end == -1 {
			// No closing tag, remove from start position to end
			input = input[:start]
			break
		}
		end += tagEnd + 9 // Absolute position after "</iframe>"

		// Remove the entire iframe block
		input = input[:start] + input[end:]
	}
	return input
}

// removeJavaScriptProtocol removes javascript: and everything following until whitespace
func removeJavaScriptProtocol(input string) string {
	for {
		start := strings.Index(input, "javascript:")
		if start == -1 {
			break
		}

		// Find the end - either whitespace or end of string
		end := start + 11 // Length of "javascript:"
		for end < len(input) && !strings.ContainsRune(" \t\n\r", rune(input[end])) {
			end++
		}

		// Remove the javascript: protocol and its content
		input = input[:start] + input[end:]
	}
	return input
}

// ValidateSearchQuery validates and sanitizes a search query
func ValidateSearchQuery(query string) (string, error) {
	if len(query) == 0 {
		return "", fmt.Errorf("search query cannot be empty")
	}

	if len(query) > 500 {
		return "", fmt.Errorf("search query too long: %d characters, maximum 500", len(query))
	}

	// Sanitize the query
	sanitized := SanitizeInput(query)

	// Check for minimum length after sanitization
	if len(sanitized) == 0 {
		return "", fmt.Errorf("search query is empty after sanitization")
	}

	return sanitized, nil
}

// ValidateMetadata validates document metadata
func ValidateMetadata(metadata map[string]string) error {
	if metadata == nil {
		return nil
	}

	// Check for maximum number of metadata fields
	if len(metadata) > 50 {
		return fmt.Errorf("too many metadata fields: %d, maximum 50", len(metadata))
	}

	for key, value := range metadata {
		// Validate key
		if len(key) == 0 {
			return fmt.Errorf("metadata key cannot be empty")
		}

		if len(key) > 100 {
			return fmt.Errorf("metadata key too long: %s (%d characters, maximum 100)", key, len(key))
		}

		// Validate value
		if len(value) > 1000 {
			return fmt.Errorf("metadata value too long for key %s: %d characters, maximum 1000", key, len(value))
		}

		// Sanitize both key and value
		sanitizedKey := SanitizeInput(key)
		sanitizedValue := SanitizeInput(value)

		if sanitizedKey != key {
			return fmt.Errorf("metadata key contains invalid characters: %s", key)
		}

		if sanitizedValue != value {
			return fmt.Errorf("metadata value contains invalid characters for key %s", key)
		}
	}

	return nil
}
