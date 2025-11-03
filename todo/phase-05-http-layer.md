# Phase 5: HTTP Layer Refactor

## Overview

Refactor HTTP handlers from 4,180 LOC of mixed concerns to <1,000 LOC of thin, focused handlers that delegate to use cases. Implement error translation middleware, response presenters, and comprehensive HTTP integration tests.

## Objectives

- [ ] Refactor all handlers to be thin (<100 LOC each)
- [ ] Create error translation middleware
- [ ] Implement response presenters
- [ ] Add HTTP-specific validation
- [ ] Write HTTP integration tests
- [ ] Reduce handler LOC from 4,180 to <1,000 (76% reduction)
- [ ] Achieve >80% test coverage

## Prerequisites

- **Phase 1 Complete**: Domain layer with aggregates
- **Phase 2 Complete**: Application layer with use cases
- **Phase 4 Complete**: Infrastructure adapters
- Fiber v2 framework knowledge
- HTTP testing patterns

## Current State

**Handler Code**: `internal/handlers/`
- **processing.go**: 1,023 LOC (business logic + HTTP)
- **document.go**: 842 LOC (mixed concerns)
- **search.go**: 567 LOC (search + HTTP)
- **health.go**: 258 LOC (health checks + HTTP)
- **classification.go**: 723 LOC (classification + HTTP)
- **metadata.go**: 467 LOC (metadata + HTTP)
- **redaction.go**: 300 LOC (redaction + HTTP)

**Total**: ~4,180 LOC

**Problems**:
- Business logic in handlers
- No separation of concerns
- Hard to test (need to mock HTTP)
- Duplicated error handling
- Inconsistent response formats
- No input validation layer

## Target State

**Handler Structure**:
- Thin handlers (<100 LOC each)
- Delegate to use cases
- Only HTTP concerns (parsing, validation, response)
- Consistent error handling via middleware
- Consistent response format via presenters

**Target LOC**: <1,000 LOC total (76% reduction)

## Task Breakdown

### Task Group 1: Response Presenters

#### Task 1.1: Base Presenter
**File**: `internal/infrastructure/http/presenter/base.go`

```go
package presenter

import (
    "github.com/gofiber/fiber/v2"
)

// Response represents a standard API response
type Response struct {
    Success bool        `json:"success"`
    Data    interface{} `json:"data,omitempty"`
    Error   *Error      `json:"error,omitempty"`
    Meta    *Meta       `json:"meta,omitempty"`
}

// Error represents an error response
type Error struct {
    Code    string      `json:"code"`
    Message string      `json:"message"`
    Details interface{} `json:"details,omitempty"`
}

// Meta represents response metadata
type Meta struct {
    RequestID string `json:"request_id,omitempty"`
    Timestamp int64  `json:"timestamp"`
    Version   string `json:"version"`
}

// Success creates a success response
func Success(c *fiber.Ctx, data interface{}) error {
    return c.JSON(&Response{
        Success: true,
        Data:    data,
        Meta:    buildMeta(c),
    })
}

// Error creates an error response
func ErrorResponse(c *fiber.Ctx, statusCode int, code, message string, details interface{}) error {
    return c.Status(statusCode).JSON(&Response{
        Success: false,
        Error: &Error{
            Code:    code,
            Message: message,
            Details: details,
        },
        Meta: buildMeta(c),
    })
}

func buildMeta(c *fiber.Ctx) *Meta {
    return &Meta{
        RequestID: c.Locals("requestid").(string),
        Timestamp: time.Now().Unix(),
        Version:   "1.0.0",
    }
}
```

---

#### Task 1.2: Document Presenter
**File**: `internal/infrastructure/http/presenter/document.go`

