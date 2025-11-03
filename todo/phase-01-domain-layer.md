# Phase 1: Domain Layer Foundation

## Overview

Create the pure domain model with aggregates, entities, and value objects. This is the foundation of the entire SOLID redesign - the domain layer contains all business logic and rules with zero infrastructure dependencies.

## Objectives

- [x] Create domain errors package with all error types
- [x] Implement document value objects (FilePath, S3URI, Hash, etc.)
- [x] Build Document aggregate with business methods
- [x] Create Legal domain entities (Case, Court, Party, Attorney)
- [x] Define Classification domain service interface
- [x] Implement domain events
- [x] Achieve 100% test coverage on domain layer
- [x] Zero infrastructure dependencies

## Prerequisites

- Go 1.24+ installed
- Understanding of DDD concepts (aggregates, entities, value objects)
- Understanding of SOLID principles
- Testify testing framework

## Current State Analysis

**Existing Domain Models** (anemic):
- `pkg/models/document.go` - 546 LOC, mostly data structure
- `pkg/models/api.go` - API models mixed with domain
- No clear aggregate boundaries
- No invariant enforcement
- No domain events

**Problems**:
- Business logic scattered in handlers
- Direct field access without validation
- Primitive obsession (strings for everything)
- No encapsulation

## Target State

**Rich Domain Models**:
- Document aggregate enforcing invariants
- Value objects for type safety
- Domain events for state changes
- 100% test coverage
- Pure Go (no external dependencies)

## Detailed Task Breakdown

### Task Group 1: Domain Errors (Priority: HIGH)

#### Task 1.1: Create base domain errors
**File**: `internal/domain/errors/errors.go`

```go
package errors

import "errors"

// Domain errors
var (
    // Document errors
    ErrDocumentNotFound       = errors.New("document not found")
    ErrEmptyDocumentID        = errors.New("document ID cannot be empty")
    ErrEmptyFileName          = errors.New("file name cannot be empty")
    ErrInvalidFilePath        = errors.New("invalid file path")
    ErrInvalidS3URI           = errors.New("invalid S3 URI")
    ErrEmptyHash              = errors.New("hash cannot be empty")
    ErrInvalidContentType     = errors.New("invalid content type")
    ErrInvalidFileSize        = errors.New("invalid file size")
    ErrCannotClassifyEmptyDocument = errors.New("cannot classify document without content")

    // Classification errors
    ErrInvalidConfidence      = errors.New("confidence must be between 0.0 and 1.0")
    ErrInvalidClassification  = errors.New("invalid classification")
    ErrClassifierNotConfigured = errors.New("classifier not configured")

    // Legal entity errors
    ErrInvalidCaseNumber      = errors.New("invalid case number format")
    ErrInvalidPartyRole       = errors.New("invalid party role")
    ErrInvalidBarNumber       = errors.New("invalid bar number")

    // Repository errors
    ErrDuplicateDocument      = errors.New("document already exists")
    ErrConcurrencyConflict    = errors.New("document was modified by another process")
)
```

**Tests to write**:
- Test error messages
- Test error wrapping
- Test error comparison

**Acceptance Criteria**:
- [x] All domain errors defined
- [x] Error tests passing
- [x] Errors documented

---

#### Task 1.2: Create custom error types
**File**: `internal/domain/errors/errors.go` (append)

```go
// ValidationError represents a validation failure
type ValidationError struct {
    Field   string
    Message string
    Value   interface{}
}

func (e *ValidationError) Error() string {
    return fmt.Sprintf("validation failed for %s: %s", e.Field, e.Message)
}

// NewValidationError creates a new validation error
func NewValidationError(field, message string, value interface{}) *ValidationError {
    return &ValidationError{
        Field:   field,
        Message: message,
        Value:   value,
    }
}
```

**Acceptance Criteria**:
- [x] Custom error types created
- [x] Error constructors added
- [x] Tests passing

---

### Task Group 2: Document Value Objects (Priority: HIGH)

#### Task 2.1: Implement DocumentID value object
**File**: `internal/domain/document/value_objects.go`

```go
package document

import (
    "fmt"
    "motion-index-fiber/internal/domain/errors"
)

// DocumentID represents a unique document identifier
type DocumentID struct {
    value string
}

// NewDocumentID creates a new DocumentID
func NewDocumentID(id string) (DocumentID, error) {
    if id == "" {
        return DocumentID{}, errors.ErrEmptyDocumentID
    }
    return DocumentID{value: id}, nil
}

// String returns the string representation
func (d DocumentID) String() string {
    return d.value
}

// Equals checks if two DocumentIDs are equal
func (d DocumentID) Equals(other DocumentID) bool {
    return d.value == other.value
}

// IsEmpty checks if the ID is empty
func (d DocumentID) IsEmpty() bool {
    return d.value == ""
}
```

