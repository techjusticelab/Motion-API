package handlers

import (
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"motion-index-fiber/internal/application/dto"
	"motion-index-fiber/internal/application/usecase/storage"
	"motion-index-fiber/internal/models"
	storageservice "motion-index-fiber/pkg/storage"
)

// StorageHandlerRefactored provides HTTP handlers for storage operations using DDD use cases.
type StorageHandlerRefactored struct {
	storageService storageservice.Service
	listUC         *storage.ListStorageObjectsUseCase
	countUC        *storage.CountStorageObjectsUseCase
	searchUC       *storage.SearchStorageObjectsUseCase
	getURLUC       *storage.GetStorageObjectURLUseCase
}

// NewStorageHandlerRefactored creates a new refactored storage handler with use case injection.
func NewStorageHandlerRefactored(storageService storageservice.Service) *StorageHandlerRefactored {
	return &StorageHandlerRefactored{
		storageService: storageService,
		listUC:         storage.NewListStorageObjectsUseCase(storageService),
		countUC:        storage.NewCountStorageObjectsUseCase(storageService),
		searchUC:       storage.NewSearchStorageObjectsUseCase(storageService),
		getURLUC:       storage.NewGetStorageObjectURLUseCase(storageService),
	}
}

// ListDocuments handles GET /api/storage/documents - List documents with pagination.
func (h *StorageHandlerRefactored) ListDocuments(c *fiber.Ctx) error {
	// Parse query parameters
	limit, _ := strconv.Atoi(c.Query("limit", "50"))
	minSize, _ := strconv.ParseInt(c.Query("min_size", "0"), 10, 64)
	var maxSize int64 = -1
	if maxSizeStr := c.Query("max_size", ""); maxSizeStr != "" {
		maxSize, _ = strconv.ParseInt(maxSizeStr, 10, 64)
	}

	req := &dto.ListStorageObjectsRequest{
		Prefix:   c.Query("prefix", "documents/"),
		Limit:    limit,
		Cursor:   c.Query("cursor", ""),
		FileType: c.Query("file_type", ""),
		MinSize:  minSize,
		MaxSize:  maxSize,
	}

	result, err := h.listUC.Execute(c.Context(), req)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.NewErrorResponse(
			"invalid_request",
			err.Error(),
			nil,
		))
	}

	return c.JSON(models.NewSuccessResponse(result, "Documents listed successfully"))
}

// GetDocumentsCount handles GET /api/storage/documents/count - Get total document count.
func (h *StorageHandlerRefactored) GetDocumentsCount(c *fiber.Ctx) error {
	minSize, _ := strconv.ParseInt(c.Query("min_size", "0"), 10, 64)
	var maxSize int64 = -1
	if maxSizeStr := c.Query("max_size", ""); maxSizeStr != "" {
		maxSize, _ = strconv.ParseInt(maxSizeStr, 10, 64)
	}

	req := &dto.CountStorageObjectsRequest{
		Prefix:   c.Query("prefix", "documents/"),
		FileType: c.Query("file_type", ""),
		MinSize:  minSize,
		MaxSize:  maxSize,
	}

	result, err := h.countUC.Execute(c.Context(), req)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.NewErrorResponse(
			"invalid_request",
			err.Error(),
			nil,
		))
	}

	return c.JSON(models.NewSuccessResponse(result, "Document count retrieved successfully"))
}

// FindDocumentsByName handles GET /api/v1/files/search - Find documents by filename pattern.
func (h *StorageHandlerRefactored) FindDocumentsByName(c *fiber.Ctx) error {
	limit, _ := strconv.Atoi(c.Query("limit", "20"))

	req := &dto.SearchStorageObjectsRequest{
		NamePattern: c.Query("name", ""),
		Prefix:      c.Query("prefix", "documents/"),
		Limit:       limit,
		ExactMatch:  c.Query("exact", "false") == "true",
	}

	result, err := h.searchUC.Execute(c.Context(), req)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.NewErrorResponse(
			"invalid_request",
			err.Error(),
			nil,
		))
	}

	return c.JSON(models.NewSuccessResponse(result, "Documents found successfully"))
}

// ServeDocument handles GET /api/v1/files/* - Serve or redirect to a document.
func (h *StorageHandlerRefactored) ServeDocument(c *fiber.Ctx) error {
	// Get and decode document path
	rawDocumentPath := c.Params("*")
	if rawDocumentPath == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Document path is required",
		})
	}

	documentPath, err := url.QueryUnescape(rawDocumentPath)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":      "Invalid URL encoding in document path",
			"details":    err.Error(),
			"suggestion": "Ensure the path is properly URL-encoded",
		})
	}

	// Parse URL parameters
	useSignedURL := c.Query("signed", "true") == "true"
	expirationParam := c.Query("expires", "1h")
	expiration, err := time.ParseDuration(expirationParam)
	if err != nil {
		expiration = time.Hour
	}

	// Get document URL using use case
	req := &dto.GetStorageObjectURLRequest{
		Path:         documentPath,
		UseSignedURL: useSignedURL,
		Expiration:   expiration,
	}

	result, err := h.getURLUC.Execute(c.Context(), req)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error":      "Document not found",
				"path":       documentPath,
				"suggestion": "Verify the document exists in storage",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Failed to get document URL",
			"details": err.Error(),
		})
	}

	// Set content type and handle download vs. redirect
	c.Set("Content-Type", result.ContentType)

	if c.Query("download", "false") == "true" {
		c.Set("Content-Disposition", "attachment")
	}

	return c.Redirect(result.URL, fiber.StatusFound)
}
