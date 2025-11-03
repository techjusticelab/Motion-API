package document

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"motion-index-fiber/internal/application/dto"
	"motion-index-fiber/internal/domain/document"
	domainerrors "motion-index-fiber/internal/domain/errors"
)

func TestUpdateMetadataUseCase_Success(t *testing.T) {
	repo := new(documentRepositoryMock)
	search := new(searchServiceMock)
	eventBus := new(eventBusMock)

	doc := mustNewDocument(t, "doc_update_metadata_success")
	doc.ClearEvents()

	repo.On("FindByID", mock.Anything, doc.ID()).Return(doc, nil)
	repo.On("Save", mock.Anything, doc).Return(nil)

	processedAt := time.Now().Add(-time.Hour)
	search.On(
		"Update",
		mock.Anything,
		doc.ID().String(),
		mock.MatchedBy(func(fields map[string]interface{}) bool {
			metadataFields, ok := fields["metadata"].(map[string]interface{})
			if !ok {
				return false
			}
			if metadataFields["language"] != "en" {
				return false
			}
			tags, ok := metadataFields["legalTags"].([]string)
			if !ok || len(tags) != 2 {
				return false
			}
			if metadataFields["aiClassified"] != true {
				return false
			}
			timePtr, ok := metadataFields["processedAt"].(*time.Time)
			if !ok || timePtr == nil {
				return false
			}
			return true
		}),
	).Return(nil)

	eventBus.On("Publish", mock.Anything, mock.Anything).Return(nil)

	useCase := NewUpdateMetadataUseCase(repo, search, eventBus)

	resp, err := useCase.Execute(context.Background(), &dto.UpdateMetadataRequest{
		DocumentID:   doc.ID().String(),
		Language:     "en",
		LegalTags:    []string{"Motion", "Suppression"},
		AIClassified: true,
		ProcessedAt:  &processedAt,
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, doc.ID().String(), resp.DocumentID)
	assert.Equal(t, "en", resp.Language)
	assert.True(t, resp.AIClassified)
	assert.NotNil(t, resp.ProcessedAt)
	assert.Len(t, resp.LegalTags, 2)

	repo.AssertExpectations(t)
	search.AssertExpectations(t)
	eventBus.AssertExpectations(t)
}

func TestUpdateMetadataUseCase_SearchOptional(t *testing.T) {
	repo := new(documentRepositoryMock)
	eventBus := new(eventBusMock)

	doc := mustNewDocument(t, "doc_update_no_search")
	doc.ClearEvents()

	repo.On("FindByID", mock.Anything, doc.ID()).Return(doc, nil)
	repo.On("Save", mock.Anything, doc).Return(nil)

	eventBus.On("Publish", mock.Anything, mock.Anything).Return(nil)

	useCase := NewUpdateMetadataUseCase(repo, nil, eventBus)

	resp, err := useCase.Execute(context.Background(), &dto.UpdateMetadataRequest{
		DocumentID: doc.ID().String(),
		Language:   "es",
		LegalTags:  []string{"evidence"},
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "es", resp.Language)

	repo.AssertExpectations(t)
	eventBus.AssertExpectations(t)
}

func TestUpdateMetadataUseCase_EventBusOptional(t *testing.T) {
	repo := new(documentRepositoryMock)
	search := new(searchServiceMock)

	doc := mustNewDocument(t, "doc_update_no_eventbus")
	doc.ClearEvents()

	repo.On("FindByID", mock.Anything, doc.ID()).Return(doc, nil)
	repo.On("Save", mock.Anything, doc).Return(nil)

	search.On("Update", mock.Anything, doc.ID().String(), mock.AnythingOfType("map[string]interface {}")).Return(nil)

	useCase := NewUpdateMetadataUseCase(repo, search, nil)

	resp, err := useCase.Execute(context.Background(), &dto.UpdateMetadataRequest{
		DocumentID: doc.ID().String(),
		Language:   "fr",
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "fr", resp.Language)

	repo.AssertExpectations(t)
	search.AssertExpectations(t)
}

func TestUpdateMetadataUseCase_RepositoryNotConfigured(t *testing.T) {
	useCase := NewUpdateMetadataUseCase(nil, nil, nil)

	_, err := useCase.Execute(context.Background(), &dto.UpdateMetadataRequest{
		DocumentID: "doc",
		Language:   "en",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "repository is not configured")
}

func TestUpdateMetadataUseCase_DocumentNotFound(t *testing.T) {
	repo := new(documentRepositoryMock)
	docID, err := document.NewDocumentID("missing")
	require.NoError(t, err)

	repo.On("FindByID", mock.Anything, docID).Return((*document.Document)(nil), nil)

	useCase := NewUpdateMetadataUseCase(repo, nil, nil)

	_, err = useCase.Execute(context.Background(), &dto.UpdateMetadataRequest{
		DocumentID: docID.String(),
		Language:   "en",
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, domainerrors.ErrDocumentNotFound)
}

func TestUpdateMetadataUseCase_RepositoryError(t *testing.T) {
	repo := new(documentRepositoryMock)
	docID, err := document.NewDocumentID("repo_error")
	require.NoError(t, err)

	repo.On("FindByID", mock.Anything, docID).Return((*document.Document)(nil), errors.New("repo failure"))

	useCase := NewUpdateMetadataUseCase(repo, nil, nil)

	_, err = useCase.Execute(context.Background(), &dto.UpdateMetadataRequest{
		DocumentID: docID.String(),
		Language:   "en",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "repo failure")
}

func TestUpdateMetadataUseCase_SaveError(t *testing.T) {
	repo := new(documentRepositoryMock)
	doc := mustNewDocument(t, "doc_update_save_error")
	doc.ClearEvents()

	repo.On("FindByID", mock.Anything, doc.ID()).Return(doc, nil)
	repo.On("Save", mock.Anything, doc).Return(errors.New("persist failed"))

	useCase := NewUpdateMetadataUseCase(repo, nil, nil)

	_, err := useCase.Execute(context.Background(), &dto.UpdateMetadataRequest{
		DocumentID: doc.ID().String(),
		Language:   "en",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "persist failed")
}

func TestUpdateMetadataUseCase_SearchError(t *testing.T) {
	repo := new(documentRepositoryMock)
	search := new(searchServiceMock)

	doc := mustNewDocument(t, "doc_update_search_error")
	doc.ClearEvents()

	repo.On("FindByID", mock.Anything, doc.ID()).Return(doc, nil)
	repo.On("Save", mock.Anything, doc).Return(nil)

	search.On("Update", mock.Anything, doc.ID().String(), mock.AnythingOfType("map[string]interface {}")).Return(errors.New("search failure"))

	useCase := NewUpdateMetadataUseCase(repo, search, nil)

	_, err := useCase.Execute(context.Background(), &dto.UpdateMetadataRequest{
		DocumentID: doc.ID().String(),
		Language:   "en",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "search failure")
}

func TestUpdateMetadataUseCase_EventBusError(t *testing.T) {
	repo := new(documentRepositoryMock)
	search := new(searchServiceMock)
	eventBus := new(eventBusMock)

	doc := mustNewDocument(t, "doc_update_eventbus_error")
	doc.ClearEvents()

	repo.On("FindByID", mock.Anything, doc.ID()).Return(doc, nil)
	repo.On("Save", mock.Anything, doc).Return(nil)
	search.On("Update", mock.Anything, doc.ID().String(), mock.AnythingOfType("map[string]interface {}")).Return(nil)
	eventBus.On("Publish", mock.Anything, mock.Anything).Return(errors.New("eventbus failure"))

	useCase := NewUpdateMetadataUseCase(repo, search, eventBus)

	_, err := useCase.Execute(context.Background(), &dto.UpdateMetadataRequest{
		DocumentID: doc.ID().String(),
		Language:   "en",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "eventbus failure")
}

func TestUpdateMetadataUseCase_InvalidRequest(t *testing.T) {
	useCase := NewUpdateMetadataUseCase(new(documentRepositoryMock), nil, nil)

	_, err := useCase.Execute(context.Background(), &dto.UpdateMetadataRequest{})
	require.Error(t, err)
}
