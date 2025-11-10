package validation

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

// ValidateSearchQuery validates and sanitizes a search query
func ValidateSearchQuery(query string) (string, error) {
	if len(query) == 0 {
		return "", fmt.Errorf("search query cannot be empty")
	}

	if len(query) > MaxSearchQueryLength {
		return "", fmt.Errorf("search query too long: %d characters, maximum %d", len(query), MaxSearchQueryLength)
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
	if len(metadata) > MaxMetadataFields {
		return fmt.Errorf("too many metadata fields: %d, maximum %d", len(metadata), MaxMetadataFields)
	}

	for key, value := range metadata {
		// Validate key
		if len(key) == 0 {
			return fmt.Errorf("metadata key cannot be empty")
		}

		if len(key) > MaxMetadataKeyLength {
			return fmt.Errorf("metadata key too long: %s (%d characters, maximum %d)", key, len(key), MaxMetadataKeyLength)
		}

		// Validate value
		if len(value) > MaxMetadataValueLength {
			return fmt.Errorf("metadata value too long for key %s: %d characters, maximum %d", key, len(value), MaxMetadataValueLength)
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
	maxSize := int64(DefaultMaxFileSize)

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