```go
package presenter

import (
    "motion-index-fiber/internal/application/dto"
)

// DocumentResponse for presenting documents
type DocumentResponse struct {
    ID           string                 `json:"id"`
    FileName     string                 `json:"file_name"`
    StorageURL   string                 `json:"storage_url,omitempty"`
    Status       string                 `json:"status"`
    DocumentType string                 `json:"document_type,omitempty"`
    Category     string                 `json:"category,omitempty"`
    Confidence   float64                `json:"confidence,omitempty"`
    CreatedAt    string                 `json:"created_at"`
    ProcessedAt  string                 `json:"processed_at,omitempty"`
    Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// PresentDocument converts DTO to presentation format
func PresentDocument(dto *dto.ProcessDocumentResponse) *DocumentResponse {
    return &DocumentResponse{
        ID:           dto.DocumentID,
        FileName:     dto.FileName,
        StorageURL:   dto.URL,
        Status:       dto.Status,
        DocumentType: dto.DocumentType,
        Category:     dto.Category,
        Confidence:   dto.Confidence,
        CreatedAt:    dto.ProcessedAt.Format(time.RFC3339),
    }
}

// PresentDocumentList converts list of DTOs
func PresentDocumentList(dtos []*dto.DocumentSummary) []*DocumentResponse {
    responses := make([]*DocumentResponse, len(dtos))
    for i, dto := range dtos {
        responses[i] = PresentDocument(dto)
    }
    return responses
}
```

---

### Task Group 2: Error Translation Middleware

#### Task 2.1: Error Translator
**File**: `internal/infrastructure/http/middleware/error_translator.go`

```go
package middleware

import (
    "errors"
    "github.com/gofiber/fiber/v2"
    "motion-index-fiber/internal/domain/errors"
    "motion-index-fiber/internal/infrastructure/http/presenter"
)

// ErrorTranslator translates domain/application errors to HTTP responses
func ErrorTranslator() fiber.Handler {
    return func(c *fiber.Ctx) error {
        err := c.Next()
        if err == nil {
            return nil
        }

        // Translate error to HTTP status and response
        statusCode, code, message := translateError(err)

        return presenter.ErrorResponse(c, statusCode, code, message, nil)
    }
}

func translateError(err error) (statusCode int, code string, message string) {
    // Domain errors
    if errors.Is(err, domainerrors.ErrDocumentNotFound) {
        return fiber.StatusNotFound, "DOCUMENT_NOT_FOUND", "Document not found"
    }
    if errors.Is(err, domainerrors.ErrInvalidConfidence) {
        return fiber.StatusBadRequest, "INVALID_CONFIDENCE", "Confidence must be between 0.0 and 1.0"
    }
    if errors.Is(err, domainerrors.ErrEmptyDocumentID) {
        return fiber.StatusBadRequest, "EMPTY_DOCUMENT_ID", "Document ID cannot be empty"
    }
    if errors.Is(err, domainerrors.ErrInvalidFilePath) {
        return fiber.StatusBadRequest, "INVALID_FILE_PATH", "Invalid file path"
    }

    // Application errors
    if errors.Is(err, apperrors.ErrValidationFailed) {
        return fiber.StatusBadRequest, "VALIDATION_FAILED", err.Error()
    }

    // Infrastructure errors
    if errors.Is(err, infraerrors.ErrStorageFailed) {
        return fiber.StatusInternalServerError, "STORAGE_FAILED", "Failed to store document"
    }
    if errors.Is(err, infraerrors.ErrSearchFailed) {
        return fiber.StatusInternalServerError, "SEARCH_FAILED", "Search operation failed"
    }

    // Default: Internal server error
    return fiber.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred"
}
```

---

### Task Group 3: Thin Handlers

#### Task 3.1: Document Handler (Refactored)
**File**: `internal/infrastructure/http/handlers/document_handler.go`

