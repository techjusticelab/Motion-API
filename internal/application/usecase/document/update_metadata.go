package document

import (
	"context"
	"fmt"

	"motion-index-fiber/internal/application/dto"
	"motion-index-fiber/internal/application/ports"
	"motion-index-fiber/internal/domain/document"
	domainerrors "motion-index-fiber/internal/domain/errors"
)

// UpdateMetadataUseCase updates metadata for an existing document aggregate.
type UpdateMetadataUseCase struct {
	repo     ports.DocumentRepository
	search   ports.SearchService
	eventBus ports.EventBus
}

// NewUpdateMetadataUseCase wires dependencies required for metadata updates.
func NewUpdateMetadataUseCase(
	repo ports.DocumentRepository,
	search ports.SearchService,
	eventBus ports.EventBus,
) *UpdateMetadataUseCase {
	return &UpdateMetadataUseCase{
		repo:     repo,
		search:   search,
		eventBus: eventBus,
	}
}

// Execute validates the request, updates metadata, persists, and propagates changes.
func (uc *UpdateMetadataUseCase) Execute(ctx context.Context, req *dto.UpdateMetadataRequest) (*dto.UpdateMetadataResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	if uc.repo == nil {
		return nil, fmt.Errorf("document repository is not configured")
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

	metadata, err := document.NewMetadata(req.Language, req.LegalTags)
	if err != nil {
		return nil, err
	}
	metadata.SetAIClassified(req.AIClassified)
	if req.ProcessedAt != nil {
		metadata.SetProcessedAt(*req.ProcessedAt)
	}

	if err := doc.UpdateMetadata(metadata); err != nil {
		return nil, err
	}

	if err := uc.repo.Save(ctx, doc); err != nil {
		return nil, err
	}

	if uc.search != nil {
		if err := uc.search.Update(ctx, doc.ID().String(), buildMetadataUpdateFields(metadata)); err != nil {
			return nil, err
		}
	}

	events := doc.GetEvents()
	if uc.eventBus != nil && len(events) > 0 {
		if err := uc.eventBus.Publish(ctx, events...); err != nil {
			return nil, err
		}
		doc.ClearEvents()
	}

	return &dto.UpdateMetadataResponse{
		DocumentID:   doc.ID().String(),
		Language:     metadata.Language(),
		LegalTags:    metadata.LegalTags(),
		AIClassified: metadata.AIClassified(),
		ProcessedAt:  metadata.ProcessedAt(),
	}, nil
}

func buildMetadataUpdateFields(metadata *document.Metadata) map[string]interface{} {
	fields := map[string]interface{}{
		"metadata": map[string]interface{}{
			"language":     metadata.Language(),
			"legalTags":    metadata.LegalTags(),
			"aiClassified": metadata.AIClassified(),
		},
	}

	if processedAt := metadata.ProcessedAt(); processedAt != nil {
		fields["metadata"].(map[string]interface{})["processedAt"] = processedAt
	}

	return fields
}
