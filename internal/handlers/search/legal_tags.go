package search

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"

	"motion-index-fiber/internal/config"
	"motion-index-fiber/pkg/search"
)

type LegalTagsHandler struct {
	cfg           *config.Config
	searchService search.Service
}

func NewLegalTagsHandler(cfg *config.Config, searchService search.Service) *LegalTagsHandler {
	return &LegalTagsHandler{
		cfg:           cfg,
		searchService: searchService,
	}
}

// GetLegalTags handles GET /legal-tags
func (h *LegalTagsHandler) GetLegalTags(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.Context(), 10*time.Second)
	defer cancel()

	tags, err := h.searchService.GetLegalTags(ctx)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to retrieve legal tags: "+err.Error())
	}

	return c.JSON(fiber.Map{
		"status": "success",
		"data":   tags,
	})
}
