package processing

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	"motion-index-fiber/internal/application/dto"
	"motion-index-fiber/internal/infrastructure/http/presenter"
)

// UpdateMetadata handles PUT /api/documents/:id/metadata - Update document metadata.
func (h *ProcessingHandlerRefactored) UpdateMetadata(c *fiber.Ctx) error {
	var body struct {
		Language     string   `json:"language"`
		LegalTags    []string `json:"legal_tags"`
		AIClassified bool     `json:"ai_classified"`
	}

	if err := c.BodyParser(&body); err != nil {
		return presenter.BadRequest(c, "Invalid request body", nil)
	}

	req := &dto.UpdateMetadataRequest{
		DocumentID:   c.Params("id"),
		Language:     body.Language,
		LegalTags:    body.LegalTags,
		AIClassified: body.AIClassified,
	}

	result, err := h.updateMetadataUC.Execute(c.Context(), req)
	if err != nil {
		return presenter.InternalError(c, fmt.Sprintf("Metadata update failed: %v", err))
	}

	return presenter.Success(c, presenter.PresentMetadata(result))
}
