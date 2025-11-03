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

func TestApplyRedactionsUseCase_Success(t *testing.T) {
	repo := new(documentRepositoryMock)
	storage := new(storageServiceMock)
	redactionSvc := new(redactionServiceMock)

	doc := mustNewDocument(t, "doc-apply")
	doc.ClearEvents()

	repo.On("FindByID", mock.Anything, doc.ID()).Return(doc, nil)
	storage.On("Retrieve", mock.Anything, doc.FilePath().String()).
		Return(io.NopCloser(bytes.NewReader([]byte("%PDF"))), nil)
	redactionSvc.On("Redact", mock.Anything, mock.Anything, mock.AnythingOfType("*ports.RedactionOptions")).
		Return(&ports.RedactionResult{
			Redactions: []ports.RedactionItem{
				{ID: "r1", Page: 1},
			},
			TotalCount: 1,
			PDFBase64:  "base64pdf",
		}, nil)

	useCase := NewApplyRedactionsUseCase(repo, storage, redactionSvc)

	resp, err := useCase.Execute(context.Background(), &dto.ApplyRedactionsRequest{
		DocumentID:   doc.ID().String(),
		ReturnBase64: true,
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, doc.ID().String(), resp.DocumentID)
	assert.Equal(t, "base64pdf", resp.PDFBase64)
	assert.Equal(t, 1, resp.TotalRedactions)

	repo.AssertExpectations(t)
	storage.AssertExpectations(t)
	redactionSvc.AssertExpectations(t)
}

func TestApplyRedactionsUseCase_CustomRedactions(t *testing.T) {
	redactionSvc := new(redactionServiceMock)

	custom := []dto.RedactionItem{
		{ID: "custom", Page: 2},
	}

	redactionSvc.On("ApplyCustom", mock.Anything, mock.Anything, mock.AnythingOfType("*ports.RedactionOptions"), mock.Anything).
		Return(&ports.RedactionResult{
			Redactions: []ports.RedactionItem{{ID: "custom", Page: 2}},
			TotalCount: 1,
		}, nil)

	useCase := NewApplyRedactionsUseCase(nil, nil, redactionSvc)

	resp, err := useCase.Execute(context.Background(), &dto.ApplyRedactionsRequest{
		PDFBase64:        base64.StdEncoding.EncodeToString([]byte("%PDF")),
		CustomRedactions: custom,
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Len(t, resp.Redactions, 1)
}

func TestApplyRedactionsUseCase_RedactionServiceMissing(t *testing.T) {
	useCase := NewApplyRedactionsUseCase(nil, nil, nil)

	_, err := useCase.Execute(context.Background(), &dto.ApplyRedactionsRequest{
		DocumentID: "doc",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "redaction service is not configured")
}

func TestApplyRedactionsUseCase_RepositoryMissing(t *testing.T) {
	redactionSvc := new(redactionServiceMock)
	useCase := NewApplyRedactionsUseCase(nil, new(storageServiceMock), redactionSvc)

	_, err := useCase.Execute(context.Background(), &dto.ApplyRedactionsRequest{
		DocumentID: "doc",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "repository is not configured")
}

func TestApplyRedactionsUseCase_DocumentNotFound(t *testing.T) {
	repo := new(documentRepositoryMock)
	storage := new(storageServiceMock)
	redactionSvc := new(redactionServiceMock)

	docID, err := document.NewDocumentID("missing")
	require.NoError(t, err)

	repo.On("FindByID", mock.Anything, docID).Return((*document.Document)(nil), nil)

	useCase := NewApplyRedactionsUseCase(repo, storage, redactionSvc)

	_, err = useCase.Execute(context.Background(), &dto.ApplyRedactionsRequest{DocumentID: docID.String()})
	require.Error(t, err)
	assert.ErrorIs(t, err, domainerrors.ErrDocumentNotFound)
}

func TestApplyRedactionsUseCase_StorageError(t *testing.T) {
	repo := new(documentRepositoryMock)
	storage := new(storageServiceMock)
	redactionSvc := new(redactionServiceMock)

	doc := mustNewDocument(t, "doc-apply-storage")
	doc.ClearEvents()

	repo.On("FindByID", mock.Anything, doc.ID()).Return(doc, nil)
	storage.On("Retrieve", mock.Anything, doc.FilePath().String()).
		Return(io.ReadCloser(nil), errors.New("retrieve failed"))

	useCase := NewApplyRedactionsUseCase(repo, storage, redactionSvc)

	_, err := useCase.Execute(context.Background(), &dto.ApplyRedactionsRequest{
		DocumentID: doc.ID().String(),
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "retrieve failed")
}

func TestApplyRedactionsUseCase_RedactionError(t *testing.T) {
	repo := new(documentRepositoryMock)
	storage := new(storageServiceMock)
	redactionSvc := new(redactionServiceMock)

	doc := mustNewDocument(t, "doc-apply-redaction-error")
	doc.ClearEvents()

	repo.On("FindByID", mock.Anything, doc.ID()).Return(doc, nil)
	storage.On("Retrieve", mock.Anything, doc.FilePath().String()).
		Return(io.NopCloser(bytes.NewReader([]byte("%PDF"))), nil)
	redactionSvc.On("Redact", mock.Anything, mock.Anything, mock.Anything).
		Return((*ports.RedactionResult)(nil), errors.New("redaction failed"))

	useCase := NewApplyRedactionsUseCase(repo, storage, redactionSvc)

	_, err := useCase.Execute(context.Background(), &dto.ApplyRedactionsRequest{
		DocumentID: doc.ID().String(),
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "redaction failed")
}
