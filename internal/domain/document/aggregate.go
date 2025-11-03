package document

import (
	"time"

	domainerrors "motion-index-fiber/internal/domain/errors"
)

// DocumentOption configures a document during construction.
type DocumentOption func(*Document) error

// NewDocument creates a new Document aggregate enforcing invariants.
func NewDocument(opts ...DocumentOption) (*Document, error) {
	now := time.Now()
	doc := &Document{
		version:   1,
		createdAt: now,
		updatedAt: now,
		events:    make([]DomainEvent, 0),
	}

	for _, opt := range opts {
		if err := opt(doc); err != nil {
			return nil, err
		}
	}

	if err := doc.validate(); err != nil {
		return nil, err
	}

	doc.addEvent(NewDocumentCreatedEvent(doc))
	return doc, nil
}

// WithID sets the document identifier.
func WithID(id string) DocumentOption {
	return func(d *Document) error {
		docID, err := NewDocumentID(id)
		if err != nil {
			return err
		}
		d.id = docID
		return nil
	}
}

// WithFileName sets the document file name.
func WithFileName(name string) DocumentOption {
	return func(d *Document) error {
		fileName, err := NewFileName(name)
		if err != nil {
			return err
		}
		d.fileName = fileName
		return nil
	}
}

// WithStoragePath sets the local storage path.
func WithStoragePath(path string) DocumentOption {
	return func(d *Document) error {
		filePath, err := NewFilePath(path)
		if err != nil {
			return err
		}
		d.filePath = filePath
		return nil
	}
}

// WithS3URIString sets the S3 URI from its string representation.
func WithS3URIString(uri string) DocumentOption {
	return func(d *Document) error {
		s3URI, err := ParseS3URI(uri)
		if err != nil {
			return err
		}
		d.s3URI = s3URI
		return nil
	}
}

// WithS3URI sets the S3 URI directly.
func WithS3URI(uri S3URI) DocumentOption {
	return func(d *Document) error {
		d.s3URI = uri
		return nil
	}
}

// WithHash sets the document hash.
func WithHash(value, algorithm string) DocumentOption {
	return func(d *Document) error {
		hash, err := NewHash(value, algorithm)
		if err != nil {
			return err
		}
		d.hash = hash
		return nil
	}
}

// WithContentType sets the MIME type.
func WithContentType(value string) DocumentOption {
	return func(d *Document) error {
		contentType, err := NewContentType(value)
		if err != nil {
			return err
		}
		d.contentType = contentType
		return nil
	}
}

// WithFileSize sets the file size in bytes.
func WithFileSize(bytes int64) DocumentOption {
	return func(d *Document) error {
		size, err := NewFileSize(bytes)
		if err != nil {
			return err
		}
		d.size = size
		return nil
	}
}

// WithText sets the extracted text.
func WithText(value string) DocumentOption {
	return func(d *Document) error {
		d.text = NewText(value)
		return nil
	}
}

// WithPageCount sets the page count.
func WithPageCount(value int) DocumentOption {
	return func(d *Document) error {
		pageCount, err := NewPageCount(value)
		if err != nil {
			return err
		}
		d.pageCount = pageCount
		return nil
	}
}

// WithWordCount sets the word count.
func WithWordCount(value int) DocumentOption {
	return func(d *Document) error {
		wordCount, err := NewWordCount(value)
		if err != nil {
			return err
		}
		d.wordCount = wordCount
		return nil
	}
}

// WithMetadata seeds the aggregate with metadata (rehydration).
func WithMetadata(metadata *Metadata) DocumentOption {
	return func(d *Document) error {
		if metadata == nil {
			return domainerrors.NewValidationError("metadata", "cannot be nil", nil)
		}
		if err := metadata.Validate(); err != nil {
			return err
		}
		d.metadata = metadata
		return nil
	}
}

// WithClassification seeds the aggregate with an existing classification.
func WithClassification(classification *Classification) DocumentOption {
	return func(d *Document) error {
		if classification == nil {
			return domainerrors.ErrInvalidClassification
		}
		if !classification.confidence.IsValid() {
			return domainerrors.ErrInvalidConfidence
		}
		d.classification = classification
		return nil
	}
}

// WithVersion overrides the aggregate version (rehydration).
func WithVersion(version int) DocumentOption {
	return func(d *Document) error {
		if version < 1 {
			return domainerrors.NewValidationError("version", "must be >= 1", version)
		}
		d.version = version
		return nil
	}
}

// WithTimestamps overrides creation/update timestamps (rehydration).
func WithTimestamps(createdAt, updatedAt time.Time) DocumentOption {
	return func(d *Document) error {
		if createdAt.IsZero() {
			return domainerrors.NewValidationError("createdAt", "must be set", createdAt)
		}
		if updatedAt.IsZero() {
			return domainerrors.NewValidationError("updatedAt", "must be set", updatedAt)
		}
		d.createdAt = createdAt
		d.updatedAt = updatedAt
		return nil
	}
}

// WithProcessedAt seeds the processed timestamp.
func WithProcessedAt(processedAt time.Time) DocumentOption {
	return func(d *Document) error {
		if processedAt.IsZero() {
			return domainerrors.NewValidationError("processedAt", "must be set", processedAt)
		}
		d.processedAt = &processedAt
		return nil
	}
}

// ApplyClassification applies a classification result to the document.
func (d *Document) ApplyClassification(classification *Classification) error {
	if classification == nil {
		return domainerrors.ErrInvalidClassification
	}
	if d.text.IsEmpty() {
		return domainerrors.ErrCannotClassifyEmptyDocument
	}
	if !classification.confidence.IsValid() {
		return domainerrors.ErrInvalidConfidence
	}

	d.classification = classification
	now := time.Now()
	classification.classifiedAt = now
	d.touch(now)

	d.addEvent(NewDocumentClassifiedEvent(d, classification))
	return nil
}

// UpdateMetadata replaces metadata after validation.
func (d *Document) UpdateMetadata(metadata *Metadata) error {
	if metadata == nil {
		return domainerrors.NewValidationError("metadata", "cannot be nil", nil)
	}
	if err := metadata.Validate(); err != nil {
		return err
	}
	d.metadata = metadata
	now := time.Now()
	d.touch(now)

	d.addEvent(NewMetadataUpdatedEvent(d))
	return nil
}

// MarkAsProcessed records that the document finished processing.
func (d *Document) MarkAsProcessed() {
	now := time.Now()
	d.processedAt = &now
	d.touch(now)
	d.addEvent(NewDocumentProcessedEvent(d))
}

func (d *Document) validate() error {
	if d.id.IsEmpty() {
		return domainerrors.ErrEmptyDocumentID
	}

	if d.fileName.IsEmpty() {
		return domainerrors.ErrEmptyFileName
	}

	if d.filePath.IsEmpty() {
		return domainerrors.ErrEmptyFilePath
	}

	if d.hash.String() == "" {
		return domainerrors.ErrInvalidHash
	}

	if d.contentType.String() == "" {
		return domainerrors.ErrInvalidContentType
	}

	if d.size.IsZero() {
		return domainerrors.ErrInvalidFileSize
	}

	if d.metadata != nil {
		if err := d.metadata.Validate(); err != nil {
			return err
		}
	}

	if d.classification != nil {
		if !d.classification.confidence.IsValid() {
			return domainerrors.ErrInvalidConfidence
		}
	}

	return nil
}

func (d *Document) touch(now time.Time) {
	d.updatedAt = now
	d.version++
}
