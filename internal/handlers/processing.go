package handlers

import (
	"github.com/gofiber/fiber/v2"
	"motion-index-fiber/internal/config"
	proc "motion-index-fiber/internal/handlers/processing"
	"motion-index-fiber/pkg/processing/pipeline"
	"motion-index-fiber/pkg/search"
	"motion-index-fiber/pkg/storage"
)

// ProcessingHandler is a thin wrapper that delegates to the processing subpackage handlers.
type ProcessingHandler struct {
	handler    *proc.Handler
	refactored *proc.ProcessingHandlerRefactored
}

// NewProcessingHandler creates a new wrapper processing handler
func NewProcessingHandler(cfg *config.Config, pipeline pipeline.Pipeline, storage storage.Service, searchSvc search.Service) *ProcessingHandler {
	return &ProcessingHandler{
		handler: proc.NewHandler(cfg, pipeline, storage, searchSvc),
	}
}

// UploadDocument delegates to processing.Handler
func (h *ProcessingHandler) UploadDocument(c *fiber.Ctx) error {
	if h.refactored != nil {
		return h.refactored.ProcessDocument(c)
	}
	return fiber.ErrNotImplemented
}

// AnalyzeRedactions delegates to processing.Handler
func (h *ProcessingHandler) AnalyzeRedactions(c *fiber.Ctx) error {
	if h.refactored != nil {
		return h.refactored.AnalyzeRedactions(c)
	}
	return fiber.ErrNotImplemented
}

// UpdateMetadata delegates to processing.Handler
func (h *ProcessingHandler) UpdateMetadata(c *fiber.Ctx) error {
	if h.refactored != nil {
		return h.refactored.UpdateMetadata(c)
	}
	return fiber.ErrNotImplemented
}

// RedactDocument delegates to processing.Handler
func (h *ProcessingHandler) RedactDocument(c *fiber.Ctx) error {
	if h.refactored != nil {
		return h.refactored.ApplyRedactions(c)
	}
	return fiber.ErrNotImplemented
}

// ProcessDocument delegates to processing.Handler
func (h *ProcessingHandler) ProcessDocument(c *fiber.Ctx) error {
	if h.refactored != nil {
		return h.refactored.ProcessDocument(c)
	}
	return fiber.ErrNotImplemented
}

// BatchProcessDocuments delegates to processing.Handler
func (h *ProcessingHandler) BatchProcessDocuments(c *fiber.Ctx) error {
	if h.refactored != nil {
		return h.refactored.BatchProcessDocuments(c)
	}
	return fiber.ErrNotImplemented
}

// Remaining helpers and refactored handlers moved to the processing subpackage.

// WithRefactored attaches refactored use cases to the processing handler.
// This mirrors the pattern used in storage refactoring and allows gradual migration.
func (h *ProcessingHandler) WithRefactored(
	processUC proc.ProcessDocumentExecutor,
	batchUC proc.BatchProcessExecutor,
	updateMetadataUC proc.UpdateMetadataExecutor,
	analyzeRedactionUC proc.AnalyzeRedactionsExecutor,
	applyRedactionUC proc.ApplyRedactionsExecutor,
) *ProcessingHandler {
	h.refactored = proc.NewProcessingHandlerRefactored(processUC, batchUC, updateMetadataUC, analyzeRedactionUC, applyRedactionUC)
	return h
}

// Expose refactored endpoints (unused by router today; available for migration)
func (h *ProcessingHandler) ProcessDocumentRefactored(c *fiber.Ctx) error {
	if h.refactored == nil {
		return fiber.ErrNotImplemented
	}
	return h.refactored.ProcessDocument(c)
}

func (h *ProcessingHandler) BatchProcessDocumentsRefactored(c *fiber.Ctx) error {
	if h.refactored == nil {
		return fiber.ErrNotImplemented
	}
	return h.refactored.BatchProcessDocuments(c)
}

func (h *ProcessingHandler) UpdateMetadataRefactored(c *fiber.Ctx) error {
	if h.refactored == nil {
		return fiber.ErrNotImplemented
	}
	return h.refactored.UpdateMetadata(c)
}

func (h *ProcessingHandler) AnalyzeRedactionsRefactored(c *fiber.Ctx) error {
	if h.refactored == nil {
		return fiber.ErrNotImplemented
	}
	return h.refactored.AnalyzeRedactions(c)
}

func (h *ProcessingHandler) ApplyRedactionsRefactored(c *fiber.Ctx) error {
	if h.refactored == nil {
		return fiber.ErrNotImplemented
	}
	return h.refactored.ApplyRedactions(c)
}
