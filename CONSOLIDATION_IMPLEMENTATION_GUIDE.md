# Models Consolidation: Implementation Guide

## Overview

This guide provides step-by-step instructions for implementing the models consolidation plan outlined in CONSOLIDATION_PLAN.md. It focuses on practical execution with minimal disruption to existing code.

---

## Pre-Implementation Checklist

- [ ] Review and approve MODELS_INVENTORY.md
- [ ] Review and approve CONSOLIDATION_PLAN.md
- [ ] Create feature branch: `git checkout -b feat/consolidate-models`
- [ ] Ensure all tests pass: `go test ./... -v`
- [ ] Backup current state: Document any custom scripts or build processes
- [ ] Notify team of refactoring timeline
- [ ] Schedule code review sessions

---

## Phase 1: Foundation (Week 1)

### Step 1.1: Create New Directory Structure

```bash
cd /home/okita/Scripts/Work/TJL/motion-api

# Create pkg/models subdirectories
mkdir -p pkg/models/core
mkdir -p pkg/models/api
mkdir -p pkg/models/search
mkdir -p pkg/models/validation

# Create internal/application helpers
mkdir -p internal/application/mappers
mkdir -p internal/application/builders
```

### Step 1.2: Move Core Document Model

**File**: `pkg/models/core/document.go`

Contents from current `pkg/models/document.go` with enhanced comments:

```go
package core

import (
	"time"
)

// Document represents a legal document in the system.
// This is the core model used by all layers.
type Document struct {
	// Identity
	ID          string            `json:"id"`
	
	// File Information
	FileName    string            `json:"file_name"`
	FilePath    string            `json:"file_path"`
	FileURL     string            `json:"file_url,omitempty"`
	S3URI       string            `json:"s3_uri,omitempty"`
	Hash        string            `json:"hash"`
	ContentType string            `json:"content_type,omitempty"`
	Size        int64             `json:"size,omitempty"`
	
	// Content
	Text      string `json:"text"`
	DocType   string `json:"doc_type"`
	Category  string `json:"category,omitempty"`
	
	// Metadata
	Metadata *DocumentMetadata `json:"metadata"`
	
	// Timestamps
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	ProcessedAt *time.Time `json:"processed_at,omitempty"`
}

// DocumentMetadata contains rich metadata about a document.
type DocumentMetadata struct {
	// ... existing fields from document.go
}

// GetCaseName returns the case name from metadata.
func (d *Document) GetCaseName() string {
	if d.Metadata != nil && d.Metadata.Case != nil {
		return d.Metadata.Case.Name
	}
	return ""
}

// GetCaseNumber returns the case number from metadata.
func (d *Document) GetCaseNumber() string {
	if d.Metadata != nil && d.Metadata.Case != nil {
		return d.Metadata.Case.Number
	}
	return ""
}

// GetCourtName returns the court name from metadata.
func (d *Document) GetCourtName() string {
	if d.Metadata != nil && d.Metadata.Court != nil {
		return d.Metadata.Court.Name
	}
	return ""
}

// GetJudgeName returns the judge name from metadata.
func (d *Document) GetJudgeName() string {
	if d.Metadata != nil && d.Metadata.Court != nil {
		return d.Metadata.Court.Judge
	}
	return ""
}

// HasLegalTag checks if the document has a specific legal tag.
func (d *Document) HasLegalTag(tag string) bool {
	if d.Metadata == nil {
		return false
	}
	for _, t := range d.Metadata.LegalTags {
		if t == tag {
			return true
		}
	}
	return false
}

// AddLegalTag adds a legal tag to the document.
func (d *Document) AddLegalTag(tag string) {
	if d.Metadata == nil {
		d.Metadata = &DocumentMetadata{}
	}
	if !d.HasLegalTag(tag) {
		d.Metadata.LegalTags = append(d.Metadata.LegalTags, tag)
	}
}
```

**Action**: Copy all content from current `pkg/models/document.go`, place in this new file, update package name to `core`.

