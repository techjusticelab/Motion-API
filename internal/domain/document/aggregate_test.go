package document

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domainerrors "motion-index-fiber/internal/domain/errors"
)

func TestNewDocument_Success(t *testing.T) {
	doc, err := NewDocument(
		WithID("doc_123"),
		WithFileName("motion.pdf"),
		WithStoragePath("documents/2024/05/motion.pdf"),
		WithS3URIString("s3://bucket/documents/2024/05/motion.pdf"),
		WithHash("a3c1e5a6b9d2f4a7c8e1f2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2d3", "SHA256"),
		WithContentType("application/pdf"),
		WithFileSize(2048),
		WithText("Motion to suppress evidence"),
		WithPageCount(12),
		WithWordCount(4500),
	)
	require.NoError(t, err)

	assert.Equal(t, "doc_123", doc.ID().String())
	assert.Equal(t, ".pdf", doc.FileName().Extension())
	assert.Equal(t, "documents/2024/05/motion.pdf", doc.FilePath().String())
	assert.Equal(t, "s3://bucket/documents/2024/05/motion.pdf", doc.S3URI().String())
	assert.Equal(t, "application/pdf", doc.ContentType().String())
	assert.Equal(t, int64(2048), doc.Size().Bytes())
	assert.Equal(t, 12, doc.PageCount().Value())
	assert.Equal(t, 4500, doc.WordCount().Value())
	assert.Equal(t, 1, doc.Version())

	events := doc.GetEvents()
	require.Len(t, events, 1)
	assert.IsType(t, &DocumentCreatedEvent{}, events[0])
}

