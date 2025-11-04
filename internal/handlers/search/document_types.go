package search

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"

	"motion-index-fiber/internal/config"
	"motion-index-fiber/pkg/search"
)

type DocumentTypesHandler struct {
	cfg           *config.Config
	searchService search.Service
}

func NewDocumentTypesHandler(cfg *config.Config, searchService search.Service) *DocumentTypesHandler {
	return &DocumentTypesHandler{
		cfg:           cfg,
		searchService: searchService,
	}
}

// GetDocumentTypes handles GET /document-types
func (h *DocumentTypesHandler) GetDocumentTypes(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.Context(), 10*time.Second)
	defer cancel()

	types, err := h.searchService.GetDocumentTypes(ctx)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to retrieve document types: "+err.Error())
	}

	return c.JSON(fiber.Map{
		"status": "success",
		"data":   types,
	})
}
