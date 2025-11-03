package redaction

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"motion-index-fiber/internal/application/dto"
	"motion-index-fiber/internal/application/ports"
	"motion-index-fiber/internal/domain/document"
	domainerrors "motion-index-fiber/internal/domain/errors"
)

func TestAnalyzeRedactionsUseCase_WithDocumentID(t *testing.T) {
	repo := new(documentRepositoryMock)
	storage := new(storageServiceMock)
	redactionSvc := new(redactionServiceMock)

	doc := mustNewDocument(t, "doc-analyze")
	doc.ClearEvents()

	repo.On("FindByID", mock.Anything, doc.ID()).Return(doc, nil)

	fileBytes := []byte("%PDF-1.4")
	storage.On("Retrieve", mock.Anything, doc.FilePath().String()).
		Return(io.NopCloser(bytes.NewReader(fileBytes)), nil)

	redactionSvc.
		On("Analyze", mock.Anything, mock.Anything, mock.AnythingOfType("*ports.RedactionOptions")).
		Return(&ports.RedactionAnalysis{
			Redactions: []ports.RedactionItem{
				{ID: "r1", Page: 1, Text: "SSN"},
			},
			TotalCount: 1,
		}, nil)

	useCase := NewAnalyzeRedactionsUseCase(repo, storage, redactionSvc)

	resp, err := useCase.Execute(context.Background(), &dto.AnalyzeRedactionsRequest{
		DocumentID: doc.ID().String(),
		Options: &dto.RedactionOptions{
			UseAI: true,
		},
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, doc.ID().String(), resp.DocumentID)
	assert.Equal(t, doc.FileName().String(), resp.FileName)
	assert.Len(t, resp.Redactions, 1)
	assert.Equal(t, 1, resp.TotalCount)

	repo.AssertExpectations(t)
	storage.AssertExpectations(t)
	redactionSvc.AssertExpectations(t)
}

func TestAnalyzeRedactionsUseCase_WithPDFBase64(t *testing.T) {
	redactionSvc := new(redactionServiceMock)

	redactionSvc.On("Analyze", mock.Anything, mock.Anything, mock.AnythingOfType("*ports.RedactionOptions")).
		Return(&ports.RedactionAnalysis{
			Redactions: []ports.RedactionItem{},
			TotalCount: 0,
		}, nil)

	useCase := NewAnalyzeRedactionsUseCase(nil, nil, redactionSvc)

	resp, err := useCase.Execute(context.Background(), &dto.AnalyzeRedactionsRequest{
		PDFBase64: base64.StdEncoding.EncodeToString([]byte("%PDF")),
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Empty(t, resp.Redactions)
	assert.Equal(t, 0, resp.TotalCount)
}

func TestAnalyzeRedactionsUseCase_RedactionServiceMissing(t *testing.T) {
	useCase := NewAnalyzeRedactionsUseCase(nil, nil, nil)

	_, err := useCase.Execute(context.Background(), &dto.AnalyzeRedactionsRequest{
		DocumentID: "doc",
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "redaction service is not configured")
}

func TestAnalyzeRedactionsUseCase_DocumentRepositoryMissing(t *testing.T) {
	redactionSvc := new(redactionServiceMock)
	useCase := NewAnalyzeRedactionsUseCase(nil, new(storageServiceMock), redactionSvc)

	_, err := useCase.Execute(context.Background(), &dto.AnalyzeRedactionsRequest{
		DocumentID: "doc",
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "repository is not configured")
}

func TestAnalyzeRedactionsUseCase_DocumentNotFound(t *testing.T) {
	repo := new(documentRepositoryMock)
	storage := new(storageServiceMock)
	redactionSvc := new(redactionServiceMock)

	docID, err := document.NewDocumentID("missing")
	require.NoError(t, err)

	repo.On("FindByID", mock.Anything, docID).Return((*document.Document)(nil), nil)

	useCase := NewAnalyzeRedactionsUseCase(repo, storage, redactionSvc)

	_, err = useCase.Execute(context.Background(), &dto.AnalyzeRedactionsRequest{
		DocumentID: docID.String(),
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, domainerrors.ErrDocumentNotFound)
}

func TestAnalyzeRedactionsUseCase_StorageError(t *testing.T) {
	repo := new(documentRepositoryMock)
	storage := new(storageServiceMock)
	redactionSvc := new(redactionServiceMock)

	doc := mustNewDocument(t, "doc-storage-error")
	doc.ClearEvents()

	repo.On("FindByID", mock.Anything, doc.ID()).Return(doc, nil)
	storage.On("Retrieve", mock.Anything, doc.FilePath().String()).
		Return(io.ReadCloser(nil), errors.New("retrieve failed"))

	useCase := NewAnalyzeRedactionsUseCase(repo, storage, redactionSvc)

	_, err := useCase.Execute(context.Background(), &dto.AnalyzeRedactionsRequest{
		DocumentID: doc.ID().String(),
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "retrieve failed")
}

func TestAnalyzeRedactionsUseCase_RedactionError(t *testing.T) {
	repo := new(documentRepositoryMock)
	storage := new(storageServiceMock)
	redactionSvc := new(redactionServiceMock)

	doc := mustNewDocument(t, "doc-redaction-error")
	doc.ClearEvents()

	repo.On("FindByID", mock.Anything, doc.ID()).Return(doc, nil)
	storage.On("Retrieve", mock.Anything, doc.FilePath().String()).
		Return(io.NopCloser(bytes.NewReader([]byte("%PDF"))), nil)
	redactionSvc.On("Analyze", mock.Anything, mock.Anything, mock.Anything).
		Return((*ports.RedactionAnalysis)(nil), errors.New("analysis failed"))

	useCase := NewAnalyzeRedactionsUseCase(repo, storage, redactionSvc)

	_, err := useCase.Execute(context.Background(), &dto.AnalyzeRedactionsRequest{
		DocumentID: doc.ID().String(),
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "analysis failed")
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
