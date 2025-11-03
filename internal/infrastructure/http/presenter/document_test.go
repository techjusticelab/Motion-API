package presenter

import (
	"testing"
	"time"

	"motion-index-fiber/internal/application/dto"

	"github.com/stretchr/testify/assert"
)

func TestPresentDocument(t *testing.T) {
	t.Run("with all fields", func(t *testing.T) {
		// Given: Complete DTO
		now := time.Now()
		processedAt := now.Add(time.Hour)
		dtoResp := &dto.ProcessDocumentResponse{
			DocumentID:   "doc_123",
			Stored:       true,
			StorageURL:   "https://example.com/doc_123.pdf",
			Classified:   true,
			Indexed:      true,
			CreatedAt:    now,
			ProcessedAt:  &processedAt,
			EventsQueued: 3,
		}

		// When: Present document
		resp := PresentDocument(dtoResp)

		// Then: All fields are correctly mapped
		assert.NotNil(t, resp)
		assert.Equal(t, "doc_123", resp.ID)
		assert.True(t, resp.Stored)
		assert.Equal(t, "https://example.com/doc_123.pdf", resp.StorageURL)
		assert.True(t, resp.Classified)
		assert.True(t, resp.Indexed)
		assert.Equal(t, now.Format(time.RFC3339), resp.CreatedAt)
		assert.Equal(t, processedAt.Format(time.RFC3339), resp.ProcessedAt)
		assert.Equal(t, 3, resp.EventsQueued)
	})

	t.Run("with minimal fields", func(t *testing.T) {
		// Given: Minimal DTO
		now := time.Now()
		dtoResp := &dto.ProcessDocumentResponse{
			DocumentID: "doc_456",
			Stored:     false,
			Classified: false,
			Indexed:    false,
			CreatedAt:  now,
		}

		// When: Present document
		resp := PresentDocument(dtoResp)

		// Then: Fields are correctly mapped
		assert.NotNil(t, resp)
		assert.Equal(t, "doc_456", resp.ID)
		assert.False(t, resp.Stored)
		assert.Empty(t, resp.StorageURL)
		assert.False(t, resp.Classified)
		assert.False(t, resp.Indexed)
		assert.Empty(t, resp.ProcessedAt)
	})

	t.Run("with nil DTO", func(t *testing.T) {
		// When: Present nil DTO
		resp := PresentDocument(nil)

		// Then: Returns nil
		assert.Nil(t, resp)
	})
}

func TestPresentBatchDocument(t *testing.T) {
	t.Run("with errors", func(t *testing.T) {
		// Given: Batch DTO with errors
		dtoResp := &dto.BatchProcessDocumentResponse{
			Total:     10,
			Succeeded: 8,
			Failed:    2,
			Errors: map[string]string{
				"doc_1": "invalid content type",
				"doc_2": "file too large",
			},
		}

		// When: Present batch document
		resp := PresentBatchDocument(dtoResp)

		// Then: All fields are correctly mapped
		assert.NotNil(t, resp)
		assert.Equal(t, 10, resp.Total)
		assert.Equal(t, 8, resp.Succeeded)
		assert.Equal(t, 2, resp.Failed)
		assert.Len(t, resp.Errors, 2)
		assert.Equal(t, "invalid content type", resp.Errors["doc_1"])
		assert.Equal(t, "file too large", resp.Errors["doc_2"])
	})

	t.Run("without errors", func(t *testing.T) {
		// Given: Batch DTO without errors
		dtoResp := &dto.BatchProcessDocumentResponse{
			Total:     5,
			Succeeded: 5,
			Failed:    0,
			Errors:    nil,
		}

		// When: Present batch document
		resp := PresentBatchDocument(dtoResp)

		// Then: Fields are correctly mapped
		assert.NotNil(t, resp)
		assert.Equal(t, 5, resp.Total)
		assert.Equal(t, 5, resp.Succeeded)
		assert.Equal(t, 0, resp.Failed)
		assert.Nil(t, resp.Errors)
	})

	t.Run("with nil DTO", func(t *testing.T) {
		// When: Present nil DTO
		resp := PresentBatchDocument(nil)

		// Then: Returns nil
		assert.Nil(t, resp)
	})
}

func TestPresentIndex(t *testing.T) {
	t.Run("when indexed", func(t *testing.T) {
		// Given: Successful index DTO
		dtoResp := &dto.IndexDocumentResponse{
			DocumentID: "doc_123",
			Indexed:    true,
		}

		// When: Present index
		resp := PresentIndex(dtoResp)

		// Then: Success message is included
		assert.NotNil(t, resp)
		assert.Equal(t, "doc_123", resp.DocumentID)
		assert.True(t, resp.Indexed)
		assert.Equal(t, "Document indexed successfully", resp.Message)
	})

	t.Run("when not indexed", func(t *testing.T) {
		// Given: Failed index DTO
		dtoResp := &dto.IndexDocumentResponse{
			DocumentID: "doc_456",
			Indexed:    false,
		}

		// When: Present index
		resp := PresentIndex(dtoResp)

		// Then: Failure message is included
		assert.NotNil(t, resp)
		assert.Equal(t, "doc_456", resp.DocumentID)
		assert.False(t, resp.Indexed)
		assert.Equal(t, "Document was not indexed", resp.Message)
	})

	t.Run("with nil DTO", func(t *testing.T) {
		// When: Present nil DTO
		resp := PresentIndex(nil)

		// Then: Returns nil
		assert.Nil(t, resp)
	})
}

func TestPresentMetadata(t *testing.T) {
	t.Run("with all fields", func(t *testing.T) {
		// Given: Complete metadata DTO
		processedAt := time.Now()
		dtoResp := &dto.UpdateMetadataResponse{
			DocumentID:   "doc_123",
			Language:     "en",
			LegalTags:    []string{"criminal", "motion"},
			AIClassified: true,
			ProcessedAt:  &processedAt,
		}

		// When: Present metadata
		resp := PresentMetadata(dtoResp)

		// Then: All fields are correctly mapped
		assert.NotNil(t, resp)
		assert.Equal(t, "doc_123", resp.DocumentID)
		assert.Equal(t, "en", resp.Language)
		assert.Equal(t, []string{"criminal", "motion"}, resp.LegalTags)
		assert.True(t, resp.AIClassified)
		assert.Equal(t, processedAt.Format(time.RFC3339), resp.ProcessedAt)
	})

	t.Run("without optional fields", func(t *testing.T) {
		// Given: Minimal metadata DTO
		dtoResp := &dto.UpdateMetadataResponse{
			DocumentID:   "doc_456",
			Language:     "en",
			LegalTags:    []string{},
			AIClassified: false,
			ProcessedAt:  nil,
		}

		// When: Present metadata
		resp := PresentMetadata(dtoResp)

		// Then: Fields are correctly mapped
		assert.NotNil(t, resp)
		assert.Equal(t, "doc_456", resp.DocumentID)
		assert.Equal(t, "en", resp.Language)
		assert.Empty(t, resp.LegalTags)
		assert.False(t, resp.AIClassified)
		assert.Empty(t, resp.ProcessedAt)
	})

	t.Run("with nil DTO", func(t *testing.T) {
		// When: Present nil DTO
		resp := PresentMetadata(nil)

		// Then: Returns nil
		assert.Nil(t, resp)
	})
}
