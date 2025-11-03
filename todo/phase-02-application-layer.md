# Phase 2: Application Layer (Use Cases)

## Overview

Create the application layer that orchestrates domain objects through use cases. This layer acts as the application's "API" - defining ports (interfaces) for infrastructure and implementing business workflows.

## Objectives

- [x] Create application port interfaces (repositories, services)
- [x] Define DTOs for all operations
- [x] Implement 8 core use cases
- [x] Add application-level validation
- [x] Write use case tests with mocked ports
- [x] Zero domain logic leakage (domain handles business rules)
- [x] Achieve >90% test coverage

## Prerequisites

- **Phase 1 Complete**: Domain layer with aggregates, entities, value objects
- Understanding of Ports & Adapters pattern
- Understanding of use case/interactor pattern
- Mock framework (testify/mock)

## Current State

**Business Logic Location**: 
- In HTTP handlers (4,180 LOC)
- Mixed with HTTP concerns
- Hard to test

**Problems**:
- Can't reuse logic (tied to HTTP)
- Hard to test (need to mock HTTP)
- No clear transactions
- No DTO validation

## Target State

**Use Cases**:
- Clear, testable business workflows
- Depend on domain aggregates
- Use ports for infrastructure
- DTOs for input/output
- Transaction boundaries defined

## Detailed Task Breakdown

### Task Group 1: Application Ports

#### Task 1.1: Define repository ports
**File**: `internal/application/ports/repositories.go`

```go
package ports

import (
    "context"

    "motion-index-fiber/internal/domain/document"
    "motion-index-fiber/internal/domain/legal"
)

// DocumentRepository orchestrates persistence for the document aggregate.
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
```

**Acceptance Criteria**:
- [x] All repository interfaces defined
- [x] Context for cancellation
- [x] Return domain objects, not DTOs

---

#### Task 1.2: Define service ports
**File**: `internal/application/ports/services.go`

```go
package ports

import (
    "context"
    "io"

    "motion-index-fiber/internal/domain/classification"
    "motion-index-fiber/internal/domain/document"
)

type ClassificationService interface {
    Classify(ctx context.Context, doc *document.Document) (*classification.Result, error)
    SupportedDocumentTypes() []string
    ProviderName() string
}

type StorageService interface {
    Store(ctx context.Context, path string, content io.Reader) (string, error)
    Retrieve(ctx context.Context, path string) (io.ReadCloser, error)
    Delete(ctx context.Context, path string) error
    URL(path string) string
}

type SearchService interface {
    Index(ctx context.Context, doc *document.Document) error
    Update(ctx context.Context, id string, fields map[string]interface{}) error
    Delete(ctx context.Context, id string) error
}

type EventBus interface {
    Publish(ctx context.Context, events ...document.DomainEvent) error
}
```

**Acceptance Criteria**:
- [x] All service ports defined
- [x] Clean abstractions
- [x] No infrastructure details leaked

---

### Task Group 2: DTOs (Data Transfer Objects)

#### Task 2.1: Document DTOs
**File**: `internal/application/dto/document_dto.go`

```go
type ProcessDocumentRequest struct {
    ID            string
    FileName      string
    Content       io.Reader
    ContentSize   int64
    ContentType   string
    StoragePath   string
    HashValue     string
    HashAlgorithm string
    StoreBinary   bool
    Classify      bool
    Index         bool
    Metadata      map[string]string
    ProcessedAt   *time.Time
}

func (r *ProcessDocumentRequest) Validate() error {
    errs := validation.NewErrorSet()
    // validation using domain value objects (document.NewDocumentID etc.)
    return errs.Error()
}

type ProcessDocumentResponse struct {
    DocumentID   string
    Stored       bool
    StorageURL   string
    Classified   bool
    Indexed      bool
    CreatedAt    time.Time
    ProcessedAt  *time.Time
    EventsQueued int
}
```

**Acceptance Criteria**:
- [x] Request/Response DTOs for each operation
- [x] Validation methods
- [x] JSON tags

---

#### Task 2.2: Classification DTOs
**File**: `internal/application/dto/classification_dto.go`

```go
type ClassifyDocumentRequest struct {
    DocumentID string
    Force      bool
}

func (r *ClassifyDocumentRequest) Validate() error {
    errs := validation.NewErrorSet()
    // ensure valid document ID
    return errs.Error()
}

type ClassificationResponse struct {
    DocumentID   string
    DocumentType string
    Category     string
    Confidence   float64
    LegalTags    []string
    ClassifiedBy string
    ClassifiedAt time.Time
}
```