**Tests**:
```go
func TestNewDocumentID_Success(t *testing.T) {
    id, err := NewDocumentID("doc_123")
    assert.NoError(t, err)
    assert.Equal(t, "doc_123", id.String())
}

func TestNewDocumentID_EmptyString(t *testing.T) {
    _, err := NewDocumentID("")
    assert.ErrorIs(t, err, errors.ErrEmptyDocumentID)
}
```

**Acceptance Criteria**:
- [x] DocumentID value object created
- [x] Validation enforced
- [x] Tests passing (100% coverage)

---

#### Task 2.2: Implement FilePath value object
**File**: `internal/domain/document/value_objects.go` (append)

```go
// FilePath represents a storage path
type FilePath struct {
    value string
}

// NewFilePath creates a validated FilePath
func NewFilePath(path string) (FilePath, error) {
    if path == "" {
        return FilePath{}, errors.ErrEmptyFilePath
    }

    // Validate storage path format
    if !isValidStoragePath(path) {
        return FilePath{}, errors.ErrInvalidFilePath
    }

    return FilePath{value: path}, nil
}

func isValidStoragePath(path string) bool {
    // Must start with "documents/"
    // No double slashes
    // No path traversal (..)
    if !strings.HasPrefix(path, "documents/") {
        return false
    }
    if strings.Contains(path, "//") || strings.Contains(path, "..") {
        return false
    }
    return true
}

func (f FilePath) String() string {
    return f.value
}
```

**Acceptance Criteria**:
- [x] FilePath validation implemented
- [x] Path format enforced
- [x] Tests passing

---

#### Task 2.3: Implement S3URI value object
**File**: `internal/domain/document/value_objects.go` (append)

```go
// S3URI represents an S3 URI (s3://bucket/key)
type S3URI struct {
    bucket string
    key    string
}

// NewS3URI creates a validated S3URI
func NewS3URI(bucket, key string) (S3URI, error) {
    if bucket == "" || key == "" {
        return S3URI{}, errors.ErrInvalidS3URI
    }
    return S3URI{bucket: bucket, key: key}, nil
}

// ParseS3URI parses an S3 URI string
func ParseS3URI(uri string) (S3URI, error) {
    // Parse "s3://bucket/key" format
    if !strings.HasPrefix(uri, "s3://") {
        return S3URI{}, errors.ErrInvalidS3URI
    }

    parts := strings.SplitN(uri[5:], "/", 2)
    if len(parts) != 2 {
        return S3URI{}, errors.ErrInvalidS3URI
    }

    return NewS3URI(parts[0], parts[1])
}

func (s S3URI) String() string {
    return fmt.Sprintf("s3://%s/%s", s.bucket, s.key)
}

func (s S3URI) Bucket() string {
    return s.bucket
}

func (s S3URI) Key() string {
    return s.key
}
```

**Acceptance Criteria**:
- [x] S3URI parsing works
- [x] Bucket and key extracted
- [x] Tests passing

---

#### Task 2.4: Implement Hash value object
**File**: `internal/domain/document/value_objects.go` (append)

```go
// Hash represents a document content hash
type Hash struct {
    value     string
    algorithm string // "SHA256", "MD5", etc.
}

// NewHash creates a validated Hash
func NewHash(value, algorithm string) (Hash, error) {
    if value == "" {
        return Hash{}, errors.ErrEmptyHash
    }

    // Validate hash format based on algorithm
    if !isValidHashFormat(value, algorithm) {
        return Hash{}, errors.ErrInvalidHash
    }

    return Hash{value: value, algorithm: algorithm}, nil
}

func isValidHashFormat(value, algorithm string) bool {
    switch algorithm {
    case "SHA256":
        return len(value) == 64 && isHexString(value)
    case "MD5":
        return len(value) == 32 && isHexString(value)
    default:
        return len(value) > 0
    }
}

func (h Hash) String() string {
    return h.value
}

func (h Hash) Algorithm() string {
    return h.algorithm
}
```

**Acceptance Criteria**:
- [x] Hash validation by algorithm
- [x] Multiple algorithms supported
- [x] Tests passing

---

