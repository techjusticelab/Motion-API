package persistence

import (
	"context"
	"sync"

	"motion-index-fiber/internal/application/ports"
	"motion-index-fiber/internal/domain/document"
)

// InMemoryDocumentRepository is a simple in-memory implementation of DocumentRepository.
type InMemoryDocumentRepository struct {
	mu   sync.RWMutex
	data map[string]*document.Document
}

// NewInMemoryDocumentRepository creates a new in-memory repository.
func NewInMemoryDocumentRepository() *InMemoryDocumentRepository {
	return &InMemoryDocumentRepository{data: make(map[string]*document.Document)}
}

// Save stores or updates a document aggregate.
func (r *InMemoryDocumentRepository) Save(ctx context.Context, doc *document.Document) error {
	_ = ctx
	r.mu.Lock()
	defer r.mu.Unlock()
	r.data[doc.ID().String()] = doc
	return nil
}

// FindByID retrieves a document by ID.
func (r *InMemoryDocumentRepository) FindByID(ctx context.Context, id document.DocumentID) (*document.Document, error) {
	_ = ctx
	r.mu.RLock()
	defer r.mu.RUnlock()
	if d, ok := r.data[id.String()]; ok {
		return d, nil
	}
	return nil, nil
}

// FindAll returns all documents matching a filter. This in-memory version ignores filters.
func (r *InMemoryDocumentRepository) FindAll(ctx context.Context, filter document.Filter) ([]*document.Document, error) {
	_ = ctx
	_ = filter
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*document.Document, 0, len(r.data))
	for _, d := range r.data {
		out = append(out, d)
	}
	return out, nil
}

// Delete removes a document by ID.
func (r *InMemoryDocumentRepository) Delete(ctx context.Context, id document.DocumentID) error {
	_ = ctx
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.data, id.String())
	return nil
}

// Exists checks whether a document exists.
func (r *InMemoryDocumentRepository) Exists(ctx context.Context, id document.DocumentID) (bool, error) {
	_ = ctx
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.data[id.String()]
	return ok, nil
}

// Count returns the number of documents matching a filter. This in-memory version ignores filters.
func (r *InMemoryDocumentRepository) Count(ctx context.Context, filter document.Filter) (int, error) {
	_ = ctx
	_ = filter
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.data), nil
}

// Ensure interface compliance
var _ ports.DocumentRepository = (*InMemoryDocumentRepository)(nil)
