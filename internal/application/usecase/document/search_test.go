package document

import (
    "context"
    "errors"
    "testing"
    "time"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"

    "motion-index-fiber/internal/application/dto"
    "motion-index-fiber/internal/application/ports"
)

func TestSearchDocumentsUseCase_Success(t *testing.T) {
	svc := new(searchServiceMock)
	now := time.Now()
	svc.On("Search", mock.Anything, mock.AnythingOfType("ports.SearchQuery")).Return(&ports.SearchResults{
		Total:    2,
		Page:     1,
		PageSize: 10,
		Hits: []ports.SearchHit{{
			ID:           "doc_1",
			FileName:     "example.pdf",
			DocumentType: "motion",
			Category:     "filing",
			Snippet:      "snippet",
			Confidence:   0.9,
			CreatedAt:    now,
		}},
	}, nil)

	useCase := NewSearchDocumentsUseCase(svc)

	req := &dto.SearchDocumentsRequest{Query: "motion", Page: 1, PageSize: 10}
	resp, err := useCase.Execute(context.Background(), req)
	assert.NoError(t, err)
	assert.Equal(t, 2, resp.Total)
	assert.Equal(t, 1, resp.TotalPages)
	assert.Len(t, resp.Documents, 1)

	svc.AssertExpectations(t)
}

func TestSearchDocumentsUseCase_InvalidRequest(t *testing.T) {
	useCase := NewSearchDocumentsUseCase(nil)
	resp, err := useCase.Execute(context.Background(), &dto.SearchDocumentsRequest{Page: 0, PageSize: 1})
	assert.Error(t, err)
	assert.Nil(t, resp)
}

func TestSearchDocumentsUseCase_ServiceError(t *testing.T) {
	svc := new(searchServiceMock)
	svc.On("Search", mock.Anything, mock.AnythingOfType("ports.SearchQuery")).Return((*ports.SearchResults)(nil), errors.New("search failed"))

	useCase := NewSearchDocumentsUseCase(svc)
	resp, err := useCase.Execute(context.Background(), &dto.SearchDocumentsRequest{Query: "motion", Page: 1, PageSize: 10})
	assert.Error(t, err)
	assert.Nil(t, resp)
}

func TestSearchDocumentsUseCase_NoService(t *testing.T) {
	useCase := NewSearchDocumentsUseCase(nil)
	req := &dto.SearchDocumentsRequest{Query: "motion", Page: 1, PageSize: 10}
	resp, err := useCase.Execute(context.Background(), req)
	assert.NoError(t, err)
	assert.Equal(t, 0, resp.Total)
}
