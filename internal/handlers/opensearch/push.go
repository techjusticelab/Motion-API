package opensearch

import (
	"context"

	"github.com/gofiber/fiber/v2"

	internalModels "motion-index-fiber/internal/models"
	pkgsearch "motion-index-fiber/pkg/search"
)

type Handler struct {
	searchService pkgsearch.Service
}

func NewHandler(searchService pkgsearch.Service) *Handler {
	return &Handler{searchService: searchService}
}

func (h *Handler) PushDocument(c *fiber.Ctx) error {
	if !h.searchService.IsHealthy() {
		return c.Status(fiber.StatusServiceUnavailable).JSON(internalModels.NewErrorResponse(
			"service_unavailable",
			"Search service is not healthy",
			nil,
		))
	}

	body := c.Body()
	if len(body) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(internalModels.NewErrorResponse(
			"invalid_payload",
			"Request body cannot be empty",
			nil,
		))
	}

	docID := c.Query("id")

	indexID, err := h.searchService.IndexRawDocument(context.Background(), docID, body)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(internalModels.NewErrorResponse(
			"indexing_failed",
			"Failed to push document to OpenSearch",
			map[string]interface{}{
				"error": err.Error(),
			},
		))
	}

	responseData := fiber.Map{
		"document_id": indexID,
	}
	if docID != "" {
		responseData["requested_id"] = docID
	}

	return c.Status(fiber.StatusCreated).JSON(internalModels.NewSuccessResponse(
		responseData,
		"Document pushed to OpenSearch successfully",
	))
}