**Acceptance Criteria**:
- [x] DTOs created
- [x] Validation added

---

#### Task 2.3: Search DTOs
**File**: `internal/application/dto/search_dto.go`

```go
type SearchDocumentsRequest struct {
    Query         string
    DocumentType  string
    Category      string
    LegalTags     []string
    FromDate      *time.Time
    ToDate        *time.Time
    MinConfidence float64
    Page          int
    PageSize      int
    SortBy        string
    SortOrder     string
}

func (r *SearchDocumentsRequest) Validate() error {
    errs := validation.NewErrorSet()
    // pagination, confidence, dates, sorting
    return errs.Error()
}

type SearchResultsResponse struct {
    Documents  []DocumentSummary
    Total      int
    Page       int
    PageSize   int
    TotalPages int
}
```

**Acceptance Criteria**:
- [x] Search DTOs with pagination
- [x] Filtering options
- [x] Validation

---

### Task Group 3: Use Cases

#### Task 3.1: ProcessDocumentUseCase
**File**: `internal/application/usecase/document/process.go`

```go
package document

import (
    "context"
    "motion-index-fiber/internal/application/dto"
    "motion-index-fiber/internal/application/ports"
    "motion-index-fiber/internal/domain/document"
)

// ProcessDocumentUseCase orchestrates document processing
type ProcessDocumentUseCase struct {
    documentRepo DocumentRepository
    storage      StorageService
    classifier   ClassificationService
    indexer      SearchService
    eventBus     EventBus
}

// NewProcessDocumentUseCase creates the use case
func NewProcessDocumentUseCase(
    repo DocumentRepository,
    storage StorageService,
    classifier ClassificationService,
    indexer SearchService,
    eventBus EventBus,
) *ProcessDocumentUseCase {
    return &ProcessDocumentUseCase{
        documentRepo: repo,
        storage:      storage,
        classifier:   classifier,
        indexer:      indexer,
        eventBus:     eventBus,
    }
}

// Execute processes a document
func (uc *ProcessDocumentUseCase) Execute(
    ctx context.Context,
    req *dto.ProcessDocumentRequest,
) (*dto.ProcessDocumentResponse, error) {
    // 1. Validate input
    if err := req.Validate(); err != nil {
        return nil, err
    }
    
    // 2. Create domain aggregate
    doc, err := document.NewDocument(
        document.WithID(req.ID),
        document.WithFileName(req.FileName),
    )
    if err != nil {
        return nil, err
    }
    
    // 3. Store file (if requested)
    if req.Store {
        url, err := uc.storage.Store(ctx, doc.GetStoragePath(), req.Content)
        if err != nil {
            return nil, err
        }
        doc.SetStorageURL(url)
    }
    
    // 4. Classify (if requested)
    if req.Classify {
        classification, err := uc.classifier.Classify(ctx, req.Text, metadata)
        if err != nil {
            return nil, err
        }
        doc.ApplyClassification(classification)
    }
    
    // 5. Save to repository
    err = uc.documentRepo.Save(ctx, doc)
    if err != nil {
        return nil, err
    }
    
    // 6. Index for search (if requested)
    if req.Index {
        err = uc.indexer.Index(ctx, doc)
        if err != nil {
            // Log but don't fail
            log.Error("indexing failed", err)
        }
    }
    
    // 7. Publish domain events
    uc.eventBus.Publish(ctx, doc.GetEvents()...)
    
    // 8. Map to response DTO
    return mapToProcessDocumentResponse(doc), nil
}
```

**Tests**: Mock all ports, test happy path, error paths, transaction rollback

**Acceptance Criteria**:
- [x] Use case implemented
- [x] All ports used
- [x] Domain aggregate used
- [x] Events published
- [x] Tests >90% coverage

---

#### Task 3.2-3.8: Additional Use Cases
Create use cases for:
- [x] `ClassifyDocumentUseCase` - Classify an existing document *(done: `internal/application/usecase/document/classify.go`)*
- [x] `SearchDocumentsUseCase` - Search for documents *(done: `internal/application/usecase/document/search.go`)*
- [x] `IndexDocumentUseCase` - Index a document for search *(done: `internal/application/usecase/document/index.go`)*
- [x] `UpdateMetadataUseCase` - Update document metadata *(done: `internal/application/usecase/document/update_metadata.go`)*
- [x] `AnalyzeRedactionsUseCase` - Analyze PII for redaction *(done: `internal/application/usecase/redaction/analyze.go`)*
- [x] `ApplyRedactionsUseCase` - Apply redactions to PDF *(done: `internal/application/usecase/redaction/apply.go`)*
- [x] `BatchProcessUseCase` - Process multiple documents *(done: `internal/application/usecase/document/batch.go`)*

