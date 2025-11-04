package handlers

import (
	"github.com/gofiber/fiber/v2"
	"motion-index-fiber/internal/config"
	"motion-index-fiber/internal/handlers/indexing"
	"motion-index-fiber/pkg/search"
)

// IndexingHandler handles direct document indexing operations
type IndexingHandler struct {
	cfg                  *config.Config
	search               search.Service
	indexDocumentHandler *indexing.IndexDocumentHandler
	refactoredHandler    *indexing.IndexingHandlerRefactored
}

// NewIndexingHandler creates a new indexing handler
func NewIndexingHandler(cfg *config.Config, search search.Service) *IndexingHandler {
	return &IndexingHandler{
		cfg:                  cfg,
		search:               search,
		indexDocumentHandler: indexing.NewIndexDocumentHandler(search),
		// refactoredHandler will be initialized when DDD use cases are available
		refactoredHandler: nil,
	}
}

// IndexDocument delegates to the index document handler
func (h *IndexingHandler) IndexDocument(c *fiber.Ctx) error {
	return h.indexDocumentHandler.IndexDocument(c)
}

// SetRefactoredHandler allows setting the DDD-based handler when use cases are available
func (h *IndexingHandler) SetRefactoredHandler(refactoredHandler *indexing.IndexingHandlerRefactored) {
	h.refactoredHandler = refactoredHandler
}

// IndexDocumentRefactored provides an alternative endpoint using DDD architecture
// This method can be used when the refactored handler is initialized
func (h *IndexingHandler) IndexDocumentRefactored(c *fiber.Ctx) error {
	if h.refactoredHandler == nil {
		// Fall back to the standard handler if refactored version is not available
		return h.IndexDocument(c)
	}
	return h.refactoredHandler.IndexDocument(c)
}
