package indexing

import (
	"context"

	"github.com/gofiber/fiber/v2"
	"motion-index-fiber/internal/application/dto"
	"motion-index-fiber/internal/infrastructure/http/presenter"
)

// IndexDocumentExecutor defines the interface for indexing documents.
type IndexDocumentExecutor interface {
	Execute(ctx context.Context, req *dto.IndexDocumentRequest) (*dto.IndexDocumentResponse, error)
}

// IndexingHandlerRefactored handles document indexing using DDD use cases.
type IndexingHandlerRefactored struct {
	indexUC IndexDocumentExecutor
}

// NewIndexingHandlerRefactored creates a refactored indexing handler with use case injection.
func NewIndexingHandlerRefactored(indexUC IndexDocumentExecutor) *IndexingHandlerRefactored {
	return &IndexingHandlerRefactored{
		indexUC: indexUC,
	}
}

// IndexDocument handles POST /api/v1/index/document - Index or reindex a document.
func (h *IndexingHandlerRefactored) IndexDocument(c *fiber.Ctx) error {
	var body struct {
		DocumentID string `json:"document_id"`
		Force      bool   `json:"force"`
	}

	if err := c.BodyParser(&body); err != nil {
		return presenter.BadRequest(c, "Invalid request body", nil)
	}

	req := &dto.IndexDocumentRequest{
		DocumentID: body.DocumentID,
		Force:      body.Force,
	}

	result, err := h.indexUC.Execute(c.Context(), req)
	if err != nil {
		if isNotFoundError(err) {
			return presenter.NotFound(c, "Document not found")
		}
		return presenter.InternalError(c, "Failed to index document")
	}

	return presenter.Success(c, presenter.PresentIndex(result))
}

// isNotFoundError checks if an error is a document not found error.
func isNotFoundError(err error) bool {
	return err != nil && (err.Error() == "document not found" ||
		err.Error() == "document with that ID does not exist")
}
