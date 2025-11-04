package ai

import (
	"context"

	"motion-index-fiber/internal/application/ports"
	"motion-index-fiber/internal/domain/classification"
	"motion-index-fiber/internal/domain/document"
	"motion-index-fiber/pkg/processing/classifier"
)

// ClassificationServiceAdapter adapts a classifier.Service to ports.ClassificationService.
type ClassificationServiceAdapter struct {
	svc      classifier.Service
	provider string
}

func NewClassificationServiceAdapter(svc classifier.Service, provider string) *ClassificationServiceAdapter {
	return &ClassificationServiceAdapter{svc: svc, provider: provider}
}

func (a *ClassificationServiceAdapter) Classify(ctx context.Context, doc *document.Document) (*classification.Result, error) {
	text := doc.Text().String()
	meta := &classifier.DocumentMetadata{
		FileName:  doc.FileName().String(),
		FileType:  doc.ContentType().String(),
		Size:      doc.Size().Bytes(),
		WordCount: len(text) / 5,
	}
	res, err := a.svc.ClassifyDocument(ctx, text, meta)
	if err != nil {
		return nil, err
	}
	return convertToDomainResult(res, a.provider)
}

func (a *ClassificationServiceAdapter) SupportedDocumentTypes() []string {
	return a.svc.GetAvailableCategories()
}
func (a *ClassificationServiceAdapter) ProviderName() string { return a.provider }

var _ ports.ClassificationService = (*ClassificationServiceAdapter)(nil)
