package presenter

import (
	"testing"
	"time"

	"motion-index-fiber/internal/application/dto"

	"github.com/stretchr/testify/assert"
)

func TestPresentSearchResults(t *testing.T) {
	t.Run("with results", func(t *testing.T) {
		// Given: Search results DTO
		now := time.Now()
		dtoResp := &dto.SearchResultsResponse{
			Documents: []dto.DocumentSummary{
				{
					ID:           "doc_1",
					FileName:     "motion.pdf",
					DocumentType: "motion",
					Category:     "filing",
					Snippet:      "Motion to suppress evidence...",
					Confidence:   0.95,
					CreatedAt:    now,
				},
				{
					ID:           "doc_2",
					FileName:     "brief.pdf",
					DocumentType: "brief",
					Category:     "argument",
					Snippet:      "Legal brief regarding...",
					Confidence:   0.88,
					CreatedAt:    now.Add(-time.Hour),
				},
			},
			Total:      2,
			Page:       1,
			PageSize:   20,
			TotalPages: 1,
		}

		// When: Present search results
		resp := PresentSearchResults(dtoResp)

		// Then: All fields are correctly mapped
		assert.NotNil(t, resp)
		assert.Len(t, resp.Documents, 2)
		assert.Equal(t, 2, resp.Total)
		assert.Equal(t, 1, resp.Page)
		assert.Equal(t, 20, resp.PageSize)
		assert.Equal(t, 1, resp.TotalPages)

		// Check first document
		assert.Equal(t, "doc_1", resp.Documents[0].ID)
		assert.Equal(t, "motion.pdf", resp.Documents[0].FileName)
		assert.Equal(t, "motion", resp.Documents[0].DocumentType)
		assert.Equal(t, "filing", resp.Documents[0].Category)
		assert.Equal(t, "Motion to suppress evidence...", resp.Documents[0].Snippet)
		assert.Equal(t, 0.95, resp.Documents[0].Confidence)
		assert.Equal(t, now.Format(time.RFC3339), resp.Documents[0].CreatedAt)

		// Check second document
		assert.Equal(t, "doc_2", resp.Documents[1].ID)
		assert.Equal(t, "brief.pdf", resp.Documents[1].FileName)
	})

	t.Run("with empty results", func(t *testing.T) {
		// Given: Empty search results DTO
		dtoResp := &dto.SearchResultsResponse{
			Documents:  []dto.DocumentSummary{},
			Total:      0,
			Page:       1,
			PageSize:   20,
			TotalPages: 0,
		}

		// When: Present search results
		resp := PresentSearchResults(dtoResp)

		// Then: Empty array is returned
		assert.NotNil(t, resp)
		assert.Empty(t, resp.Documents)
		assert.Equal(t, 0, resp.Total)
		assert.Equal(t, 0, resp.TotalPages)
	})

	t.Run("with nil DTO", func(t *testing.T) {
		// When: Present nil DTO
		resp := PresentSearchResults(nil)

		// Then: Returns nil
		assert.Nil(t, resp)
	})
}

func TestPresentDocumentSummary(t *testing.T) {
	t.Run("with all fields", func(t *testing.T) {
		// Given: Complete document summary
		now := time.Now()
		doc := dto.DocumentSummary{
			ID:           "doc_123",
			FileName:     "test.pdf",
			DocumentType: "motion",
			Category:     "criminal",
			Snippet:      "Test snippet...",
			Confidence:   0.92,
			CreatedAt:    now,
		}

		// When: Present document summary
		resp := PresentDocumentSummary(doc)

		// Then: All fields are correctly mapped
		assert.Equal(t, "doc_123", resp.ID)
		assert.Equal(t, "test.pdf", resp.FileName)
		assert.Equal(t, "motion", resp.DocumentType)
		assert.Equal(t, "criminal", resp.Category)
		assert.Equal(t, "Test snippet...", resp.Snippet)
		assert.Equal(t, 0.92, resp.Confidence)
		assert.Equal(t, now.Format(time.RFC3339), resp.CreatedAt)
	})

	t.Run("with minimal fields", func(t *testing.T) {
		// Given: Minimal document summary
		now := time.Now()
		doc := dto.DocumentSummary{
			ID:        "doc_456",
			FileName:  "minimal.pdf",
			CreatedAt: now,
		}

		// When: Present document summary
		resp := PresentDocumentSummary(doc)

		// Then: Fields are correctly mapped
		assert.Equal(t, "doc_456", resp.ID)
		assert.Equal(t, "minimal.pdf", resp.FileName)
		assert.Empty(t, resp.DocumentType)
		assert.Empty(t, resp.Category)
		assert.Empty(t, resp.Snippet)
		assert.Equal(t, 0.0, resp.Confidence)
	})
}

func TestPresentDocumentSummaries(t *testing.T) {
	t.Run("with multiple summaries", func(t *testing.T) {
		// Given: Multiple document summaries
		now := time.Now()
		dtos := []dto.DocumentSummary{
			{
				ID:        "doc_1",
				FileName:  "file1.pdf",
				CreatedAt: now,
			},
			{
				ID:        "doc_2",
				FileName:  "file2.pdf",
				CreatedAt: now.Add(-time.Hour),
			},
		}

		// When: Present document summaries
		resp := PresentDocumentSummaries(dtos)

		// Then: All summaries are mapped
		assert.Len(t, resp, 2)
		assert.Equal(t, "doc_1", resp[0].ID)
		assert.Equal(t, "file1.pdf", resp[0].FileName)
		assert.Equal(t, "doc_2", resp[1].ID)
		assert.Equal(t, "file2.pdf", resp[1].FileName)
	})

	t.Run("with empty slice", func(t *testing.T) {
		// Given: Empty slice
		dtos := []dto.DocumentSummary{}

		// When: Present document summaries
		resp := PresentDocumentSummaries(dtos)

		// Then: Empty slice is returned
		assert.NotNil(t, resp)
		assert.Empty(t, resp)
	})

	t.Run("with nil slice", func(t *testing.T) {
		// When: Present nil slice
		resp := PresentDocumentSummaries(nil)

		// Then: Returns nil
		assert.Nil(t, resp)
	})
}