### Step 1.3: Move Legal Types

**File**: `pkg/models/core/legal.go`

Contents from current `pkg/models/legal.go`:

```go
package core

// CaseInfo represents case information.
type CaseInfo struct {
	Number   string `json:"number"`
	Name     string `json:"name"`
	Type     string `json:"type"`
	Status   string `json:"status,omitempty"`
	OpenedAt *string `json:"opened_at,omitempty"`
	ClosedAt *string `json:"closed_at,omitempty"`
}

// CourtInfo represents court information.
type CourtInfo struct {
	Name     string `json:"name"`
	Level    string `json:"level"`
	Division string `json:"division,omitempty"`
	Judge    string `json:"judge,omitempty"`
}

// Party represents a case party.
type Party struct {
	Name     string `json:"name"`
	Role     string `json:"role"`
	Type     string `json:"type,omitempty"`
	Contact  string `json:"contact,omitempty"`
}

// Attorney represents a case attorney.
type Attorney struct {
	Name     string `json:"name"`
	BarNumber string `json:"bar_number,omitempty"`
	Role     string `json:"role"`
	Contact  string `json:"contact,omitempty"`
}

// Judge represents a judge.
type Judge struct {
	Name     string `json:"name"`
	Judicial string `json:"judicial,omitempty"`
}

// Charge represents a criminal charge.
type Charge struct {
	Code        string `json:"code"`
	Description string `json:"description"`
	Severity    string `json:"severity,omitempty"`
}

// Authority represents an authority/jurisdiction.
type Authority struct {
	Name  string `json:"name"`
	Type  string `json:"type,omitempty"`
	Code  string `json:"code,omitempty"`
}

// LegalTag represents a legal tag/category.
type LegalTag struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Category    string `json:"category,omitempty"`
}
```

**Action**: Copy all legal types from current `pkg/models/legal.go`.

### Step 1.4: Move Common Types

**File**: `pkg/models/core/types.go`

Contents with DocumentType enum and helpers:

```go
package core

// DocumentType represents the type of legal document.
type DocumentType string

const (
	// Motions
	DocTypeMotion DocumentType = "motion"
	DocTypeMotionToSuppressEvidence DocumentType = "motion_to_suppress_evidence"
	DocTypeMotionToDismiss DocumentType = "motion_to_dismiss"
	// ... other document types from legal.go
	
	// Orders
	DocTypeOrder DocumentType = "order"
	DocTypeOrderToDismiss DocumentType = "order_to_dismiss"
	// ... other orders
	
	// Pleadings
	DocTypeComplaint DocumentType = "complaint"
	DocTypeIndictment DocumentType = "indictment"
	// ... other pleadings
)

// IsMotion checks if document is a motion.
func (dt DocumentType) IsMotion() bool {
	return dt == DocTypeMotion || 
		dt == DocTypeMotionToSuppressEvidence ||
		dt == DocTypeMotionToDismiss
		// ... check other motion types
}

// IsOrder checks if document is an order.
func (dt DocumentType) IsOrder() bool {
	return dt == DocTypeOrder ||
		dt == DocTypeOrderToDismiss
		// ... check other order types
}

// IsPleading checks if document is a pleading.
func (dt DocumentType) IsPleading() bool {
	return dt == DocTypeComplaint ||
		dt == DocTypeIndictment
		// ... check other pleading types
}

// Category represents document category.
type Category string

const (
	CategoryFiling Category = "filing"
	CategoryOrder Category = "order"
	CategoryMotion Category = "motion"
	CategoryNotice Category = "notice"
	CategoryAppendix Category = "appendix"
)

// Confidence represents classification confidence (0.0-1.0).
type Confidence float64

func (c Confidence) IsValid() bool {
	return c >= 0.0 && c <= 1.0
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

**Action**: Extract DocumentType enum, Category enum, and all type definitions from current files.

### Step 1.5: Update Import Paths in `pkg/models/`

Update the main package file that exports from subdirectories:

**File**: `pkg/models/models.go` (NEW)

```go
// Package models provides all core data models used throughout the system.
package models

