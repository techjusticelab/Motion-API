package handlers

import (
	"github.com/gofiber/fiber/v2"
	"motion-index-fiber/internal/config"
	"motion-index-fiber/internal/handlers/search"
	pkgsearch "motion-index-fiber/pkg/search"
)

// SearchHandler handles search-related HTTP requests
type SearchHandler struct {
	cfg                       *config.Config
	searchService             pkgsearch.Service
	searchDocumentsHandler    *search.SearchDocumentsHandler
	legalTagsHandler          *search.LegalTagsHandler
	documentTypesHandler      *search.DocumentTypesHandler
	documentStatsHandler      *search.DocumentStatsHandler
	fieldOptionsHandler       *search.FieldOptionsHandler
	metadataFieldsHandler     *search.MetadataFieldsHandler
	documentHandler           *search.DocumentHandler
	documentRedactionsHandler *search.DocumentRedactionsHandler
	refactoredHandler         *search.SearchHandlerRefactored
}

// NewSearchHandler creates a new search handler
func NewSearchHandler(cfg *config.Config, searchService pkgsearch.Service) *SearchHandler {
	return &SearchHandler{
		cfg:                       cfg,
		searchService:             searchService,
		searchDocumentsHandler:    search.NewSearchDocumentsHandler(cfg, searchService),
		legalTagsHandler:          search.NewLegalTagsHandler(cfg, searchService),
		documentTypesHandler:      search.NewDocumentTypesHandler(cfg, searchService),
		documentStatsHandler:      search.NewDocumentStatsHandler(cfg, searchService),
		fieldOptionsHandler:       search.NewFieldOptionsHandler(cfg, searchService),
		metadataFieldsHandler:     search.NewMetadataFieldsHandler(cfg, searchService),
		documentHandler:           search.NewDocumentHandler(cfg, searchService),
		documentRedactionsHandler: search.NewDocumentRedactionsHandler(cfg, searchService),
		// refactoredHandler will be initialized when DDD use cases are available
		refactoredHandler: nil,
	}
}

// SearchDocuments delegates to the search documents handler
func (h *SearchHandler) SearchDocuments(c *fiber.Ctx) error {
	return h.searchDocumentsHandler.SearchDocuments(c)
}

// GetLegalTags delegates to the legal tags handler
func (h *SearchHandler) GetLegalTags(c *fiber.Ctx) error {
	return h.legalTagsHandler.GetLegalTags(c)
}

// GetDocumentTypes delegates to the document types handler
func (h *SearchHandler) GetDocumentTypes(c *fiber.Ctx) error {
	return h.documentTypesHandler.GetDocumentTypes(c)
}

// GetDocumentStats delegates to the document stats handler
func (h *SearchHandler) GetDocumentStats(c *fiber.Ctx) error {
	return h.documentStatsHandler.GetDocumentStats(c)
}

// GetFieldOptions delegates to the field options handler
func (h *SearchHandler) GetFieldOptions(c *fiber.Ctx) error {
	return h.fieldOptionsHandler.GetFieldOptions(c)
}

// GetMetadataFields delegates to the metadata fields handler
func (h *SearchHandler) GetMetadataFields(c *fiber.Ctx) error {
	return h.metadataFieldsHandler.GetMetadataFields(c)
}

// GetMetadataFieldValues delegates to the metadata fields handler
func (h *SearchHandler) GetMetadataFieldValues(c *fiber.Ctx) error {
	return h.metadataFieldsHandler.GetMetadataFieldValues(c)
}

// PostMetadataFieldValues delegates to the metadata fields handler
func (h *SearchHandler) PostMetadataFieldValues(c *fiber.Ctx) error {
	return h.metadataFieldsHandler.PostMetadataFieldValues(c)
}

// GetDocument delegates to the document handler
func (h *SearchHandler) GetDocument(c *fiber.Ctx) error {
	return h.documentHandler.GetDocument(c)
}

// DeleteDocument delegates to the document handler
func (h *SearchHandler) DeleteDocument(c *fiber.Ctx) error {
	return h.documentHandler.DeleteDocument(c)
}

// GetDocumentRedactions delegates to the document redactions handler
func (h *SearchHandler) GetDocumentRedactions(c *fiber.Ctx) error {
	return h.documentRedactionsHandler.GetDocumentRedactions(c)
}
