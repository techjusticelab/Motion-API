package presenter

import (
	"motion-index-fiber/internal/application/dto"
)

// RedactionAnalysisResponse represents redaction analysis in HTTP response
type RedactionAnalysisResponse struct {
	DocumentID string                  `json:"document_id,omitempty"`
	FileName   string                  `json:"file_name,omitempty"`
	Redactions []RedactionItemResponse `json:"redactions"`
	TotalCount int                     `json:"total_count"`
}

// RedactionItemResponse represents a single redaction item
type RedactionItemResponse struct {
	ID        string    `json:"id"`
	Page      int       `json:"page"`
	Text      string    `json:"text,omitempty"`
	BBox      []float64 `json:"bbox,omitempty"`
	Type      string    `json:"type"`
	Citation  string    `json:"citation,omitempty"`
	Reason    string    `json:"reason,omitempty"`
	LegalCode string    `json:"legal_code,omitempty"`
	Applied   bool      `json:"applied"`
}

// RedactionResultResponse represents applied redactions in HTTP response
type RedactionResultResponse struct {
	DocumentID      string                  `json:"document_id,omitempty"`
	FileName        string                  `json:"file_name,omitempty"`
	Redactions      []RedactionItemResponse `json:"redactions"`
	TotalRedactions int                     `json:"total_redactions"`
	PDFBase64       string                  `json:"pdf_base64,omitempty"`
}

// PresentRedactionAnalysis converts AnalyzeRedactionsResponse to presentation format
func PresentRedactionAnalysis(dto *dto.AnalyzeRedactionsResponse) *RedactionAnalysisResponse {
	if dto == nil {
		return nil
	}

	return &RedactionAnalysisResponse{
		DocumentID: dto.DocumentID,
		FileName:   dto.FileName,
		Redactions: convertRedactionItems(dto.Redactions),
		TotalCount: dto.TotalCount,
	}
}

// PresentRedactionResult converts ApplyRedactionsResponse to presentation format
func PresentRedactionResult(dto *dto.ApplyRedactionsResponse) *RedactionResultResponse {
	if dto == nil {
		return nil
	}

	return &RedactionResultResponse{
		DocumentID:      dto.DocumentID,
		FileName:        dto.FileName,
		Redactions:      convertRedactionItems(dto.Redactions),
		TotalRedactions: dto.TotalRedactions,
		PDFBase64:       dto.PDFBase64,
	}
}

// convertRedactionItems converts DTO redaction items to response format
func convertRedactionItems(items []dto.RedactionItem) []RedactionItemResponse {
	result := make([]RedactionItemResponse, len(items))
	for i, item := range items {
		result[i] = RedactionItemResponse{
			ID:        item.ID,
			Page:      item.Page,
			Text:      item.Text,
			BBox:      item.BBox,
			Type:      item.Type,
			Citation:  item.Citation,
			Reason:    item.Reason,
			LegalCode: item.LegalCode,
			Applied:   item.Applied,
		}
	}
	return result
}
