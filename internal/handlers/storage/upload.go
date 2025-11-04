package storage

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"motion-index-fiber/internal/config"
	"motion-index-fiber/internal/models"
	"motion-index-fiber/pkg/storage"
)

type UploadHandler struct {
	cfg     *config.Config
	storage storage.Service
}

func NewUploadHandler(cfg *config.Config, storage storage.Service) *UploadHandler {
	return &UploadHandler{
		cfg:     cfg,
		storage: storage,
	}
}

// UploadDocumentToS3 handles POST /upload/s3 - Upload documents to DigitalOcean Spaces
func (h *UploadHandler) UploadDocumentToS3(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.Context(), 10*time.Minute)
	defer cancel()

	// Parse multipart form data
	form, err := c.MultipartForm()
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.NewErrorResponse(
			"invalid_form_data",
			"Failed to parse multipart form data",
			map[string]interface{}{"error": err.Error()},
		))
	}

	// Get uploaded files
	files := form.File["file"]
	if len(files) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(models.NewErrorResponse(
			"no_file_provided",
			"No file was provided in the request",
			map[string]interface{}{"field": "file"},
		))
	}

	var uploadedFiles []map[string]interface{}
	var errors []string

	// Process each uploaded file
	for _, file := range files {
		// Validate file type
		if !isValidFileType(file.Filename) {
			errors = append(errors, fmt.Sprintf("Invalid file type for %s. Allowed types: pdf, doc, docx, ppt, pptx, txt", file.Filename))
			continue
		}

		// Validate file size (max 50MB)
		const maxFileSize = 50 * 1024 * 1024 // 50MB
		if file.Size > maxFileSize {
			errors = append(errors, fmt.Sprintf("File %s is too large. Maximum size: 50MB", file.Filename))
			continue
		}

		// Generate random ID
		randomID, err := generateRandomID()
		if err != nil {
			errors = append(errors, fmt.Sprintf("Failed to generate ID for %s: %v", file.Filename, err))
			continue
		}

		// Create file path in unprocessed folder
		ext := filepath.Ext(file.Filename)
		filename := fmt.Sprintf("%s_%s%s", randomID, sanitizeFilename(strings.TrimSuffix(file.Filename, ext)), ext)
		storagePath := fmt.Sprintf("unprocessed/%s", filename)

		// Open the uploaded file
		src, err := file.Open()
		if err != nil {
			errors = append(errors, fmt.Sprintf("Failed to open file %s: %v", file.Filename, err))
			continue
		}

		// Create upload metadata
		metadata := &storage.UploadMetadata{
			ContentType: getContentTypeFromExtension(ext),
			Size:        file.Size,
			FileName:    file.Filename,
			Tags: map[string]string{
				"status":    "unprocessed",
				"upload_id": randomID,
			},
		}

		// Upload to DigitalOcean Spaces
		result, err := h.storage.Upload(ctx, storagePath, src, metadata)
		src.Close()

		if err != nil {
			errors = append(errors, fmt.Sprintf("Failed to upload file %s: %v", file.Filename, err))
			continue
		}

		// Add successful upload to response
		uploadedFiles = append(uploadedFiles, map[string]interface{}{
			"original_filename": file.Filename,
			"storage_path":      storagePath,
			"file_id":           randomID,
			"size":              file.Size,
			"content_type":      getContentTypeFromExtension(ext),
			"upload_url":        result.URL,
			"uploaded_at":       result.UploadedAt,
			"etag":              result.ETag,
		})
	}

	// Prepare response
	response := map[string]interface{}{
		"uploaded_files": uploadedFiles,
		"total_uploaded": len(uploadedFiles),
		"total_files":    len(files),
	}

	// Include errors if any
	if len(errors) > 0 {
		response["errors"] = errors
		response["total_errors"] = len(errors)
	}

	// Return appropriate status code
	if len(uploadedFiles) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(models.NewErrorResponse(
			"upload_failed",
			"No files were successfully uploaded",
			response,
		))
	}

	if len(errors) > 0 {
		return c.Status(fiber.StatusPartialContent).JSON(models.NewSuccessResponse(
			response,
			"Some files uploaded successfully, but some failed",
		))
	}

	return c.JSON(models.NewSuccessResponse(response, "All files uploaded successfully"))
}

// isValidFileType checks if the file type is allowed
func isValidFileType(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	validExtensions := map[string]bool{
		".pdf":  true,
		".doc":  true,
		".docx": true,
		".ppt":  true,
		".pptx": true,
		".txt":  true,
	}
	return validExtensions[ext]
}

// generateRandomID creates a random ID for file naming
func generateRandomID() (string, error) {
	bytes := make([]byte, 16)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// sanitizeFilename removes problematic characters from filename
func sanitizeFilename(filename string) string {
	// Replace problematic characters with underscores
	reg := regexp.MustCompile(`[<>:"/\\|?*\x00-\x1f]`)
	sanitized := reg.ReplaceAllString(filename, "_")

	// Remove multiple consecutive underscores
	reg = regexp.MustCompile(`_+`)
	sanitized = reg.ReplaceAllString(sanitized, "_")

	// Trim underscores from start and end
	sanitized = strings.Trim(sanitized, "_")

	// Ensure filename is not empty
	if sanitized == "" {
		sanitized = "unnamed"
	}

	return sanitized
}

// getContentTypeFromExtension returns the MIME type for a file extension
func getContentTypeFromExtension(ext string) string {
	switch strings.ToLower(ext) {
	case ".pdf":
		return "application/pdf"
	case ".docx":
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	case ".doc":
		return "application/msword"
	case ".pptx":
		return "application/vnd.openxmlformats-officedocument.presentationml.presentation"
	case ".ppt":
		return "application/vnd.ms-powerpoint"
	case ".txt":
		return "text/plain"
	default:
		return "application/octet-stream"
	}
}
