# Document Domain

The document domain is the core aggregate of the Motion-Index-Fiber system. It represents a legal document with all its associated metadata, classification, and processing state.

## Structure

```
document/
├── aggregate.go        # Document aggregate root with business logic
├── entity.go          # Document entity definition
├── value_objects.go   # Value objects (FilePath, Hash, S3URI, etc.)
├── repository.go      # Repository interface
├── events.go          # Domain events
└── document_test.go   # Comprehensive tests
```

## Document Aggregate

The Document aggregate is the root entity that manages the lifecycle of a legal document.

### Aggregate Root: Document

```go
type Document struct {
    // Identity
    id       DocumentID
    version  int

    // File Information
    fileName    FileName
    filePath    FilePath
    s3URI       S3URI
    hash        Hash
    contentType ContentType
    size        FileSize

    // Content
    text        Text
    pageCount   PageCount
    wordCount   WordCount

    // Classification
    classification *Classification

    // Metadata
    metadata    *Metadata

    // Timestamps
    createdAt   time.Time
    updatedAt   time.Time
    processedAt *time.Time

    // Domain Events
    events []DomainEvent
}
```

### Value Objects

#### FilePath
Represents a storage path with validation.

```go
type FilePath struct {
    value string
}

func NewFilePath(path string) (FilePath, error) {
    if path == "" {
        return FilePath{}, ErrEmptyFilePath
    }
    if !isValidStoragePath(path) {
        return FilePath{}, ErrInvalidFilePath
    }
    return FilePath{value: path}, nil
}
```

#### S3URI
Represents an S3-format URI for cloud storage.

```go
type S3URI struct {
    bucket string
    key    string
}

func NewS3URI(bucket, key string) (S3URI, error) {
    if bucket == "" || key == "" {
        return S3URI{}, ErrInvalidS3URI
    }
    return S3URI{bucket: bucket, key: key}, nil
}

func (u S3URI) String() string {
    return fmt.Sprintf("s3://%s/%s", u.bucket, u.key)
}
```

#### Hash
Represents a document content hash with validation.

```go
type Hash struct {
    value     string
    algorithm string // e.g., "SHA256"
}

func NewHash(value, algorithm string) (Hash, error) {
    if value == "" {
        return Hash{}, ErrEmptyHash
    }
    return Hash{value: value, algorithm: algorithm}, nil
}
```

#### Classification
Represents the result of AI classification.

```go
type Classification struct {
    documentType DocumentType
    category     Category
    confidence   Confidence
    legalTags    []string
    classifiedAt time.Time
    classifiedBy string // Provider name
}

func NewClassification(
    docType DocumentType,
    category Category,
    confidence Confidence,
) (*Classification, error) {
    if err := confidence.Validate(); err != nil {
        return nil, err
    }
    return &Classification{
        documentType: docType,
        category:     category,
        confidence:   confidence,
        classifiedAt: time.Now(),
    }, nil
}
```

#### Confidence
Represents classification confidence (0.0 - 1.0).

```go
type Confidence float64

func NewConfidence(value float64) (Confidence, error) {
    if value < 0.0 || value > 1.0 {
        return 0, ErrInvalidConfidence
    }
    return Confidence(value), nil
}

func (c Confidence) IsHigh() bool {
    return c >= 0.8
}

func (c Confidence) IsMedium() bool {
    return c >= 0.5 && c < 0.8
}

func (c Confidence) IsLow() bool {
    return c < 0.5
}
```

## Business Methods

### Creating a Document

```go
func NewDocument(opts ...DocumentOption) (*Document, error) {
    doc := &Document{
        id:        generateDocumentID(),
        version:   1,
        createdAt: time.Now(),
        updatedAt: time.Now(),
        events:    make([]DomainEvent, 0),
    }

    // Apply options
    for _, opt := range opts {
        if err := opt(doc); err != nil {
            return nil, err
        }
    }

    // Validate invariants
    if err := doc.validate(); err != nil {
        return nil, err
    }

    // Emit domain event
    doc.addEvent(NewDocumentCreatedEvent(doc))

    return doc, nil
}
```

### Document Options (Functional Options Pattern)

```go
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
```

### Applying Classification

```go
func (d *Document) ApplyClassification(classification *Classification) error {
    // Business rule: Can't classify without content
    if d.text.IsEmpty() {
        return ErrCannotClassifyEmptyDocument
    }

    // Business rule: Confidence must be valid
    if !classification.confidence.IsValid() {
        return ErrInvalidConfidence
    }

    d.classification = classification
    d.updatedAt = time.Now()

    // Emit domain event
    d.addEvent(NewDocumentClassifiedEvent(d, classification))

    return nil
}
```

### Updating Metadata

