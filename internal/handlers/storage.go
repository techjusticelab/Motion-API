package handlers

import (
	"github.com/gofiber/fiber/v2"
	"motion-index-fiber/internal/config"
	"motion-index-fiber/internal/handlers/storage"
	pkgextractor "motion-index-fiber/pkg/processing/extractor"
	pkgsearch "motion-index-fiber/pkg/search"
	pkgstorage "motion-index-fiber/pkg/storage"
)

type StorageHandler struct {
	cfg               *config.Config
	storage           pkgstorage.Service
	search            pkgsearch.Service
	extractor         pkgextractor.Service
	listHandler       *storage.ListHandler
	countHandler      *storage.CountHandler
	serveHandler      *storage.ServeHandler
	searchHandler     *storage.SearchHandler
	uploadHandler     *storage.UploadHandler
	refactoredHandler *storage.RefactoredHandler
}

func NewStorageHandler(cfg *config.Config, storageService pkgstorage.Service, searchService pkgsearch.Service, extractorService pkgextractor.Service) *StorageHandler {
	return &StorageHandler{
		cfg:               cfg,
		storage:           storageService,
		search:            searchService,
		extractor:         extractorService,
		listHandler:       storage.NewListHandler(cfg, storageService),
		countHandler:      storage.NewCountHandler(cfg, storageService),
		serveHandler:      storage.NewServeHandler(cfg, storageService),
		searchHandler:     storage.NewSearchHandler(cfg, storageService),
		uploadHandler:     storage.NewUploadHandler(cfg, storageService, extractorService, searchService),
		refactoredHandler: storage.NewRefactoredHandler(storageService),
	}
}

// ListDocuments delegates to the list handler
func (h *StorageHandler) ListDocuments(c *fiber.Ctx) error {
	return h.listHandler.ListDocuments(c)
}

// GetDocumentsCount delegates to the count handler
func (h *StorageHandler) GetDocumentsCount(c *fiber.Ctx) error {
	return h.countHandler.GetDocumentsCount(c)
}

// ServeDocument delegates to the serve handler
func (h *StorageHandler) ServeDocument(c *fiber.Ctx) error {
	return h.serveHandler.ServeDocument(c)
}

// FindDocumentsByName delegates to the search handler
func (h *StorageHandler) FindDocumentsByName(c *fiber.Ctx) error {
	return h.searchHandler.FindDocumentsByName(c)
}

// UploadDocumentToS3 delegates to the upload handler
func (h *StorageHandler) UploadDocumentToS3(c *fiber.Ctx) error {
	return h.uploadHandler.UploadDocumentToS3(c)
}

func (h *StorageHandler) UploadDocumentAndIndex(c *fiber.Ctx) error {
	return h.uploadHandler.UploadDocumentAndIndex(c)
}

// SetRefactoredHandler allows setting the DDD-based handler when use cases are available
func (h *StorageHandler) SetRefactoredHandler(refactoredHandler *storage.RefactoredHandler) {
	h.refactoredHandler = refactoredHandler
}

// ListDocumentsRefactored provides an alternative endpoint using DDD architecture
// This method can be used when the refactored handler is initialized
func (h *StorageHandler) ListDocumentsRefactored(c *fiber.Ctx) error {
	if h.refactoredHandler == nil {
		// Fall back to the standard handler if refactored version is not available
		return h.ListDocuments(c)
	}
	return h.refactoredHandler.ListDocuments(c)
}

// GetDocumentsCountRefactored provides an alternative endpoint using DDD architecture
// This method can be used when the refactored handler is initialized
func (h *StorageHandler) GetDocumentsCountRefactored(c *fiber.Ctx) error {
	if h.refactoredHandler == nil {
		// Fall back to the standard handler if refactored version is not available
		return h.GetDocumentsCount(c)
	}
	return h.refactoredHandler.GetDocumentsCount(c)
}

// FindDocumentsByNameRefactored provides an alternative endpoint using DDD architecture
// This method can be used when the refactored handler is initialized
func (h *StorageHandler) FindDocumentsByNameRefactored(c *fiber.Ctx) error {
	if h.refactoredHandler == nil {
		// Fall back to the standard handler if refactored version is not available
		return h.FindDocumentsByName(c)
	}
	return h.refactoredHandler.FindDocumentsByName(c)
}

// ServeDocumentRefactored provides an alternative endpoint using DDD architecture
// This method can be used when the refactored handler is initialized
func (h *StorageHandler) ServeDocumentRefactored(c *fiber.Ctx) error {
	if h.refactoredHandler == nil {
		// Fall back to the standard handler if refactored version is not available
		return h.ServeDocument(c)
	}
	return h.refactoredHandler.ServeDocument(c)
}
