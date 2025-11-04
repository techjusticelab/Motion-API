package storage

import (
	"context"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"motion-index-fiber/internal/config"
	"motion-index-fiber/internal/models"
	"motion-index-fiber/pkg/storage"
)

type SearchHandler struct {
	cfg     *config.Config
	storage storage.Service
}

func NewSearchHandler(cfg *config.Config, storage storage.Service) *SearchHandler {
	return &SearchHandler{
		cfg:     cfg,
		storage: storage,
	}
}

// FindDocumentsByName handles GET /api/v1/files/search - Find documents by filename pattern
func (h *SearchHandler) FindDocumentsByName(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.Context(), 60*time.Second)
	defer cancel()

	// Get search parameters
	namePattern := c.Query("name", "")
	prefix := c.Query("prefix", "documents/")
	limitStr := c.Query("limit", "20")
	exactMatch := c.Query("exact", "false") == "true"

	if namePattern == "" {
		return c.Status(fiber.StatusBadRequest).JSON(models.NewErrorResponse(
			"missing_parameter",
			"Search pattern is required",
			map[string]interface{}{"parameter": "name"},
		))
	}

	// Parse limit
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 || limit > 100 {
		limit = 20
	}

	// List all documents from storage
	objects, err := h.storage.List(ctx, prefix)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.NewErrorResponse(
			"storage_error",
			"Failed to search documents",
			map[string]interface{}{"error": err.Error()},
		))
	}

	// Filter by name pattern
	var matches []map[string]interface{}
	namePattern = strings.ToLower(namePattern)

	for _, obj := range objects {
		// Skip directories
		if strings.HasSuffix(obj.Path, "/") {
			continue
		}

		filename := strings.ToLower(filepath.Base(obj.Path))

		var isMatch bool
		if exactMatch {
			isMatch = filename == namePattern || filename == namePattern+filepath.Ext(filename)
		} else {
			isMatch = strings.Contains(filename, namePattern)
		}

		if isMatch {
			// Generate both direct and CDN URLs
			directURL := h.storage.GetURL(obj.Path)
			signedURL, _ := h.storage.GetSignedURL(obj.Path, time.Hour)

			matches = append(matches, map[string]interface{}{
				"path":          obj.Path,
				"filename":      filepath.Base(obj.Path),
				"size":          obj.Size,
				"last_modified": obj.LastModified,
				"file_type":     strings.ToLower(filepath.Ext(obj.Path)),
				"direct_url":    directURL,
				"signed_url":    signedURL,
				"api_url":       fmt.Sprintf("/api/v1/files/%s", strings.TrimPrefix(obj.Path, "documents/")),
			})

			if len(matches) >= limit {
				break
			}
		}
	}

	response := map[string]interface{}{
		"documents":      matches,
		"total_found":    len(matches),
		"search_pattern": namePattern,
		"exact_match":    exactMatch,
		"limit":          limit,
	}

	return c.JSON(models.NewSuccessResponse(response, "Documents found successfully"))
}
