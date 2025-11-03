package document

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"motion-index-fiber/internal/application/dto"
	"motion-index-fiber/internal/domain/classification"
	"motion-index-fiber/internal/domain/document"
)

type classifierMock struct{ mock.Mock }

func (m *classifierMock) Classify(ctx context.Context, doc *document.Document) (*classification.Result, error) {
	args := m.Called(ctx, doc)
	if res, ok := args.Get(0).(*classification.Result); ok {
		return res, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *classifierMock) SupportedDocumentTypes() []string {
	args := m.Called()
	if res, ok := args.Get(0).([]string); ok {
		return res
	}
	return nil
}

func (m *classifierMock) ProviderName() string {
	args := m.Called()
	return args.String(0)
}

type eventBusNoop struct{}

func (eventBusNoop) Publish(ctx context.Context, events ...document.DomainEvent) error { return nil }

func TestClassifyDocumentUseCase_Execute_Success(t *testing.T) {
	repo := new(documentRepositoryMock)
	classifier := new(classifierMock)
	search := new(searchServiceMock)

	doc, _ := document.NewDocument(
		document.WithID("doc_123"),
		document.WithFileName("example.pdf"),
		document.WithStoragePath("documents/example.pdf"),
		document.WithHash("a3c1e5a6b9d2f4a7c8e1f2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2d3", "SHA256"),
		document.WithContentType("application/pdf"),
		document.WithFileSize(1024),
		document.WithText("content"),
	)

	repo.On("FindByID", mock.Anything, mock.AnythingOfType("document.DocumentID")).Return(doc, nil)
	repo.On("Save", mock.Anything, mock.Anything).Return(nil)
	search.On("Index", mock.Anything, mock.Anything).Return(nil)

	docType, _ := document.NewDocumentType("motion")
	category, _ := document.NewCategory("filing")
	confidence, _ := document.NewConfidence(0.9)
	result, _ := classification.NewResult(docType, category, confidence, "classifier", []string{"motion"}, "")

	classifier.On("Classify", mock.Anything, doc).Return(result, nil)

	useCase := NewClassifyDocumentUseCase(repo, classifier, eventBusNoop{}, search)

	resp, err := useCase.Execute(context.Background(), &dto.ClassifyDocumentRequest{DocumentID: "doc_123"})
	assert.NoError(t, err)
	assert.Equal(t, "motion", resp.DocumentType)
	assert.Equal(t, "filing", resp.Category)
	assert.InDelta(t, 0.9, resp.Confidence, 0.001)

	repo.AssertExpectations(t)
	classifier.AssertExpectations(t)
	search.AssertExpectations(t)
}

func TestClassifyDocumentUseCase_Execute_DocumentNotFound(t *testing.T) {
	repo := new(documentRepositoryMock)
	repo.On("FindByID", mock.Anything, mock.AnythingOfType("document.DocumentID")).Return(nil, nil)

	useCase := NewClassifyDocumentUseCase(repo, nil, nil, nil)
	resp, err := useCase.Execute(context.Background(), &dto.ClassifyDocumentRequest{DocumentID: "missing"})
	assert.Error(t, err)
	assert.Nil(t, resp)
}

func TestClassifyDocumentUseCase_Execute_RepositoryError(t *testing.T) {
	repo := new(documentRepositoryMock)
	repo.On("FindByID", mock.Anything, mock.AnythingOfType("document.DocumentID")).Return(nil, errors.New("fail"))

	useCase := NewClassifyDocumentUseCase(repo, nil, nil, nil)
	resp, err := useCase.Execute(context.Background(), &dto.ClassifyDocumentRequest{DocumentID: "doc_123"})
	assert.Error(t, err)
	assert.Nil(t, resp)
}

func TestClassifyDocumentUseCase_Execute_ClassifierMissing(t *testing.T) {
	repo := new(documentRepositoryMock)

	doc, _ := document.NewDocument(
		document.WithID("doc_123"),
		document.WithFileName("example.pdf"),
		document.WithStoragePath("documents/example.pdf"),
		document.WithHash("a3c1e5a6b9d2f4a7c8e1f2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2d3", "SHA256"),
		document.WithContentType("application/pdf"),
		document.WithFileSize(1024),
		document.WithText("content"),
	)
	repo.On("FindByID", mock.Anything, mock.AnythingOfType("document.DocumentID")).Return(doc, nil)

	useCase := NewClassifyDocumentUseCase(repo, nil, nil, nil)
	resp, err := useCase.Execute(context.Background(), &dto.ClassifyDocumentRequest{DocumentID: "doc_123"})
	assert.Error(t, err)
	assert.Nil(t, resp)
}

func TestClassifyDocumentUseCase_Execute_UsesExistingClassification(t *testing.T) {
	repo := new(documentRepositoryMock)

	docType, _ := document.NewDocumentType("motion")
	category, _ := document.NewCategory("filing")
	confidence, _ := document.NewConfidence(0.8)
	classification, _ := document.NewClassification(docType, category, confidence, "classifier", []string{"motion"})

	doc, _ := document.NewDocument(
		document.WithID("doc_123"),
		document.WithFileName("example.pdf"),
		document.WithStoragePath("documents/example.pdf"),
		document.WithHash("a3c1e5a6b9d2f4a7c8e1f2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2d3", "SHA256"),
		document.WithContentType("application/pdf"),
		document.WithFileSize(1024),
		document.WithText("content"),
		document.WithClassification(classification),
	)

	repo.On("FindByID", mock.Anything, mock.AnythingOfType("document.DocumentID")).Return(doc, nil)

	useCase := NewClassifyDocumentUseCase(repo, nil, nil, nil)
	resp, err := useCase.Execute(context.Background(), &dto.ClassifyDocumentRequest{DocumentID: "doc_123"})
	assert.NoError(t, err)
	assert.InDelta(t, 0.8, resp.Confidence, 0.001)
}

func TestClassifyDocumentUseCase_Execute_ForceReclassification(t *testing.T) {
	repo := new(documentRepositoryMock)
	classifier := new(classifierMock)

	docType, _ := document.NewDocumentType("motion")
	category, _ := document.NewCategory("filing")
	confidence, _ := document.NewConfidence(0.8)
	existing, _ := document.NewClassification(docType, category, confidence, "classifier", []string{"motion"})

	doc, _ := document.NewDocument(
		document.WithID("doc_123"),
		document.WithFileName("example.pdf"),
		document.WithStoragePath("documents/example.pdf"),
		document.WithHash("a3c1e5a6b9d2f4a7c8e1f2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2d3", "SHA256"),
		document.WithContentType("application/pdf"),
		document.WithFileSize(1024),
		document.WithText("content"),
		document.WithClassification(existing),
	)

	repo.On("FindByID", mock.Anything, mock.AnythingOfType("document.DocumentID")).Return(doc, nil)
	repo.On("Save", mock.Anything, mock.Anything).Return(nil)

	newConfidence, _ := document.NewConfidence(0.95)
	result, _ := classification.NewResult(docType, category, newConfidence, "classifier", []string{"motion"}, "")
	classifier.On("Classify", mock.Anything, doc).Return(result, nil)

	useCase := NewClassifyDocumentUseCase(repo, classifier, eventBusNoop{}, nil)
	resp, err := useCase.Execute(context.Background(), &dto.ClassifyDocumentRequest{DocumentID: "doc_123", Force: true})
	assert.NoError(t, err)
	assert.InDelta(t, 0.95, resp.Confidence, 0.001)

	repo.AssertExpectations(t)
	classifier.AssertExpectations(t)
}

func TestClassifyDocumentUseCase_Execute_ClassifierError(t *testing.T) {
	repo := new(documentRepositoryMock)
	classifier := new(classifierMock)

	doc, _ := document.NewDocument(
		document.WithID("doc_123"),
		document.WithFileName("example.pdf"),
		document.WithStoragePath("documents/example.pdf"),
		document.WithHash("a3c1e5a6b9d2f4a7c8e1f2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2d3", "SHA256"),
		document.WithContentType("application/pdf"),
		document.WithFileSize(1024),
		document.WithText("content"),
	)
	repo.On("FindByID", mock.Anything, mock.AnythingOfType("document.DocumentID")).Return(doc, nil)
	classifier.On("Classify", mock.Anything, doc).Return(nil, errors.New("fail"))

	useCase := NewClassifyDocumentUseCase(repo, classifier, nil, nil)
	resp, err := useCase.Execute(context.Background(), &dto.ClassifyDocumentRequest{DocumentID: "doc_123", Force: true})
	assert.Error(t, err)
	assert.Nil(t, resp)
}

func TestClassifyDocumentUseCase_Execute_SaveError(t *testing.T) {
	repo := new(documentRepositoryMock)
	classifier := new(classifierMock)

	doc, _ := document.NewDocument(
		document.WithID("doc_123"),
		document.WithFileName("example.pdf"),
		document.WithStoragePath("documents/example.pdf"),
		document.WithHash("a3c1e5a6b9d2f4a7c8e1f2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2d3", "SHA256"),
		document.WithContentType("application/pdf"),
		document.WithFileSize(1024),
		document.WithText("content"),
	)

	repo.On("FindByID", mock.Anything, mock.AnythingOfType("document.DocumentID")).Return(doc, nil)
	repo.On("Save", mock.Anything, mock.Anything).Return(errors.New("save failed"))

	docType, _ := document.NewDocumentType("motion")
	category, _ := document.NewCategory("filing")
	confidence, _ := document.NewConfidence(0.9)
	result, _ := classification.NewResult(docType, category, confidence, "classifier", []string{"motion"}, "")
	classifier.On("Classify", mock.Anything, doc).Return(result, nil)

	useCase := NewClassifyDocumentUseCase(repo, classifier, nil, nil)
	resp, err := useCase.Execute(context.Background(), &dto.ClassifyDocumentRequest{DocumentID: "doc_123", Force: true})
	assert.Error(t, err)
	assert.Nil(t, resp)
}

func TestClassifyDocumentUseCase_Execute_EventBusError(t *testing.T) {
	repo := new(documentRepositoryMock)
	classifier := new(classifierMock)
	eventBus := new(eventBusMock)

	doc, _ := document.NewDocument(
		document.WithID("doc_123"),
		document.WithFileName("example.pdf"),
		document.WithStoragePath("documents/example.pdf"),
		document.WithHash("a3c1e5a6b9d2f4a7c8e1f2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2d3", "SHA256"),
		document.WithContentType("application/pdf"),
		document.WithFileSize(1024),
		document.WithText("content"),
	)

	repo.On("FindByID", mock.Anything, mock.AnythingOfType("document.DocumentID")).Return(doc, nil)
	repo.On("Save", mock.Anything, mock.Anything).Return(nil)
	eventBus.On("Publish", mock.Anything, mock.Anything).Return(errors.New("bus"))

	docType, _ := document.NewDocumentType("motion")
	category, _ := document.NewCategory("filing")
	confidence, _ := document.NewConfidence(0.9)
	result, _ := classification.NewResult(docType, category, confidence, "classifier", []string{"motion"}, "")
	classifier.On("Classify", mock.Anything, doc).Return(result, nil)

	useCase := NewClassifyDocumentUseCase(repo, classifier, eventBus, nil)
	resp, err := useCase.Execute(context.Background(), &dto.ClassifyDocumentRequest{DocumentID: "doc_123"})
	assert.Error(t, err)
	assert.Nil(t, resp)
}

func TestMapClassificationToDTONil(t *testing.T) {
	dto := mapClassificationToDTO("doc_123", nil)
	assert.Equal(t, "doc_123", dto.DocumentID)
	assert.Equal(t, 0.0, dto.Confidence)
}
