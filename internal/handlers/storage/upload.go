package storage

import (
	"bytes"
	"context"
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"motion-index-fiber/internal/config"
	internalModels "motion-index-fiber/internal/models"
	pkgmodels "motion-index-fiber/pkg/models"
	pkgextractor "motion-index-fiber/pkg/processing/extractor"
	pkgsearch "motion-index-fiber/pkg/search"
	"motion-index-fiber/pkg/storage"
)

type UploadHandler struct {
	cfg       *config.Config
	storage   storage.Service
	extractor pkgextractor.Service
	search    pkgsearch.Service
}

func NewUploadHandler(cfg *config.Config, storage storage.Service, extractor pkgextractor.Service, search pkgsearch.Service) *UploadHandler {
	return &UploadHandler{
		cfg:       cfg,
		storage:   storage,
		extractor: extractor,
		search:    search,
	}
}

// UploadDocumentToS3 handles POST /upload/s3 - Upload documents to DigitalOcean Spaces
func (h *UploadHandler) UploadDocumentToS3(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.Context(), 10*time.Minute)
	defer cancel()

	// Parse multipart form data
	form, err := c.MultipartForm()
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(internalModels.NewErrorResponse(
			"invalid_form_data",
			"Failed to parse multipart form data",
			map[string]interface{}{"error": err.Error()},
		))
	}

	// Get uploaded files
	files := form.File["file"]
	if len(files) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(internalModels.NewErrorResponse(
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
		return c.Status(fiber.StatusBadRequest).JSON(internalModels.NewErrorResponse(
			"upload_failed",
			"No files were successfully uploaded",
			response,
		))
	}

	if len(errors) > 0 {
		return c.Status(fiber.StatusPartialContent).JSON(internalModels.NewSuccessResponse(
			response,
			"Some files uploaded successfully, but some failed",
		))
	}

	return c.JSON(internalModels.NewSuccessResponse(response, "All files uploaded successfully"))
}

