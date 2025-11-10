# Models Quick Reference Card

## Where Are Models Located?

```
pkg/models/
├── core/           # 🎯 Core domain models (Document, legal types)
├── api/            # 🌐 HTTP request/response DTOs
├── search/         # 🔍 Search and aggregation models
├── validation/     # ✓ Input validation and sanitization
└── time.go         # 📅 DateRange and time utilities

internal/domain/    # 🧠 DDD layer - Rich domain logic
├── document/       # Document aggregate, value objects, events
├── legal/          # Legal entities (Case, Party, Attorney)
├── classification/ # Classification service contracts
└── errors/         # Domain-specific errors

internal/application/dto/  # 🔄 Application layer DTOs
├── document.go     # Document use case DTOs
├── classification.go # Classification use case DTOs
└── search.go       # Search use case DTOs
```

---

## Core Models (Most Common)

### Document & Metadata

```go
// Import from pkg/models/core
import "github.com/motion-index-fiber/pkg/models/core"

// Create a document
doc := &core.Document{
    ID:          "doc_123",
    FileName:    "motion.pdf",
    FilePath:    "documents/2024/05/motion.pdf",
    Text:        "Motion to suppress evidence...",
    DocType:     "motion",
    Category:    "filing",
    Hash:        "sha256hash...",
    ContentType: "application/pdf",
    CreatedAt:   time.Now(),
}

// Access metadata
if doc.Metadata != nil {
    caseNum := doc.GetCaseNumber()      // Helper method
    caseName := doc.GetCaseName()       // Helper method
    judgeName := doc.GetJudgeName()     // Helper method
}

// Manage legal tags
doc.AddLegalTag("suppression_motion")
if doc.HasLegalTag("suppression_motion") {
    // do something
}
```

### Legal Types

```go
import "github.com/motion-index-fiber/pkg/models/core"

// Case information
caseInfo := &core.CaseInfo{
    Number: "2024-CR-12345",
    Name:   "State v. Doe",
    Type:   "criminal",
}

// Party
party := &core.Party{
    Name: "John Doe",
    Role: "defendant",
    Type: "individual",
}

// Attorney
attorney := &core.Attorney{
    Name:      "Jane Smith",
    BarNumber: "CA123456",
    Role:      "defense_counsel",
}

// Court
court := &core.CourtInfo{
    Name:  "Superior Court of California",
    Level: "trial",
    Judge: "Hon. Robert Jones",
}
```

### Document Types & Enums

```go
import "github.com/motion-index-fiber/pkg/models/core"

// DocumentType constants
const (
    // Motions
    DocTypeMotion                      core.DocumentType = "motion"
    DocTypeMotionToSuppressEvidence    core.DocumentType = "motion_to_suppress_evidence"
    DocTypeMotionToDismiss             core.DocumentType = "motion_to_dismiss"
    
    // Orders
    DocTypeOrder                       core.DocumentType = "order"
    DocTypeOrderToDismiss              core.DocumentType = "order_to_dismiss"
    
    // Pleadings
    DocTypeComplaint                   core.DocumentType = "complaint"
    DocTypeIndictment                  core.DocumentType = "indictment"
)

// Check type using helper methods
docType := core.DocTypeMotion
if docType.IsMotion() {
    // Handle motion
}
if docType.IsOrder() {
    // Handle order
}

// Category
category := core.CategoryFiling
category := core.CategoryOrder
category := core.CategoryMotion

// Confidence (0.0 - 1.0)
conf := core.Confidence(0.95)
if conf.IsHigh() {
    // High confidence
}
if conf.IsMedium() {
    // Medium confidence
}
```

---

## API Models

### Requests

```go
import "github.com/motion-index-fiber/pkg/models/api"

// Process document
req := &api.ProcessDocumentRequest{
    FileName:    "motion.pdf",
    FilePath:    "documents/2024/05/motion.pdf",
    ContentType: "application/pdf",
    Metadata: map[string]interface{}{
        "case_number": "2024-CR-12345",
    },
}

// Validate request
if err := req.Validate(); err != nil {
    log.Fatalf("Invalid request: %v", err)
}

// Search documents
searchReq := &api.SearchDocumentsRequest{
    Query:    "motion to suppress",
    DocType:  "motion",
    Size:     10,
    From:     0,
}

// Classify document
classifyReq := &api.ClassifyDocumentRequest{
    DocumentID: "doc_123",
    Text:       "Full document text...",
    Metadata: map[string]interface{}{
        "filing_date": "2024-03-15",
    },
}
```

