package document

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"motion-index-fiber/internal/application/dto"
	"motion-index-fiber/internal/domain/classification"
	"motion-index-fiber/internal/domain/document"
)

func TestProcessDocumentUseCase_Execute_Success(t *testing.T) {
	ctx := context.Background()

	repo := new(documentRepositoryMock)
	storage := new(storageServiceMock)
	eventBus := new(eventBusMock)

	req := &dto.ProcessDocumentRequest{
		ID:            "doc_123",
		FileName:      "example.pdf",
		Content:       bytes.NewReader([]byte("content")),
		ContentSize:   7,
		ContentType:   "application/pdf",
		StoragePath:   "documents/2024/05/example.pdf",
		HashValue:     "a3c1e5a6b9d2f4a7c8e1f2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2d3",
		HashAlgorithm: "SHA256",
		StoreBinary:   true,
		Classify:      false,
		Index:         false,
	}

	storage.On("Store", mock.Anything, req.StoragePath, mock.Anything).Return("https://storage/doc.pdf", nil)
	repo.On("Save", mock.Anything, mock.AnythingOfType("*document.Document")).Return(nil)
	eventBus.On("Publish", mock.Anything, mock.Anything).Return(nil)

	useCase := NewProcessDocumentUseCase(repo, storage, nil, nil, eventBus)

	resp, err := useCase.Execute(ctx, req)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "doc_123", resp.DocumentID)
	assert.True(t, resp.Stored)

	storage.AssertExpectations(t)
	repo.AssertExpectations(t)
	eventBus.AssertExpectations(t)
}

func TestProcessDocumentUseCase_ClassifyAndIndex(t *testing.T) {
	ctx := context.Background()

	repo := new(documentRepositoryMock)
	classifier := new(classificationServiceMock)
	search := new(searchServiceMock)
	eventBus := new(eventBusMock)

	req := &dto.ProcessDocumentRequest{
		ID:            "doc_123",
		FileName:      "example.pdf",
		ContentType:   "application/pdf",
		StoragePath:   "documents/2024/05/example.pdf",
		ContentSize:   7,
		Text:          "content",
		HashValue:     "a3c1e5a6b9d2f4a7c8e1f2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2d3",
		HashAlgorithm: "SHA256",
		Classify:      true,
		Index:         true,
	}

	repo.On("Save", mock.Anything, mock.AnythingOfType("*document.Document")).Return(nil)
	search.On("Index", mock.Anything, mock.AnythingOfType("*document.Document")).Return(nil)
	eventBus.On("Publish", mock.Anything, mock.Anything).Return(nil)
	docType, _ := document.NewDocumentType("motion")
	category, _ := document.NewCategory("filing")
	confidence, _ := document.NewConfidence(0.95)
	classResult, _ := classification.NewResult(docType, category, confidence, "classifier", []string{"motion"}, "")
	classifier.On("Classify", mock.Anything, mock.AnythingOfType("*document.Document")).Return(classResult, nil)

	useCase := NewProcessDocumentUseCase(repo, nil, classifier, search, eventBus)

	resp, err := useCase.Execute(ctx, req)
	assert.NoError(t, err)
	assert.True(t, resp.Classified)
	assert.True(t, resp.Indexed)

	repo.AssertExpectations(t)
	classifier.AssertExpectations(t)
	search.AssertExpectations(t)
	eventBus.AssertExpectations(t)
}

func TestProcessDocumentUseCase_Execute_InvalidRequest(t *testing.T) {
	useCase := NewProcessDocumentUseCase(nil, nil, nil, nil, nil)
	resp, err := useCase.Execute(context.Background(), &dto.ProcessDocumentRequest{})
	assert.Error(t, err)
	assert.Nil(t, resp)
}

func TestProcessDocumentUseCase_Execute_StorageFailure(t *testing.T) {
	req := &dto.ProcessDocumentRequest{
		ID:            "doc_123",
		FileName:      "example.pdf",
		Content:       bytes.NewReader([]byte("content")),
		ContentSize:   7,
		ContentType:   "application/pdf",
		StoragePath:   "documents/2024/05/example.pdf",
		HashValue:     "a3c1e5a6b9d2f4a7c8e1f2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2d3",
		HashAlgorithm: "SHA256",
		StoreBinary:   true,
	}

	storage := new(storageServiceMock)
	storage.On("Store", mock.Anything, req.StoragePath, mock.Anything).Return("", errors.New("store failed"))

	useCase := NewProcessDocumentUseCase(new(documentRepositoryMock), storage, nil, nil, nil)
	resp, err := useCase.Execute(context.Background(), req)
	assert.Error(t, err)
	assert.Nil(t, resp)
}

func TestProcessDocumentUseCase_Execute_ClassifyServiceMissing(t *testing.T) {
	req := &dto.ProcessDocumentRequest{
		ID:            "doc_123",
		FileName:      "example.pdf",
		ContentType:   "application/pdf",
		StoragePath:   "documents/example.pdf",
		ContentSize:   1,
		Text:          "content",
		HashValue:     "a3c1e5a6b9d2f4a7c8e1f2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2d3",
		HashAlgorithm: "SHA256",
		Classify:      true,
	}

	repo := new(documentRepositoryMock)
	repo.On("Save", mock.Anything, mock.AnythingOfType("*document.Document")).Return(nil)

	useCase := NewProcessDocumentUseCase(repo, nil, nil, nil, nil)
	resp, err := useCase.Execute(context.Background(), req)
	assert.Error(t, err)
	assert.Nil(t, resp)
}

