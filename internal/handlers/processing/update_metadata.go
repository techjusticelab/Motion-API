package processing

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	internalModels "motion-index-fiber/internal/models"
)

// UpdateMetadata updates metadata for an existing document
func (h *Handler) UpdateMetadata(c *fiber.Ctx) error {
	var request internalModels.UpdateMetadataRequest

	// Parse request body
	if err := c.BodyParser(&request); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(internalModels.NewErrorResponse(
			"parse_error",
			"Failed to parse request body",
			map[string]interface{}{"error": err.Error()},
		))
	}

	// Validate request
	if request.DocumentID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(internalModels.NewErrorResponse(
			"validation_error",
			"Document ID is required",
			nil,
		))
	}

	// Update metadata using search service
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Convert map[string]string to map[string]interface{}
	metadata := make(map[string]interface{})
	for k, v := range request.Metadata {
		metadata[k] = v
	}

	err := h.searchSvc.UpdateDocumentMetadata(ctx, request.DocumentID, metadata)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(internalModels.NewErrorResponse(
			"update_error",
			"Failed to update document metadata",
			map[string]interface{}{"error": err.Error()},
		))
	}

	response := &internalModels.UpdateMetadataResponse{
		DocumentID: request.DocumentID,
		UpdatedAt:  time.Now(),
		Status:     "success",
	}

	return c.JSON(internalModels.NewSuccessResponse(response, "Metadata updated successfully"))
}