```go
func (d *Document) UpdateMetadata(metadata *Metadata) error {
    // Validate metadata
    if err := metadata.Validate(); err != nil {
        return err
    }

    d.metadata = metadata
    d.updatedAt = time.Now()
    d.version++

    // Emit domain event
    d.addEvent(NewMetadataUpdatedEvent(d))

    return nil
}
```

### Marking as Processed

```go
func (d *Document) MarkAsProcessed() {
    now := time.Now()
    d.processedAt = &now
    d.updatedAt = now

    // Emit domain event
    d.addEvent(NewDocumentProcessedEvent(d))
}
```

## Invariants

The Document aggregate enforces these business rules:

1. **ID must be unique** and non-empty
2. **File name must be valid** (no path separators, valid extension)
3. **File path must be valid** storage path format
4. **Classification confidence** must be between 0.0 and 1.0
5. **Hash must be valid** (non-empty, valid format)
6. **Content type must be supported** (PDF, DOCX, TXT)
7. **File size must be positive** and within limits

```go
func (d *Document) validate() error {
    if d.id.IsEmpty() {
        return ErrEmptyDocumentID
    }

    if d.fileName.IsEmpty() {
        return ErrEmptyFileName
    }

    if d.classification != nil {
        if !d.classification.confidence.IsValid() {
            return ErrInvalidConfidence
        }
    }

    return nil
}
```

## Domain Events

### DocumentCreated

```go
type DocumentCreatedEvent struct {
    DocumentID  string
    FileName    string
    ContentType string
    OccurredAt  time.Time
}
```

### DocumentClassified

```go
type DocumentClassifiedEvent struct {
    DocumentID   string
    DocumentType string
    Category     string
    Confidence   float64
    ClassifiedBy string
    OccurredAt   time.Time
}
```

### DocumentProcessed

```go
type DocumentProcessedEvent struct {
    DocumentID  string
    ProcessedAt time.Time
}
```

### MetadataUpdated

```go
type MetadataUpdatedEvent struct {
    DocumentID string
    Version    int
    UpdatedAt  time.Time
}
```

## Repository Interface

```go
type DocumentRepository interface {
    // Save persists a document aggregate
    Save(ctx context.Context, document *Document) error

    // FindByID retrieves a document by its ID
    FindByID(ctx context.Context, id DocumentID) (*Document, error)

    // FindAll retrieves documents matching the filter
    FindAll(ctx context.Context, filter Filter) ([]*Document, error)

    // Delete removes a document
    Delete(ctx context.Context, id DocumentID) error

    // Exists checks if a document exists
    Exists(ctx context.Context, id DocumentID) (bool, error)

    // Count returns the number of documents matching the filter
    Count(ctx context.Context, filter Filter) (int, error)
}
```

## Filter Specification

```go
type Filter struct {
    // Pagination
    Limit  int
    Offset int

    // Filters
    DocumentType  *DocumentType
    Category      *Category
    MinConfidence *Confidence
    FromDate      *time.Time
    ToDate        *time.Time

    // Search
    TextQuery string
    LegalTags []string

    // Sorting
    SortBy    string
    SortOrder SortOrder
}
```

## Testing Guidelines

### Test Coverage

- **Value Objects**: 100% coverage (all validation paths)
- **Aggregate Methods**: 100% coverage (all business logic)
- **Invariant Enforcement**: Test all invariant violations
- **Event Emission**: Verify events are emitted correctly

### Example Test

```go
func TestDocument_ApplyClassification_Success(t *testing.T) {
    // Given: A document with text
    doc, err := NewDocument(
        WithID("doc_123"),
        WithFileName("motion.pdf"),
        WithText("Motion to suppress evidence"),
    )
    require.NoError(t, err)

    // When: Classification is applied
    classification := &Classification{
        documentType: DocTypeMotion,
        confidence:   NewConfidence(0.95),
    }
    err = doc.ApplyClassification(classification)

    // Then: Classification succeeds
    assert.NoError(t, err)
    assert.Equal(t, classification, doc.classification)

    // And: Event is emitted
    events := doc.GetEvents()
    assert.Len(t, events, 2) // Created + Classified
    assert.IsType(t, &DocumentClassifiedEvent{}, events[1])
}
```

## Migration Notes

When migrating from `pkg/models/document.go`:

1. Extract value objects from primitive types
2. Add business methods to aggregate
3. Implement invariant validation
4. Add domain events
5. Create repository interface
6. Write comprehensive tests

**Old Code** (anemic model):
```go
doc := &models.Document{
    ID:       "123",
    FileName: "test.pdf",
}
```

**New Code** (rich domain model):
```go
doc, err := document.NewDocument(
    document.WithID("123"),
    document.WithFileName("test.pdf"),
)
```