// Re-export core models
export (
	Document
	DocumentMetadata
	CaseInfo
	CourtInfo
	Party
	Attorney
	Judge
	Charge
	Authority
	LegalTag
	DocumentType
	Category
	Confidence
)

// Re-export from subpackages for backward compatibility
from "github.com/motion-index-fiber/pkg/models/core" import (
	Document
	DocumentMetadata
	CaseInfo
	CourtInfo
	Party
	Attorney
	Judge
	Charge
	Authority
	LegalTag
	DocumentType
	Category
	Confidence
)
```

**Action**: Create an index file that re-exports all public types for backward compatibility.

### Step 1.6: Verify Phase 1

```bash
# Run tests to verify no import issues
go test ./pkg/models/core/... -v

# Check that types are accessible
go build ./...
```

---

## Phase 2: API Models (Week 2)

### Step 2.1: Create API Request Models

**File**: `pkg/models/api/request.go`

Move all request DTOs from `internal/models/requests.go`:

```go
package api

// ProcessDocumentRequest is the request to process a document.
type ProcessDocumentRequest struct {
	FileName    string            `json:"file_name"`
	FilePath    string            `json:"file_path"`
	ContentType string            `json:"content_type"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// BatchProcessRequest is the request to batch process documents.
type BatchProcessRequest struct {
	Documents []ProcessDocumentRequest `json:"documents"`
}

// ClassifyDocumentRequest is the request to classify a document.
type ClassifyDocumentRequest struct {
	DocumentID string            `json:"document_id"`
	Text       string            `json:"text"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// UpdateClassificationRequest is the request to update classification.
type UpdateClassificationRequest struct {
	DocumentID string `json:"document_id"`
	DocumentType string `json:"document_type"`
	Category   string `json:"category"`
	Confidence float64 `json:"confidence"`
}

// SearchDocumentsRequest is the request to search documents.
type SearchDocumentsRequest struct {
	Query     string                 `json:"query,omitempty"`
	DocType   string                 `json:"doc_type,omitempty"`
	CaseNumber string                `json:"case_number,omitempty"`
	Filters   map[string]interface{} `json:"filters,omitempty"`
	Size      int                    `json:"size"`
	From      int                    `json:"from"`
}

// UpdateMetadataRequest is the request to update document metadata.
type UpdateMetadataRequest struct {
	DocumentID string                 `json:"document_id"`
	Metadata   map[string]interface{} `json:"metadata"`
}

// DeleteDocumentRequest is the request to delete a document.
type DeleteDocumentRequest struct {
	DocumentID string `json:"document_id"`
}

// GetDocumentRequest is the request to retrieve a document.
type GetDocumentRequest struct {
	DocumentID string `json:"document_id"`
}

// GetDocumentStatsRequest is the request to get document statistics.
type GetDocumentStatsRequest struct {
	DocumentID string `json:"document_id"`
}

// HealthCheckRequest is the request to check system health.
type HealthCheckRequest struct {
	// Empty - health check has no parameters
}

// AnalyzeRedactionsRequest is the request to analyze redactions.
type AnalyzeRedactionsRequest struct {
	DocumentID string `json:"document_id"`
	FilePath   string `json:"file_path"`
}

// BulkUploadRequest is the request to bulk upload documents.
type BulkUploadRequest struct {
	Documents []ProcessDocumentRequest `json:"documents"`
}
```

**Action**: Copy all request types, update package name to `api`.

### Step 2.2: Create API Response Models

**File**: `pkg/models/api/response.go`

Move all response DTOs from `internal/models/responses.go`:

```go
package api

import (
	"time"
	"github.com/motion-index-fiber/pkg/models/core"
)

// APIResponse is the standard HTTP response envelope.
type APIResponse struct {
	Success   bool        `json:"success"`
	Message   string      `json:"message,omitempty"`
	Data      interface{} `json:"data,omitempty"`
	Error     *APIError   `json:"error,omitempty"`
	RequestID string      `json:"request_id,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
}

// APIError represents an error in the API response.
type APIError struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

// ProcessDocumentResponse is the response from processing a document.
type ProcessDocumentResponse struct {
	DocumentID  string              `json:"document_id"`
	FileName    string              `json:"file_name"`
	Status      string              `json:"status"`
	Extraction  *ExtractionResult   `json:"extraction,omitempty"`
	Classification *ClassificationResult `json:"classification,omitempty"`
	ProcessedAt time.Time           `json:"processed_at"`
}

// BatchProcessResponse is the response from batch processing.
type BatchProcessResponse struct {
	TotalProcessed int                       `json:"total_processed"`
	Successful     int                       `json:"successful"`
	Failed         int                       `json:"failed"`
	Results        []ProcessDocumentResponse `json:"results"`
}

// ExtractionResult contains extraction results.
type ExtractionResult struct {
	Text       string `json:"text"`
	PageCount  int    `json:"page_count"`
	WordCount  int    `json:"word_count"`
	Language   string `json:"language,omitempty"`
	ExtractedAt time.Time `json:"extracted_at"`
}

// ClassificationResult contains classification results.
type ClassificationResult struct {
	DocumentType string    `json:"document_type"`
	Category     string    `json:"category"`
	Confidence   float64   `json:"confidence"`
	LegalTags    []string  `json:"legal_tags,omitempty"`
	ClassifiedAt time.Time `json:"classified_at"`
	ClassifiedBy string    `json:"classified_by,omitempty"`
}

// ProcessingStep represents a step in document processing.
type ProcessingStep struct {
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	StartedAt time.Time `json:"started_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	Duration  int       `json:"duration_ms,omitempty"`
	Error     string    `json:"error,omitempty"`
}

// SearchDocumentsResponse is the response from search.
type SearchDocumentsResponse struct {
	Query       string         `json:"query"`
	TotalHits   int64          `json:"total_hits"`
	Results     []SearchDocument `json:"results"`
	Aggregations map[string]interface{} `json:"aggregations,omitempty"`
	Page        int            `json:"page"`
	PageSize    int            `json:"page_size"`
}

// SearchDocument represents a single search result.
type SearchDocument struct {
	Document core.Document `json:"document"`
	Score    float64      `json:"score"`
	Highlights map[string][]string `json:"highlights,omitempty"`
}

// DocumentStatsResponse contains document statistics.
type DocumentStatsResponse struct {
	TotalDocuments   int64                  `json:"total_documents"`
	ByDocumentType   map[string]int64       `json:"by_document_type"`
	ByCategory       map[string]int64       `json:"by_category"`
	ByStatus         map[string]int64       `json:"by_status"`
	AverageConfidence float64              `json:"average_confidence"`
	LastIndexedAt    *time.Time             `json:"last_indexed_at,omitempty"`
}

// ClassificationResponse is the response from classification.
type ClassificationResponse struct {
	DocumentID    string                `json:"document_id"`
	Classification ClassificationResult `json:"classification"`
	Processing    []ProcessingStep      `json:"processing,omitempty"`
}

// IndexResult represents the result of an index operation.
type IndexResult struct {
	DocumentID string    `json:"document_id"`
	Success    bool      `json:"success"`
	Message    string    `json:"message,omitempty"`
	IndexedAt  time.Time `json:"indexed_at"`
}

// IndexDocumentResponse is the response from indexing.
type IndexDocumentResponse struct {
	Results []IndexResult `json:"results"`
	TotalIndexed int      `json:"total_indexed"`
	TotalFailed  int      `json:"total_failed"`
}
```

**Action**: Copy all response types, update package name to `api`.

### Step 2.3: Update API Models with Validation Helpers

Add validation methods to request models:

```go
// In pkg/models/api/request.go

// Validate validates the ProcessDocumentRequest.
func (r *ProcessDocumentRequest) Validate() error {
	if r.FileName == "" {
		return NewValidationError("file_name", "File name is required")
	}
	if r.FilePath == "" {
		return NewValidationError("file_path", "File path is required")
	}
	if r.ContentType == "" {
		return NewValidationError("content_type", "Content type is required")
	}
	return nil
}

// Validate validates the SearchDocumentsRequest.
func (r *SearchDocumentsRequest) Validate() error {
	if r.Query == "" {
		return NewValidationError("query", "Query is required")
	}
	if r.Size < 1 || r.Size > 1000 {
		return NewValidationError("size", "Size must be between 1 and 1000")
	}
	if r.From < 0 {
		return NewValidationError("from", "From must be greater than or equal to 0")
	}
	return nil
}

// ValidationError represents a validation error.
type ValidationError struct {
	Field   string
	Message string
}

func NewValidationError(field, message string) error {
	return &ValidationError{Field: field, Message: message}
}

func (e *ValidationError) Error() string {
	return e.Field + ": " + e.Message
}
```

### Step 2.4: Move Search Models

**File**: `pkg/models/search/request.go` and `pkg/models/search/response.go`

Move all search-specific models from current `pkg/models/search.go`.

**Action**: Extract search models into separate files maintaining logical grouping.

---

## Phase 3: Update Imports (Week 3-4)

### Step 3.1: Create Import Mapping

**File**: `IMPORT_MIGRATION_MAP.md`

Document all import path changes:

```markdown
# Import Migration Map

## Old → New Paths

### Core Models
- `github.com/motion-index-fiber/pkg/models.Document` → `github.com/motion-index-fiber/pkg/models/core.Document`
- `github.com/motion-index-fiber/pkg/models.DocumentMetadata` → `github.com/motion-index-fiber/pkg/models/core.DocumentMetadata`
- `github.com/motion-index-fiber/pkg/models.DocumentType` → `github.com/motion-index-fiber/pkg/models/core.DocumentType`

### API Models
- `github.com/motion-index-fiber/internal/models.ProcessDocumentRequest` → `github.com/motion-index-fiber/pkg/models/api.ProcessDocumentRequest`
- `github.com/motion-index-fiber/internal/models.APIResponse` → `github.com/motion-index-fiber/pkg/models/api.APIResponse`

### Search Models
- `github.com/motion-index-fiber/pkg/models.SearchRequest` → `github.com/motion-index-fiber/pkg/models/search.SearchRequest`
- `github.com/motion-index-fiber/pkg/models.SearchResult` → `github.com/motion-index-fiber/pkg/models/search.SearchResult`

### Validation Models
- `github.com/motion-index-fiber/internal/models.FileValidationRules` → `github.com/motion-index-fiber/pkg/models/validation.FileValidationRules`
- `github.com/motion-index-fiber/internal/models.SanitizeInput` → `github.com/motion-index-fiber/pkg/models/validation.SanitizeInput`
```

### Step 3.2: Use IDE Refactoring

1. Open the project in VS Code/GoLand
2. Use "Find and Replace in Files" feature
3. Enable "Use Regular Expression"
4. For each mapping:
   - Find: `from github.com/motion-index-fiber/pkg/models import \(([^)]+)\)`
   - Replace: `from github.com/motion-index-fiber/pkg/models/core import ($1)`
5. Test with `go build ./...`

### Step 3.3: Batch Update by Package

**Batch 1**: Internal handlers (50+ files)

```bash
# Update all handler imports
find ./internal/handlers -name "*.go" -type f | xargs sed -i \
  's|pkg/models|pkg/models/core|g'

