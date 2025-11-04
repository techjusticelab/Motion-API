package search

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"

	"motion-index-fiber/internal/config"
	"motion-index-fiber/pkg/search"
)

type DocumentStatsHandler struct {
	cfg           *config.Config
	searchService search.Service
}

func NewDocumentStatsHandler(cfg *config.Config, searchService search.Service) *DocumentStatsHandler {
	return &DocumentStatsHandler{
		cfg:           cfg,
		searchService: searchService,
	}
}

// GetDocumentStats handles GET /document-stats
func (h *DocumentStatsHandler) GetDocumentStats(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.Context(), 15*time.Second)
	defer cancel()

	stats, err := h.searchService.GetDocumentStats(ctx)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to retrieve document stats: "+err.Error())
	}

	return c.JSON(fiber.Map{
		"status": "success",
		"data":   stats,
	})
}
