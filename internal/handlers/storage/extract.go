package storage

import (
	"context"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"motion-index-fiber/internal/config"
	"motion-index-fiber/internal/models"
	pkgextractor "motion-index-fiber/pkg/processing/extractor"
	pkgsearch "motion-index-fiber/pkg/search"
	pkgstorage "motion-index-fiber/pkg/storage"
)

// ExtractHandler handles text extraction from stored documents.
type ExtractHandler struct {
	cfg       *config.Config
	storage   pkgstorage.Service
	search    pkgsearch.Service
	extractor pkgextractor.Service
}

// NewExtractHandler constructs a new ExtractHandler instance.
func NewExtractHandler(cfg *config.Config, storage pkgstorage.Service, search pkgsearch.Service, extractor pkgextractor.Service) *ExtractHandler {
	return &ExtractHandler{
		cfg:       cfg,
		storage:   storage,
		search:    search,
		extractor: extractor,
	}
}

type extractRequest struct {
	DocID       string `json:"doc_id"`
	S3ID        string `json:"s3_id"`
	StoragePath string `json:"storage_path"`
}

// ExtractText downloads a document from storage and returns the extracted text.
func (h *ExtractHandler) ExtractText(c *fiber.Ctx) error {
	var req extractRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.NewErrorResponse(
			"invalid_request",
			"Failed to parse request body",
			map[string]interface{}{"error": err.Error()},
		))
	}

	docID := strings.TrimSpace(req.DocID)
	storagePath := firstNonEmpty(sanitizeStorageKey(req.StoragePath), sanitizeStorageKey(req.S3ID))
	ctx, cancel := context.WithTimeout(c.Context(), 5*time.Minute)
	defer cancel()

	if storagePath == "" && docID != "" && h.search != nil {
		fetchedDoc, err := h.search.GetDocument(ctx, docID)
		if err != nil {
			return c.Status(fiber.StatusNotFound).JSON(models.NewErrorResponse(
				"document_not_found",
				"Document could not be found",
				map[string]interface{}{"doc_id": docID},
			))
		}
		storagePath = sanitizeStorageKey(fetchedDoc.FilePath)
	}

	if storagePath == "" {
		return c.Status(fiber.StatusBadRequest).JSON(models.NewErrorResponse(
			"storage_path_missing",
			"An s3_id or storage_path must be provided",
			nil,
		))
	}

	reader, err := h.storage.Download(ctx, storagePath)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(models.NewErrorResponse(
			"download_failed",
			"Failed to download file from storage",
			map[string]interface{}{"error": err.Error()},
		))
	}
	defer reader.Close()

	tempFile, err := os.CreateTemp("", "extract-*")
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.NewErrorResponse(
			"temp_file_failed",
			"Failed to create temporary file for extraction",
			map[string]interface{}{"error": err.Error()},
		))
	}
	defer func() {
		tempFile.Close()
		os.Remove(tempFile.Name())
	}()

	size, err := io.Copy(tempFile, reader)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.NewErrorResponse(
			"read_failed",
			"Failed to read file contents",
			map[string]interface{}{"error": err.Error()},
		))
	}

	if _, err := tempFile.Seek(0, io.SeekStart); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.NewErrorResponse(
			"seek_failed",
			"Failed to prepare file for extraction",
			map[string]interface{}{"error": err.Error()},
		))
	}

	fileName := path.Base(storagePath)
	metadata := &pkgextractor.DocumentMetadata{
		FileName: fileName,
		MimeType: pkgstorage.GetContentTypeFromFilename(fileName),
		Size:     size,
		Format:   strings.TrimPrefix(strings.ToLower(filepath.Ext(fileName)), "."),
	}

	result, err := h.extractor.ExtractText(ctx, tempFile, metadata)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.NewErrorResponse(
			"extraction_failed",
			"Failed to extract text from document",
			map[string]interface{}{"error": err.Error()},
		))
	}

	response := map[string]interface{}{
		"doc_id":             docID,
		"storage_path":       storagePath,
		"file_name":          fileName,
		"text":               result.Text,
		"word_count":         result.WordCount,
		"char_count":         result.CharCount,
		"page_count":         result.PageCount,
		"language":           result.Language,
		"duration_ms":        result.Duration,
		"metadata":           result.Metadata,
		"extraction_success": result.Success,
	}

	return c.JSON(models.NewSuccessResponse(response, "Text extracted successfully"))
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func sanitizeStorageKey(key string) string {
	key = strings.TrimSpace(key)
	if key == "" {
		return ""
	}

	// Normalise path separators
	key = strings.ReplaceAll(key, "\\", "/")

	// Remove leading relative indicators
	for strings.HasPrefix(key, "./") {
		key = strings.TrimPrefix(key, "./")
	}
	key = strings.TrimPrefix(key, "/")

	cleaned := path.Clean(key)
	cleaned = strings.TrimPrefix(cleaned, "../")
	cleaned = strings.TrimPrefix(cleaned, "./")

	if cleaned == "." || cleaned == "" {
		return ""
	}

	if strings.Contains(cleaned, "..") {
		return ""
	}

	return cleaned
}