#### Task 2.5: Implement Confidence value object
**File**: `internal/domain/document/value_objects.go` (append)

```go
// Confidence represents classification confidence (0.0 - 1.0)
type Confidence float64

// NewConfidence creates a validated Confidence
func NewConfidence(value float64) (Confidence, error) {
    if value < 0.0 || value > 1.0 {
        return 0, errors.ErrInvalidConfidence
    }
    return Confidence(value), nil
}

func (c Confidence) Value() float64 {
    return float64(c)
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

func (c Confidence) IsValid() bool {
    return c >= 0.0 && c <= 1.0
}
```

**Acceptance Criteria**:
- [x] Confidence range validated (0.0-1.0)
- [x] Helper methods (IsHigh, IsMedium, IsLow)
- [x] Tests passing

---

### Task Group 3: Document Entity (Priority: HIGH)

#### Task 3.1: Create Document entity structure
**File**: `internal/domain/document/entity.go`

```go
package document

import "time"

// Document is the aggregate root for document management
type Document struct {
    // Identity
    id      DocumentID
    version int // For optimistic locking

    // File Information
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

    // Classification
    classification *Classification

    // Metadata
    metadata *Metadata

    // Timestamps
    createdAt   time.Time
    updatedAt   time.Time
    processedAt *time.Time

    // Domain Events
    events []DomainEvent
}
```

**Acceptance Criteria**:
- [x] Private fields (encapsulation)
- [x] Domain events array
- [x] Version for optimistic locking

---

#### Task 3.2: Implement Document getters
**File**: `internal/domain/document/entity.go` (append)

```go
// Getters (read-only access)
func (d *Document) ID() DocumentID           { return d.id }
func (d *Document) Version() int             { return d.version }
func (d *Document) FileName() FileName       { return d.fileName }
func (d *Document) FilePath() FilePath       { return d.filePath }
func (d *Document) S3URI() S3URI             { return d.s3URI }
func (d *Document) Hash() Hash               { return d.hash }
func (d *Document) Classification() *Classification { return d.classification }
func (d *Document) CreatedAt() time.Time     { return d.createdAt }
func (d *Document) UpdatedAt() time.Time     { return d.updatedAt }
func (d *Document) ProcessedAt() *time.Time  { return d.processedAt }
```

**Acceptance Criteria**:
- [x] All getters implemented
- [x] Read-only access enforced

---

### Task Group 4: Document Aggregate (Priority: HIGH)

#### Task 4.1: Implement Document factory
**File**: `internal/domain/document/aggregate.go`

```go
package document

import (
    "time"
    domainerrors "motion-index-fiber/internal/domain/errors"
)

// DocumentOption is a functional option for Document creation
type DocumentOption func(*Document) error

// NewDocument creates a new Document with options
func NewDocument(opts ...DocumentOption) (*Document, error) {
    doc := &Document{
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

// Functional options
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

**Acceptance Criteria**:
- [x] Factory with functional options
- [x] Invariant validation on creation
- [x] Domain event emitted
- [x] Tests passing

---

#### Task 4.2: Implement ApplyClassification business method
**File**: `internal/domain/document/aggregate.go` (append)

```go
// ApplyClassification applies a classification to the document
func (d *Document) ApplyClassification(classification *Classification) error {
    // Business rule: Can't classify without content
    if d.text.IsEmpty() {
        return domainerrors.ErrCannotClassifyEmptyDocument
    }

    // Business rule: Confidence must be valid
    if !classification.confidence.IsValid() {
        return domainerrors.ErrInvalidConfidence
    }

    d.classification = classification
    d.updatedAt = time.Now()
    d.version++

    // Emit domain event
    d.addEvent(NewDocumentClassifiedEvent(d, classification))

    return nil
}
```

**Acceptance Criteria**:
- [x] Business rules enforced
- [x] Version incremented
- [x] Event emitted
- [x] Tests passing

---

#### Task 4.3: Implement UpdateMetadata business method
**File**: `internal/domain/document/aggregate.go` (append)

```go
// UpdateMetadata updates the document metadata
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

**Acceptance Criteria**:
- [x] Metadata validated
- [x] Version incremented
- [x] Event emitted

---

#### Task 4.4: Implement MarkAsProcessed business method
**File**: `internal/domain/document/aggregate.go` (append)

```go
// MarkAsProcessed marks the document as processed
func (d *Document) MarkAsProcessed() {
    now := time.Now()
    d.processedAt = &now
    d.updatedAt = now
    d.version++

    // Emit domain event
    d.addEvent(NewDocumentProcessedEvent(d))
}
```

