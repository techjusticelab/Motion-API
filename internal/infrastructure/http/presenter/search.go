package presenter

import (
	"time"

	"motion-index-fiber/internal/application/dto"
)

// SearchResultsResponse represents search results in the HTTP response
type SearchResultsResponse struct {
	Documents  []DocumentSummaryResponse `json:"documents"`
	Total      int                       `json:"total"`
	Page       int                       `json:"page"`
	PageSize   int                       `json:"page_size"`
	TotalPages int                       `json:"total_pages"`
}

// DocumentSummaryResponse represents a document summary in search results
type DocumentSummaryResponse struct {
	ID           string  `json:"id"`
	FileName     string  `json:"file_name"`
	DocumentType string  `json:"document_type,omitempty"`
	Category     string  `json:"category,omitempty"`
	Snippet      string  `json:"snippet,omitempty"`
	Confidence   float64 `json:"confidence,omitempty"`
	CreatedAt    string  `json:"created_at"`
}

// PresentSearchResults converts SearchResultsResponse DTO to presentation format
func PresentSearchResults(dto *dto.SearchResultsResponse) *SearchResultsResponse {
	if dto == nil {
		return nil
	}

	documents := make([]DocumentSummaryResponse, 0, len(dto.Documents))
	for _, doc := range dto.Documents {
		documents = append(documents, DocumentSummaryResponse{
			ID:           doc.ID,
			FileName:     doc.FileName,
			DocumentType: doc.DocumentType,
			Category:     doc.Category,
			Snippet:      doc.Snippet,
			Confidence:   doc.Confidence,
			CreatedAt:    doc.CreatedAt.Format(time.RFC3339),
		})
	}

	return &SearchResultsResponse{
		Documents:  documents,
		Total:      dto.Total,
		Page:       dto.Page,
		PageSize:   dto.PageSize,
		TotalPages: dto.TotalPages,
	}
}

// PresentDocumentSummary converts a single DocumentSummary to presentation format
func PresentDocumentSummary(doc dto.DocumentSummary) DocumentSummaryResponse {
	return DocumentSummaryResponse{
		ID:           doc.ID,
		FileName:     doc.FileName,
		DocumentType: doc.DocumentType,
		Category:     doc.Category,
		Snippet:      doc.Snippet,
		Confidence:   doc.Confidence,
		CreatedAt:    doc.CreatedAt.Format(time.RFC3339),
	}
}

// PresentDocumentSummaries converts a slice of DocumentSummary to presentation format
func PresentDocumentSummaries(dtos []dto.DocumentSummary) []DocumentSummaryResponse {
	if dtos == nil {
		return nil
	}

	summaries := make([]DocumentSummaryResponse, 0, len(dtos))
	for _, doc := range dtos {
		summaries = append(summaries, PresentDocumentSummary(doc))
	}

	return summaries
}
