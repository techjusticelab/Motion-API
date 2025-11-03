package document

import (
	"context"
	"io"

	"github.com/stretchr/testify/mock"

	"motion-index-fiber/internal/application/dto"
	"motion-index-fiber/internal/application/ports"
	"motion-index-fiber/internal/domain/classification"
	"motion-index-fiber/internal/domain/document"
)

type documentRepositoryMock struct{ mock.Mock }

func (m *documentRepositoryMock) Save(ctx context.Context, doc *document.Document) error {
	args := m.Called(ctx, doc)
	return args.Error(0)
}

func (m *documentRepositoryMock) FindByID(ctx context.Context, id document.DocumentID) (*document.Document, error) {
	args := m.Called(ctx, id)
	if res, ok := args.Get(0).(*document.Document); ok {
		return res, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *documentRepositoryMock) FindAll(ctx context.Context, filter document.Filter) ([]*document.Document, error) {
	args := m.Called(ctx, filter)
	if res, ok := args.Get(0).([]*document.Document); ok {
		return res, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *documentRepositoryMock) Delete(ctx context.Context, id document.DocumentID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *documentRepositoryMock) Exists(ctx context.Context, id document.DocumentID) (bool, error) {
	args := m.Called(ctx, id)
	return args.Bool(0), args.Error(1)
}

func (m *documentRepositoryMock) Count(ctx context.Context, filter document.Filter) (int, error) {
	args := m.Called(ctx, filter)
	return args.Int(0), args.Error(1)
}

type storageServiceMock struct{ mock.Mock }

func (m *storageServiceMock) Store(ctx context.Context, path string, content io.Reader) (string, error) {
	args := m.Called(ctx, path, content)
	return args.String(0), args.Error(1)
}

func (m *storageServiceMock) Retrieve(ctx context.Context, path string) (io.ReadCloser, error) {
	args := m.Called(ctx, path)
	if res, ok := args.Get(0).(io.ReadCloser); ok {
		return res, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *storageServiceMock) Delete(ctx context.Context, path string) error {
	args := m.Called(ctx, path)
	return args.Error(0)
}

func (m *storageServiceMock) URL(path string) string {
	args := m.Called(path)
	return args.String(0)
}

type classificationServiceMock struct{ mock.Mock }

func (m *classificationServiceMock) Classify(ctx context.Context, doc *document.Document) (*classification.Result, error) {
	args := m.Called(ctx, doc)
	if res, ok := args.Get(0).(*classification.Result); ok {
		return res, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *classificationServiceMock) SupportedDocumentTypes() []string {
	args := m.Called()
	if res, ok := args.Get(0).([]string); ok {
		return res
	}
	return nil
}

func (m *classificationServiceMock) ProviderName() string {
	args := m.Called()
	return args.String(0)
}

type searchServiceMock struct{ mock.Mock }

func (m *searchServiceMock) Index(ctx context.Context, doc *document.Document) error {
	args := m.Called(ctx, doc)
	return args.Error(0)
}

func (m *searchServiceMock) Update(ctx context.Context, id string, fields map[string]interface{}) error {
	args := m.Called(ctx, id, fields)
	return args.Error(0)
}

func (m *searchServiceMock) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *searchServiceMock) Search(ctx context.Context, query ports.SearchQuery) (*ports.SearchResults, error) {
	args := m.Called(ctx, query)
	if res, ok := args.Get(0).(*ports.SearchResults); ok {
		return res, args.Error(1)
	}
	return nil, args.Error(1)
}

type eventBusMock struct{ mock.Mock }

func (m *eventBusMock) Publish(ctx context.Context, events ...document.DomainEvent) error {
	args := m.Called(ctx, events)
	return args.Error(0)
}

type processUseCaseMock struct{ mock.Mock }

func (m *processUseCaseMock) Execute(ctx context.Context, req *dto.ProcessDocumentRequest) (*dto.ProcessDocumentResponse, error) {
	args := m.Called(ctx, req)
	if res, ok := args.Get(0).(*dto.ProcessDocumentResponse); ok {
		return res, args.Error(1)
	}
	return nil, args.Error(1)
}
