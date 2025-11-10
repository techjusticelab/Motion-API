# Motion-Index-Fiber: Models Consolidation Plan

## Executive Summary

This document outlines a consolidation strategy to unify 100+ data structures currently distributed across 4 separate locations into a cohesive, well-organized structure that maintains clear separation of concerns while reducing duplication.

**Current State**: 24 files, 4 locations, 100+ types with overlapping concepts
**Desired State**: Unified structure with clear layer boundaries and single source of truth

---

## Phase 1: Analysis & Planning (COMPLETED)

### Deliverables
- [x] Complete inventory of all models (MODELS_INVENTORY.md)
- [x] Dependency analysis
- [x] Consolidation opportunity identification
- [x] Impact assessment

### Key Findings
1. **High Duplication**: Document model defined in multiple layers
2. **Type Alias Bridges**: Using type aliases to bridge between layers (SearchDocumentsRequest)
3. **Scattered Validation**: Validation logic in 3+ different locations
4. **Clear Layers**: Domain (DDD), Application (DTOs), and HTTP (requests/responses)

---

## Phase 2: Consolidation Structure (PROPOSED)

### Recommended New Layout

```
motion-index-fiber/
├── pkg/
│   └── models/
│       ├── core/                      # Core domain models
│       │   ├── document.go            # Document, DocumentMetadata (from current document.go)
│       │   ├── legal.go               # Legal types (from current legal.go)
│       │   └── types.go               # Common types (DocumentType, Category, etc.)
│       │
│       ├── api/                       # HTTP API models
│       │   ├── request.go             # All request DTOs
│       │   │   ├── ProcessDocumentRequest, BatchProcessRequest
│       │   │   ├── ClassifyDocumentRequest, UpdateClassificationRequest
│       │   │   ├── SearchDocumentsRequest
│       │   │   ├── AnalyzeRedactionsRequest, BulkUploadRequest
│       │   │   ├── GetDocumentRequest, GetDocumentStatsRequest
│       │   │   └── HealthCheckRequest
│       │   │
│       │   ├── response.go            # All response DTOs
│       │   │   ├── ProcessDocumentResponse, BatchProcessResponse
│       │   │   ├── ExtractionResult, ClassificationResult
│       │   │   ├── SearchDocumentsResponse, DocumentStatsResponse
│       │   │   ├── IndexDocumentResponse
│       │   │   ├── ExtractionResult, ClassificationResult
│       │   │   └── ProcessingStep, IndexResult
│       │   │
│       │   ├── envelope.go            # APIResponse, APIError
│       │   └── errors.go              # HTTP error codes/messages
│       │
│       ├── search/                    # Search-specific models
│       │   ├── request.go             # SearchRequest + options
│       │   ├── result.go              # SearchResult, SearchDocument
│       │   ├── aggregation.go         # Aggregation types, BulkResult
│       │   └── filters.go             # Filter, SortOptions, Pagination
│       │
│       ├── validation/                # Input validation
│       │   ├── rules.go               # FileValidationRules, defaults
│       │   ├── sanitizers.go          # XSS/injection prevention
│       │   └── validators.go          # Validation functions
│       │
│       └── time.go                    # DateRange (unchanged)
│
├── internal/
│   ├── domain/                         # DDD layer (UNCHANGED)
│   │   ├── document/
│   │   ├── classification/
│   │   ├── legal/
│   │   └── errors/
│   │
│   ├── application/
│   │   ├── dto/
│   │   │   ├── mappers.go             # NEW: DTO ↔ Domain mappers
│   │   │   ├── validators.go          # Application layer validation
│   │   │   └── builders.go            # NEW: DTO builders for responses
│   │   │
│   │   ├── ports/
│   │   ├── usecase/
│   │   └── validation/
│   │
│   ├── handlers/                       # Remove local models (use pkg/models)
│   ├── infrastructure/
│   └── config/
│
└── cmd/
    ├── server/
    ├── api-classifier/
    ├── setup-index/
    └── inspect-index/
```

### Key Changes by Location

#### `pkg/models/core/document.go`
**MOVE FROM**: `pkg/models/document.go`
**MOVE FROM**: Parts of `pkg/models/document_part2.go` (mapping helpers)
**CONTAINS**:
- `Document` struct
- `DocumentMetadata` struct
- OpenSearch mapping helpers (keeping here for compatibility)
- Helper methods (GetCaseName, GetDefenseAttorneys, etc.)

#### `pkg/models/core/legal.go`
**MOVE FROM**: `pkg/models/legal.go` (unchanged)
**CONTAINS**: All legal types (CaseInfo, Party, Attorney, Judge, Charge, Authority, DocumentType, LegalTag)

