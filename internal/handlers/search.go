package handlers

import (
	"context"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"

	internalModels "motion-index-fiber/internal/models"
	"motion-index-fiber/pkg/models"
	"motion-index-fiber/pkg/search"
)

// SearchHandler handles search-related HTTP requests
type SearchHandler struct {
	searchService search.Service
}

// NewSearchHandler creates a new search handler
func NewSearchHandler(searchService search.Service) *SearchHandler {
	return &SearchHandler{
		searchService: searchService,
	}
}

// SearchDocuments handles POST /search
func (h *SearchHandler) SearchDocuments(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.Context(), 30*time.Second)
	defer cancel()

	var req models.SearchRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid request body: "+err.Error())
	}

	// Parse query parameters if body is empty
	if req.Query == "" {
		req.Query = c.Query("q", "")
	}
	if req.Size == 0 {
		if sizeStr := c.Query("size"); sizeStr != "" {
			if size, err := strconv.Atoi(sizeStr); err == nil {
				req.Size = size
			}
		}
	}
	if req.From == 0 {
		if fromStr := c.Query("from"); fromStr != "" {
			if from, err := strconv.Atoi(fromStr); err == nil {
				req.From = from
			}
		}
	}

	// Validate request
	if err := validateSearchRequest(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	// Execute search
	result, err := h.searchService.SearchDocuments(ctx, &req)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Search failed: "+err.Error())
	}

	return c.JSON(fiber.Map{
		"status":  "success",
		"data":    result,
		"message": "Search completed successfully",
	})
}

// GetLegalTags handles GET /legal-tags
func (h *SearchHandler) GetLegalTags(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.Context(), 10*time.Second)
	defer cancel()

	tags, err := h.searchService.GetLegalTags(ctx)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to retrieve legal tags: "+err.Error())
	}

	return c.JSON(fiber.Map{
		"status": "success",
		"data":   tags,
	})
}

// GetDocumentTypes handles GET /document-types
func (h *SearchHandler) GetDocumentTypes(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.Context(), 10*time.Second)
	defer cancel()

	types, err := h.searchService.GetDocumentTypes(ctx)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to retrieve document types: "+err.Error())
	}

	return c.JSON(fiber.Map{
		"status": "success",
		"data":   types,
	})
}

// GetDocumentStats handles GET /document-stats
func (h *SearchHandler) GetDocumentStats(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.Context(), 15*time.Second)
	defer cancel()

	stats, err := h.searchService.GetDocumentStats(ctx)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to retrieve document stats: "+err.Error())
	}

	return c.JSON(fiber.Map{
		"status": "success",
		"data":   stats,
	})
}

// GetFieldOptions handles GET /field-options
func (h *SearchHandler) GetFieldOptions(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.Context(), 15*time.Second)
	defer cancel()

	options, err := h.searchService.GetAllFieldOptions(ctx)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to retrieve field options: "+err.Error())
	}

	return c.JSON(fiber.Map{
		"status": "success",
		"data":   options,
	})
}

// GetMetadataFields handles GET /metadata-fields (without parameters)
func (h *SearchHandler) GetMetadataFields(c *fiber.Ctx) error {
	// Return the list of available metadata fields with their types
	// This is a static list for now, but could be made dynamic based on search service
	fields := []map[string]interface{}{
		{"id": "case_name", "name": "Case Name", "type": "string"},
		{"id": "case_number", "name": "Case Number", "type": "string"},
		{"id": "author", "name": "Author", "type": "string"},
		{"id": "judge", "name": "Judge", "type": "string"},
		{"id": "court", "name": "Court", "type": "string"},
		{"id": "legal_tags", "name": "Legal Tags", "type": "array"},
		{"id": "doc_type", "name": "Document Type", "type": "string"},
		{"id": "category", "name": "Category", "type": "string"},
		{"id": "status", "name": "Status", "type": "string"},
		{"id": "created_at", "name": "Created Date", "type": "date"},
	}

	response := map[string]interface{}{
		"fields": fields,
	}

	return c.JSON(internalModels.NewSuccessResponse(response, "Metadata fields retrieved successfully"))
}