### Responses

```go
import "github.com/motion-index-fiber/pkg/models/api"

// Standard API response
resp := &api.APIResponse{
    Success:   true,
    Message:   "Document processed successfully",
    Data:      processedDoc,
    RequestID: "req_123",
    Timestamp: time.Now(),
}

// Error response
errResp := &api.APIResponse{
    Success: false,
    Error: &api.APIError{
        Code:    "VALIDATION_ERROR",
        Message: "Invalid document format",
        Details: map[string]interface{}{
            "field":  "content_type",
            "reason": "Must be PDF, DOCX, or TXT",
        },
    },
    Timestamp: time.Now(),
}

// Processing response
procResp := &api.ProcessDocumentResponse{
    DocumentID: "doc_123",
    FileName:   "motion.pdf",
    Status:     "completed",
    Extraction: &api.ExtractionResult{
        Text:      "Extracted text...",
        PageCount: 5,
        WordCount: 1250,
    },
    Classification: &api.ClassificationResult{
        DocumentType: "motion",
        Category:     "filing",
        Confidence:   0.95,
        LegalTags:    []string{"suppression", "evidence"},
    },
    ProcessedAt: time.Now(),
}

// Search response
searchResp := &api.SearchDocumentsResponse{
    Query:     "motion to suppress",
    TotalHits: 42,
    Results: []api.SearchDocument{
        {
            Document:  doc1,
            Score:     0.95,
            Highlights: map[string][]string{
                "text": {"<em>motion</em> to <em>suppress</em>..."},
            },
        },
    },
    Page:     1,
    PageSize: 10,
}
```

---

## Search Models

```go
import "github.com/motion-index-fiber/pkg/models/search"

// SearchRequest (same as api.SearchDocumentsRequest)
req := &search.SearchRequest{
    Query:     "suppress evidence",
    DocType:   "motion",
    Size:      20,
    From:      0,
}

// SearchResult
result := &search.SearchResult{
    ID:        "doc_123",
    Score:     0.95,
    Highlights: map[string][]string{},
}

// Aggregations
agg := &search.AggregationResponse{
    Aggregations: map[string]interface{}{
        "by_type": map[string]int64{
            "motion": 42,
            "order":  28,
        },
    },
}

// Bulk operations
bulkResult := &search.BulkResult{
    Took:   150,
    Errors: false,
    Items:  []search.BulkResultItem{},
}
```

---

## Domain Models (Rich Domain Logic)

```go
import "github.com/motion-index-fiber/internal/domain/document"

// Create document with validation (functional options pattern)
doc, err := document.NewDocument(
    document.WithID("doc_123"),
    document.WithFileName("motion.pdf"),
    document.WithFilePath("documents/2024/05/motion.pdf"),
    document.WithContentType("application/pdf"),
    document.WithHash("sha256hash...", "SHA256"),
    document.WithFileSize(2048),
    document.WithText("Motion to suppress evidence"),
)
if err != nil {
    log.Fatalf("Failed to create document: %v", err)
}

// Apply classification
classification, err := document.NewClassification(
    document.DocTypeMotion,
    document.CategoryFiling,
    document.Confidence(0.95),
)
if err != nil {
    log.Fatalf("Invalid classification: %v", err)
}

err = doc.ApplyClassification(classification)
if err != nil {
    log.Fatalf("Failed to classify: %v", err)
}

// Get domain events
events := doc.GetEvents()
for _, event := range events {
    // Handle domain events
    log.Printf("Event: %T", event)
}
```

---

## Validation Models

