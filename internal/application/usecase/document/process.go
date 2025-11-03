package document

import (
	"context"
	"fmt"
	"io"
	"strings"

	"motion-index-fiber/internal/application/dto"
	"motion-index-fiber/internal/application/ports"
	"motion-index-fiber/internal/domain/document"
)

// ProcessDocumentUseCase orchestrates ingestion of a new document.
type ProcessDocumentUseCase struct {
	repo              ports.DocumentRepository
	storage           ports.StorageService
	classifier        ports.ClassificationService
	search            ports.SearchService
	eventBus          ports.EventBus
	classifyByDefault bool
	indexByDefault    bool
}

// NewProcessDocumentUseCase wires dependencies for the use case.
func NewProcessDocumentUseCase(
	repo ports.DocumentRepository,
	storage ports.StorageService,
	classifier ports.ClassificationService,
	search ports.SearchService,
	eventBus ports.EventBus,
) *ProcessDocumentUseCase {
	return &ProcessDocumentUseCase{
		repo:       repo,
		storage:    storage,
		classifier: classifier,
		search:     search,
		eventBus:   eventBus,
	}
}

// Execute ingests the document, optionally storing, classifying, and indexing it.
func (uc *ProcessDocumentUseCase) Execute(ctx context.Context, req *dto.ProcessDocumentRequest) (*dto.ProcessDocumentResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	doc, err := uc.buildAggregate(req)
	if err != nil {
		return nil, err
	}

	var storageURL string
	if req.StoreBinary {
		if uc.storage == nil {
			return nil, fmt.Errorf("storage service is not configured")
		}
		storageURL, err = uc.storeContent(ctx, req.StoragePath, req.Content)
		if err != nil {
			return nil, err
		}
	}

	if req.Classify || uc.classifyByDefault {
		if err := uc.classifyDocument(ctx, doc); err != nil {
			return nil, err
		}
	}

	if err := uc.repo.Save(ctx, doc); err != nil {
		return nil, err
	}

	indexed := req.Index || uc.indexByDefault
	if req.Index || uc.indexByDefault {
		if err := uc.indexDocument(ctx, doc); err != nil {
			return nil, err
		}
	}

	eventCount := len(doc.GetEvents())
	if uc.eventBus != nil {
		if err := uc.eventBus.Publish(ctx, doc.GetEvents()...); err != nil {
			return nil, err
		}
		doc.ClearEvents()
	}

	return &dto.ProcessDocumentResponse{
		DocumentID:   doc.ID().String(),
		Stored:       storageURL != "",
		StorageURL:   storageURL,
		Classified:   doc.HasClassification(),
		Indexed:      indexed,
		CreatedAt:    doc.CreatedAt(),
		ProcessedAt:  doc.ProcessedAt(),
		EventsQueued: eventCount,
	}, nil
}

func (uc *ProcessDocumentUseCase) buildAggregate(req *dto.ProcessDocumentRequest) (*document.Document, error) {
	options := []document.DocumentOption{
		document.WithID(req.ID),
		document.WithFileName(req.FileName),
		document.WithHash(req.HashValue, req.HashAlgorithm),
	}

	if req.StoragePath != "" {
		options = append(options, document.WithStoragePath(req.StoragePath))
	}

	if req.ContentType != "" {
		options = append(options, document.WithContentType(req.ContentType))
	}

	if req.ContentSize > 0 {
		options = append(options, document.WithFileSize(req.ContentSize))
	}

	if strings.TrimSpace(req.Text) != "" {
		options = append(options, document.WithText(req.Text))
	}

	return document.NewDocument(options...)
}

func (uc *ProcessDocumentUseCase) storeContent(ctx context.Context, path string, reader io.Reader) (string, error) {
	url, err := uc.storage.Store(ctx, path, reader)
	if err != nil {
		return "", err
	}
	return url, nil
}

func (uc *ProcessDocumentUseCase) classifyDocument(ctx context.Context, doc *document.Document) error {
	if uc.classifier == nil {
		return fmt.Errorf("classifier service is not configured")
	}
	result, err := uc.classifier.Classify(ctx, doc)
	if err != nil {
		return err
	}
	classification, err := result.ToDocumentClassification()
	if err != nil {
		return err
	}
	return doc.ApplyClassification(classification)
}

func (uc *ProcessDocumentUseCase) indexDocument(ctx context.Context, doc *document.Document) error {
	if uc.search == nil {
		return nil
	}
	return uc.search.Index(ctx, doc)
}