```go
package handlers

import (
    "github.com/gofiber/fiber/v2"
    "motion-index-fiber/internal/application/dto"
    "motion-index-fiber/internal/application/usecases/document"
    "motion-index-fiber/internal/infrastructure/http/presenter"
)

// DocumentHandler handles document-related HTTP requests
type DocumentHandler struct {
    processUseCase *document.ProcessDocumentUseCase
    updateUseCase  *document.UpdateMetadataUseCase
}

// NewDocumentHandler creates a new document handler
func NewDocumentHandler(
    processUseCase *document.ProcessDocumentUseCase,
    updateUseCase *document.UpdateMetadataUseCase,
) *DocumentHandler {
    return &DocumentHandler{
        processUseCase: processUseCase,
        updateUseCase:  updateUseCase,
    }
}

// ProcessDocument handles document processing requests
func (h *DocumentHandler) ProcessDocument(c *fiber.Ctx) error {
    // 1. Parse multipart form
    file, err := c.FormFile("file")
    if err != nil {
        return fiber.NewError(fiber.StatusBadRequest, "File is required")
    }

    // 2. Build request DTO
    req := &dto.ProcessDocumentRequest{
        ID:          c.FormValue("id"),
        FileName:    file.Filename,
        ContentType: file.Header.Get("Content-Type"),
        Size:        file.Size,
        Content:     file,
        ExtractText: c.FormValue("extract_text") == "true",
        Classify:    c.FormValue("classify") == "true",
        Store:       c.FormValue("store") == "true",
        Index:       c.FormValue("index") == "true",
    }

    // 3. Execute use case
    resp, err := h.processUseCase.Execute(c.Context(), req)
    if err != nil {
        return err // Will be caught by error translator middleware
    }

    // 4. Present response
    return presenter.Success(c, presenter.PresentDocument(resp))
}

// UpdateMetadata handles metadata update requests
func (h *DocumentHandler) UpdateMetadata(c *fiber.Ctx) error {
    // 1. Parse request body
    var req dto.UpdateMetadataRequest
    if err := c.BodyParser(&req); err != nil {
        return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
    }

    // 2. Extract document ID from URL
    req.DocumentID = c.Params("id")

    // 3. Execute use case
    resp, err := h.updateUseCase.Execute(c.Context(), &req)
    if err != nil {
        return err
    }

    // 4. Present response
    return presenter.Success(c, presenter.PresentDocument(resp))
}

// GetDocument handles document retrieval requests
func (h *DocumentHandler) GetDocument(c *fiber.Ctx) error {
    // Implementation similar to above
    // ~30 LOC
}
```

**Result**: ~100 LOC (down from 842 LOC)

---

#### Task 3.2: Search Handler (Refactored)
**File**: `internal/infrastructure/http/handlers/search_handler.go`

```go
package handlers

import (
    "github.com/gofiber/fiber/v2"
    "motion-index-fiber/internal/application/dto"
    "motion-index-fiber/internal/application/usecases/document"
    "motion-index-fiber/internal/infrastructure/http/presenter"
)

// SearchHandler handles search-related HTTP requests
type SearchHandler struct {
    searchUseCase *document.SearchDocumentsUseCase
}

// NewSearchHandler creates a new search handler
func NewSearchHandler(searchUseCase *document.SearchDocumentsUseCase) *SearchHandler {
    return &SearchHandler{searchUseCase: searchUseCase}
}

// SearchDocuments handles document search requests
func (h *SearchHandler) SearchDocuments(c *fiber.Ctx) error {
    // 1. Parse query parameters
    req := &dto.SearchDocumentsRequest{
        Query:        c.Query("q"),
        DocumentType: c.Query("document_type"),
        Category:     c.Query("category"),
        Page:         c.QueryInt("page", 1),
        PageSize:     c.QueryInt("page_size", 20),
        SortBy:       c.Query("sort_by", "created_at"),
        SortOrder:    c.Query("sort_order", "desc"),
    }

    // 2. Execute use case
    resp, err := h.searchUseCase.Execute(c.Context(), req)
    if err != nil {
        return err
    }

    // 3. Present response
    return presenter.Success(c, map[string]interface{}{
        "documents":   presenter.PresentDocumentList(resp.Documents),
        "total":       resp.Total,
        "page":        resp.Page,
        "page_size":   resp.PageSize,
        "total_pages": resp.TotalPages,
    })
}
```

