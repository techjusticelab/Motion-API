package document

import (
	"context"
	"fmt"

	"motion-index-fiber/internal/application/dto"
)

// ProcessDocumentExecutor defines the behaviour required from the process use case.
type ProcessDocumentExecutor interface {
	Execute(ctx context.Context, req *dto.ProcessDocumentRequest) (*dto.ProcessDocumentResponse, error)
}

// BatchProcessUseCase orchestrates processing multiple documents sequentially.
type BatchProcessUseCase struct {
	processor ProcessDocumentExecutor
}

// NewBatchProcessUseCase constructs a batch processor from the single document use case.
func NewBatchProcessUseCase(processor ProcessDocumentExecutor) *BatchProcessUseCase {
	return &BatchProcessUseCase{processor: processor}
}

// Execute processes each document request, aggregating successes and failures.
func (uc *BatchProcessUseCase) Execute(ctx context.Context, req *dto.BatchProcessDocumentRequest) (*dto.BatchProcessDocumentResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	if uc.processor == nil {
		return nil, fmt.Errorf("process document use case is not configured")
	}

	total := len(req.Documents)
	successes := 0
	errorsMap := make(map[string]string)

	for idx, documentReq := range req.Documents {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		_, err := uc.processor.Execute(ctx, documentReq)
		if err != nil {
			key := documentReq.ID
			if key == "" {
				key = fmt.Sprintf("documents[%d]", idx)
			}
			errorsMap[key] = err.Error()
			continue
		}

		successes++
	}

	var errorReport map[string]string
	if len(errorsMap) > 0 {
		errorReport = errorsMap
	}

	failed := total - successes

	return &dto.BatchProcessDocumentResponse{
		Total:     total,
		Succeeded: successes,
		Failed:    failed,
		Errors:    errorReport,
	}, nil
}
