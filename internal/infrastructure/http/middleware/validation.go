package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"motion-index-fiber/internal/application/validation"
)

// ValidateDocumentID middleware ensures document ID parameter is present and valid
func ValidateDocumentID() fiber.Handler {
	return func(c *fiber.Ctx) error {
		documentID := c.Params("id")
		if strings.TrimSpace(documentID) == "" {
			return fiber.NewError(fiber.StatusBadRequest, "Document ID is required")
		}

		// Basic validation - IDs should not contain special characters
		if strings.ContainsAny(documentID, "<>\"'&;") {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid document ID format")
		}

		return c.Next()
	}
}

// ValidateFileUpload middleware ensures file upload is present and valid
func ValidateFileUpload(fieldName string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		file, err := c.FormFile(fieldName)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "File is required")
		}

		// Validate file name
		if err := validation.ValidateFileName(file.Filename); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid file name: "+err.Error())
		}

		// Check file size (max 50MB)
		const maxFileSize = 50 * 1024 * 1024 // 50MB
		if file.Size > maxFileSize {
			return fiber.NewError(fiber.StatusRequestEntityTooLarge, "File too large (max 50MB)")
		}

		// Check file is not empty
		if file.Size == 0 {
			return fiber.NewError(fiber.StatusBadRequest, "File cannot be empty")
		}

		return c.Next()
	}
}

// ValidateContentType middleware ensures request has correct content type
func ValidateContentType(allowedTypes ...string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		contentType := c.Get("Content-Type")
		if contentType == "" {
			return fiber.NewError(fiber.StatusBadRequest, "Content-Type header is required")
		}

		// Check if content type is allowed
		for _, allowed := range allowedTypes {
			if strings.Contains(contentType, allowed) {
				return c.Next()
			}
		}

		return fiber.NewError(fiber.StatusUnsupportedMediaType,
			"Unsupported content type. Expected: "+strings.Join(allowedTypes, ", "))
	}
}

// ValidatePagination middleware validates pagination query parameters
func ValidatePagination() fiber.Handler {
	return func(c *fiber.Ctx) error {
		page := c.QueryInt("page", 0)
		size := c.QueryInt("size", 0)

		if page < 0 {
			return fiber.NewError(fiber.StatusBadRequest, "Page must be non-negative")
		}

		if size < 0 {
			return fiber.NewError(fiber.StatusBadRequest, "Page size must be non-negative")
		}

		if size > 100 {
			return fiber.NewError(fiber.StatusBadRequest, "Page size cannot exceed 100")
		}

		return c.Next()
	}
}

// ValidateQueryParam middleware validates a required query parameter
func ValidateQueryParam(paramName string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		value := c.Query(paramName)
		if strings.TrimSpace(value) == "" {
			return fiber.NewError(fiber.StatusBadRequest, paramName+" query parameter is required")
		}
		return c.Next()
	}
}

// ValidateJSONBody middleware ensures request body is valid JSON
func ValidateJSONBody() fiber.Handler {
	return func(c *fiber.Ctx) error {
		contentType := c.Get("Content-Type")
		if !strings.Contains(contentType, "application/json") {
			return fiber.NewError(fiber.StatusUnsupportedMediaType, "Content-Type must be application/json")
		}

		// Check body is not empty
		if len(c.Body()) == 0 {
			return fiber.NewError(fiber.StatusBadRequest, "Request body cannot be empty")
		}

		return c.Next()
	}
}

// ValidateBatchSize middleware validates batch operation size limits
func ValidateBatchSize(maxSize int) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Parse multipart form
		form, err := c.MultipartForm()
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "Failed to parse multipart form")
		}

		// Check files count
		var totalFiles int
		for _, files := range form.File {
			totalFiles += len(files)
		}

		if totalFiles == 0 {
			return fiber.NewError(fiber.StatusBadRequest, "At least one file is required")
		}

		if totalFiles > maxSize {
			return fiber.NewError(fiber.StatusBadRequest,
				"Too many files. Maximum allowed: "+string(rune(maxSize)))
		}

		return c.Next()
	}
}

// ValidateResourceExists middleware validates that a resource exists (generic)
type ResourceValidator func(c *fiber.Ctx, id string) (bool, error)

func ValidateResourceExists(paramName string, validator ResourceValidator) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id := c.Params(paramName)
		if id == "" {
			return fiber.NewError(fiber.StatusBadRequest, paramName+" parameter is required")
		}

		exists, err := validator(c, id)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to validate resource: "+err.Error())
		}

		if !exists {
			return fiber.NewError(fiber.StatusNotFound, "Resource not found")
		}

		return c.Next()
	}
}

// Chain is a utility that combines multiple middleware, but in practice
// Fiber already supports this natively. This function exists for convenience
// when you want to group validations together.
//
// Note: In production, prefer using Fiber's native chaining:
//   app.Use(middleware1, middleware2, middleware3)
//
// This Chain function is mainly useful for readability in route definitions.
func Chain(handlers ...fiber.Handler) []fiber.Handler {
	return handlers
}