**Result**: ~50 LOC (down from 567 LOC)

---

#### Task 3.3: Classification Handler (Refactored)
**File**: `internal/infrastructure/http/handlers/classification_handler.go`

Similar structure, ~80 LOC (down from 723 LOC)

---

#### Task 3.4: Health Handler (Refactored)
**File**: `internal/infrastructure/http/handlers/health_handler.go`

Similar structure, ~60 LOC (down from 258 LOC)

---

### Task Group 4: Request Validation

#### Task 4.1: Validation Middleware
**File**: `internal/infrastructure/http/middleware/validation.go`

```go
package middleware

import (
    "github.com/go-playground/validator/v10"
    "github.com/gofiber/fiber/v2"
    "motion-index-fiber/internal/infrastructure/http/presenter"
)

var validate = validator.New()

// ValidateRequest validates request DTO
func ValidateRequest(req interface{}) fiber.Handler {
    return func(c *fiber.Ctx) error {
        if err := c.BodyParser(req); err != nil {
            return presenter.ErrorResponse(c, fiber.StatusBadRequest,
                "INVALID_REQUEST", "Invalid request body", nil)
        }

        if err := validate.Struct(req); err != nil {
            validationErrors := err.(validator.ValidationErrors)
            details := make(map[string]string)
            for _, err := range validationErrors {
                details[err.Field()] = err.Tag()
            }
            return presenter.ErrorResponse(c, fiber.StatusBadRequest,
                "VALIDATION_FAILED", "Request validation failed", details)
        }

        return c.Next()
    }
}
```

---

### Task Group 5: Router Setup

#### Task 5.1: Router Configuration
**File**: `internal/infrastructure/http/router/router.go`

```go
package router

import (
    "github.com/gofiber/fiber/v2"
    "motion-index-fiber/internal/infrastructure/http/handlers"
    "motion-index-fiber/internal/infrastructure/http/middleware"
)

// SetupRoutes configures all HTTP routes
func SetupRoutes(app *fiber.App, handlers *handlers.Handlers) {
    // Middleware
    app.Use(middleware.RequestID())
    app.Use(middleware.Logger())
    app.Use(middleware.ErrorTranslator())

    // API v1
    v1 := app.Group("/api/v1")

    // Health endpoints
    v1.Get("/health", handlers.Health.Check)
    v1.Get("/health/detailed", handlers.Health.Detailed)

    // Document endpoints
    docs := v1.Group("/documents")
    docs.Post("/", handlers.Document.ProcessDocument)
    docs.Get("/:id", handlers.Document.GetDocument)
    docs.Put("/:id/metadata", handlers.Document.UpdateMetadata)
    docs.Delete("/:id", handlers.Document.DeleteDocument)

    // Search endpoints
    v1.Get("/search", handlers.Search.SearchDocuments)

    // Classification endpoints
    classify := v1.Group("/classify")
    classify.Post("/:id", handlers.Classification.ClassifyDocument)
    classify.Get("/:id", handlers.Classification.GetClassification)

    // Redaction endpoints
    redact := v1.Group("/redactions")
    redact.Post("/:id/analyze", handlers.Redaction.AnalyzeRedactions)
    redact.Post("/:id/apply", handlers.Redaction.ApplyRedactions)
}
```

---

### Task Group 6: HTTP Integration Tests

#### Task 6.1: Test Utilities
**File**: `internal/infrastructure/http/testing/utils.go`

