package storage

import (
	"context"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"motion-index-fiber/internal/config"
	"motion-index-fiber/internal/models"
	"motion-index-fiber/pkg/storage"
)

type CountHandler struct {
	cfg     *config.Config
	storage storage.Service
}

func NewCountHandler(cfg *config.Config, storage storage.Service) *CountHandler {
	return &CountHandler{
		cfg:     cfg,
		storage: storage,
	}
}

// GetDocumentsCount handles GET /api/storage/documents/count - Get total document count
func (h *CountHandler) GetDocumentsCount(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.Context(), 60*time.Second)
	defer cancel()

	// Parse query parameters for filtering
	prefix := c.Query("prefix", "documents/")
	fileType := c.Query("file_type", "")
	minSizeStr := c.Query("min_size", "0")
	maxSizeStr := c.Query("max_size", "")

	// Parse size filters
	minSize, _ := strconv.ParseInt(minSizeStr, 10, 64)
	var maxSize int64 = -1
	if maxSizeStr != "" {
		maxSize, _ = strconv.ParseInt(maxSizeStr, 10, 64)
	}

	// List and filter documents
	objects, err := h.storage.List(ctx, prefix)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.NewErrorResponse(
			"storage_error",
			"Failed to count documents",
			map[string]interface{}{"error": err.Error()},
		))
	}

	// Create a temporary list handler to use the filter function
	listHandler := &ListHandler{cfg: h.cfg, storage: h.storage}
	filtered := listHandler.filterDocuments(objects, fileType, minSize, maxSize)

	response := map[string]interface{}{
		"total_count": len(filtered),
		"prefix":      prefix,
		"applied_filters": map[string]interface{}{
			"file_type": fileType,
			"min_size":  minSize,
			"max_size":  maxSize,
		},
	}

	return c.JSON(models.NewSuccessResponse(response, "Document count retrieved successfully"))
}
