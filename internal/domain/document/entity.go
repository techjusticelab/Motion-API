package document

import (
	"strings"
	"time"

	domainerrors "motion-index-fiber/internal/domain/errors"
)

// Document is the aggregate root managing legal document state.
type Document struct {
	// Identity
	id      DocumentID
	version int

	// File information
	fileName    FileName
	filePath    FilePath
	s3URI       S3URI
	hash        Hash
	contentType ContentType
	size        FileSize

	// Content
	text      Text
	pageCount PageCount
	wordCount WordCount

	// Classification & metadata
	classification *Classification
	metadata       *Metadata

	// Timestamps
	createdAt   time.Time
	updatedAt   time.Time
	processedAt *time.Time

	// Domain events raised by the aggregate
	events []DomainEvent
}

// ID returns the aggregate identifier.
func (d *Document) ID() DocumentID {
	return d.id
}

// Version returns the optimistic lock version.
func (d *Document) Version() int {
	return d.version
}

// FileName returns the stored file name.
func (d *Document) FileName() FileName {
	return d.fileName
}

// FilePath returns the storage path.
func (d *Document) FilePath() FilePath {
	return d.filePath
}

// S3URI returns the remote storage reference.
func (d *Document) S3URI() S3URI {
	return d.s3URI
}

// Hash returns the content hash.
func (d *Document) Hash() Hash {
	return d.hash
}

// ContentType returns the MIME type.
func (d *Document) ContentType() ContentType {
	return d.contentType
}

// Size returns the file size.
func (d *Document) Size() FileSize {
	return d.size
}

// Text returns the extracted text content.
func (d *Document) Text() Text {
	return d.text
}

// PageCount returns the number of pages.
func (d *Document) PageCount() PageCount {
	return d.pageCount
}

// WordCount returns the number of words.
func (d *Document) WordCount() WordCount {
	return d.wordCount
}

// Classification returns the latest classification, if any.
func (d *Document) Classification() *Classification {
	return d.classification
}

// Metadata returns associated metadata, if any.
func (d *Document) Metadata() *Metadata {
	return d.metadata
}

// CreatedAt returns the creation timestamp.
func (d *Document) CreatedAt() time.Time {
	return d.createdAt
}

// UpdatedAt returns the last updated timestamp.
func (d *Document) UpdatedAt() time.Time {
	return d.updatedAt
}

// ProcessedAt returns when the document finished processing.
func (d *Document) ProcessedAt() *time.Time {
	return d.processedAt
}

// HasClassification reports whether the document has a classification.
func (d *Document) HasClassification() bool {
	return d.classification != nil
}

// GetEvents returns a copy of queued domain events.
func (d *Document) GetEvents() []DomainEvent {
	if len(d.events) == 0 {
		return nil
	}
	events := make([]DomainEvent, len(d.events))
	copy(events, d.events)
	return events
}

// ClearEvents removes queued domain events.
func (d *Document) ClearEvents() {
	d.events = nil
}

// addEvent queues a domain event.
func (d *Document) addEvent(event DomainEvent) {
	if event == nil {
		return
	}
	d.events = append(d.events, event)
}

// Metadata captures ancillary document metadata.
type Metadata struct {
	language     string
	legalTags    []string
	aiClassified bool
	processedAt  *time.Time
}

// NewMetadata constructs metadata with validation.
func NewMetadata(language string, legalTags []string) (*Metadata, error) {
	tagsCopy := make([]string, len(legalTags))
	copy(tagsCopy, legalTags)

	m := &Metadata{
		language:  strings.TrimSpace(language),
		legalTags: tagsCopy,
	}
	if err := m.Validate(); err != nil {
		return nil, err
	}
	return m, nil
}

// Language returns the metadata language.
func (m *Metadata) Language() string {
	return m.language
}

// LegalTags returns a copy of legal tags.
func (m *Metadata) LegalTags() []string {
	if len(m.legalTags) == 0 {
		return nil
	}
	tags := make([]string, len(m.legalTags))
	copy(tags, m.legalTags)
	return tags
}

// ProcessedAt returns the processing timestamp, if set.
func (m *Metadata) ProcessedAt() *time.Time {
	return m.processedAt
}

