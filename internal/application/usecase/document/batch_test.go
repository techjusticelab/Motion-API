package document

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"motion-index-fiber/internal/application/dto"
)

func TestBatchProcessUseCase_Success(t *testing.T) {
	processor := new(processUseCaseMock)

	req := &dto.BatchProcessDocumentRequest{
		Documents: []*dto.ProcessDocumentRequest{
			{
				ID:            "doc_1",
				FileName:      "a.pdf",
				ContentType:   "application/pdf",
				StoragePath:   "documents/a.pdf",
				ContentSize:   1,
				HashValue:     "a3c1e5a6b9d2f4a7c8e1f2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2d3",
				HashAlgorithm: "SHA256",
			},
			{
				ID:            "doc_2",
				FileName:      "b.pdf",
				ContentType:   "application/pdf",
				StoragePath:   "documents/b.pdf",
				ContentSize:   1,
				HashValue:     "b3c1e5a6b9d2f4a7c8e1f2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2d3",
				HashAlgorithm: "SHA256",
			},
		},
	}
	require.NoError(t, req.Validate())

	processor.On("Execute", mock.Anything, req.Documents[0]).
		Return(&dto.ProcessDocumentResponse{DocumentID: "doc_1"}, nil)
	processor.On("Execute", mock.Anything, req.Documents[1]).
		Return(&dto.ProcessDocumentResponse{DocumentID: "doc_2"}, nil)

	useCase := NewBatchProcessUseCase(processor)

	resp, err := useCase.Execute(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, 2, resp.Total)
	assert.Equal(t, 2, resp.Succeeded)
	assert.Equal(t, 0, resp.Failed)
	assert.Nil(t, resp.Errors)

	processor.AssertExpectations(t)
}

func TestBatchProcessUseCase_AggregatesFailures(t *testing.T) {
	processor := new(processUseCaseMock)

	req := &dto.BatchProcessDocumentRequest{
		Documents: []*dto.ProcessDocumentRequest{
			{
				ID:            "doc_ok",
				FileName:      "ok.pdf",
				ContentType:   "application/pdf",
				StoragePath:   "documents/ok.pdf",
				ContentSize:   1,
				HashValue:     "c3c1e5a6b9d2f4a7c8e1f2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2d3",
				HashAlgorithm: "SHA256",
			},
			{
				ID:            "doc_fail",
				FileName:      "fail.pdf",
				ContentType:   "application/pdf",
				StoragePath:   "documents/fail.pdf",
				ContentSize:   1,
				HashValue:     "d3c1e5a6b9d2f4a7c8e1f2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2d3",
				HashAlgorithm: "SHA256",
			},
		},
	}
	require.NoError(t, req.Validate())

	processor.On("Execute", mock.Anything, req.Documents[0]).
		Return(&dto.ProcessDocumentResponse{DocumentID: "doc_ok"}, nil)
	processor.On("Execute", mock.Anything, req.Documents[1]).
		Return((*dto.ProcessDocumentResponse)(nil), errors.New("classification failed"))

	useCase := NewBatchProcessUseCase(processor)

	resp, err := useCase.Execute(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, 2, resp.Total)
	assert.Equal(t, 1, resp.Succeeded)
	assert.Equal(t, 1, resp.Failed)
	require.NotNil(t, resp.Errors)
	assert.Contains(t, resp.Errors, "doc_fail")
	assert.Equal(t, "classification failed", resp.Errors["doc_fail"])

	processor.AssertExpectations(t)
}

func TestBatchProcessUseCase_ProcessorNotConfigured(t *testing.T) {
	useCase := NewBatchProcessUseCase(nil)

	_, err := useCase.Execute(context.Background(), &dto.BatchProcessDocumentRequest{
		Documents: []*dto.ProcessDocumentRequest{
			{
				ID:            "doc",
				FileName:      "a.pdf",
				ContentType:   "application/pdf",
				StoragePath:   "documents/a.pdf",
				ContentSize:   1,
				HashValue:     "03c1e5a6b9d2f4a7c8e1f2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2d3",
				HashAlgorithm: "SHA256",
			},
		},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "process document use case is not configured")
}

func TestBatchProcessUseCase_InvalidRequest(t *testing.T) {
	useCase := NewBatchProcessUseCase(new(processUseCaseMock))

	_, err := useCase.Execute(context.Background(), &dto.BatchProcessDocumentRequest{})
	require.Error(t, err)
}

func TestBatchProcessUseCase_ContextCancelled(t *testing.T) {
	processor := new(processUseCaseMock)

	req := &dto.BatchProcessDocumentRequest{
		Documents: []*dto.ProcessDocumentRequest{
			{
				ID:            "doc_1",
				FileName:      "a.pdf",
				ContentType:   "application/pdf",
				StoragePath:   "documents/a.pdf",
				ContentSize:   1,
				HashValue:     "e3c1e5a6b9d2f4a7c8e1f2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2d3",
				HashAlgorithm: "SHA256",
			},
			{
				ID:            "doc_2",
				FileName:      "b.pdf",
				ContentType:   "application/pdf",
				StoragePath:   "documents/b.pdf",
				ContentSize:   1,
				HashValue:     "f3c1e5a6b9d2f4a7c8e1f2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2d3",
				HashAlgorithm: "SHA256",
			},
		},
	}
	require.NoError(t, req.Validate())

	ctx, cancel := context.WithCancel(context.Background())

	processor.On("Execute", mock.Anything, req.Documents[0]).
		Run(func(args mock.Arguments) {
			cancel()
		}).
		Return(&dto.ProcessDocumentResponse{DocumentID: "doc_1"}, nil)

	useCase := NewBatchProcessUseCase(processor)

	_, err := useCase.Execute(ctx, req)

	require.Error(t, err)
	assert.True(t, errors.Is(err, context.Canceled))
	processor.AssertNumberOfCalls(t, "Execute", 1)
}
