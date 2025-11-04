package search

import (
	"context"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"

	"motion-index-fiber/internal/config"
	internalModels "motion-index-fiber/internal/models"
	"motion-index-fiber/pkg/models"
	"motion-index-fiber/pkg/search"
)

type MetadataFieldsHandler struct {
	cfg           *config.Config
	searchService search.Service
}

func NewMetadataFieldsHandler(cfg *config.Config, searchService search.Service) *MetadataFieldsHandler {
	return &MetadataFieldsHandler{
		cfg:           cfg,
		searchService: searchService,
	}
}

// GetMetadataFields handles GET /metadata-fields (without parameters)
func (h *MetadataFieldsHandler) GetMetadataFields(c *fiber.Ctx) error {
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
func (h *MetadataFieldsHandler) GetMetadataFieldValues(c *fiber.Ctx) error {
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
func (h *MetadataFieldsHandler) PostMetadataFieldValues(c *fiber.Ctx) error {
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
		"field":           req.Field,
		"values":          values,
		"filters_applied": req.Filters,
		"total_returned":  len(values),
	}, "Metadata field values retrieved successfully"))
}
