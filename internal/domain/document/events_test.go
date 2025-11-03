package document

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func buildTestDocument(t *testing.T) *Document {
	t.Helper()
	doc, err := NewDocument(
		WithID("doc_events"),
		WithFileName("events.pdf"),
		WithStoragePath("documents/2024/05/events.pdf"),
		WithHash("23c1e5a6b9d2f4a7c8e1f2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2d0", "SHA256"),
		WithContentType("application/pdf"),
		WithFileSize(2048),
		WithText("Event test text"),
	)
	require.NoError(t, err)
	return doc
}

func TestNewDocumentCreatedEvent(t *testing.T) {
	doc := buildTestDocument(t)
	event := NewDocumentCreatedEvent(doc)

	assert.Equal(t, doc.ID().String(), event.AggregateID())
	assert.Equal(t, "DocumentCreated", event.EventType())
	assert.Equal(t, doc.FileName().String(), event.FileName)
	assert.Equal(t, doc.ContentType().String(), event.ContentType)
	assert.False(t, event.OccurredAt().IsZero())
}

func TestNewDocumentClassifiedEvent(t *testing.T) {
	doc := buildTestDocument(t)
	documentType, _ := NewDocumentType("motion")
	category, _ := NewCategory("filing")
	confidence, _ := NewConfidence(0.9)
	classification, _ := NewClassification(documentType, category, confidence, "classifier", []string{"motion"})

	event := NewDocumentClassifiedEvent(doc, classification)

	assert.Equal(t, doc.ID().String(), event.AggregateID())
	assert.Equal(t, "DocumentClassified", event.EventType())
	assert.Equal(t, "motion", event.DocumentType)
	assert.Equal(t, "filing", event.Category)
	assert.InDelta(t, 0.9, event.Confidence, 0.0001)
	assert.Equal(t, "classifier", event.ClassifiedBy)
	assert.False(t, event.OccurredAt().IsZero())
}

func TestNewDocumentProcessedEvent(t *testing.T) {
	doc := buildTestDocument(t)
	now := time.Now()
	doc.processedAt = &now

	event := NewDocumentProcessedEvent(doc)

	assert.Equal(t, doc.ID().String(), event.AggregateID())
	assert.Equal(t, "DocumentProcessed", event.EventType())
	assert.Equal(t, now, event.ProcessedAt)
	assert.False(t, event.OccurredAt().IsZero())
}

func TestNewMetadataUpdatedEvent(t *testing.T) {
	doc := buildTestDocument(t)
	doc.version = 3
	doc.updatedAt = time.Now()

	event := NewMetadataUpdatedEvent(doc)

	assert.Equal(t, doc.ID().String(), event.AggregateID())
	assert.Equal(t, "MetadataUpdated", event.EventType())
	assert.Equal(t, 3, event.Version)
	assert.Equal(t, doc.UpdatedAt(), event.UpdatedAt)
	assert.False(t, event.OccurredAt().IsZero())
}