func TestProcessDocumentUseCase_Execute_ClassifierError(t *testing.T) {
	req := &dto.ProcessDocumentRequest{
		ID:            "doc_123",
		FileName:      "example.pdf",
		ContentType:   "application/pdf",
		StoragePath:   "documents/example.pdf",
		ContentSize:   1,
		Text:          "content",
		HashValue:     "a3c1e5a6b9d2f4a7c8e1f2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2d3",
		HashAlgorithm: "SHA256",
		Classify:      true,
	}

	repo := new(documentRepositoryMock)
	classifier := new(classificationServiceMock)
	classifier.On("Classify", mock.Anything, mock.AnythingOfType("*document.Document")).Return(nil, errors.New("classify failed"))

	useCase := NewProcessDocumentUseCase(repo, nil, classifier, nil, nil)
	resp, err := useCase.Execute(context.Background(), req)
	assert.Error(t, err)
	assert.Nil(t, resp)
	classifier.AssertExpectations(t)
}

func TestProcessDocumentUseCase_ClassifyByDefault(t *testing.T) {
	repo := new(documentRepositoryMock)
	classifier := new(classificationServiceMock)

	repo.On("Save", mock.Anything, mock.AnythingOfType("*document.Document")).Return(nil)

	docType, _ := document.NewDocumentType("motion")
	category, _ := document.NewCategory("filing")
	confidence, _ := document.NewConfidence(0.9)
	result, _ := classification.NewResult(docType, category, confidence, "classifier", []string{"motion"}, "")
	classifier.On("Classify", mock.Anything, mock.AnythingOfType("*document.Document")).Return(result, nil)

	useCase := NewProcessDocumentUseCase(repo, nil, classifier, nil, nil)
	useCase.classifyByDefault = true

	req := &dto.ProcessDocumentRequest{
		ID:            "doc_123",
		FileName:      "example.pdf",
		ContentType:   "application/pdf",
		StoragePath:   "documents/example.pdf",
		ContentSize:   1,
		Text:          "content",
		HashValue:     "a3c1e5a6b9d2f4a7c8e1f2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2d3",
		HashAlgorithm: "SHA256",
		Classify:      false,
	}

	resp, err := useCase.Execute(context.Background(), req)
	assert.NoError(t, err)
	assert.True(t, resp.Classified)

	repo.AssertExpectations(t)
	classifier.AssertExpectations(t)
}

func TestProcessDocumentUseCase_EventBusError(t *testing.T) {
	repo := new(documentRepositoryMock)
	eventBus := new(eventBusMock)

	repo.On("Save", mock.Anything, mock.AnythingOfType("*document.Document")).Return(nil)
	eventBus.On("Publish", mock.Anything, mock.Anything).Return(errors.New("bus"))

	useCase := NewProcessDocumentUseCase(repo, nil, nil, nil, eventBus)

	req := &dto.ProcessDocumentRequest{
		ID:            "doc_123",
		FileName:      "example.pdf",
		ContentType:   "application/pdf",
		StoragePath:   "documents/example.pdf",
		ContentSize:   1,
		Text:          "content",
		HashValue:     "a3c1e5a6b9d2f4a7c8e1f2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2d3",
		HashAlgorithm: "SHA256",
	}

	resp, err := useCase.Execute(context.Background(), req)
	assert.Error(t, err)
	assert.Nil(t, resp)
}

func TestProcessDocumentUseCase_IndexWithoutSearchService(t *testing.T) {
	req := &dto.ProcessDocumentRequest{
		ID:            "doc_123",
		FileName:      "example.pdf",
		ContentType:   "application/pdf",
		StoragePath:   "documents/example.pdf",
		ContentSize:   1,
		Text:          "content",
		HashValue:     "a3c1e5a6b9d2f4a7c8e1f2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2d3",
		HashAlgorithm: "SHA256",
		Index:         true,
	}

	repo := new(documentRepositoryMock)
	repo.On("Save", mock.Anything, mock.AnythingOfType("*document.Document")).Return(nil)

	useCase := NewProcessDocumentUseCase(repo, nil, nil, nil, nil)
	resp, err := useCase.Execute(context.Background(), req)
	assert.NoError(t, err)
	assert.True(t, resp.Indexed)
}

func TestProcessDocumentUseCase_Execute_StorageServiceMissing(t *testing.T) {
	req := &dto.ProcessDocumentRequest{
		ID:            "doc_123",
		FileName:      "example.pdf",
		Content:       bytes.NewReader([]byte("content")),
		ContentSize:   7,
		ContentType:   "application/pdf",
		StoragePath:   "documents/example.pdf",
		Text:          "content",
		HashValue:     "a3c1e5a6b9d2f4a7c8e1f2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2d3",
		HashAlgorithm: "SHA256",
		StoreBinary:   true,
	}

	useCase := NewProcessDocumentUseCase(new(documentRepositoryMock), nil, nil, nil, nil)
	resp, err := useCase.Execute(context.Background(), req)
	assert.Error(t, err)
	assert.Nil(t, resp)
}