```go
import "github.com/motion-index-fiber/pkg/models/validation"

// File validation rules
rules := &validation.FileValidationRules{
    AllowedMimeTypes: []string{"application/pdf", "application/vnd.openxmlformats-officedocument.wordprocessingml.document"},
    MaxFileSize:      10485760, // 10 MB
    AllowedExtensions: []string{".pdf", ".docx", ".txt"},
}

// Validate file
if !rules.IsValidMimeType("application/pdf") {
    log.Fatal("Invalid MIME type")
}

// Sanitize input
sanitized := validation.SanitizeInput(userInput)

// Prevent XSS
clean := validation.PreventXSS(userInput)
```

---

## Time Models

```go
import "github.com/motion-index-fiber/pkg/models"

// DateRange for filtering
dr := &models.DateRange{
    From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
    To:   time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC),
}

// Check if date is in range
inRange := dr.Contains(time.Now())

// Backward compatible field names
dr.Start = dr.From  // Alias
dr.End = dr.To      // Alias
```

---

## Application Layer DTOs

```go
import "github.com/motion-index-fiber/internal/application/dto"

// Document DTO with validation
docDTO := &dto.ProcessDocumentRequest{
    FileName:    "motion.pdf",
    FilePath:    "documents/motion.pdf",
    ContentType: "application/pdf",
}

if err := docDTO.Validate(); err != nil {
    log.Fatalf("Invalid DTO: %v", err)
}

// Classification DTO
classifyDTO := &dto.ClassifyDocumentRequest{
    DocumentID: "doc_123",
    Text:       "Full text...",
    Metadata: map[string]interface{}{
        "filing_date": "2024-03-15",
    },
}

// Search DTO
searchDTO := &dto.SearchDocumentsRequest{
    Query: "suppress",
    Size:  10,
    From:  0,
}
```

---

## Domain Layer (DDD Concepts)

### Value Objects (Type-Safe Primitives)

```go
import "github.com/motion-index-fiber/internal/domain/document"

// DocumentID (validated, immutable)
docID, err := document.NewDocumentID("doc_123")
if err != nil {
    log.Fatalf("Invalid document ID: %v", err)
}

// FilePath (validated storage path)
path, err := document.NewFilePath("documents/2024/05/motion.pdf")
if err != nil {
    log.Fatalf("Invalid file path: %v", err)
}

// Hash (with algorithm tracking)
hash, err := document.NewHash("sha256hash...", "SHA256")
if err != nil {
    log.Fatalf("Invalid hash: %v", err)
}

// S3URI (cloud storage path)
uri, err := document.NewS3URI("my-bucket", "documents/motion.pdf")
if err != nil {
    log.Fatalf("Invalid S3 URI: %v", err)
}
```

### Domain Events

```go
import "github.com/motion-index-fiber/internal/domain/document"

// Events are emitted automatically when using aggregate methods
doc, _ := document.NewDocument(
    document.WithID("doc_123"),
    document.WithFileName("motion.pdf"),
)

events := doc.GetEvents()
for _, event := range events {
    switch e := event.(type) {
    case *document.DocumentCreatedEvent:
        log.Printf("Document created: %s", e.DocumentID)
    case *document.DocumentClassifiedEvent:
        log.Printf("Document classified: %s with confidence %f", 
            e.DocumentID, e.Confidence)
    case *document.DocumentProcessedEvent:
        log.Printf("Document processed: %s", e.DocumentID)
    }
}
```

---

## Dependency Injection Pattern

```go
import (
    "github.com/motion-index-fiber/pkg/models/core"
    "github.com/motion-index-fiber/pkg/models/api"
)

// Handler with injected dependencies
type DocumentHandler struct {
    service DocumentService
    logger  Logger
}

func NewDocumentHandler(service DocumentService, logger Logger) *DocumentHandler {
    return &DocumentHandler{
        service: service,
        logger:  logger,
    }
}

func (h *DocumentHandler) ProcessDocument(req *api.ProcessDocumentRequest) (*api.ProcessDocumentResponse, error) {
    // Validate request
    if err := req.Validate(); err != nil {
        h.logger.Error("Invalid request", err)
        return nil, err
    }
    
    // Call service (which uses domain models)
    result, err := h.service.ProcessDocument(req)
    if err != nil {
        h.logger.Error("Processing failed", err)
        return nil, err
    }
    
    // Return response
    return result, nil
}
```

---

## Common Patterns

### Creating Models