func TestNewDocument_MissingRequiredFields(t *testing.T) {
	baseOpts := []DocumentOption{
		WithFileName("motion.pdf"),
		WithStoragePath("documents/2024/05/motion.pdf"),
		WithHash("a3c1e5a6b9d2f4a7c8e1f2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2d3", "SHA256"),
		WithContentType("application/pdf"),
		WithFileSize(2048),
	}

	cases := []struct {
		name    string
		opts    []DocumentOption
		wantErr error
	}{
		{
			name:    "missing id",
			opts:    baseOpts,
			wantErr: domainerrors.ErrEmptyDocumentID,
		},
		{
			name:    "missing file name",
			opts:    append([]DocumentOption{WithID("doc_1")}, WithStoragePath("documents/2024/05/motion.pdf"), WithHash("a3c1e5a6b9d2f4a7c8e1f2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2d3", "SHA256"), WithContentType("application/pdf"), WithFileSize(2048)),
			wantErr: domainerrors.ErrEmptyFileName,
		},
		{
			name:    "missing file path",
			opts:    append([]DocumentOption{WithID("doc_1"), WithFileName("motion.pdf")}, WithHash("a3c1e5a6b9d2f4a7c8e1f2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2d3", "SHA256"), WithContentType("application/pdf"), WithFileSize(2048)),
			wantErr: domainerrors.ErrEmptyFilePath,
		},
		{
			name:    "missing hash",
			opts:    append([]DocumentOption{WithID("doc_1"), WithFileName("motion.pdf"), WithStoragePath("documents/2024/05/motion.pdf")}, WithContentType("application/pdf"), WithFileSize(2048)),
			wantErr: domainerrors.ErrInvalidHash,
		},
		{
			name:    "missing content type",
			opts:    append([]DocumentOption{WithID("doc_1"), WithFileName("motion.pdf"), WithStoragePath("documents/2024/05/motion.pdf"), WithHash("a3c1e5a6b9d2f4a7c8e1f2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2d3", "SHA256")}, WithFileSize(2048)),
			wantErr: domainerrors.ErrInvalidContentType,
		},
		{
			name:    "missing file size",
			opts:    append([]DocumentOption{WithID("doc_1"), WithFileName("motion.pdf"), WithStoragePath("documents/2024/05/motion.pdf"), WithHash("a3c1e5a6b9d2f4a7c8e1f2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2d3", "SHA256"), WithContentType("application/pdf")}),
			wantErr: domainerrors.ErrInvalidFileSize,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewDocument(tt.opts...)
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestDocument_ApplyClassification_Success(t *testing.T) {
	doc, err := NewDocument(
		WithID("doc_456"),
		WithFileName("order.pdf"),
		WithStoragePath("documents/2024/05/order.pdf"),
		WithHash("b3c1e5a6b9d2f4a7c8e1f2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2d4", "SHA256"),
		WithContentType("application/pdf"),
		WithFileSize(4096),
		WithText("Order granting motion"),
	)
	require.NoError(t, err)

	documentType, err := NewDocumentType("order")
	require.NoError(t, err)
	category, err := NewCategory("decision")
	require.NoError(t, err)
	confidence, err := NewConfidence(0.92)
	require.NoError(t, err)

	classification, err := NewClassification(documentType, category, confidence, "classifier", []string{"order", "decision"})
	require.NoError(t, err)

	err = doc.ApplyClassification(classification)
	require.NoError(t, err)

	assert.True(t, doc.HasClassification())
	assert.Equal(t, classification, doc.Classification())
	assert.Greater(t, doc.Version(), 1)

	events := doc.GetEvents()
	require.Len(t, events, 2)
	assert.IsType(t, &DocumentClassifiedEvent{}, events[1])
}

func TestDocument_ApplyClassification_NoText(t *testing.T) {
	doc, err := NewDocument(
		WithID("doc_789"),
		WithFileName("order.pdf"),
		WithStoragePath("documents/2024/05/order.pdf"),
		WithHash("c3c1e5a6b9d2f4a7c8e1f2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2d5", "SHA256"),
		WithContentType("application/pdf"),
		WithFileSize(1024),
	)
	require.NoError(t, err)

	documentType, _ := NewDocumentType("order")
	category, _ := NewCategory("decision")
	confidence, _ := NewConfidence(0.92)
	classification, _ := NewClassification(documentType, category, confidence, "classifier", nil)

	err = doc.ApplyClassification(classification)
	assert.ErrorIs(t, err, domainerrors.ErrCannotClassifyEmptyDocument)
	assert.False(t, doc.HasClassification())
}

func TestDocument_ApplyClassification_InvalidConfidence(t *testing.T) {
	doc, err := NewDocument(
		WithID("doc_confidence"),
		WithFileName("order.pdf"),
		WithStoragePath("documents/2024/05/order.pdf"),
		WithHash("d3c1e5a6b9d2f4a7c8e1f2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2d6", "SHA256"),
		WithContentType("application/pdf"),
		WithFileSize(1024),
		WithText("Order body"),
	)
	require.NoError(t, err)

	documentType, _ := NewDocumentType("order")
	category, _ := NewCategory("decision")

	classification := &Classification{
		documentType: documentType,
		category:     category,
		confidence:   Confidence(1.5),
		classifiedBy: "classifier",
	}

	err = doc.ApplyClassification(classification)
	assert.ErrorIs(t, err, domainerrors.ErrInvalidConfidence)
}

func TestDocument_ApplyClassification_NilClassification(t *testing.T) {
	doc, err := NewDocument(
		WithID("doc_nil_classification"),
		WithFileName("order.pdf"),
		WithStoragePath("documents/2024/05/order.pdf"),
		WithHash("d4c1e5a6b9d2f4a7c8e1f2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2d6", "SHA256"),
		WithContentType("application/pdf"),
		WithFileSize(1024),
		WithText("Order body"),
	)
	require.NoError(t, err)

	err = doc.ApplyClassification(nil)
	assert.ErrorIs(t, err, domainerrors.ErrInvalidClassification)
}

func TestDocument_UpdateMetadata(t *testing.T) {
	doc, err := NewDocument(
		WithID("doc_metadata"),
		WithFileName("metadata.pdf"),
		WithStoragePath("documents/2024/05/metadata.pdf"),
		WithHash("e3c1e5a6b9d2f4a7c8e1f2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2d7", "SHA256"),
		WithContentType("application/pdf"),
		WithFileSize(512),
		WithText("Metadata sample text"),
	)
	require.NoError(t, err)

	metadata, err := NewMetadata("en", []string{"motion", "suppression"})
	require.NoError(t, err)

	err = doc.UpdateMetadata(metadata)
	require.NoError(t, err)

	assert.Equal(t, metadata, doc.Metadata())
	assert.Greater(t, doc.Version(), 1)

	events := doc.GetEvents()
	require.Len(t, events, 2)
	assert.IsType(t, &MetadataUpdatedEvent{}, events[1])
}

func TestDocument_UpdateMetadata_Invalid(t *testing.T) {
	doc, err := NewDocument(
		WithID("doc_metadata_invalid"),
		WithFileName("metadata.pdf"),
		WithStoragePath("documents/2024/05/metadata.pdf"),
		WithHash("f3c1e5a6b9d2f4a7c8e1f2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2d8", "SHA256"),
		WithContentType("application/pdf"),
		WithFileSize(512),
	)
	require.NoError(t, err)

	metadata := &Metadata{language: ""}
	err = doc.UpdateMetadata(metadata)
	var validationErr *domainerrors.ValidationError
	assert.ErrorAs(t, err, &validationErr)
}

func TestDocument_MarkAsProcessed(t *testing.T) {
	doc, err := NewDocument(
		WithID("doc_processed"),
		WithFileName("processed.pdf"),
		WithStoragePath("documents/2024/05/processed.pdf"),
		WithHash("03c1e5a6b9d2f4a7c8e1f2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2d9", "SHA256"),
		WithContentType("application/pdf"),
		WithFileSize(256),
	)
	require.NoError(t, err)

	initialVersion := doc.Version()
	doc.MarkAsProcessed()

	assert.NotNil(t, doc.ProcessedAt())
	assert.Greater(t, doc.Version(), initialVersion)

	events := doc.GetEvents()
	require.Len(t, events, 2)
	assert.IsType(t, &DocumentProcessedEvent{}, events[1])
}

func TestDocument_GetEventsReturnsCopy(t *testing.T) {
	doc, err := NewDocument(
		WithID("doc_events"),
		WithFileName("events.pdf"),
		WithStoragePath("documents/2024/05/events.pdf"),
		WithHash("13c1e5a6b9d2f4a7c8e1f2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2d9", "SHA256"),
		WithContentType("application/pdf"),
		WithFileSize(1024),
	)
	require.NoError(t, err)

	events := doc.GetEvents()
	require.Len(t, events, 1)
	events[0] = nil

	eventsAfter := doc.GetEvents()
	require.Len(t, eventsAfter, 1)
	assert.NotNil(t, eventsAfter[0])

	doc.ClearEvents()
	assert.Empty(t, doc.GetEvents())
}

func TestNewDocument_WithRehydrationOptions(t *testing.T) {
	s3URI, err := NewS3URI("bucket", "documents/2024/05/file.pdf")
	require.NoError(t, err)

	hashValue := "33c1e5a6b9d2f4a7c8e1f2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2db"
	createdAt := time.Date(2024, time.May, 10, 8, 30, 0, 0, time.UTC)
	updatedAt := createdAt.Add(2 * time.Hour)

	metadata, err := NewMetadata("en", []string{"Motion", "motion"})
	require.NoError(t, err)

	documentType, err := NewDocumentType("motion")
	require.NoError(t, err)
	category, err := NewCategory("filing")
	require.NoError(t, err)
	confidence, err := NewConfidence(0.9)
	require.NoError(t, err)
	classification, err := NewClassification(documentType, category, confidence, "classifier", []string{"motion"})
	require.NoError(t, err)

	doc, err := NewDocument(
		WithID("doc_rehydrate"),
		WithFileName("rehydrate.pdf"),
		WithStoragePath("documents/2024/05/rehydrate.pdf"),
		WithS3URI(s3URI),
		WithHash(hashValue, "SHA256"),
		WithContentType("application/pdf"),
		WithFileSize(1024),
		WithText("Rehydration example"),
		WithMetadata(metadata),
		WithClassification(classification),
		WithVersion(5),
		WithTimestamps(createdAt, updatedAt),
		WithProcessedAt(updatedAt),
	)
	require.NoError(t, err)

	assert.Equal(t, s3URI.String(), doc.S3URI().String())
	assert.Equal(t, hashValue, doc.Hash().String())
	assert.Equal(t, "Rehydration example", doc.Text().String())
	assert.Equal(t, createdAt, doc.CreatedAt())
	assert.NotNil(t, doc.ProcessedAt())
	assert.Equal(t, 5, doc.Version())
	assert.Equal(t, metadata, doc.Metadata())
	assert.NotNil(t, doc.Classification())
}

func TestNewDocument_WithOptionFailures(t *testing.T) {
	baseOpts := []DocumentOption{
		WithID("doc_opts"),
		WithFileName("opts.pdf"),
		WithStoragePath("documents/2024/05/opts.pdf"),
		WithHash("43c1e5a6b9d2f4a7c8e1f2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2dc", "SHA256"),
		WithContentType("application/pdf"),
		WithFileSize(1024),
	}

	t.Run("WithID invalid", func(t *testing.T) {
		_, err := NewDocument(
			WithID(""),
			WithFileName("opts.pdf"),
			WithStoragePath("documents/2024/05/opts.pdf"),
			WithHash("43c1e5a6b9d2f4a7c8e1f2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2dc", "SHA256"),
			WithContentType("application/pdf"),
			WithFileSize(1024),
		)
		assert.ErrorIs(t, err, domainerrors.ErrEmptyDocumentID)
	})

	t.Run("WithFileName invalid", func(t *testing.T) {
		_, err := NewDocument(
			WithID("doc_bad_name"),
			WithFileName("../opts.pdf"),
			WithStoragePath("documents/2024/05/opts.pdf"),
			WithHash("43c1e5a6b9d2f4a7c8e1f2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2dc", "SHA256"),
			WithContentType("application/pdf"),
			WithFileSize(1024),
		)
		assert.Error(t, err)
	})

	t.Run("WithStoragePath invalid", func(t *testing.T) {
		_, err := NewDocument(
			WithID("doc_bad_path"),
			WithFileName("opts.pdf"),
			WithStoragePath("uploads/opts.pdf"),
			WithHash("43c1e5a6b9d2f4a7c8e1f2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2dc", "SHA256"),
			WithContentType("application/pdf"),
			WithFileSize(1024),
		)
		assert.Error(t, err)
	})

	t.Run("WithHash invalid", func(t *testing.T) {
		_, err := NewDocument(
			WithID("doc_bad_hash"),
			WithFileName("opts.pdf"),
			WithStoragePath("documents/2024/05/opts.pdf"),
			WithHash("", "SHA256"),
			WithContentType("application/pdf"),
			WithFileSize(1024),
		)
		assert.ErrorIs(t, err, domainerrors.ErrEmptyHash)
	})

	t.Run("WithContentType invalid", func(t *testing.T) {
		_, err := NewDocument(
			WithID("doc_bad_type"),
			WithFileName("opts.pdf"),
			WithStoragePath("documents/2024/05/opts.pdf"),
			WithHash("43c1e5a6b9d2f4a7c8e1f2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2dc", "SHA256"),
			WithContentType("image/png"),
			WithFileSize(1024),
		)
		assert.ErrorIs(t, err, domainerrors.ErrInvalidContentType)
	})

	t.Run("WithFileSize invalid", func(t *testing.T) {
		_, err := NewDocument(
			WithID("doc_bad_size"),
			WithFileName("opts.pdf"),
			WithStoragePath("documents/2024/05/opts.pdf"),
			WithHash("43c1e5a6b9d2f4a7c8e1f2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2dc", "SHA256"),
			WithContentType("application/pdf"),
			WithFileSize(0),
		)
		assert.ErrorIs(t, err, domainerrors.ErrInvalidFileSize)
	})

	t.Run("WithPageCount invalid", func(t *testing.T) {
		_, err := NewDocument(
			append(baseOpts, WithPageCount(-1))...,
		)
		assert.Error(t, err)
	})

	t.Run("WithWordCount invalid", func(t *testing.T) {
		_, err := NewDocument(
			append(baseOpts, WithWordCount(-1))...,
		)
		assert.Error(t, err)
	})

	t.Run("WithS3URIString invalid", func(t *testing.T) {
		_, err := NewDocument(append(baseOpts, WithS3URIString("https://bucket/key"))...)
		assert.Error(t, err)
	})

	t.Run("WithMetadata nil", func(t *testing.T) {
		_, err := NewDocument(append(baseOpts, WithMetadata(nil))...)
		assert.Error(t, err)
	})

	t.Run("WithMetadata invalid content", func(t *testing.T) {
		metadata := &Metadata{language: ""}
		_, err := NewDocument(append(baseOpts, WithMetadata(metadata))...)
		assert.Error(t, err)
	})

	t.Run("WithClassification nil", func(t *testing.T) {
		_, err := NewDocument(append(baseOpts, WithClassification(nil))...)
		assert.ErrorIs(t, err, domainerrors.ErrInvalidClassification)
	})

	t.Run("WithClassification invalid confidence", func(t *testing.T) {
		classification := &Classification{confidence: Confidence(2)}
		_, err := NewDocument(append(baseOpts, WithClassification(classification))...)
		assert.ErrorIs(t, err, domainerrors.ErrInvalidConfidence)
	})

	t.Run("WithVersion invalid", func(t *testing.T) {
		_, err := NewDocument(append(baseOpts, WithVersion(0))...)
		assert.Error(t, err)
	})

	t.Run("WithTimestamps invalid", func(t *testing.T) {
		_, err := NewDocument(append(baseOpts, WithTimestamps(time.Time{}, time.Now()))...)
		assert.Error(t, err)
	})

	t.Run("WithTimestamps invalid updatedAt", func(t *testing.T) {
		_, err := NewDocument(append(baseOpts, WithTimestamps(time.Now(), time.Time{}))...)
		assert.Error(t, err)
	})

	t.Run("WithProcessedAt invalid", func(t *testing.T) {
		_, err := NewDocument(append(baseOpts, WithProcessedAt(time.Time{}))...)
		assert.Error(t, err)
	})
}

func TestMetadataStateHelpers(t *testing.T) {
	metadata, err := NewMetadata("en", nil)
	require.NoError(t, err)

	assert.False(t, metadata.AIClassified())
	assert.Nil(t, metadata.ProcessedAt())
	assert.Nil(t, metadata.LegalTags())

	now := time.Now()
	metadata.SetProcessedAt(now)
	metadata.SetAIClassified(true)

	require.NotNil(t, metadata.ProcessedAt())
	assert.WithinDuration(t, now, *metadata.ProcessedAt(), time.Second)
	assert.True(t, metadata.AIClassified())
}

func TestDocumentAddEventIgnoresNil(t *testing.T) {
	doc, err := NewDocument(
		WithID("doc_event_nil"),
		WithFileName("event.pdf"),
		WithStoragePath("documents/2024/05/event.pdf"),
		WithHash("53c1e5a6b9d2f4a7c8e1f2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2dd", "SHA256"),
		WithContentType("application/pdf"),
		WithFileSize(1024),
	)
	require.NoError(t, err)

	initialEvents := len(doc.GetEvents())
	doc.addEvent(nil)
	assert.Equal(t, initialEvents, len(doc.GetEvents()))
}

func TestNewMetadataValidation(t *testing.T) {
	_, err := NewMetadata("  ", []string{"  "})
	assert.Error(t, err)
}

func TestDocumentTypeAndCategoryValidation(t *testing.T) {
	_, err := NewDocumentType(" ")
	assert.ErrorIs(t, err, domainerrors.ErrInvalidClassification)

	_, err = NewCategory(" ")
	assert.ErrorIs(t, err, domainerrors.ErrInvalidClassification)
}

func TestDocumentValidateDetectsInvalidState(t *testing.T) {
	doc := &Document{}

	assert.ErrorIs(t, doc.validate(), domainerrors.ErrEmptyDocumentID)

	docID, _ := NewDocumentID("doc_validate")
	doc.id = docID
	assert.ErrorIs(t, doc.validate(), domainerrors.ErrEmptyFileName)

	fileName, _ := NewFileName("validate.pdf")
	doc.fileName = fileName
	assert.ErrorIs(t, doc.validate(), domainerrors.ErrEmptyFilePath)

	filePath, _ := NewFilePath("documents/2024/05/validate.pdf")
	doc.filePath = filePath
	assert.ErrorIs(t, doc.validate(), domainerrors.ErrInvalidHash)

	hash, _ := NewHash("63c1e5a6b9d2f4a7c8e1f2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2de", "SHA256")
	doc.hash = hash
	assert.ErrorIs(t, doc.validate(), domainerrors.ErrInvalidContentType)

	contentType, _ := NewContentType("application/pdf")
	doc.contentType = contentType
	assert.ErrorIs(t, doc.validate(), domainerrors.ErrInvalidFileSize)

	size, _ := NewFileSize(1024)
	doc.size = size

	doc.metadata = &Metadata{language: ""}
	err := doc.validate()
	var validationErr *domainerrors.ValidationError
	assert.ErrorAs(t, err, &validationErr)
	assert.Equal(t, "language", validationErr.Field)

	doc.metadata.language = "en"
	doc.metadata.legalTags = []string{"Motion"}
	assert.NoError(t, doc.metadata.Validate())

	doc.classification = &Classification{confidence: Confidence(2)}
	assert.ErrorIs(t, doc.validate(), domainerrors.ErrInvalidConfidence)

	doc.classification.confidence, _ = NewConfidence(0.8)
	assert.NoError(t, doc.validate())
}

func TestUpdateMetadataNil(t *testing.T) {
	doc, err := NewDocument(
		WithID("doc_update_nil"),
		WithFileName("update.pdf"),
		WithStoragePath("documents/2024/05/update.pdf"),
		WithHash("73c1e5a6b9d2f4a7c8e1f2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2df", "SHA256"),
		WithContentType("application/pdf"),
		WithFileSize(1024),
	)
	require.NoError(t, err)

	err = doc.UpdateMetadata(nil)
	assert.Error(t, err)
}

func TestClassificationLegalTagsEmpty(t *testing.T) {
	docType, _ := NewDocumentType("motion")
	category, _ := NewCategory("filing")
	confidence, _ := NewConfidence(0.9)

	classification, err := NewClassification(docType, category, confidence, "classifier", nil)
	require.NoError(t, err)
	assert.Nil(t, classification.LegalTags())
}

func TestMetadataValidateNilPointer(t *testing.T) {
	var metadata *Metadata
	err := metadata.Validate()
	var validationErr *domainerrors.ValidationError
	assert.ErrorAs(t, err, &validationErr)
	assert.Equal(t, "metadata", validationErr.Field)
}

func TestMetadata_SanitizeAndValidate(t *testing.T) {
	metadata, err := NewMetadata(" EN ", []string{"Motion", "motion", "  suppression "})
	require.NoError(t, err)

	assert.Equal(t, "EN", strings.ToUpper(metadata.Language()))
	tags := metadata.LegalTags()
	assert.Len(t, tags, 2)
	assert.Contains(t, tags, "Motion")
	assert.Contains(t, tags, "suppression")
}

func TestClassification_NewClassification(t *testing.T) {
	documentType, err := NewDocumentType("motion")
	require.NoError(t, err)
	category, err := NewCategory("filing")
	require.NoError(t, err)
	confidence, err := NewConfidence(0.85)
	require.NoError(t, err)

	classification, err := NewClassification(documentType, category, confidence, "classifier", []string{"motion"})
	require.NoError(t, err)

	assert.Equal(t, documentType, classification.DocumentType())
	assert.Equal(t, category, classification.Category())
	assert.Equal(t, confidence, classification.Confidence())
	assert.Contains(t, classification.LegalTags(), "motion")
	assert.Equal(t, "classifier", classification.ClassifiedBy())
	assert.False(t, classification.ClassifiedAt().IsZero())
}

func TestClassification_InvalidInputs(t *testing.T) {
	category, _ := NewCategory("filing")
	confidence, _ := NewConfidence(0.75)

	_, err := NewClassification(DocumentType{}, category, confidence, "classifier", nil)
	assert.ErrorIs(t, err, domainerrors.ErrInvalidClassification)

	documentType, _ := NewDocumentType("motion")
	_, err = NewClassification(documentType, Category{}, confidence, "classifier", nil)
	assert.ErrorIs(t, err, domainerrors.ErrInvalidClassification)

	invalidConfidence := Confidence(1.5)
	_, err = NewClassification(documentType, category, invalidConfidence, "classifier", nil)
	assert.ErrorIs(t, err, domainerrors.ErrInvalidConfidence)
}
