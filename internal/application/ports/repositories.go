package ports

import (
	"context"

	"motion-index-fiber/internal/domain/document"
	"motion-index-fiber/internal/domain/legal"
)

// DocumentRepository describes persistence operations for the document aggregate.
type DocumentRepository interface {
	Save(ctx context.Context, doc *document.Document) error
	FindByID(ctx context.Context, id document.DocumentID) (*document.Document, error)
	FindAll(ctx context.Context, filter document.Filter) ([]*document.Document, error)
	Delete(ctx context.Context, id document.DocumentID) error
	Exists(ctx context.Context, id document.DocumentID) (bool, error)
	Count(ctx context.Context, filter document.Filter) (int, error)
}

// CaseRepository exposes persistence for legal case entities.
type CaseRepository interface {
	Save(ctx context.Context, c *legal.Case) error
	FindByNumber(ctx context.Context, number legal.CaseNumber) (*legal.Case, error)
}
