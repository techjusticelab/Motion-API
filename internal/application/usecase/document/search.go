package document

import (
	"context"

	"motion-index-fiber/internal/application/dto"
	"motion-index-fiber/internal/application/ports"
)

// SearchDocumentsUseCase orchestrates search requests.
type SearchDocumentsUseCase struct {
	search ports.SearchService
}

// NewSearchDocumentsUseCase wires dependencies for document search.
func NewSearchDocumentsUseCase(search ports.SearchService) *SearchDocumentsUseCase {
	return &SearchDocumentsUseCase{search: search}
}

// Execute validates the request, delegates to the search service, and maps results.
func (uc *SearchDocumentsUseCase) Execute(ctx context.Context, req *dto.SearchDocumentsRequest) (*dto.SearchResultsResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	if uc.search == nil {
		return &dto.SearchResultsResponse{Documents: []dto.DocumentSummary{}, Total: 0, Page: req.Page, PageSize: req.PageSize}, nil
	}

	query := ports.SearchQuery{
		Query:         req.Query,
		DocumentType:  req.DocumentType,
		Category:      req.Category,
		LegalTags:     req.LegalTags,
		FromDate:      req.FromDate,
		ToDate:        req.ToDate,
		MinConfidence: req.MinConfidence,
		Page:          req.Page,
		PageSize:      req.PageSize,
		SortBy:        req.SortBy,
		SortOrder:     req.SortOrder,
	}

	results, err := uc.search.Search(ctx, query)
	if err != nil {
		return nil, err
	}

	resp := &dto.SearchResultsResponse{
		Total:    results.Total,
		Page:     results.Page,
		PageSize: results.PageSize,
	}

	resp.TotalPages = computeTotalPages(results.Total, results.PageSize)
	resp.Documents = make([]dto.DocumentSummary, len(results.Hits))
	for i, hit := range results.Hits {
		resp.Documents[i] = dto.DocumentSummary{
			ID:           hit.ID,
			FileName:     hit.FileName,
			DocumentType: hit.DocumentType,
			Category:     hit.Category,
			Snippet:      hit.Snippet,
			Confidence:   hit.Confidence,
			CreatedAt:    hit.CreatedAt,
		}
	}

	return resp, nil
}

func computeTotalPages(total, pageSize int) int {
	if pageSize <= 0 {
		return 0
	}
	pages := total / pageSize
	if total%pageSize != 0 {
		pages++
	}
	return pages
}
