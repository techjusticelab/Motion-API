package redaction

import (
	"context"
	"io"

	"github.com/stretchr/testify/mock"

	"motion-index-fiber/internal/application/ports"
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

type redactionServiceMock struct{ mock.Mock }

func (m *redactionServiceMock) Analyze(ctx context.Context, reader io.Reader, options *ports.RedactionOptions) (*ports.RedactionAnalysis, error) {
	args := m.Called(ctx, reader, options)
	if res, ok := args.Get(0).(*ports.RedactionAnalysis); ok {
		return res, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *redactionServiceMock) Redact(ctx context.Context, reader io.Reader, options *ports.RedactionOptions) (*ports.RedactionResult, error) {
	args := m.Called(ctx, reader, options)
	if res, ok := args.Get(0).(*ports.RedactionResult); ok {
		return res, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *redactionServiceMock) ApplyCustom(ctx context.Context, reader io.Reader, options *ports.RedactionOptions, redactions []ports.RedactionItem) (*ports.RedactionResult, error) {
	args := m.Called(ctx, reader, options, redactions)
	if res, ok := args.Get(0).(*ports.RedactionResult); ok {
		return res, args.Error(1)
	}
	return nil, args.Error(1)
}