// UploadDocumentAndIndex handles POST /upload/index - upload, extract, and index document in OpenSearch
func (h *UploadHandler) UploadDocumentAndIndex(c *fiber.Ctx) error {
	if h.extractor == nil || h.search == nil {
		return c.Status(fiber.StatusInternalServerError).JSON(internalModels.NewErrorResponse(
			"service_unavailable",
			"Required services are not configured",
			nil,
		))
	}

	ctx, cancel := context.WithTimeout(c.Context(), 15*time.Minute)
	defer cancel()

	fileHeader, err := c.FormFile("file")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(internalModels.NewErrorResponse(
			"file_required",
			"A file must be provided",
			map[string]interface{}{"error": err.Error()},
		))
	}

	if !isValidFileType(fileHeader.Filename) {
		return c.Status(fiber.StatusBadRequest).JSON(internalModels.NewErrorResponse(
			"invalid_file_type",
			"Invalid file type. Allowed types: pdf, doc, docx, ppt, pptx, txt",
			map[string]interface{}{"file": fileHeader.Filename},
		))
	}

	const maxFileSize = 50 * 1024 * 1024
	if fileHeader.Size > maxFileSize {
		return c.Status(fiber.StatusBadRequest).JSON(internalModels.NewErrorResponse(
			"file_too_large",
			"File exceeds the maximum allowed size of 50MB",
			map[string]interface{}{"file": fileHeader.Filename, "size": fileHeader.Size},
		))
	}

	src, err := fileHeader.Open()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(internalModels.NewErrorResponse(
			"file_open_failed",
			"Failed to open uploaded file",
			map[string]interface{}{"error": err.Error()},
		))
	}
	defer src.Close()

	fileBytes, err := io.ReadAll(src)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(internalModels.NewErrorResponse(
			"file_read_failed",
			"Failed to read uploaded file",
			map[string]interface{}{"error": err.Error()},
		))
	}

	ext := filepath.Ext(fileHeader.Filename)
	contentType := getContentTypeFromExtension(ext)
	cleanName := sanitizeFilename(strings.TrimSuffix(fileHeader.Filename, ext))
	if cleanName == "" {
		cleanName = "document"
	}

	docID := generateDocumentID(cleanName)
	storagePath := fmt.Sprintf("uploads/%s%s", docID, ext)

	uploadMetadata := &storage.UploadMetadata{
		ContentType: contentType,
		Size:        fileHeader.Size,
		FileName:    fileHeader.Filename,
		Tags: map[string]string{
			"doc_id": docID,
			"status": "uploaded",
		},
	}

	uploadResult, err := h.storage.Upload(ctx, storagePath, bytes.NewReader(fileBytes), uploadMetadata)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(internalModels.NewErrorResponse(
			"upload_failed",
			"Failed to upload file to storage",
			map[string]interface{}{"error": err.Error()},
		))
	}

	cdnURL := h.storage.GetURL(uploadResult.Path)
	if cdnURL == "" {
		cdnURL = uploadResult.URL
	}

	extractionMetadata := &pkgextractor.DocumentMetadata{
		FileName: fileHeader.Filename,
		MimeType: contentType,
		Size:     fileHeader.Size,
		Format:   strings.TrimPrefix(strings.ToLower(ext), "."),
	}

	extractionResult, err := h.extractor.ExtractText(ctx, bytes.NewReader(fileBytes), extractionMetadata)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(internalModels.NewErrorResponse(
			"extraction_failed",
			"Failed to extract text from document",
			map[string]interface{}{"error": err.Error()},
		))
	}
	if extractionResult == nil || !extractionResult.Success {
		reason := "unknown"
		if extractionResult != nil && extractionResult.Error != "" {
			reason = extractionResult.Error
		}
		return c.Status(fiber.StatusInternalServerError).JSON(internalModels.NewErrorResponse(
			"extraction_incomplete",
			"Text extraction did not complete successfully",
			map[string]interface{}{"reason": reason},
		))
	}

	now := time.Now()
	metadata := pkgmodels.NewDocumentMetadata()
	metadata.DocumentName = fileHeader.Filename
	metadata.DocumentType = pkgmodels.DocTypeUnknown
	metadata.ProcessedAt = now
	metadata.AIClassified = false
	metadata.Status = "uploaded"
	metadata.WordCount = extractionResult.WordCount
	metadata.Pages = extractionResult.PageCount
	if extractionResult.Language != "" {
		metadata.Language = extractionResult.Language
	}
	metadata.Summary = ""
	metadata.Subject = ""
	metadata.SetLegacyFields()

	hash := fmt.Sprintf("%x", md5.Sum(fileBytes))

	var s3URI string
	if bucket := strings.TrimSpace(h.cfg.Storage.Bucket); bucket != "" {
		s3URI = fmt.Sprintf("s3://%s/%s", bucket, uploadResult.Path)
	}

	document := &pkgmodels.Document{
		ID:          docID,
		FileName:    fileHeader.Filename,
		FilePath:    uploadResult.Path,
		FileURL:     cdnURL,
		S3URI:       s3URI,
		Text:        extractionResult.Text,
		DocType:     pkgmodels.DocTypeUnknown.String(),
		Category:    "",
		Hash:        hash,
		CreatedAt:   now,
		UpdatedAt:   now,
		Metadata:    metadata,
		Size:        fileHeader.Size,
		ContentType: contentType,
	}

	indexID, err := h.search.IndexDocument(ctx, document)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(internalModels.NewErrorResponse(
			"indexing_failed",
			"Failed to index document in OpenSearch",
			map[string]interface{}{
				"error":        err.Error(),
				"storage_path": uploadResult.Path,
			},
		))
	}

	response := map[string]interface{}{
		"document_id":   docID,
		"index_id":      indexID,
		"file_name":     fileHeader.Filename,
		"storage_path":  uploadResult.Path,
		"file_url":      cdnURL,
		"cdn_url":       cdnURL,
		"size":          fileHeader.Size,
		"content_type":  contentType,
		"text":          extractionResult.Text,
		"word_count":    extractionResult.WordCount,
		"page_count":    extractionResult.PageCount,
		"metadata":      metadata,
		"uploaded_at":   uploadResult.UploadedAt,
		"extraction_ms": extractionResult.Duration,
	}

	return c.Status(fiber.StatusCreated).JSON(internalModels.NewSuccessResponse(
		response,
		"File uploaded, text extracted, and document indexed successfully",
	))
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

func generateDocumentID(name string) string {
	timestamp := strconv.FormatInt(time.Now().UnixNano(), 10)
	clean := sanitizeFilename(name)
	if clean == "" {
		clean = "document"
	}
	return fmt.Sprintf("doc_%s_%s", timestamp, clean)
}