#### `pkg/models/core/types.go`
**NEW FILE**
**CONTAINS**:
- DocumentType enum and methods
- Category enum (extracted from value objects)
- LegalTag struct
- Common enums and constants

#### `pkg/models/api/request.go`
**CONSOLIDATE FROM**:
- `internal/models/requests.go` (ProcessDocumentRequest, BatchProcessRequest, etc.)
- `internal/application/dto/document.go` (similar request types)
- `internal/application/dto/classification.go` (ClassifyDocumentRequest)
- `internal/application/dto/search.go` (SearchDocumentsRequest)

**CONTAINS**:
```go
// Processing
type ProcessDocumentRequest struct { ... }
type ProcessOptions struct { ... }
type BatchProcessRequest struct { ... }
type UpdateMetadataRequest struct { ... }
type DeleteDocumentRequest struct { ... }

// Classification
type ClassifyDocumentRequest struct { ... }
type UpdateClassificationRequest struct { ... }

// Search (moved from search/request.go for now)
type SearchDocumentsRequest struct { ... }

// Redaction
type AnalyzeRedactionsRequest struct { ... }
type RedactDocumentRequest struct { ... }
type RedactionItem struct { ... }
type RedactionOptions struct { ... }

// Storage
type BulkUploadRequest struct { ... }

// Utility
type GetDocumentRequest struct { ... }
type GetDocumentStatsRequest struct { ... }
type HealthCheckRequest struct { ... }
```

#### `pkg/models/api/response.go`
**CONSOLIDATE FROM**:
- `internal/models/responses.go` (ProcessDocumentResponse, ExtractionResult, etc.)
- `internal/application/dto/document.go` (ProcessDocumentResponse)
- `internal/application/dto/classification.go` (ClassificationResponse)

**CONTAINS**:
```go
// Processing
type ProcessDocumentResponse struct { ... }
type BatchProcessResponse struct { ... }
type ExtractionResult struct { ... }
type ClassificationResult struct { ... }
type ProcessingStep struct { ... }
type IndexResult struct { ... }

// Search
type SearchDocumentsResponse struct { ... }
type DocumentStatsResponse struct { ... }

// Indexing
type IndexDocumentResponse struct { ... }

// Redaction
type RedactionAnalysisResponse struct { ... }
```

#### `pkg/models/api/envelope.go`
**MOVE FROM**: `pkg/models/api.go`
**CONTAINS**:
- `APIResponse` struct
- `APIError` struct
- Constructor functions
- Helper methods (SetRequestID, IsError, etc.)

#### `pkg/models/search/request.go`
**MOVE FROM**: Part of `pkg/models/search.go`
**CONTAINS**:
- `SearchRequest` struct
- `MetadataFieldValuesRequest` struct
- `Filters` struct
- `SortOptions` struct
- `PaginationOptions` struct
- `HighlightOptions` struct
- Validation: `ValidateSearchRequest()`, `ApplyDefaults()`

#### `pkg/models/search/result.go`
**MOVE FROM**: Part of `pkg/models/search.go`
**CONTAINS**:
- `SearchResult` struct
- `SearchDocument` struct
- `SearchResponse` struct
- `SearchError` struct
- Constructor: `NewSearchRequest()`, `NewSearchResult()`

#### `pkg/models/search/aggregation.go`
**MOVE FROM**: Part of `pkg/models/search.go`
**CONTAINS**:
- `TagCount`, `TypeCount`, `FieldValue`
- `DocumentStats`, `FieldStat`
- `FieldOptions`
- `BulkResult`, `BulkResultItem`, `BulkItemResult`, `BulkError`, `BulkFailedDoc`
- `IndexStats`, `ShardInfo`, `PerformanceStats`
- `AggregationResponse`, `AggregationBucket`

#### `pkg/models/search/filters.go`
**NEW FILE** (extract from request.go for clarity)
**CONTAINS**:
- `Filters` struct
- `SortOptions` struct
- `SortOrder` enum
- `PaginationOptions` struct
- `HighlightOptions` struct
- Constants: `DefaultSearchSize`, `MaxSearchSize`

#### `pkg/models/validation/rules.go`
**MOVE FROM**: `internal/models/validation.go`
**CONTAINS**:
- `FileValidationRules` struct
- `ValidationError` struct
- `DefaultFileValidationRules()` function
- Validation rule constants

#### `pkg/models/validation/sanitizers.go`
**EXTRACT FROM**: `internal/models/validation.go`
**CONTAINS**:
- `SanitizeInput()` function
- `removeScriptBlocks()` function
- `removeIframeBlocks()` function
- `removeJavaScriptProtocol()` function
- All XSS/injection prevention logic

