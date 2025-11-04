package search

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"

	"motion-index-fiber/internal/config"
	"motion-index-fiber/pkg/search"
)

type DocumentHandler struct {
	cfg           *config.Config
	searchService search.Service
}

func NewDocumentHandler(cfg *config.Config, searchService search.Service) *DocumentHandler {
	return &DocumentHandler{
		cfg:           cfg,
		searchService: searchService,
	}
}

// GetDocument handles GET /documents/{id}
func (h *DocumentHandler) GetDocument(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.Context(), 10*time.Second)
	defer cancel()

	docID := c.Params("id")
	if docID == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Document ID is required")
	}

	document, err := h.searchService.GetDocument(ctx, docID)
	if err != nil {
		if err.Error() == "document not found" {
			return fiber.NewError(fiber.StatusNotFound, "Document not found")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to retrieve document: "+err.Error())
	}

	return c.JSON(fiber.Map{
		"status": "success",
		"data":   document,
	})
}

// DeleteDocument handles DELETE /documents/{id} (protected endpoint)
func (h *DocumentHandler) DeleteDocument(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.Context(), 10*time.Second)
	defer cancel()

	docID := c.Params("id")
	if docID == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Document ID is required")
	}

	err := h.searchService.DeleteDocument(ctx, docID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to delete document: "+err.Error())
	}

	return c.JSON(fiber.Map{
		"status":  "success",
		"message": "Document deleted successfully",
	})
}