```go
package testing

import (
    "bytes"
    "encoding/json"
    "io"
    "net/http/httptest"
    "testing"

    "github.com/gofiber/fiber/v2"
    "github.com/stretchr/testify/require"
)

// TestApp creates a test Fiber app
func TestApp(t *testing.T) *fiber.App {
    app := fiber.New(fiber.Config{
        ErrorHandler: func(c *fiber.Ctx, err error) error {
            return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
                "error": err.Error(),
            })
        },
    })
    return app
}

// MakeRequest makes an HTTP request to the test app
func MakeRequest(t *testing.T, app *fiber.App, method, path string, body interface{}) *httptest.ResponseRecorder {
    var bodyReader io.Reader
    if body != nil {
        jsonBody, err := json.Marshal(body)
        require.NoError(t, err)
        bodyReader = bytes.NewReader(jsonBody)
    }

    req := httptest.NewRequest(method, path, bodyReader)
    req.Header.Set("Content-Type", "application/json")

    resp, err := app.Test(req)
    require.NoError(t, err)

    return resp
}

// ParseResponse parses JSON response
func ParseResponse(t *testing.T, resp *httptest.ResponseRecorder, target interface{}) {
    body, err := io.ReadAll(resp.Body)
    require.NoError(t, err)

    err = json.Unmarshal(body, target)
    require.NoError(t, err)
}
```

---

#### Task 6.2: Document Handler Tests
**File**: `internal/infrastructure/http/handlers/document_handler_test.go`

```go
package handlers_test

import (
    "net/http"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
    "motion-index-fiber/internal/application/dto"
    "motion-index-fiber/internal/infrastructure/http/handlers"
    httptest "motion-index-fiber/internal/infrastructure/http/testing"
)

func TestDocumentHandler_ProcessDocument_Success(t *testing.T) {
    // Given: Mock use case
    mockUseCase := new(MockProcessDocumentUseCase)
    mockUseCase.On("Execute", mock.Anything, mock.Anything).
        Return(&dto.ProcessDocumentResponse{
            DocumentID: "doc_123",
            Status:     "processed",
        }, nil)

    // And: Handler with mock
    handler := handlers.NewDocumentHandler(mockUseCase, nil)
    app := httptest.TestApp(t)
    app.Post("/documents", handler.ProcessDocument)

    // When: Request is made
    resp := httptest.MakeRequest(t, app, "POST", "/documents", map[string]interface{}{
        "file_name": "test.pdf",
    })

    // Then: Response is success
    assert.Equal(t, http.StatusOK, resp.StatusCode)

    var response map[string]interface{}
    httptest.ParseResponse(t, resp, &response)
    assert.True(t, response["success"].(bool))
    assert.Equal(t, "doc_123", response["data"].(map[string]interface{})["id"])
}

func TestDocumentHandler_ProcessDocument_ValidationError(t *testing.T) {
    // Given: Handler
    handler := handlers.NewDocumentHandler(nil, nil)
    app := httptest.TestApp(t)
    app.Post("/documents", handler.ProcessDocument)

    // When: Invalid request is made
    resp := httptest.MakeRequest(t, app, "POST", "/documents", map[string]interface{}{
        // Missing required fields
    })

    // Then: Validation error is returned
    assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

    var response map[string]interface{}
    httptest.ParseResponse(t, resp, &response)
    assert.False(t, response["success"].(bool))
    assert.Equal(t, "VALIDATION_FAILED", response["error"].(map[string]interface{})["code"])
}
```

---

## Files to Create

| File | Purpose | LOC |
|------|---------|-----|
| `http/presenter/base.go` | Base presenter | ~100 |
| `http/presenter/document.go` | Document presenter | ~150 |
| `http/presenter/search.go` | Search presenter | ~100 |
| `http/middleware/error_translator.go` | Error translation | ~200 |
| `http/middleware/validation.go` | Request validation | ~100 |
| `http/middleware/request_id.go` | Request ID middleware | ~50 |
| `http/handlers/document_handler.go` | Document handler | ~100 |
| `http/handlers/search_handler.go` | Search handler | ~80 |
| `http/handlers/classification_handler.go` | Classification handler | ~80 |
| `http/handlers/health_handler.go` | Health handler | ~60 |
| `http/handlers/redaction_handler.go` | Redaction handler | ~70 |
| `http/router/router.go` | Router setup | ~150 |
| `http/testing/utils.go` | Test utilities | ~150 |
| Plus 10 test files | Handler tests | ~2000 |