```go
// ✅ Good: Use constructors with validation
doc, err := document.NewDocument(
    document.WithID("doc_123"),
    document.WithFileName("motion.pdf"),
)
if err != nil {
    // Handle validation error
}

// ❌ Bad: Direct struct creation (bypasses validation)
doc := &Document{ID: "doc_123"}

// ✅ Good: Use request DTOs in handlers
processReq := &api.ProcessDocumentRequest{...}
if err := processReq.Validate(); err != nil {
    // Return validation error
}

// ❌ Bad: Validate after using data
process(req.FilePath)
if err := req.Validate(); err != nil {
    // Too late, already used invalid data
}
```

### Handling Errors

```go
import "github.com/motion-index-fiber/internal/domain/errors"

// Domain errors (from domain layer)
doc, err := document.NewDocument(opts...)
if err != nil {
    // Could be: ErrEmptyDocumentID, ErrInvalidFileName, etc.
    if errors.Is(err, document.ErrEmptyDocumentID) {
        // Handle missing ID
    }
}

// API errors (in handlers)
if err := req.Validate(); err != nil {
    return c.JSON(fiber.StatusBadRequest, &api.APIResponse{
        Success: false,
        Error: &api.APIError{
            Code:    "VALIDATION_ERROR",
            Message: err.Error(),
        },
    })
}
```

### Working with Metadata

```go
// Initialize metadata if needed
if doc.Metadata == nil {
    doc.Metadata = &core.DocumentMetadata{}
}

// Set nested objects
doc.Metadata.Case = &core.CaseInfo{
    Number: "2024-CR-12345",
    Name:   "State v. Doe",
}

// Use helper methods
caseName := doc.GetCaseName()     // Returns "" if nil
caseNum := doc.GetCaseNumber()    // Returns "" if nil
judgeName := doc.GetJudgeName()   // Returns "" if nil
```

---

## Anti-Patterns to Avoid

```go
// ❌ Primitive obsession: Use value objects!
docID := "123" // Bad: plain string

// ✅ Good: Use validated value objects
docID, _ := document.NewDocumentID("123")

// ❌ Leaky abstractions: Domain logic in handlers
if doc.Confidence > 0.8 {
    // Business logic in wrong layer
}

// ✅ Good: Use domain methods
if doc.Classification.Confidence.IsHigh() {
    // Business logic in domain layer
}

// ❌ Silent failures: Always handle errors
_ = doc.ApplyClassification(classification)

// ✅ Good: Check errors
if err := doc.ApplyClassification(classification); err != nil {
    log.Fatalf("Failed to classify: %v", err)
}

// ❌ Circular dependencies: Handler → Service → Handler
// Solution: Restructure to: Handler → Service → Repository

// ❌ God objects: One model doing everything
// Solution: Break into smaller focused types
```

---

## Testing Models

```go
import (
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/motion-index-fiber/internal/domain/document"
)

func TestNewDocument(t *testing.T) {
    doc, err := document.NewDocument(
        document.WithID("doc_123"),
        document.WithFileName("motion.pdf"),
    )
    
    assert.NoError(t, err)
    assert.NotNil(t, doc)
    assert.Equal(t, "doc_123", doc.ID)
    assert.Equal(t, "motion.pdf", doc.FileName)
    
    // Verify events
    events := doc.GetEvents()
    assert.Len(t, events, 1)
    assert.IsType(t, &document.DocumentCreatedEvent{}, events[0])
}

func TestValidation(t *testing.T) {
    // Test validation errors
    _, err := document.NewDocument(
        document.WithID(""), // Empty ID should fail
    )
    
    assert.Error(t, err)
    assert.True(t, errors.Is(err, document.ErrEmptyDocumentID))
}
```

---

## See Also

- **MODELS_INVENTORY.md** - Complete reference of all models (100+ types)
- **CONSOLIDATION_PLAN.md** - Consolidation strategy and timeline
- **CONSOLIDATION_IMPLEMENTATION_GUIDE.md** - Step-by-step implementation instructions
- **CLAUDE.md** - Project architecture and guidelines
- **internal/domain/document/CLAUDE.md** - Domain layer documentation

---

**Last Updated**: November 4, 2024  
**Status**: Ready for implementation
