package document

import (
	"context"
	"fmt"

	"motion-index-fiber/internal/application/dto"
	"motion-index-fiber/internal/application/ports"
	"motion-index-fiber/internal/domain/document"
	domainerrors "motion-index-fiber/internal/domain/errors"
)

// IndexDocumentUseCase reindexes an existing document in the search service.
type IndexDocumentUseCase struct {
	repo   ports.DocumentRepository
	search ports.SearchService
}

// NewIndexDocumentUseCase wires dependencies required to index documents.
func NewIndexDocumentUseCase(
	repo ports.DocumentRepository,
	search ports.SearchService,
) *IndexDocumentUseCase {
	return &IndexDocumentUseCase{
		repo:   repo,
		search: search,
	}
}

// Execute retrieves the document aggregate and forwards it to the search service.
func (uc *IndexDocumentUseCase) Execute(ctx context.Context, req *dto.IndexDocumentRequest) (*dto.IndexDocumentResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	if uc.repo == nil {
		return nil, fmt.Errorf("document repository is not configured")
	}
	if uc.search == nil {
		return nil, fmt.Errorf("search service is not configured")
	}

	docID, err := document.NewDocumentID(req.DocumentID)
	if err != nil {
		return nil, err
	}

	doc, err := uc.repo.FindByID(ctx, docID)
	if err != nil {
		return nil, err
	}
	if doc == nil {
		return nil, domainerrors.ErrDocumentNotFound
	}

	if err := uc.search.Index(ctx, doc); err != nil {
		return nil, err
	}

	return &dto.IndexDocumentResponse{
		DocumentID: doc.ID().String(),
		Indexed:    true,
	}, nil
}
