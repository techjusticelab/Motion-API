package search

import (
	"context"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"motion-index-fiber/internal/application/dto"
	"motion-index-fiber/internal/infrastructure/http/presenter"
	"motion-index-fiber/pkg/search"
)

// SearchDocumentsExecutor defines the interface for searching documents.
type SearchDocumentsExecutor interface {
	Execute(ctx context.Context, req *dto.SearchDocumentsRequest) (*dto.SearchResultsResponse, error)
}

// SearchHandlerRefactored handles search operations using DDD use cases.
type SearchHandlerRefactored struct {
	searchUC      SearchDocumentsExecutor
	searchService search.Service // Direct service for simple pass-through operations
}

// NewSearchHandlerRefactored creates a refactored search handler with use case injection.
func NewSearchHandlerRefactored(searchUC SearchDocumentsExecutor, searchService search.Service) *SearchHandlerRefactored {
	return &SearchHandlerRefactored{
		searchUC:      searchUC,
		searchService: searchService,
	}
}

// SearchDocuments handles POST /search - Search for documents.
func (h *SearchHandlerRefactored) SearchDocuments(c *fiber.Ctx) error {
	var body struct {
		Query         string   `json:"query"`
		DocumentType  string   `json:"document_type"`
		Category      string   `json:"category"`
		LegalTags     []string `json:"legal_tags"`
		Page          int      `json:"page"`
		PageSize      int      `json:"page_size"`
		SortBy        string   `json:"sort_by"`
		SortOrder     string   `json:"sort_order"`
		MinConfidence float64  `json:"min_confidence"`
	}

	if err := c.BodyParser(&body); err != nil {
		// Try query params if body parsing fails
		body.Query = c.Query("q", "")
		if sizeStr := c.Query("size"); sizeStr != "" {
			body.PageSize, _ = strconv.Atoi(sizeStr)
		}
		if fromStr := c.Query("from"); fromStr != "" {
			from, _ := strconv.Atoi(fromStr)
			body.Page = from / body.PageSize
		}
	}

	req := &dto.SearchDocumentsRequest{
		Query:         body.Query,
		DocumentType:  body.DocumentType,
		Category:      body.Category,
		LegalTags:     body.LegalTags,
		Page:          body.Page,
		PageSize:      body.PageSize,
		SortBy:        body.SortBy,
		SortOrder:     body.SortOrder,
		MinConfidence: body.MinConfidence,
	}

	result, err := h.searchUC.Execute(c.Context(), req)
	if err != nil {
		return presenter.InternalError(c, "Search failed")
	}

	return presenter.Success(c, presenter.PresentSearchResults(result))
}

// GetDocument handles GET /documents/{id} - Get a specific document.
func (h *SearchHandlerRefactored) GetDocument(c *fiber.Ctx) error {
	docID := c.Params("id")
	if docID == "" {
		return presenter.BadRequest(c, "Document ID is required", nil)
	}

	document, err := h.searchService.GetDocument(c.Context(), docID)
	if err != nil {
		if err.Error() == "document not found" {
			return presenter.NotFound(c, "Document not found")
		}
		return presenter.InternalError(c, "Failed to retrieve document")
	}

	return presenter.Success(c, document)
}

// DeleteDocument handles DELETE /documents/{id} - Delete a document.
func (h *SearchHandlerRefactored) DeleteDocument(c *fiber.Ctx) error {
	docID := c.Params("id")
	if docID == "" {
		return presenter.BadRequest(c, "Document ID is required", nil)
	}

	err := h.searchService.DeleteDocument(c.Context(), docID)
	if err != nil {
		return presenter.InternalError(c, "Failed to delete document")
	}

	return presenter.Success(c, fiber.Map{"message": "Document deleted successfully"})
}