#### `pkg/models/validation/validators.go`
**MOVE FROM**: `internal/models/validation.go`
**CONTAINS**:
- `ValidateFile()` function
- `ValidateFiles()` function
- `ValidateSearchQuery()` function
- `ValidateMetadata()` function
- `ValidateStruct()` function
- `FormatValidationErrors()` function
- Custom validators: `validateFileExtension()`, `validateFileSize()`, etc.

#### `internal/application/dto/mappers.go`
**NEW FILE**
**PURPOSE**: Convert between HTTP DTOs and domain models
**CONTAINS**:
```go
// Request mappers
func (req *ProcessDocumentRequest) ToApplicationDTO() (*dto.ProcessDocumentRequest, error) { ... }
func (req *ClassifyDocumentRequest) ToApplicationDTO() (*dto.ClassifyDocumentRequest, error) { ... }

// Response builders
func NewProcessDocumentResponseFromDomain(doc *domain.Document) *ProcessDocumentResponse { ... }
func NewClassificationResponseFromDomain(result *classification.Result) *ClassificationResponse { ... }
```

#### `internal/application/dto/validators.go`
**NEW FILE**
**PURPOSE**: Application-layer validation (uses domain validation)
**CONTAINS**:
```go
func (req *ProcessDocumentRequest) Validate() error { ... }
func (req *ClassifyDocumentRequest) Validate() error { ... }
func (req *SearchDocumentsRequest) Validate() error { ... }
```

#### `internal/models/` REMOVAL
**STATUS**: DELETE AFTER MIGRATION
**REASON**: All content moved to `pkg/models/` with proper structure

---

## Phase 3: Consolidation Strategy

### Step 1: Create New pkg/models Structure
1. Create subdirectories: `core/`, `api/`, `search/`, `validation/`
2. Create new files in each subdirectory
3. Copy existing types with no modifications
4. Update package imports (all will be `pkg/models/*`)

### Step 2: Update Imports Across Codebase
**Priority 1 - High dependency files** (28+ files importing pkg/models):
- Update to use new package structure: `pkg/models/core`, `pkg/models/api`, etc.

**Priority 2 - HTTP layer** (handlers, middleware):
- Change from `internal/models` to `pkg/models/api`
- Update response builders to use new structure

**Priority 3 - Application layer**:
- Create DTO mappers in `internal/application/dto/mappers.go`
- Update use cases to use mappers

### Step 3: Delete Deprecated Packages
1. Delete `internal/models/` directory (all content moved)
2. Delete type aliases in `internal/models/requests.go`
3. Remove unused imports

### Step 4: Update Documentation
1. Update CLAUDE.md with new structure
2. Update README.md with model organization
3. Update API documentation

---

## Phase 4: Implementation Timeline

### Week 1: Foundation
- [ ] Create new directory structure
- [ ] Create core/document.go, core/legal.go
- [ ] Create api/envelope.go, api/request.go, api/response.go
- [ ] Create search/request.go, search/result.go, search/aggregation.go

### Week 2: Validation & Utilities
- [ ] Create validation/rules.go, validation/sanitizers.go, validation/validators.go
- [ ] Update pkg/models/time.go (minor changes)
- [ ] Update pkg/models/document_part2.go (mapping helpers)

### Week 3: Application Layer DTOs
- [ ] Create internal/application/dto/mappers.go
- [ ] Create internal/application/dto/validators.go
- [ ] Create internal/application/dto/builders.go

### Week 4: Update Imports (Batch 1)
- [ ] Update handlers (12 files)
- [ ] Update search service and builders
- [ ] Update pipeline processors

### Week 5: Update Imports (Batch 2)
- [ ] Update command-line tools
- [ ] Update infrastructure adapters
- [ ] Update remaining services

### Week 6: Testing & Cleanup
- [ ] Run full test suite
- [ ] Fix any import errors
- [ ] Delete internal/models/ directory
- [ ] Update documentation

---

## Risk Assessment

