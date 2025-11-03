package handlers

import (
	"context"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"motion-index-fiber/internal/application/dto"
	"motion-index-fiber/internal/infrastructure/http/presenter"
	"motion-index-fiber/pkg/models"
	"motion-index-fiber/pkg/search"
)

// SearchUseCase defines the interface for document search operations
type SearchUseCase interface {
	Execute(ctx context.Context, req *dto.SearchDocumentsRequest) (*dto.SearchResultsResponse, error)
}

// SearchHandler handles search-related HTTP requests
type SearchHandler struct {
	searchUseCase SearchUseCase
	searchService search.Service // For aggregation endpoints
}

// NewSearchHandler creates a new search handler
func NewSearchHandler(searchUseCase SearchUseCase, searchService search.Service) *SearchHandler {
	return &SearchHandler{
		searchUseCase: searchUseCase,
		searchService: searchService,
	}
}

// SearchDocuments handles document search requests
func (h *SearchHandler) SearchDocuments(c *fiber.Ctx) error {
	// Parse request from body or query params
	req := &dto.SearchDocumentsRequest{
		Query:        c.Query("q", c.Query("query", "")),
		DocumentType: c.Query("document_type", ""),
		Category:     c.Query("category", ""),
		Page:         c.QueryInt("page", 1),
		PageSize:     c.QueryInt("page_size", 20),
		SortBy:       c.Query("sort_by", "created_at"),
		SortOrder:    c.Query("sort_order", "desc"),
	}

	// Parse MinConfidence
	if minConfStr := c.Query("min_confidence"); minConfStr != "" {
		if conf, err := strconv.ParseFloat(minConfStr, 64); err == nil {
			req.MinConfidence = conf
		}
	}

	// Parse legal tags
	if tags := c.Query("legal_tags"); tags != "" {
		// Simple comma-separated tags parsing
		req.LegalTags = []string{tags}
	}

	// Execute search use case
	results, err := h.searchUseCase.Execute(c.Context(), req)
	if err != nil {
		return err // Middleware will translate
	}

	// Present results
	return presenter.Success(c, presenter.PresentSearchResults(results))
}

// GetLegalTags handles GET /legal-tags
func (h *SearchHandler) GetLegalTags(c *fiber.Ctx) error {
	tags, err := h.searchService.GetLegalTags(c.Context())
	if err != nil {
		return err
	}

	return presenter.Success(c, map[string]interface{}{
		"tags": tags,
	})
}

// GetDocumentTypes handles GET /document-types
func (h *SearchHandler) GetDocumentTypes(c *fiber.Ctx) error {
	types, err := h.searchService.GetDocumentTypes(c.Context())
	if err != nil {
		return err
	}

	return presenter.Success(c, map[string]interface{}{
		"types": types,
	})
}

// GetDocumentStats handles GET /document-stats
func (h *SearchHandler) GetDocumentStats(c *fiber.Ctx) error {
	stats, err := h.searchService.GetDocumentStats(c.Context())
	if err != nil {
		return err
	}

	return presenter.Success(c, stats)
}

// GetFieldOptions handles GET /field-options
func (h *SearchHandler) GetFieldOptions(c *fiber.Ctx) error {
	options, err := h.searchService.GetAllFieldOptions(c.Context())
	if err != nil {
		return err
	}

	return presenter.Success(c, options)
}

// GetMetadataFieldValues handles GET /metadata-fields/{field}
func (h *SearchHandler) GetMetadataFieldValues(c *fiber.Ctx) error {
	field := c.Params("field")
	if field == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Field parameter is required")
	}

	prefix := c.Query("prefix", "")
	size := c.QueryInt("size", 50)

	values, err := h.searchService.GetMetadataFieldValues(c.Context(), field, prefix, size)
	if err != nil {
		return err
	}

	return presenter.Success(c, map[string]interface{}{
		"field":  field,
		"values": values,
	})
}

// PostMetadataFieldValues handles POST /metadata-field-values with custom filters
func (h *SearchHandler) PostMetadataFieldValues(c *fiber.Ctx) error {
	var req models.MetadataFieldValuesRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
	}

	if req.Field == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Field parameter is required")
	}

	// Set defaults
	if req.Size <= 0 {
		req.Size = 50
	}
	if req.Size > 1000 {
		req.Size = 1000
	}

	values, err := h.searchService.GetMetadataFieldValuesWithFilters(c.Context(), &req)
	if err != nil {
		return err
	}

	return presenter.Success(c, map[string]interface{}{
		"field":           req.Field,
		"values":          values,
		"filters_applied": req.Filters,
		"total_returned":  len(values),
	})
}

// GetDocument handles GET /documents/{id}
func (h *SearchHandler) GetDocument(c *fiber.Ctx) error {
	docID := c.Params("id")
	if docID == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Document ID is required")
	}

	document, err := h.searchService.GetDocument(c.Context(), docID)
	if err != nil {
		return err // Middleware will translate to 404 if not found
	}

	return presenter.Success(c, document)
}

// DeleteDocument handles DELETE /documents/{id}
func (h *SearchHandler) DeleteDocument(c *fiber.Ctx) error {
	docID := c.Params("id")
	if docID == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Document ID is required")
	}

	err := h.searchService.DeleteDocument(c.Context(), docID)
	if err != nil {
		return err
	}

	return presenter.Success(c, map[string]interface{}{
		"message": "Document deleted successfully",
	})
}