**Acceptance Criteria**:
- [x] ProcessedAt timestamp set
- [x] Event emitted

---

#### Task 4.5: Implement invariant validation
**File**: `internal/domain/document/aggregate.go` (append)

```go
// validate checks all business invariants
func (d *Document) validate() error {
    if d.id.IsEmpty() {
        return domainerrors.ErrEmptyDocumentID
    }

    if d.fileName.IsEmpty() {
        return domainerrors.ErrEmptyFileName
    }

    if d.classification != nil {
        if !d.classification.confidence.IsValid() {
            return domainerrors.ErrInvalidConfidence
        }
    }

    return nil
}
```

**Acceptance Criteria**:
- [x] All invariants checked
- [x] Validation tests passing

---

### Task Group 5: Domain Events (Priority: MEDIUM)

#### Task 5.1: Create domain event interface
**File**: `internal/domain/document/events.go`

```go
package document

import "time"

// DomainEvent is the interface for all domain events
type DomainEvent interface {
    OccurredAt() time.Time
    AggregateID() string
    EventType() string
}
```

---

#### Task 5.2: Implement DocumentCreatedEvent
**File**: `internal/domain/document/events.go` (append)

```go
// DocumentCreatedEvent is emitted when a document is created
type DocumentCreatedEvent struct {
    DocumentID  string
    FileName    string
    ContentType string
    occurredAt  time.Time
}

func NewDocumentCreatedEvent(doc *Document) *DocumentCreatedEvent {
    return &DocumentCreatedEvent{
        DocumentID:  doc.id.String(),
        FileName:    doc.fileName.String(),
        ContentType: doc.contentType.String(),
        occurredAt:  time.Now(),
    }
}

func (e *DocumentCreatedEvent) OccurredAt() time.Time { return e.occurredAt }
func (e *DocumentCreatedEvent) AggregateID() string   { return e.DocumentID }
func (e *DocumentCreatedEvent) EventType() string     { return "DocumentCreated" }
```

**Acceptance Criteria**:
- [x] Event struct created
- [x] Constructor implemented
- [x] Interface methods implemented

---

#### Task 5.3: Implement DocumentClassifiedEvent
**File**: `internal/domain/document/events.go` (append)

```go
// DocumentClassifiedEvent is emitted when a document is classified
type DocumentClassifiedEvent struct {
    DocumentID   string
    DocumentType string
    Category     string
    Confidence   float64
    ClassifiedBy string
    occurredAt   time.Time
}

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

func (e *DocumentClassifiedEvent) OccurredAt() time.Time { return e.occurredAt }
func (e *DocumentClassifiedEvent) AggregateID() string   { return e.DocumentID }
func (e *DocumentClassifiedEvent) EventType() string     { return "DocumentClassified" }
```

**Acceptance Criteria**:
- [x] Event with classification data
- [x] Tests passing

---

### Task Group 6: Repository Interface (Priority: MEDIUM)

#### Task 6.1: Define DocumentRepository interface
**File**: `internal/domain/document/repository.go`

```go
package document

import "context"

// DocumentRepository defines the interface for document persistence
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

**Acceptance Criteria**:
- [x] Interface defined
- [x] All CRUD operations included
- [x] Context for cancellation

---

#### Task 6.2: Define Filter specification
**File**: `internal/domain/document/repository.go` (append)

```go
// Filter specifies criteria for querying documents
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

type SortOrder string