### Low Risk (Safe to Consolidate Immediately)
✅ Moving `pkg/models/api.go` content
✅ Moving `pkg/models/legal.go` content
✅ Splitting `pkg/models/search.go` into multiple files
✅ Creating validation/* structure

### Medium Risk (Requires Careful Testing)
⚠️ Moving `internal/models/` to `pkg/models/api/`
- Reason: High dependency count, many import statements to update
- Mitigation: Use IDE refactoring tools, run tests after each batch

⚠️ Creating DTO mappers
- Reason: New code that must precisely convert between layers
- Mitigation: Write comprehensive tests for all mappers

### No Risk (Already in Good Shape)
✅ Domain layer (internal/domain/) - no changes needed
✅ Application ports/interfaces - no changes needed
✅ Test structures - mostly isolated

---

## Migration Checklist

### Before Starting
- [ ] Create feature branch: `feature/consolidate-models`
- [ ] Create milestone in issue tracker
- [ ] Notify team of changes
- [ ] Ensure CI/CD pipeline is green

### Code Changes
- [ ] Create new directory structure
- [ ] Move files with content preservation
- [ ] Update import statements (IDE batch refactor)
- [ ] Create new DTO mappers
- [ ] Remove deprecated internal/models/

### Testing
- [ ] Run `go test ./...` for each batch of changes
- [ ] Run integration tests
- [ ] Manual testing of API endpoints
- [ ] Performance testing (ensure no regressions)

### Documentation
- [ ] Update MODELS_INVENTORY.md with new structure
- [ ] Update CLAUDE.md section on models
- [ ] Update inline code comments
- [ ] Create MODELS_MIGRATION.md (migration guide for future work)

### Deployment
- [ ] Code review
- [ ] Merge to main
- [ ] Tag release (if appropriate)
- [ ] Update changelog

---

## Dependencies After Consolidation

### Import Pattern (New)
```go
// HTTP layer
import "motion-index-fiber/pkg/models/api"    // requests, responses
import "motion-index-fiber/pkg/models/core"   // Document, DocumentMetadata
import "motion-index-fiber/pkg/models/search" // SearchRequest, SearchResult

// Application layer  
import "motion-index-fiber/internal/application/dto"  // Mappers, builders
import "motion-index-fiber/internal/domain"           // Aggregates, services

// Domain layer
import "motion-index-fiber/internal/domain/document"  // Document aggregate
import "motion-index-fiber/internal/domain/errors"    // Domain errors
```

### No Internal Model Imports
```go
// ❌ Old pattern (avoid)
import "motion-index-fiber/internal/models"

// ✅ New pattern (use)
import "motion-index-fiber/pkg/models/api"
```

---

## Benefits After Consolidation

### Code Organization
✅ Single source of truth for each model
✅ Clear layer boundaries (core, api, search, validation)
✅ Reduced duplication (100+ types now organized)
✅ Easier to locate models (organized by concern)

### Maintainability
✅ Adding new request/response types has clear location
✅ Validation logic centralized
✅ Mappers explicit (not hidden in layers)
✅ Clear separation: DTOs vs. Domain vs. HTTP

### Type Safety
✅ No type aliases between layers
✅ Explicit mappers instead of implicit conversions
✅ Better IDE support (proper package structure)
✅ Clearer dependency graph

### Testing
✅ Easier to mock models
✅ DTOs testable in isolation
✅ Validation testable independently
✅ Fewer import interdependencies

### Performance
✅ No change expected (structural only)
✅ Potentially faster compilation (smaller packages)
✅ Clearer dependency tree for analysis

---

## Success Criteria

### Metrics
- [ ] All tests pass (100% success rate)
- [ ] No circular imports
- [ ] Import statements follow new pattern consistently
- [ ] Code compiles with no warnings
- [ ] API behavior unchanged (backward compatible)

### Quality Gates
- [ ] No increase in LOC (consolidation should reduce)
- [ ] Code coverage maintained or improved
- [ ] No performance regression in benchmarks
- [ ] All handlers passing integration tests

### Documentation
- [ ] MODELS_INVENTORY.md updated for new structure
- [ ] CLAUDE.md updated with new patterns
- [ ] No outdated comments in code
- [ ] All new types have GoDoc comments

---

## Rollback Plan

If consolidation causes issues:

### Option 1: Partial Rollback (Stage-based)
1. Keep domain layer (internal/domain/) as-is
2. Only consolidate HTTP/API models
3. Revert internal/models/ cleanup
4. Try again with smaller scope

### Option 2: Full Rollback
1. Revert to commit before consolidation started
2. Analyze issues
3. Plan more carefully
4. Try again in next sprint

### Mitigation Steps
- [ ] Keep old branch available for 30 days
- [ ] Tag pre-consolidation commit
- [ ] Document any issues encountered
- [ ] Create follow-up issues for improvements

---

## Long-Term Maintenance

### Guidelines After Consolidation
1. **New Request Type**: Add to `pkg/models/api/request.go`
2. **New Response Type**: Add to `pkg/models/api/response.go`
3. **New Domain Model**: Add to `internal/domain/`
4. **New Mapper**: Add to `internal/application/dto/mappers.go`
5. **New Validation**: Add to `pkg/models/validation/validators.go`

### Code Review Checklist
- [ ] Models in `pkg/models/` only (not internal/)
- [ ] DTOs in `internal/application/dto/` for layer transitions
- [ ] Domain logic in `internal/domain/`
- [ ] No imports of internal/models/
- [ ] Mappers explicit (not type aliases)

### Future Improvements
1. Consider splitting `pkg/models/search/` further if it grows
2. Add builder pattern for complex models
3. Add validation decorators for reusability
4. Consider code generation for mapper boilerplate

---

