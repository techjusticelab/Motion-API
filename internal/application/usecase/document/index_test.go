package document

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"motion-index-fiber/internal/application/dto"
	"motion-index-fiber/internal/domain/document"
	domainerrors "motion-index-fiber/internal/domain/errors"
)

func TestIndexDocumentUseCase_Success(t *testing.T) {
	repo := new(documentRepositoryMock)
	search := new(searchServiceMock)

	doc := mustNewDocument(t, "doc_index_success")
	doc.ClearEvents()

	repo.On("FindByID", mock.Anything, doc.ID()).Return(doc, nil)
	search.On("Index", mock.Anything, doc).Return(nil)

	useCase := NewIndexDocumentUseCase(repo, search)

	resp, err := useCase.Execute(context.Background(), &dto.IndexDocumentRequest{
		DocumentID: doc.ID().String(),
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, doc.ID().String(), resp.DocumentID)
	assert.True(t, resp.Indexed)

	repo.AssertExpectations(t)
	search.AssertExpectations(t)
}

func TestIndexDocumentUseCase_RepositoryNotConfigured(t *testing.T) {
	useCase := NewIndexDocumentUseCase(nil, new(searchServiceMock))

	_, err := useCase.Execute(context.Background(), &dto.IndexDocumentRequest{DocumentID: "doc"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "repository is not configured")
}

func TestIndexDocumentUseCase_SearchNotConfigured(t *testing.T) {
	useCase := NewIndexDocumentUseCase(new(documentRepositoryMock), nil)

	_, err := useCase.Execute(context.Background(), &dto.IndexDocumentRequest{DocumentID: "doc"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "search service is not configured")
}

func TestIndexDocumentUseCase_DocumentNotFound(t *testing.T) {
	repo := new(documentRepositoryMock)
	search := new(searchServiceMock)

	docID, err := document.NewDocumentID("missing")
	require.NoError(t, err)

	repo.On("FindByID", mock.Anything, docID).Return((*document.Document)(nil), nil)

	useCase := NewIndexDocumentUseCase(repo, search)

	_, err = useCase.Execute(context.Background(), &dto.IndexDocumentRequest{DocumentID: docID.String()})
	require.Error(t, err)
	assert.ErrorIs(t, err, domainerrors.ErrDocumentNotFound)
}

func TestIndexDocumentUseCase_RepositoryError(t *testing.T) {
	repo := new(documentRepositoryMock)
	search := new(searchServiceMock)

	docID, err := document.NewDocumentID("repo_error")
	require.NoError(t, err)

	repo.On("FindByID", mock.Anything, docID).Return((*document.Document)(nil), errors.New("load failed"))

	useCase := NewIndexDocumentUseCase(repo, search)

	_, err = useCase.Execute(context.Background(), &dto.IndexDocumentRequest{DocumentID: docID.String()})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "load failed")
}

func TestIndexDocumentUseCase_SearchError(t *testing.T) {
	repo := new(documentRepositoryMock)
	search := new(searchServiceMock)

	doc := mustNewDocument(t, "doc_index_error")
	doc.ClearEvents()

	repo.On("FindByID", mock.Anything, doc.ID()).Return(doc, nil)
	search.On("Index", mock.Anything, doc).Return(errors.New("index failure"))

	useCase := NewIndexDocumentUseCase(repo, search)

	_, err := useCase.Execute(context.Background(), &dto.IndexDocumentRequest{DocumentID: doc.ID().String()})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "index failure")
}

func TestIndexDocumentUseCase_InvalidRequest(t *testing.T) {
	useCase := NewIndexDocumentUseCase(new(documentRepositoryMock), new(searchServiceMock))

	_, err := useCase.Execute(context.Background(), &dto.IndexDocumentRequest{})
	require.Error(t, err)
}

func mustNewDocument(t *testing.T, id string) *document.Document {
	t.Helper()

	doc, err := document.NewDocument(
		document.WithID(id),
		document.WithFileName("example.pdf"),
		document.WithStoragePath("documents/example.pdf"),
		document.WithHash("a3c1e5a6b9d2f4a7c8e1f2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2d3", "SHA256"),
		document.WithContentType("application/pdf"),
		document.WithFileSize(1024),
	)
	require.NoError(t, err)
	return doc
}