const (
    SortAsc  SortOrder = "asc"
    SortDesc SortOrder = "desc"
)
```

**Acceptance Criteria**:
- [x] Filter specification complete
- [x] Pagination support
- [x] Sorting support

---

### Task Group 7: Testing (Priority: HIGH)

#### Task 7.1: Write value object tests
**File**: `internal/domain/document/value_objects_test.go`

Test every value object:
- Valid creation
- Invalid creation (each validation rule)
- Equality checks
- String representation
- Helper methods

**Target**: 100% coverage

---

#### Task 7.2: Write aggregate tests
**File**: `internal/domain/document/aggregate_test.go`

Test scenarios:
- Document creation with options
- ApplyClassification success
- ApplyClassification failures (no content, invalid confidence)
- UpdateMetadata success
- UpdateMetadata validation failures
- MarkAsProcessed
- Event emission
- Version incrementing
- Invariant enforcement

**Target**: 100% coverage

---

#### Task 7.3: Write event tests
**File**: `internal/domain/document/events_test.go`

Test each event:
- Event creation
- Event data correctness
- Interface implementation

**Target**: 100% coverage

---

## Files Inventory

### Files to Create

| File | Purpose | Estimated LOC |
|------|---------|---------------|
| `errors/errors.go` | Domain errors | ~150 |
| `errors/errors_test.go` | Error tests | ~200 |
| `document/value_objects.go` | All value objects | ~400 |
| `document/value_objects_test.go` | VO tests | ~500 |
| `document/entity.go` | Document entity | ~200 |
| `document/aggregate.go` | Business logic | ~350 |
| `document/aggregate_test.go` | Aggregate tests | ~800 |
| `document/events.go` | Domain events | ~150 |
| `document/events_test.go` | Event tests | ~200 |
| `document/repository.go` | Repository interface | ~100 |
| `legal/case.go` | Case aggregate | ~200 |
| `legal/court.go` | Court value object | ~100 |
| `legal/party.go` | Party entity | ~150 |
| `legal/attorney.go` | Attorney entity | ~150 |
| `legal/legal_test.go` | Legal tests | ~400 |
| `classification/classifier.go` | Classifier interface | ~100 |
| `classification/result.go` | Classification VO | ~150 |

**Total**: ~17 files, ~4,200 LOC

### Files to Reference (No Changes)

- `pkg/models/document.go` - For migration reference
- `pkg/models/legal.go` - For legal entity structure

## Testing Strategy

### Unit Tests
- Test all value objects (validation, equality, string representation)
- Test all aggregate methods (business logic, invariants, events)
- Test domain events (creation, data correctness)
- Use table-driven tests for multiple scenarios

### Coverage Goals
- **Value Objects**: 100% coverage
- **Aggregates**: 100% coverage
- **Events**: 100% coverage
- **Overall Domain**: 100% coverage

### Test Structure
```go
func TestNewDocument_Success(t *testing.T) {
    // Given
    opts := []DocumentOption{
        WithID("doc_123"),
        WithFileName("test.pdf"),
    }

    // When
    doc, err := NewDocument(opts...)

    // Then
    assert.NoError(t, err)
    assert.Equal(t, "doc_123", doc.ID().String())
    assert.Len(t, doc.GetEvents(), 1) // DocumentCreatedEvent
}
```

## Acceptance Criteria

### Must Have
- [x] All value objects created with validation
- [x] Document aggregate with business methods
- [x] Domain events for state changes
- [x] Repository interface defined
- [x] 100% test coverage on domain layer
- [x] Zero infrastructure dependencies
- [x] All tests passing
- [x] CLAUDE.md updated with examples

### Nice to Have
- [ ] Benchmark tests for performance
- [ ] Property-based tests for value objects
- [ ] Documentation examples

## Risk Assessment

### High Risk
- **Complexity**: Rich domain models can be complex
  - **Mitigation**: Start simple, add complexity incrementally

- **Over-engineering**: Too many value objects
  - **Mitigation**: Only create VOs for complex concepts, not everything

### Medium Risk
- **Learning curve**: Team may not know DDD
  - **Mitigation**: Pair programming, code reviews, training

### Low Risk
- **Pure Go**: No external dependencies means no version conflicts

## Dependencies

### Internal Dependencies
- None (this is the foundation)

### External Dependencies
- Go standard library only
- `testify` for testing (dev dependency)

## Rollback Strategy

If domain layer causes issues:
1. Keep old `pkg/models` in place
2. Don't replace anything yet
3. Domain layer is additive only
4. Can be removed without affecting existing code

## Success Metrics

### Code Quality
- Test coverage: 100%
- Cyclomatic complexity: <10 per function
- No infrastructure dependencies: 0

### Design
- Value objects: 8+
- Aggregates: 2 (Document, Legal)
- Domain events: 4+
- Repository interfaces: 2

### Team
- Code reviews: 100% reviewed before merge
- Understanding: All team members understand DDD concepts

## Notes

- **Start Small**: Begin with DocumentID, FileName value objects
- **Iterate**: Don't try to create everything at once
- **TDD**: Write tests first, then implementation
- **Review Often**: Get feedback early and often
- **Document**: Update CLAUDE.md with examples as you go

## Next Phase

Once Phase 1 is complete:
- **Phase 2**: Application layer (use cases)
- Use domain aggregates in use cases
- Define application ports
- Create DTOs

---

**Phase Status**: 🟡 In Progress
**Last Updated**: 2025-10-27
**Next Task**: Create domain errors package