# Verify
go build ./internal/handlers/...
go test ./internal/handlers/... -v
```

**Batch 2**: Services and use cases (30+ files)

```bash
# Update application layer
find ./internal/application -name "*.go" -type f | xargs sed -i \
  's|pkg/models|pkg/models/core|g'

# Verify
go build ./internal/application/...
go test ./internal/application/... -v
```

**Batch 3**: Infrastructure and storage (20+ files)

```bash
# Update infrastructure
find ./pkg -name "*.go" -type f | xargs sed -i \
  's|pkg/models|pkg/models/core|g'

# Verify
go build ./pkg/...
go test ./pkg/... -v
```

### Step 3.4: Manual Verification

For each batch:

```bash
# Check for compilation errors
go build ./...

# Run tests
go test ./... -v

# Check for circular imports
go test -run=TestNoCircularImports ./...

# Format code
go fmt ./...
```

---

## Phase 4: Cleanup & Finalization (Week 5-6)

### Step 4.1: Remove Old Files

Once all imports are updated and tests pass:

```bash
# Backup old files (optional)
mkdir backup_old_models
mv pkg/models/document.go backup_old_models/
mv pkg/models/legal.go backup_old_models/
mv pkg/models/search.go backup_old_models/
mv internal/models/ backup_old_models/

