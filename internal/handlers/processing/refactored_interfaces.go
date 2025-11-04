package processing

import (
	"context"
	"motion-index-fiber/internal/application/dto"
)

type ProcessDocumentExecutor interface {
	Execute(ctx context.Context, req *dto.ProcessDocumentRequest) (*dto.ProcessDocumentResponse, error)
}

type BatchProcessExecutor interface {
	Execute(ctx context.Context, req *dto.BatchProcessDocumentRequest) (*dto.BatchProcessDocumentResponse, error)
}

type UpdateMetadataExecutor interface {
	Execute(ctx context.Context, req *dto.UpdateMetadataRequest) (*dto.UpdateMetadataResponse, error)
}

type AnalyzeRedactionsExecutor interface {
	Execute(ctx context.Context, req *dto.AnalyzeRedactionsRequest) (*dto.AnalyzeRedactionsResponse, error)
}

type ApplyRedactionsExecutor interface {
	Execute(ctx context.Context, req *dto.ApplyRedactionsRequest) (*dto.ApplyRedactionsResponse, error)
}
