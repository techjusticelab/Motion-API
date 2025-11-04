package indexing

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/gofiber/fiber/v2"

	internalModels "motion-index-fiber/internal/models"
	"motion-index-fiber/pkg/search"
)

type IndexDocumentHandler struct {
	search search.Service
}

func NewIndexDocumentHandler(search search.Service) *IndexDocumentHandler {
	return &IndexDocumentHandler{
		search: search,
	}
}

// IndexDocument handles POST /api/v1/index/document - Direct document indexing
func (h *IndexDocumentHandler) IndexDocument(c *fiber.Ctx) error {
	var request internalModels.IndexDocumentRequest
	if err := c.BodyParser(&request); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(internalModels.NewErrorResponse(
			"parse_error",
			"Failed to parse request body",
			map[string]interface{}{"error": err.Error()},
		))
	}

	// Validate required fields
	if err := validateIndexRequest(&request); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(internalModels.NewErrorResponse(
			"validation_error",
			err.Error(),
			nil,
		))
	}

	// Check search service health
	if !h.search.IsHealthy() {
		return c.Status(fiber.StatusServiceUnavailable).JSON(internalModels.NewErrorResponse(
			"service_unavailable",
			"Search service is not healthy",
			nil,
		))
	}

	ctx := context.Background()

	// Index the document
	log.Printf("[INDEXING] Processing document: %s", request.DocumentID)
	indexID, err := indexDocument(ctx, h.search, &request)
	if err != nil {
		log.Printf("[INDEXING] ❌ Failed to index document %s: %v", request.DocumentID, err)
		return c.Status(fiber.StatusInternalServerError).JSON(internalModels.NewErrorResponse(
			"indexing_failed",
			fmt.Sprintf("Failed to index document: %v", err),
			map[string]interface{}{
				"document_id": request.DocumentID,
				"error":       err.Error(),
			},
		))
	}
	log.Printf("[INDEXING] ✅ Successfully indexed document %s with ID: %s", request.DocumentID, indexID)

	// Create response
	response := &internalModels.IndexDocumentResponse{
		DocumentID: request.DocumentID,
		IndexID:    indexID,
		Success:    true,
		Message:    "Document indexed successfully",
		IndexedAt:  time.Now(),
	}

	return c.JSON(internalModels.NewSuccessResponse(response, "Document indexed successfully"))
}
