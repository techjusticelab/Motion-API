package search

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"

	"motion-index-fiber/internal/config"
	"motion-index-fiber/pkg/search"
)

type FieldOptionsHandler struct {
	cfg           *config.Config
	searchService search.Service
}

func NewFieldOptionsHandler(cfg *config.Config, searchService search.Service) *FieldOptionsHandler {
	return &FieldOptionsHandler{
		cfg:           cfg,
		searchService: searchService,
	}
}

// GetFieldOptions handles GET /field-options
func (h *FieldOptionsHandler) GetFieldOptions(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.Context(), 15*time.Second)
	defer cancel()

	options, err := h.searchService.GetAllFieldOptions(ctx)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to retrieve field options: "+err.Error())
	}

	return c.JSON(fiber.Map{
		"status": "success",
		"data":   options,
	})
}
