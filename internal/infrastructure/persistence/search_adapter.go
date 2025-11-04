package persistence

import (
	"context"
	"time"

	"motion-index-fiber/internal/application/ports"
	"motion-index-fiber/internal/domain/document"
	infraerrors "motion-index-fiber/internal/infrastructure/errors"
	"motion-index-fiber/pkg/models"
	"motion-index-fiber/pkg/search"
)

// SearchAdapter adapts the existing pkg/search to the application port.
type SearchAdapter struct {
	service search.Service
}

// NewSearchAdapter creates a new search adapter.
func NewSearchAdapter(service search.Service) *SearchAdapter {
	return &SearchAdapter{
		service: service,
	}
}

// Index implements the SearchService port.
func (a *SearchAdapter) Index(ctx context.Context, doc *document.Document) error {
	// Convert domain document to pkg model
	pkgDoc := convertDomainDocumentToPkgModel(doc)

	// Index the document
	_, err := a.service.IndexDocument(ctx, pkgDoc)
	if err != nil {
		return infraerrors.TranslateSearchError(err)
	}

	return nil
}

// Update implements the SearchService port.
func (a *SearchAdapter) Update(ctx context.Context, id string, fields map[string]interface{}) error {
	err := a.service.UpdateDocumentMetadata(ctx, id, fields)
	if err != nil {
		return infraerrors.TranslateSearchError(err)
	}

	return nil
}

// Delete implements the SearchService port.
func (a *SearchAdapter) Delete(ctx context.Context, id string) error {
	err := a.service.DeleteDocument(ctx, id)
	if err != nil {
		return infraerrors.TranslateSearchError(err)
	}

	return nil
}

// Search implements the SearchService port.
func (a *SearchAdapter) Search(ctx context.Context, query ports.SearchQuery) (*ports.SearchResults, error) {
	// Convert application query to pkg search request
	req := &models.SearchRequest{
		Query:       query.Query,
		DocType:     query.DocumentType,
		LegalTags:   query.LegalTags,
		From:        (query.Page - 1) * query.PageSize,
		Size:        query.PageSize,
		FuzzySearch: true, // Enable fuzzy matching
	}

	// Add date range if specified
	if query.FromDate != nil || query.ToDate != nil {
		req.DateRange = &models.DateRange{
			From: query.FromDate,
			To:   query.ToDate,
		}
	}

	// Add sorting if specified
	if query.SortBy != "" {
		req.SortBy = query.SortBy
		req.SortOrder = query.SortOrder
	}

	// Execute search
	result, err := a.service.SearchDocuments(ctx, req)
	if err != nil {
		return nil, infraerrors.TranslateSearchError(err)
	}

	// Convert pkg result to application result
	return convertPkgResultToApplicationResult(result, query.Page, query.PageSize), nil
}

// convertDomainDocumentToPkgModel converts a domain document to pkg model.
func convertDomainDocumentToPkgModel(doc *document.Document) *models.Document {
	pkgDoc := &models.Document{
		ID:          doc.ID().String(),
		FileName:    doc.FileName().String(),
		ContentType: doc.ContentType().String(),
		FilePath:    doc.FilePath().String(),
		Hash:        doc.Hash().String(),
		Text:        doc.Text().String(),
		Size:        doc.Size().Bytes(),
		CreatedAt:   doc.CreatedAt(),
		UpdatedAt:   doc.UpdatedAt(),
	}

	// Add classification metadata if available
	if classification := doc.Classification(); classification != nil {
		docType := models.ParseDocumentType(classification.DocumentType().String())
		pkgDoc.Metadata = &models.DocumentMetadata{
			DocumentType: docType,
			LegalTags:    classification.LegalTags(),
			Confidence:   classification.Confidence().Value(),
		}
	}

	return pkgDoc
}

// convertPkgResultToApplicationResult converts pkg search result to application result.
func convertPkgResultToApplicationResult(result *models.SearchResult, page, pageSize int) *ports.SearchResults {
	hits := make([]ports.SearchHit, len(result.Documents))

	for i, doc := range result.Documents {
		hit := ports.SearchHit{
			ID:        doc.ID,
			Snippet:   extractSnippet(doc.Highlights),
			CreatedAt: extractCreatedAt(doc.Document),
		}

		// Extract fields from document source
		if fileName, ok := doc.Document["file_name"].(string); ok {
			hit.FileName = fileName
		}
		if metadata, ok := doc.Document["metadata"].(map[string]interface{}); ok {
			if docType, ok := metadata["document_type"].(string); ok {
				hit.DocumentType = docType
			}
			if category, ok := metadata["legal_category"].(string); ok {
				hit.Category = category
			}
			if confidence, ok := metadata["confidence"].(float64); ok {
				hit.Confidence = confidence
			}
		}

		hits[i] = hit
	}

	return &ports.SearchResults{
		Total:    int(result.TotalHits),
		Page:     page,
		PageSize: pageSize,
		Hits:     hits,
	}
}

// extractSnippet extracts the first highlight snippet.
func extractSnippet(highlights map[string][]string) string {
	for _, snippets := range highlights {
		if len(snippets) > 0 {
			return snippets[0]
		}
	}
	return ""
}

// extractCreatedAt extracts created_at timestamp from document source.
func extractCreatedAt(docSource interface{}) time.Time {
	docMap, ok := docSource.(map[string]interface{})
	if !ok {
		return time.Now()
	}

	createdAtStr, ok := docMap["created_at"].(string)
	if !ok {
		return time.Now()
	}

	createdAt, err := time.Parse(time.RFC3339, createdAtStr)
	if err != nil {
		return time.Now()
	}

	return createdAt
}

// Ensure SearchAdapter implements the port
var _ ports.SearchService = (*SearchAdapter)(nil)
