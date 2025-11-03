package document

import (
	"context"
	"fmt"

	"motion-index-fiber/internal/application/dto"
	"motion-index-fiber/internal/application/ports"
	"motion-index-fiber/internal/domain/document"
)

// ClassifyDocumentUseCase runs classification for an existing document.
type ClassifyDocumentUseCase struct {
	repo       ports.DocumentRepository
	classifier ports.ClassificationService
	eventBus   ports.EventBus
	search     ports.SearchService
}

// NewClassifyDocumentUseCase constructs the use case.
func NewClassifyDocumentUseCase(
	repo ports.DocumentRepository,
	classifier ports.ClassificationService,
	eventBus ports.EventBus,
	search ports.SearchService,
) *ClassifyDocumentUseCase {
	return &ClassifyDocumentUseCase{
		repo:       repo,
		classifier: classifier,
		eventBus:   eventBus,
		search:     search,
	}
}

// Execute performs classification and persists the result.
func (uc *ClassifyDocumentUseCase) Execute(ctx context.Context, req *dto.ClassifyDocumentRequest) (*dto.ClassificationResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, err
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
		return nil, fmt.Errorf("document %s not found", req.DocumentID)
	}

	if doc.HasClassification() && !req.Force {
		return mapClassificationToDTO(req.DocumentID, doc.Classification()), nil
	}

	if uc.classifier == nil {
		return nil, fmt.Errorf("classification service is not configured")
	}

	result, err := uc.classifier.Classify(ctx, doc)
	if err != nil {
		return nil, err
	}

	classification, err := result.ToDocumentClassification()
	if err != nil {
		return nil, err
	}

	if err := doc.ApplyClassification(classification); err != nil {
		return nil, err
	}

	if err := uc.repo.Save(ctx, doc); err != nil {
		return nil, err
	}

	if uc.search != nil {
		_ = uc.search.Index(ctx, doc)
	}

	if uc.eventBus != nil {
		if err := uc.eventBus.Publish(ctx, doc.GetEvents()...); err != nil {
			return nil, err
		}
		doc.ClearEvents()
	}

	return mapClassificationToDTO(req.DocumentID, classification), nil
}

func mapClassificationToDTO(documentID string, classification *document.Classification) *dto.ClassificationResponse {
	if classification == nil {
		return &dto.ClassificationResponse{DocumentID: documentID}
	}
	return &dto.ClassificationResponse{
		DocumentID:   documentID,
		DocumentType: classification.DocumentType().String(),
		Category:     classification.Category().String(),
		Confidence:   classification.Confidence().Value(),
		LegalTags:    classification.LegalTags(),
		ClassifiedBy: classification.ClassifiedBy(),
		ClassifiedAt: classification.ClassifiedAt(),
	}
}
