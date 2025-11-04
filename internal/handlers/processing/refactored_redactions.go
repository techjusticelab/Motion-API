package processing

import (
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"
	"motion-index-fiber/internal/application/dto"
	"motion-index-fiber/internal/infrastructure/http/presenter"
)

// AnalyzeRedactions handles POST /api/documents/redactions/analyze - Analyze document for redactions.
func (h *ProcessingHandlerRefactored) AnalyzeRedactions(c *fiber.Ctx) error {
	contentType := c.Get("Content-Type")

	var req *dto.AnalyzeRedactionsRequest

	if strings.Contains(contentType, "multipart/form-data") {
		req = h.parseRedactionFromMultipart(c)
	} else {
		req = h.parseRedactionFromJSON(c)
	}

	if req == nil {
		return presenter.BadRequest(c, "Invalid request", nil)
	}

	result, err := h.analyzeRedactionUC.Execute(c.Context(), req)
	if err != nil {
		return presenter.InternalError(c, fmt.Sprintf("Redaction analysis failed: %v", err))
	}

	return presenter.Success(c, result)
}

// ApplyRedactions handles POST /api/documents/redactions/apply - Apply redactions to document.
func (h *ProcessingHandlerRefactored) ApplyRedactions(c *fiber.Ctx) error {
	var body struct {
		DocumentID       string                `json:"document_id"`
		PDFBase64        string                `json:"pdf_base64"`
		CustomRedactions []dto.RedactionItem   `json:"custom_redactions"`
		Options          *dto.RedactionOptions `json:"options"`
	}

	if err := c.BodyParser(&body); err != nil {
		return presenter.BadRequest(c, "Invalid request body", nil)
	}

	req := &dto.ApplyRedactionsRequest{
		DocumentID:       body.DocumentID,
		PDFBase64:        body.PDFBase64,
		Options:          body.Options,
		CustomRedactions: body.CustomRedactions,
		ReturnBase64:     true,
	}

	result, err := h.applyRedactionUC.Execute(c.Context(), req)
	if err != nil {
		return presenter.InternalError(c, fmt.Sprintf("Redaction application failed: %v", err))
	}

	return presenter.Success(c, result)
}