# Delete if backup is not needed
rm -rf backup_old_models/
```

### Step 4.2: Verify Final State

```bash
# Full test suite
go test ./... -v -coverprofile=coverage.out

# Coverage report
go tool cover -html=coverage.out

# Build production binary
go build -ldflags="-s -w" -o bin/server cmd/server/main.go

# Check for unused imports
go mod tidy
```

### Step 4.3: Update Documentation

1. Update CLAUDE.md with new import paths
2. Update README.md with new structure
3. Add examples of correct imports
4. Document any breaking changes

### Step 4.4: Code Review & Merge

1. Push feature branch to GitHub
2. Create Pull Request with detailed description
3. Link to CONSOLIDATION_PLAN.md and MODELS_INVENTORY.md
4. Include migration checklist
5. Assign reviewers
6. Address feedback
7. Merge to main branch

---

## Rollback Plan

If issues arise during consolidation:

### Minor Issues (specific imports failing)

```bash
# Revert specific file changes
git checkout -- internal/handlers/some_handler.go

# Re-run tests
go test ./... -v
```

### Major Issues (widespread compilation errors)

```bash
# Revert entire branch
git checkout main
git branch -D feat/consolidate-models

# Return to previous state
go test ./... -v
```

### Database/Data Issues

Since this is purely structural refactoring, no data migration is needed. All code changes are:
- Import path updates
- File location changes
- Package name changes
- NO changes to data structures or business logic

---

## Testing Checklist

- [ ] All unit tests pass
- [ ] All integration tests pass
- [ ] No circular imports
- [ ] No unused imports
- [ ] Code coverage maintained above current level
- [ ] API endpoints functional
- [ ] Search functionality working
- [ ] Document processing complete
- [ ] Classification working
- [ ] Health checks passing
- [ ] No performance regression

---

## Success Criteria

- [x] Phase 1: New directory structure created and verified
- [x] Phase 2: API models moved and updated
- [x] Phase 3: All imports updated without errors
- [x] Phase 4: Old files removed, final tests passing
- [ ] All 50+ files with correct new imports
- [ ] Test coverage at or above 80%
- [ ] Zero circular imports
- [ ] API documentation updated
- [ ] Team trained on new structure
- [ ] One sprint cycle without issues

---

## Maintenance Guidelines

### After Consolidation

1. **Always import from subpackages**: `import "github.com/motion-index-fiber/pkg/models/core"`
2. **Don't add to old locations**: All new models go to appropriate subpackage
3. **Keep re-exports updated**: If adding to models.go, also update re-exports
4. **Document new models**: Add godoc comments for all public types
5. **Test new models**: Include unit tests with 100% coverage
6. **Review on PR**: Ensure imports follow new pattern

### Creating New Models

**Good Practice**:
```go
// In pkg/models/search/advanced_filters.go
package search

// AdvancedFilter represents an advanced search filter
type AdvancedFilter struct {
    // fields
}
```

**Anti-Pattern**:
```go
// DON'T create models in internal/models after consolidation
package models // ❌ Old location

type AdvancedFilter struct {
    // fields
}
```