**Total**: ~25 files, ~3,440 LOC (new), ~3,180 LOC deleted (net: +260 LOC, 76% handler reduction)

---

## Files to Delete/Refactor

### Files to Delete (~3,180 LOC)
- `internal/handlers/processing.go` (1,023 LOC - business logic moved to use cases)
- Most of `internal/handlers/document.go` (842 LOC → 100 LOC)
- Most of `internal/handlers/search.go` (567 LOC → 80 LOC)
- Most of `internal/handlers/classification.go` (723 LOC → 80 LOC)
- Most of `internal/handlers/metadata.go` (467 LOC → merged into document_handler)
- Most of `internal/handlers/redaction.go` (300 LOC → 70 LOC)
- Most of `internal/handlers/health.go` (258 LOC → 60 LOC)

**Total Deleted**: ~3,180 LOC of mixed-concern handler code

---

## Testing Strategy

### HTTP Integration Tests
- Test full request → response cycle
- Use real Fiber app with test utilities
- Mock use cases, not infrastructure
- Test success and error paths

### Coverage Goals
- Handlers: >80% coverage
- Presenters: >90% coverage
- Middleware: >85% coverage
- Router: 100% coverage

### Test Categories
1. **Happy Path Tests**: Valid requests return correct responses
2. **Validation Tests**: Invalid requests return validation errors
3. **Error Translation Tests**: Domain errors translated to HTTP errors
4. **Presenter Tests**: DTOs correctly converted to response format

---

## Acceptance Criteria

- [ ] All handlers refactored to <100 LOC each
- [ ] Handler LOC reduced from 4,180 to <1,000 (76% reduction)
- [ ] Error translation middleware working
- [ ] Response presenters consistent across all endpoints
- [ ] HTTP integration tests passing (>80% coverage)
- [ ] All handlers delegate to use cases
- [ ] No business logic in handlers
- [ ] Consistent error handling across all endpoints

---

## Success Metrics

- Handler LOC: 4,180 → <1,000 (76% reduction)
- Average handler LOC: ~700 → <100 (86% reduction)
- Test coverage: >80%
- HTTP integration tests: >50 tests
- Error handling consistency: 100%

---

## Migration Strategy

### Step 1: Create Infrastructure
1. Create presenter package
2. Create middleware package
3. Create router package
4. Create test utilities

### Step 2: Refactor One Handler at a Time
1. Start with simplest handler (health)
2. Extract business logic to use case (if not done)
3. Make handler thin (delegate to use case)
4. Update tests
5. Verify endpoint still works
6. Repeat for next handler

### Step 3: Delete Old Code
1. After all handlers refactored
2. Delete old handler files
3. Update imports
4. Run full test suite

---

## Risk Assessment

### Risks
1. **Breaking API contracts**: Changing response formats
2. **Performance regression**: Additional middleware overhead
3. **Test coverage gaps**: Missing edge cases
4. **Incomplete migration**: Some handlers still have business logic

### Mitigation
1. **Contract tests**: Verify response format compatibility
2. **Performance benchmarks**: Measure before/after
3. **Coverage reports**: Track coverage throughout migration
4. **Code reviews**: Ensure all business logic moved to use cases

---

## Dependencies

- Phase 2 complete (use cases available)
- Phase 4 complete (adapters available)
- Fiber v2 installed
- validator/v10 installed
- testify installed

---

## Rollback Strategy

If issues arise:
1. Feature flag: Route to old or new handlers
2. Gradual rollout: Migrate one endpoint at a time
3. Instant rollback: Revert to old handlers if needed

---

**Phase Status**: ⚪ Not Started
**Dependencies**: Phase 2 (use cases), Phase 4 (adapters)
**Next Phase**: Phase 6 (Dependency Injection)
