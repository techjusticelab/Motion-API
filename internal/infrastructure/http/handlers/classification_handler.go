package handlers

import (
	"context"

	"github.com/gofiber/fiber/v2"
	"motion-index-fiber/internal/application/dto"
	"motion-index-fiber/internal/infrastructure/http/presenter"
)

// ClassifyDocumentUseCase defines the interface for document classification
type ClassifyDocumentUseCase interface {
	Execute(ctx context.Context, req *dto.ClassifyDocumentRequest) (*dto.ClassificationResponse, error)
}

// ClassificationHandler handles document classification HTTP requests
type ClassificationHandler struct {
	classifyUseCase ClassifyDocumentUseCase
}

// NewClassificationHandler creates a new classification handler
func NewClassificationHandler(classifyUseCase ClassifyDocumentUseCase) *ClassificationHandler {
	return &ClassificationHandler{
		classifyUseCase: classifyUseCase,
	}
}

// ClassifyDocument handles POST /documents/:id/classify
func (h *ClassificationHandler) ClassifyDocument(c *fiber.Ctx) error {
	// Parse document ID from URL
	documentID := c.Params("id")
	if documentID == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Document ID is required")
	}

	// Parse request body for optional parameters
	var body struct {
		Force bool `json:"force"`
	}
	_ = c.BodyParser(&body) // Ignore parsing errors, use defaults

	// Build use case request
	req := &dto.ClassifyDocumentRequest{
		DocumentID: documentID,
		Force:      body.Force,
	}

	// Execute classification use case
	result, err := h.classifyUseCase.Execute(c.Context(), req)
	if err != nil {
		return err // Middleware will translate
	}

	// Present results
	return presenter.Success(c, presenter.PresentClassificationResult(result))
}

// ReclassifyDocument handles PUT /documents/:id/classify (force reclassification)
func (h *ClassificationHandler) ReclassifyDocument(c *fiber.Ctx) error {
	// Parse document ID from URL
	documentID := c.Params("id")
	if documentID == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Document ID is required")
	}

	// Build use case request with force=true
	req := &dto.ClassifyDocumentRequest{
		DocumentID: documentID,
		Force:      true,
	}

	// Execute classification use case
	result, err := h.classifyUseCase.Execute(c.Context(), req)
	if err != nil {
		return err // Middleware will translate
	}

	// Present results
	return presenter.Success(c, presenter.PresentClassificationResult(result))
}