// SetProcessedAt records the processing timestamp.
func (m *Metadata) SetProcessedAt(t time.Time) {
	m.processedAt = &t
}

// AIClassified reports whether AI classification contributed to the metadata.
func (m *Metadata) AIClassified() bool {
	return m.aiClassified
}

// SetAIClassified toggles the AI classification indicator.
func (m *Metadata) SetAIClassified(flag bool) {
	m.aiClassified = flag
}

// Validate enforces metadata invariants.
func (m *Metadata) Validate() error {
	if m == nil {
		return domainerrors.NewValidationError("metadata", "cannot be nil", nil)
	}

	m.language = strings.TrimSpace(m.language)
	if m.language == "" {
		return domainerrors.NewValidationError("language", "cannot be empty", m.language)
	}

	m.legalTags = sanitizeTags(m.legalTags)
	return nil
}

// Classification represents AI/ML derived categorisation.
type Classification struct {
	documentType DocumentType
	category     Category
	confidence   Confidence
	legalTags    []string
	classifiedAt time.Time
	classifiedBy string
}

// NewClassification constructs a validated classification instance.
func NewClassification(documentType DocumentType, category Category, confidence Confidence, classifiedBy string, legalTags []string) (*Classification, error) {
	if documentType.IsZero() {
		return nil, domainerrors.ErrInvalidClassification
	}

	if category.IsZero() {
		return nil, domainerrors.ErrInvalidClassification
	}

	if !confidence.IsValid() {
		return nil, domainerrors.ErrInvalidConfidence
	}

	classification := &Classification{
		documentType: documentType,
		category:     category,
		confidence:   confidence,
		legalTags:    sanitizeTags(legalTags),
		classifiedAt: time.Now(),
		classifiedBy: strings.TrimSpace(classifiedBy),
	}
	return classification, nil
}

// DocumentType returns the document type.
func (c *Classification) DocumentType() DocumentType {
	return c.documentType
}

// Category returns the classification category.
func (c *Classification) Category() Category {
	return c.category
}

// Confidence returns the confidence measure.
func (c *Classification) Confidence() Confidence {
	return c.confidence
}

// LegalTags returns a copy of associated legal tags.
func (c *Classification) LegalTags() []string {
	if len(c.legalTags) == 0 {
		return nil
	}
	tags := make([]string, len(c.legalTags))
	copy(tags, c.legalTags)
	return tags
}

// ClassifiedAt returns the classification timestamp.
func (c *Classification) ClassifiedAt() time.Time {
	return c.classifiedAt
}

// ClassifiedBy returns the classifier identifier.
func (c *Classification) ClassifiedBy() string {
	return c.classifiedBy
}

// DocumentType represents a domain-specific document type.
type DocumentType struct {
	value string
}

// NewDocumentType constructs a document type.
func NewDocumentType(value string) (DocumentType, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return DocumentType{}, domainerrors.ErrInvalidClassification
	}
	return DocumentType{value: strings.ToLower(trimmed)}, nil
}

// String returns the type as a string.
func (dt DocumentType) String() string {
	return dt.value
}

// IsZero reports whether the type is unset.
func (dt DocumentType) IsZero() bool {
	return dt.value == ""
}

// Category represents a high-level classification category.
type Category struct {
	value string
}

// NewCategory constructs a validated category.
func NewCategory(value string) (Category, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return Category{}, domainerrors.ErrInvalidClassification
	}
	return Category{value: strings.ToLower(trimmed)}, nil
}

// String returns the category name.
func (c Category) String() string {
	return c.value
}

// IsZero reports whether the category is unset.
func (c Category) IsZero() bool {
	return c.value == ""
}

// sanitizeTags trims and de-duplicates legal tags.
func sanitizeTags(tags []string) []string {
	if len(tags) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(tags))
	result := make([]string, 0, len(tags))
	for _, tag := range tags {
		trimmed := strings.TrimSpace(tag)
		if trimmed == "" {
			continue
		}
		key := strings.ToLower(trimmed)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, trimmed)
	}
	if len(result) == 0 {
		return nil
	}
	return result
}
