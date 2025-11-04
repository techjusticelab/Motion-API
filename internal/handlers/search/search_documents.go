package search

import (
	"context"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"

	"motion-index-fiber/internal/config"
	"motion-index-fiber/pkg/models"
	"motion-index-fiber/pkg/search"
)

type SearchDocumentsHandler struct {
	cfg           *config.Config
	searchService search.Service
}

func NewSearchDocumentsHandler(cfg *config.Config, searchService search.Service) *SearchDocumentsHandler {
	return &SearchDocumentsHandler{
		cfg:           cfg,
		searchService: searchService,
	}
}

// SearchDocuments handles POST /search
func (h *SearchDocumentsHandler) SearchDocuments(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.Context(), 30*time.Second)
	defer cancel()

	var req models.SearchRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid request body: "+err.Error())
	}

	// Parse query parameters if body is empty
	if req.Query == "" {
		req.Query = c.Query("q", "")
	}
	if req.Size == 0 {
		if sizeStr := c.Query("size"); sizeStr != "" {
			if size, err := strconv.Atoi(sizeStr); err == nil {
				req.Size = size
			}
		}
	}
	if req.From == 0 {
		if fromStr := c.Query("from"); fromStr != "" {
			if from, err := strconv.Atoi(fromStr); err == nil {
				req.From = from
			}
		}
	}

	// Validate request
	if err := validateSearchRequest(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	// Execute search
	result, err := h.searchService.SearchDocuments(ctx, &req)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Search failed: "+err.Error())
	}

	return c.JSON(fiber.Map{
		"status":  "success",
		"data":    result,
		"message": "Search completed successfully",
	})
}