// GetMetadataFieldValues handles GET /metadata-fields/{field}
func (h *SearchHandler) GetMetadataFieldValues(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.Context(), 10*time.Second)
	defer cancel()

	field := c.Params("field")
	if field == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Field parameter is required")
	}

	// Parse query parameters
	prefix := c.Query("prefix", "")
	size := 50
	if sizeStr := c.Query("size"); sizeStr != "" {
		if s, err := strconv.Atoi(sizeStr); err == nil && s > 0 {
			size = s
		}
	}

	values, err := h.searchService.GetMetadataFieldValues(ctx, field, prefix, size)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to retrieve field values: "+err.Error())
	}

	return c.JSON(fiber.Map{
		"status": "success",
		"data":   values,
	})
}

// PostMetadataFieldValues handles POST /metadata-field-values with custom filters
func (h *SearchHandler) PostMetadataFieldValues(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.Context(), 30*time.Second)
	defer cancel()

	var req models.MetadataFieldValuesRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(internalModels.NewErrorResponse(
			"validation_error",
			"Invalid request body: "+err.Error(),
			nil,
		))
	}

	// Validate required field
	if req.Field == "" {
		return c.Status(fiber.StatusBadRequest).JSON(internalModels.NewErrorResponse(
			"validation_error",
			"Field parameter is required",
			nil,
		))
	}

	// Set default size if not provided
	if req.Size <= 0 {
		req.Size = 50
	}
	if req.Size > 1000 {
		req.Size = 1000
	}

	values, err := h.searchService.GetMetadataFieldValuesWithFilters(ctx, &req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(internalModels.NewErrorResponse(
			"search_error",
			"Failed to retrieve field values: "+err.Error(),
			nil,
		))
	}

	return c.JSON(internalModels.NewSuccessResponse(map[string]interface{}{
		"field":  req.Field,
		"values": values,
		"filters_applied": req.Filters,
		"total_returned": len(values),
	}, "Metadata field values retrieved successfully"))
}

// GetDocument handles GET /documents/{id}
func (h *SearchHandler) GetDocument(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.Context(), 10*time.Second)
	defer cancel()

	docID := c.Params("id")
	if docID == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Document ID is required")
	}

	document, err := h.searchService.GetDocument(ctx, docID)
	if err != nil {
		if err.Error() == "document not found" {
			return fiber.NewError(fiber.StatusNotFound, "Document not found")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to retrieve document: "+err.Error())
	}

	return c.JSON(fiber.Map{
		"status": "success",
		"data":   document,
	})
}

// DeleteDocument handles DELETE /documents/{id} (protected endpoint)
func (h *SearchHandler) DeleteDocument(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.Context(), 10*time.Second)
	defer cancel()

	docID := c.Params("id")
	if docID == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Document ID is required")
	}

	err := h.searchService.DeleteDocument(ctx, docID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to delete document: "+err.Error())
	}

	return c.JSON(fiber.Map{
		"status":  "success",
		"message": "Document deleted successfully",
	})
}

// GetDocumentRedactions gets redaction analysis for a specific document
func (h *SearchHandler) GetDocumentRedactions(c *fiber.Ctx) error {
	docID := c.Params("id")
	if docID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(internalModels.NewErrorResponse(
			"validation_error",
			"Document ID is required",
			nil,
		))
	}

	// For now, return a placeholder response indicating no redaction analysis found
	// TODO: Implement actual redaction analysis retrieval from storage or search service
	// This would typically involve:
	// 1. Querying the document from search service to get metadata
	// 2. Checking if redaction analysis exists for this document
	// 3. Returning the redaction analysis data

	return c.Status(fiber.StatusNotFound).JSON(internalModels.NewErrorResponse(
		"not_found",
		"No redaction analysis found for this document",
		map[string]interface{}{
			"document_id": docID,
		},
	))
}

// validateSearchRequest validates a search request
func validateSearchRequest(req *models.SearchRequest) error {
	if req.Size > models.MaxSearchSize {
		req.Size = models.MaxSearchSize
	}
	if req.Size <= 0 {
		req.Size = models.DefaultSearchSize
	}
	if req.From < 0 {
		req.From = 0
	}

	// Validate sort order
	if req.SortOrder != "" && req.SortOrder != "asc" && req.SortOrder != "desc" {
		req.SortOrder = "desc"
	}

	return nil
}