Each follows the same pattern:
1. Inject dependencies (ports) via constructor
2. Execute method with DTO request
3. Validate input
4. Use domain aggregates
5. Call ports for infrastructure
6. Publish events
7. Return DTO response

---

### Task Group 4: Validation

#### Task 4.1: DTO validators
**File**: `internal/application/validation/validators.go`

```go
package validation

import "errors"

func ValidateFileName(name string) error {
    if _, err := document.NewFileName(name); err != nil {
        return err
    }
    return nil
}

func ValidatePagination(page, size int) error {
    if page == 0 && size == 0 {
        return nil
    }
    if page <= 0 || size <= 0 {
        return errors.New("page and size must be > 0")
    }
    return nil
}
```

**Acceptance Criteria**:
- [x] Common validators
- [x] Reusable across DTOs
- [x] Clear error messages

---

## Testing Strategy

### Use Case Tests (with Mocks)

```go
func TestProcessDocumentUseCase_Execute_Success(t *testing.T) {
    // Given: Mocked dependencies
    mockRepo := new(MockDocumentRepository)
    mockStorage := new(MockStorageService)
    mockClassifier := new(MockClassificationService)
    mockIndexer := new(MockSearchService)
    mockEventBus := new(MockEventBus)
    
    // Setup expectations
    mockStorage.On("Store", mock.Anything, mock.Anything, mock.Anything).
        Return("https://storage.com/doc.pdf", nil)
    mockRepo.On("Save", mock.Anything, mock.Anything).Return(nil)
    mockIndexer.On("Index", mock.Anything, mock.Anything).Return(nil)
    mockEventBus.On("Publish", mock.Anything, mock.Anything).Return(nil)
    
    // When: Execute use case
    uc := NewProcessDocumentUseCase(mockRepo, mockStorage, mockClassifier, mockIndexer, mockEventBus)
    req := &dto.ProcessDocumentRequest{
        ID: "doc_123",
        FileName: "test.pdf",
        Store: true,
    }
    resp, err := uc.Execute(context.Background(), req)
    
    // Then: Success
    assert.NoError(t, err)
    assert.NotNil(t, resp)
    assert.Equal(t, "doc_123", resp.DocumentID)
    
    // Verify all mocks called
    mockRepo.AssertExpectations(t)
    mockStorage.AssertExpectations(t)
}
```

### Coverage Goals
- Use cases: >90%
- DTOs: 100%
- Validators: 100%

---

## Files Inventory

### Files to Create (~30 files)

| File | Purpose | LOC |
|------|---------|-----|
| `ports/repositories.go` | Repository interfaces | ~200 |
| `ports/services.go` | Service interfaces | ~200 |
| `dto/document_dto.go` | Document DTOs | ~300 |
| `dto/classification_dto.go` | Classification DTOs | ~200 |
| `dto/search_dto.go` | Search DTOs | ~250 |
| `usecases/document/process_document.go` | Process use case | ~250 |
| `usecases/document/classify_document.go` | Classify use case | ~150 |
| `usecases/document/search_documents.go` | Search use case | ~200 |
| `usecases/document/update_metadata.go` | Update use case | ~100 |
| `validation/validators.go` | Common validators | ~150 |
| Plus 8 test files | Tests | ~2000 |

**Total**: ~20 files, ~4,200 LOC

---

## Acceptance Criteria

- [ ] All ports defined (repositories, services)
- [ ] All DTOs created with validation
- [ ] 8 use cases implemented
- [ ] All use cases tested with mocks (>90% coverage)
- [ ] No infrastructure dependencies
- [ ] No domain logic in use cases (orchestration only)

## Success Metrics

- Use case test coverage: >90%
- DTO validation coverage: 100%
- Ports clearly defined: 5-8 interfaces
- Use cases: 8 implemented

---

**Phase Status**: 🟢 Completed
**Dependencies**: Phase 1 complete
**Next Phase**: Phase 3 (Configuration)
