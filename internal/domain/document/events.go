package document

import "time"

// DomainEvent represents a domain significant occurrence.
type DomainEvent interface {
	OccurredAt() time.Time
	AggregateID() string
	EventType() string
}

// DocumentCreatedEvent is emitted when a document aggregate is created.
type DocumentCreatedEvent struct {
	DocumentID  string
	FileName    string
	ContentType string
	occurredAt  time.Time
}

// NewDocumentCreatedEvent constructs a creation event.
func NewDocumentCreatedEvent(doc *Document) *DocumentCreatedEvent {
	return &DocumentCreatedEvent{
		DocumentID:  doc.id.String(),
		FileName:    doc.fileName.String(),
		ContentType: doc.contentType.String(),
		occurredAt:  time.Now(),
	}
}

// OccurredAt returns when the event occurred.
func (e *DocumentCreatedEvent) OccurredAt() time.Time {
	return e.occurredAt
}

// AggregateID returns the document identifier.
func (e *DocumentCreatedEvent) AggregateID() string {
	return e.DocumentID
}

// EventType returns the event type name.
func (e *DocumentCreatedEvent) EventType() string {
	return "DocumentCreated"
}

// DocumentClassifiedEvent is emitted when a document is classified.
type DocumentClassifiedEvent struct {
	DocumentID   string
	DocumentType string
	Category     string
	Confidence   float64
	ClassifiedBy string
	occurredAt   time.Time
}

// NewDocumentClassifiedEvent constructs a classification event.
func NewDocumentClassifiedEvent(doc *Document, classification *Classification) *DocumentClassifiedEvent {
	return &DocumentClassifiedEvent{
		DocumentID:   doc.id.String(),
		DocumentType: classification.documentType.String(),
		Category:     classification.category.String(),
		Confidence:   classification.confidence.Value(),
		ClassifiedBy: classification.classifiedBy,
		occurredAt:   time.Now(),
	}
}

// OccurredAt returns when the event occurred.
func (e *DocumentClassifiedEvent) OccurredAt() time.Time {
	return e.occurredAt
}

// AggregateID returns the document identifier.
func (e *DocumentClassifiedEvent) AggregateID() string {
	return e.DocumentID
}

// EventType returns the event type name.
func (e *DocumentClassifiedEvent) EventType() string {
	return "DocumentClassified"
}

// DocumentProcessedEvent is emitted when a document finishes processing.
type DocumentProcessedEvent struct {
	DocumentID  string
	ProcessedAt time.Time
	occurredAt  time.Time
}

// NewDocumentProcessedEvent constructs a processed event.
func NewDocumentProcessedEvent(doc *Document) *DocumentProcessedEvent {
	processed := time.Now()
	if doc.processedAt != nil {
		processed = *doc.processedAt
	}
	return &DocumentProcessedEvent{
		DocumentID:  doc.id.String(),
		ProcessedAt: processed,
		occurredAt:  time.Now(),
	}
}

// OccurredAt returns when the event occurred.
func (e *DocumentProcessedEvent) OccurredAt() time.Time {
	return e.occurredAt
}

// AggregateID returns the document identifier.
func (e *DocumentProcessedEvent) AggregateID() string {
	return e.DocumentID
}

// EventType returns the event type name.
func (e *DocumentProcessedEvent) EventType() string {
	return "DocumentProcessed"
}

// MetadataUpdatedEvent is emitted when document metadata changes.
type MetadataUpdatedEvent struct {
	DocumentID string
	Version    int
	UpdatedAt  time.Time
	occurredAt time.Time
}

// NewMetadataUpdatedEvent constructs a metadata update event.
func NewMetadataUpdatedEvent(doc *Document) *MetadataUpdatedEvent {
	return &MetadataUpdatedEvent{
		DocumentID: doc.id.String(),
		Version:    doc.version,
		UpdatedAt:  doc.updatedAt,
		occurredAt: time.Now(),
	}
}

// OccurredAt returns when the event occurred.
func (e *MetadataUpdatedEvent) OccurredAt() time.Time {
	return e.occurredAt
}

// AggregateID returns the document identifier.
func (e *MetadataUpdatedEvent) AggregateID() string {
	return e.DocumentID
}

// EventType returns the event type name.
func (e *MetadataUpdatedEvent) EventType() string {
	return "MetadataUpdated"
}
