package middleware

import (
	"fmt"
	"mime/multipart"
	"strings"

	"github.com/gofiber/fiber/v2"
	"motion-index-fiber/internal/models"
)

// FileUploadConfig defines configuration for file upload middleware
type FileUploadConfig struct {
	MaxFileSize       int64    // Maximum file size in bytes
	AllowedExtensions []string // Allowed file extensions
	AllowedMimeTypes  []string // Allowed MIME types
	MaxFiles          int      // Maximum number of files per request
	FieldName         string   // Form field name for file uploads
	BatchFieldName    string   // Form field name for batch uploads
	ValidationRules   *models.FileValidationRules
}

// DefaultFileUploadConfig returns default file upload configuration
func DefaultFileUploadConfig() *FileUploadConfig {
	rules := models.DefaultFileValidationRules()
	return &FileUploadConfig{
		MaxFileSize:       rules.MaxSize,
		AllowedExtensions: rules.AllowedExtensions,
		AllowedMimeTypes:  rules.AllowedMimeTypes,
		MaxFiles:          10,
		FieldName:         "file",
		BatchFieldName:    "files",
		ValidationRules:   rules,
	}
}

// FileUploadMiddleware creates middleware for file upload validation
func FileUploadMiddleware(config ...*FileUploadConfig) fiber.Handler {
	cfg := DefaultFileUploadConfig()
	if len(config) > 0 && config[0] != nil {
		cfg = config[0]
	}

	return func(c *fiber.Ctx) error {
		// Skip if this is not a multipart form request
		contentType := c.Get("Content-Type")
		if !strings.Contains(contentType, "multipart/form-data") {
			return c.Next()
		}

		// Parse multipart form
		form, err := c.MultipartForm()
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(models.NewErrorResponse(
				"multipart_error",
				"Failed to parse multipart form",
				map[string]interface{}{"error": err.Error()},
			))
		}

		// Check for single file upload
		if files, exists := form.File[cfg.FieldName]; exists && len(files) > 0 {
			if err := validateSingleFile(files[0], cfg); err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(models.NewErrorResponse(
					"file_validation_error",
					err.Error(),
					map[string]interface{}{"field": cfg.FieldName},
				))
			}
		}

		// Check for batch file upload
		if files, exists := form.File[cfg.BatchFieldName]; exists && len(files) > 0 {
			if err := validateMultipleFiles(files, cfg); err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(models.NewErrorResponse(
					"file_validation_error",
					err.Error(),
					map[string]interface{}{"field": cfg.BatchFieldName},
				))
			}
		}

		return c.Next()
	}
}

// validateSingleFile validates a single file upload
func validateSingleFile(file *multipart.FileHeader, config *FileUploadConfig) error {
	// Validate file using models validation
	if err := models.ValidateFile(file, config.ValidationRules); err != nil {
		return err
	}

	// Additional middleware-specific validations can be added here
	return nil
}

// validateMultipleFiles validates multiple file uploads
func validateMultipleFiles(files []*multipart.FileHeader, config *FileUploadConfig) error {
	// Check maximum number of files
	if len(files) > config.MaxFiles {
		return fmt.Errorf("too many files: %d, maximum allowed: %d", len(files), config.MaxFiles)
	}

	// Validate each file
	for i, file := range files {
		if err := validateSingleFile(file, config); err != nil {
			return fmt.Errorf("file %d (%s): %w", i+1, file.Filename, err)
		}
	}

	return nil
}

// FileUploadStats tracks file upload statistics
type FileUploadStats struct {
	TotalUploads      int64
	TotalBytes        int64
	SuccessfulUploads int64
	FailedUploads     int64
	RejectedFiles     int64
}

// StatsMiddleware tracks file upload statistics
func StatsMiddleware(stats *FileUploadStats) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Skip if this is not a multipart form request
		contentType := c.Get("Content-Type")
		if !strings.Contains(contentType, "multipart/form-data") {
			return c.Next()
		}

		// Parse multipart form to count files and bytes
		form, err := c.MultipartForm()
		if err != nil {
			return c.Next()
		}

		var fileCount int64
		var totalBytes int64

		// Count files and bytes from all form fields
		for _, files := range form.File {
			for _, file := range files {
				fileCount++
				totalBytes += file.Size
			}
		}

		// Update stats
		stats.TotalUploads += fileCount
		stats.TotalBytes += totalBytes

		// Store original status for post-processing
		err = c.Next()

		// Update success/failure stats based on response status
		if c.Response().StatusCode() >= 400 {
			stats.FailedUploads += fileCount
		} else {
			stats.SuccessfulUploads += fileCount
		}

		return err
	}
}

// SecurityMiddleware provides additional security checks for file uploads
func SecurityMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Skip if this is not a multipart form request
		contentType := c.Get("Content-Type")
		if !strings.Contains(contentType, "multipart/form-data") {
			return c.Next()
		}

		// Parse multipart form
		form, err := c.MultipartForm()
		if err != nil {
			return c.Next()
		}

		// Security checks for all uploaded files
		for fieldName, files := range form.File {
			for _, file := range files {
				// Check for suspicious filenames
				if err := validateSecureFilename(file.Filename); err != nil {
					return c.Status(fiber.StatusBadRequest).JSON(models.NewErrorResponse(
						"security_violation",
						fmt.Sprintf("Suspicious filename detected in field %s: %s", fieldName, err.Error()),
						map[string]interface{}{"field": fieldName, "filename": file.Filename},
					))
				}

				// Additional security checks can be added here:
				// - Virus scanning
				// - Content verification
				// - Magic number validation
			}
		}

		return c.Next()
	}
}

// validateSecureFilename checks for potentially dangerous filenames
func validateSecureFilename(filename string) error {
	// Check for path traversal attempts
	if strings.Contains(filename, "..") {
		return fmt.Errorf("path traversal attempt detected")
	}

	// Check for null bytes
	if strings.Contains(filename, "\x00") {
		return fmt.Errorf("null byte in filename")
	}

	// Check for suspicious characters
	suspicious := []string{
		"<", ">", ":", "\"", "|", "?", "*",
		"\r", "\n", "\t",
	}

	for _, char := range suspicious {
		if strings.Contains(filename, char) {
			return fmt.Errorf("suspicious character '%s' in filename", char)
		}
	}

	// Check for suspicious extensions (double extensions, executable files, etc.)
	lower := strings.ToLower(filename)
	dangerousExtensions := []string{
		".exe", ".scr", ".bat", ".cmd", ".com", ".pif", ".vbs", ".js", ".jar",
		".php", ".asp", ".aspx", ".jsp", ".sh", ".py", ".rb", ".pl",
	}

	for _, ext := range dangerousExtensions {
		if strings.HasSuffix(lower, ext) {
			return fmt.Errorf("dangerous file extension: %s", ext)
		}
	}

	return nil
}

// RequestSizeMiddleware limits the total request size
func RequestSizeMiddleware(maxSize int64) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Check Content-Length header
		contentLength := int64(c.Request().Header.ContentLength())
		if contentLength > maxSize {
			return c.Status(fiber.StatusRequestEntityTooLarge).JSON(models.NewErrorResponse(
				"request_too_large",
				fmt.Sprintf("Request size %d bytes exceeds maximum %d bytes", contentLength, maxSize),
				map[string]interface{}{
					"content_length": fmt.Sprintf("%d", contentLength),
					"max_size":       fmt.Sprintf("%d", maxSize),
				},
			))
		}

		return c.Next()
	}
}
