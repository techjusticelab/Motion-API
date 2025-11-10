# Motion-Index-Fiber: Complete Models & Data Structures Inventory

## Executive Summary

The codebase contains models and data structures distributed across **4 main locations**:
- `pkg/models/` - Core domain and data transfer objects
- `internal/models/` - HTTP request/response wrappers
- `internal/domain/` - Rich domain entities with business logic (DDD)
- `internal/application/dto/` - Application layer data transfer objects

**Total Files**: 24 Go files defining models
**Total Structs/Types**: 100+ types across all packages

---

## 1. `pkg/models/` - Core Models (6 files)

### Location: `/home/okita/Scripts/Work/TJL/motion-api/pkg/models/`

#### Files and Structures:

**1.1 `document.go` - Primary Document Model**
- **`Document`** - Main document aggregate
  - `ID`: unique identifier
  - `FileName`: original file name
  - `FilePath`: storage location
  - `FileURL`: public access URL
  - `S3URI`: DigitalOcean Spaces URI
  - `Text`: extracted document text
  - `DocType`: legal document classification
  - `Category`: document category
  - `Hash`: content integrity hash
  - `CreatedAt`, `UpdatedAt`: timestamps
  - `Metadata`: DocumentMetadata (rich metadata)
  - `Size`: file size in bytes
  - `ContentType`: MIME type

- **`DocumentMetadata`** - Rich metadata container
  - Basic: `DocumentName`, `Subject`, `Summary`, `DocumentType`
  - Case: `Case` (CaseInfo object)
  - Court: `Court` (CourtInfo object)
  - People: `Parties`, `Attorneys`, `Judge`
  - Dates (enhanced): `FilingDate`, `EventDate`, `HearingDate`, `DecisionDate`, `ServedDate`
  - Status & Properties: `Language`, `Pages`, `WordCount`, `Status`
  - Legal: `LegalTags`, `Charges`, `Authorities`
  - Processing: `ProcessedAt`, `Confidence`, `AIClassified`

**Methods:**
- `GetCaseName()`, `GetCaseNumber()`, `GetCourtName()`, `GetJudgeName()`
- `HasLegalTag(tag string)`, `AddLegalTag(tag string)`
- `GetPrimaryAttorney()`, `GetDefenseAttorneys()`, `GetProsecutionAttorneys()`
- `IsMotion()`, `IsOrder()`, `IsPleading()`, `GetDocumentCategory()`

**Purpose**: Core data model representing legal documents with full metadata hierarchy

**Dependents**:
- `internal/models/requests.go` (re-exports)
- `internal/models/responses.go` (re-exports)
- 28 handler/service files

---

**1.2 `api.go` - API Response Wrappers**
- **`APIResponse`** - Standard API envelope
  - `Success`: boolean flag
  - `Message`: operation message
  - `Data`: generic payload
  - `Error`: error details (APIError)
  - `RequestID`: tracing ID
  - `Timestamp`: response time

- **`APIError`** - Structured error object
  - `Code`: error code
  - `Message`: human-readable message
  - `Details`: additional error context
  - `Field`: field-specific errors

**Methods:**
- `NewSuccessResponse()`, `NewErrorResponse()`, `NewValidationErrorResponse()`
- `SetRequestID()`, `IsError()`, `GetErrorCode()`, `GetErrorMessage()`
- `WithData()`, `WithMessage()` (chainable)

**Constants:** Error codes and messages (validation, auth, not found, etc.)

**Purpose**: Standardized HTTP response formatting across all endpoints

**Dependents**: All HTTP handlers and API endpoints

---

**1.3 `legal.go` - Legal Domain Types**
- **`CaseInfo`** - Case identification
  - `CaseNumber`: docket number
  - `CaseName`: case title
  - `CaseType`: criminal/civil/traffic
  - `Chapter`: bankruptcy chapter
  - `Docket`: full docket identifier
  - `NatureOfSuit`: legal category

- **`CourtInfo`** - Court details
  - `CourtID`: unique identifier
  - `CourtName`: court name
  - `Jurisdiction`: federal/state/local
  - `Level`: trial/appellate/supreme
  - `District`, `Division`, `County`: geographic info

- **`Party`** - Case party (defendant, plaintiff, etc.)
  - `Name`: party name
  - `Role`: defendant/plaintiff/appellant
  - `PartyType`: individual/corporation/government
  - `Date`: party-specific date

- **`Attorney`** - Legal counsel
  - `Name`: attorney name
  - `BarNumber`: bar license number
  - `Role`: defense/prosecution/counsel
  - `Organization`: firm/agency
  - `ContactInfo`: contact details

- **`Judge`** - Presiding judge
  - `Name`: judge name
  - `Title`: judge title
  - `JudgeID`: unique identifier

- **`Charge`** - Criminal charge
  - `Statute`: legal statute code
  - `Description`: charge description
  - `Grade`: felony/misdemeanor
  - `Class`: A/B/C classification
  - `Count`: count number

- **`Authority`** - Legal citation
  - `Citation`: case/statute citation
  - `CaseTitle`: case name
  - `Type`: case_law/statute/regulation
  - `Precedent`: boolean flag
  - `Page`: reference page/section

- **`DocumentType`** (string enum)
  - Motions: 9 types (suppress, dismiss, compel, etc.)
  - Orders/Rulings: 5 types
  - Pleadings: 5 types
  - Administrative: 7 types
  - Methods: `IsMotion()`, `IsOrder()`, `IsPleading()`, `GetCategory()`

- **`LegalTag`** - Classification tag
  - `Name`: tag name
  - `Category`: motion/evidence/procedural/substantive/criminal/civil
  - `Description`: tag description

**Functions:**
- `GetAllDocumentTypes()`, `ParseDocumentType()`
- `GetCommonLegalTags()`

**Purpose**: Legal domain terminology and entity definitions

**Dependents**: All legal document processing modules

---

**1.4 `search.go` - Search & Query Models**
- **`SearchRequest`** - Query parameters
  - Text: `Query`
  - Filters: `DocType`, `CaseNumber`, `CaseName`, `Judge`, `Court`, `Author`, `Status`, `LegalTags`
  - Dates: `DateRange`
  - Pagination: `Size`, `From`
  - Sorting: `SortBy`, `SortOrder`
  - Features: `IncludeHighlights`, `FuzzySearch`
  - Nested: `Filters`, `Sort`, `Pagination`, `Highlight`

- **`SearchResult`** - Query results
  - `TotalHits`: hit count
  - `MaxScore`: relevance score
  - `Documents`: [SearchDocument]
  - `Aggregations`: facet results
  - `Took`: query time
  - `TimedOut`: timeout flag

- **`SearchDocument`** - Individual result
  - `ID`: document ID
  - `Score`: relevance score
  - `Document`: raw document map
  - `Highlights`: matching excerpts

- **Aggregation Types**:
  - `TagCount`, `TypeCount`, `FieldValue`
  - `DocumentStats`, `FieldStat`
  - `FieldOptions`: available filter values
  - `AggregationBucket`: facet bucket

- **Bulk Operations**:
  - `BulkResult`: bulk operation result
  - `BulkResultItem`, `BulkItemResult`: per-item results
  - `BulkError`: error details
  - `BulkFailedDoc`: failed document info

- **Helper Types**:
  - `SortOrder`: enum (asc/desc)
  - `Filters`: structured filter object
  - `SortOptions`: sort configuration
  - `PaginationOptions`: pagination config
  - `HighlightOptions`: highlight config
  - `AggregationResponse`: aggregation wrapper

**Constants:** `DefaultSearchSize` (20), `MaxSearchSize` (100)

**Methods:**
- `ValidateSearchRequest()`: validation & defaults
- `ApplyDefaults()`: request defaults
- `GetEffectiveSize()`, `GetEffectiveFrom()`: computed fields
- `HasFilters()`, `GetFilterCount()`: filter utilities
- `NewSearchRequest()`, `NewSearchResult()`: constructors

**Purpose**: Search API request/response modeling and aggregation

**Dependents**: Search handlers, query builders, aggregation service

---

**1.5 `time.go` - Date/Time Models**
- **`DateRange`** - Date filtering
  - `From`, `To`: primary date fields
  - `Start`, `End`: alternative field names (backward compatibility)
  - Methods:
    - `IsValid()`: validates range
    - `GetFrom()`, `GetTo()`: smart getters
    - `SetFrom()`, `SetTo()`: smart setters
    - `IsEmpty()`: no constraints
    - `Contains(t)`: range membership test

**Purpose**: Date range filtering for search and reporting

**Dependents**: Search requests, metadata queries, statistics

---

**1.6 `document_part2.go` - OpenSearch Mapping Helpers**

Contains functions that build OpenSearch index mappings:

- `GetDocumentMapping()` - Full document mapping
- `getMetadataMapping()` - Metadata nested mapping
- `getCaseMapping()`, `getCourtMapping()`
- `getPartiesMapping()`, `getAttorneysMapping()`, `getJudgeMapping()`
- `getChargesMapping()`, `getAuthoritiesMapping()`

Defines field types, analyzers, and synonyms for legal terminology

**Purpose**: OpenSearch schema definition and index setup

**Dependents**: Index setup commands, search configuration

---

### Summary Table: `pkg/models/` Dependencies

| File | Primary Purpose | Used By | Count |
|------|---|---|---|
| document.go | Document data model | 28+ files | High |
| api.go | HTTP response envelope | All handlers | High |
| legal.go | Legal domain types | Domain layer, handlers | High |
| search.go | Search/aggregation models | Search handlers, services | High |
| time.go | Date range filtering | Search, metadata | Medium |
| document_part2.go | Index mapping helpers | Index setup, search | Low |

---

## 2. `internal/models/` - HTTP Layer Models (4 files)

### Location: `/home/okita/Scripts/Work/TJL/motion-api/internal/models/`

**Key Point**: These are HTTP-specific wrappers, mostly delegating to `pkg/models`

#### Files and Structures:

**2.1 `requests.go` - HTTP Request DTOs**
- **`ProcessDocumentRequest`** - File upload with metadata
  - File handling: multipart file
  - Category, description
  - Case info: name, number
  - Author, judge, court
  - Legal tags
  - Processing options

- **`ProcessOptions`** - Feature flags
  - `ExtractText`, `ClassifyDoc`, `IndexDocument`, `StoreDocument`
  - Timeout and retry settings

- **`BatchProcessRequest`** - Multiple file upload
  - Similar to ProcessDocumentRequest but accepts array of files

- **`UpdateMetadataRequest`** - Metadata updates
  - Document ID
  - Metadata map
  - Case/court/author/judge/legal tags updates

- **`DeleteDocumentRequest`** - Deletion request
  - Document ID
  - Deletion reason

- **`AnalyzeRedactionsRequest`** - Redaction analysis
  - File upload
  - Sensitivity level (low/medium/high)

- **`BulkUploadRequest`** - Bulk file upload
  - Multiple files
  - Category
  - Case info
  - Auto-process flag

- **`GetDocumentRequest`** - Retrieve document
  - Document ID
  - Format (json/full/metadata)

- **`GetDocumentStatsRequest`** - Statistics query
  - Date range
  - Group by (date/category/court/judge/author)
  - Granularity (day/week/month/year)

- **`HealthCheckRequest`** - System health check
  - Component filter
  - Deep check flag

- **Type Aliases**:
  - `SearchDocumentsRequest = models.SearchRequest`
  - `DateRange = models.DateRange`

**Default/Helper Functions:**
- `DefaultProcessOptions()`: standard options
- Validation methods on ProcessOptions

**Purpose**: HTTP request unmarshaling and validation

**Dependents**: Handlers, middleware, controllers

---

**2.2 `responses.go` - HTTP Response DTOs**
- **`ExtractionResult`** - Text extraction outcome
  - Text, page count, language

- **`ClassificationResult`** - Classification outcome
  - Category, confidence, tags

- **`ProcessDocumentResponse`** - Processing result
  - Document ID, file name, status
  - Processing time
  - Extraction/classification/index results
  - Storage result (URL, CDN URL)
  - Processing steps
  - Metadata
  - Timestamps

- **`BatchProcessResponse`** - Batch result
  - Batch ID, counts (total/success/failure)
  - Individual results
  - Batch errors
  - Processing time

- **`BatchProcessError`** - Per-file error
  - File name, error message, code

- **`IndexResult`** - Indexing outcome
  - Document ID, index name, success, error

- **`ProcessingStep`** - Pipeline step result
  - Name, status, times, duration, error

- **`SearchDocumentsResponse`** - Search results
  - Query, hits, scores, time
  - Pagination info
  - Documents
  - Aggregations
  - Suggestions

- **`DocumentStatsResponse`** - Statistics

- **Type Aliases**:
  - `APIResponse = models.APIResponse`
  - `APIError = models.APIError`
  - `DocumentMetadata = models.DocumentMetadata`

**Purpose**: HTTP response marshaling and structure

**Dependents**: All handlers returning responses

---

**2.3 `indexing.go` - Indexing Specific Models**
- **`IndexDocumentRequest`** - Single document indexing
  - Document ID, path, text
  - Classification result (from classifier)
  - File metadata (name, type, size, URL)

- **`IndexDocumentResponse`** - Indexing result
  - Document ID, index ID
  - Success flag, message, error
  - Indexed at timestamp

**Purpose**: OpenSearch indexing request/response

**Dependents**: Indexing handlers, index service

---

**2.4 `validation.go` - Input Validation Logic**

**Validation Functions:**
- `ValidateStruct()`: struct-level validation
- `FormatValidationErrors()`: error formatting
- `ValidateFile()`: single file validation
- `ValidateFiles()`: batch file validation
- `ValidateSearchQuery()`: query string validation
- `ValidateMetadata()`: metadata map validation
- `SanitizeInput()`: XSS/injection prevention

**Custom Validators:**
- `validateFileExtension()`: extension whitelist
- `validateFileSize()`: size limits
- `validateLegalCategory()`: category enum
- `removeScriptBlocks()`: sanitization
- `removeIframeBlocks()`: sanitization
- `removeJavaScriptProtocol()`: sanitization

**Configuration:**
- **`FileValidationRules`** - Configurable validation
  - Max size: 100MB default
  - Allowed extensions: pdf, doc, docx, txt, rtf, html, htm
  - Allowed MIME types
  - Min size: 1 byte

**Purpose**: Input validation and sanitization across HTTP layer

**Dependents**: All request handlers, middleware

---

### Summary: `internal/models/` Structure

These are HTTP/presentation layer models that wrap core `pkg/models` types or add HTTP-specific concerns. Heavy use of validation and error formatting.

---

## 3. `internal/domain/` - Rich Domain Models (DDD) (12 files)

### Location: `/home/okita/Scripts/Work/TJL/motion-api/internal/domain/`

**Key Characteristic**: Business logic-first, infrastructure-independent, event-sourced

#### 3.1 Document Aggregate (7 files in `document/`)

**`aggregate.go` - Document Aggregate Root**
- **`Document`** - Rich aggregate with behavior
  - Private fields: `id`, `version`, `fileName`, `filePath`, `s3URI`, `hash`
  - Content: `text`, `pageCount`, `wordCount`
  - `classification`: Classification aggregate
  - `metadata`: Metadata
  - Timestamps: `createdAt`, `updatedAt`, `processedAt`
  - Events: `events []DomainEvent`

- **DocumentOption** - Functional option pattern
  - `WithID()`, `WithFileName()`, `WithStoragePath()`
  - `WithS3URIString()`, `WithContentType()`, `WithHash()`
  - `WithText()`, `WithPageCount()`, `WithMetadata()`
  - etc. (20+ option functions)

- **Methods** (enforce invariants):
  - `NewDocument(opts ...DocumentOption)` - factory with validation
  - `ApplyClassification()` - state transition
  - `UpdateMetadata()` - metadata updates
  - `MarkAsProcessed()` - lifecycle management
  - `GetEvents()` - event sourcing
  - Accessors with validation

**Purpose**: Aggregate root protecting document invariants with behavior

---

**`value_objects.go` - Document Value Objects**

Value objects are immutable, validated, self-validating:

- **`DocumentID`** - String-based identifier
  - Validates non-empty
  - Methods: `String()`, `Equals()`, `IsEmpty()`

- **`FileName`** - File name with validation
  - Validates: non-empty, length ≤ 255, no path separators, valid extension
  - Allowed: .pdf, .doc, .docx, .txt, .rtf
  - Methods: `String()`, `Extension()`, `IsEmpty()`

- **`FilePath`** - Storage path
  - Validates: valid storage path format
  - Methods: `String()`, `IsEmpty()`

- **`FileSize`** - File size in bytes
  - Validates: ≤ 100MB
  - Methods: `Bytes()`, `IsValid()`

- **`ContentType`** - MIME type
  - Validates against whitelist
  - Allowed: application/pdf, application/msword, text/plain, etc.
  - Methods: `String()`, `IsValid()`

- **`Hash`** - Content hash
  - Validates: non-empty, valid algorithm (SHA256, MD5)
  - Methods: `Value()`, `Algorithm()`, `String()`

- **`S3URI`** - S3 cloud storage reference
  - Parses/validates S3 URI format
  - Methods: `Bucket()`, `Key()`, `String()`, `ParseS3URI()`

- **`Text`** - Document text content
  - Methods: `Content()`, `Length()`, `IsEmpty()`

- **`PageCount`** - Page count
  - Validates: ≥ 0
  - Methods: `Count()`, `IsValid()`

- **`WordCount`** - Word count
  - Validates: ≥ 0
  - Methods: `Count()`, `IsValid()`

- **`DocumentType`** - Classification type
  - Value object wrapping document type
  - Methods: `String()`, `IsZero()`, `IsValid()`

- **`Category`** - Document category
  - Value object for category
  - Methods: `String()`, `IsZero()`, `IsValid()`

- **`Confidence`** - Classification confidence (0.0-1.0)
  - Validates: 0.0 ≤ confidence ≤ 1.0
  - Methods: `Float64()`, `IsValid()`, `IsHigh()`, `IsMedium()`, `IsLow()`

**Purpose**: Type-safe, validated representations of document concepts

---

**`entity.go` - Document Entity**
- **`Document` entity definition** (separate from aggregate for persistence)
- Likely contains ORM/database mapping concerns

---

**`events.go` - Domain Events**
- **`DomainEvent` interface**
  - `OccurredAt() time.Time`
  - `AggregateID() string`
  - `EventType() string`

- **`DocumentCreatedEvent`** - Document creation
  - DocumentID, FileName, ContentType
  - OccurredAt timestamp

- **`DocumentClassifiedEvent`** - Classification applied
  - DocumentID, DocumentType, Category
  - Confidence, ClassifiedBy
  - OccurredAt

- **Event factory methods**: `NewDocumentCreatedEvent()`, `NewDocumentClassifiedEvent()`

**Purpose**: Event sourcing for audit trail and projections

---

**`repository.go` - Repository Interface**
- **`DocumentRepository` interface** (ports)
  - `Save(ctx, doc)`
  - `FindByID(ctx, id)`
  - `FindAll(ctx, filter)`
  - `Delete(ctx, id)`
  - `Exists(ctx, id)`
  - `Count(ctx, filter)`

**Purpose**: Abstraction for document persistence (implemented in infrastructure)

---

#### 3.2 Classification Domain (2 files in `classification/`)

**`classifier.go` - Classifier Interface**
- **`Classifier` interface** - Domain service contract
  - `Classify(ctx, text, metadata)` → Result
  - `GetSupportedCategories()` → []string
  - `IsConfigured()` → bool

**Purpose**: Domain contract for classification without depending on specific AI providers

---

**`result.go` - Classification Result Value Object**
- **`Result`** - Classification outcome
  - Private: `documentType`, `category`, `confidence`
  - `legalTags`: []string
  - `classifiedAt`: time.Time
  - `provider`: string
  - `explanation`: string

- **Constructor**: `NewResult()` - validates all invariants
- **Methods** (accessors with safe defaults):
  - `DocumentType()`, `Category()`, `Confidence()`
  - `LegalTags()` (returns copy), `ClassifiedAt()`
  - `Provider()`, `Explanation()`

**Purpose**: Immutable classification result with validation

---

#### 3.3 Legal Domain (4 files in `legal/`)

**`case.go` - Case Entity**
- **`CaseNumber` value object**
  - Validates: matches pattern `^[A-Za-z0-9\-:/]+$`
  - Methods: `String()`, `IsValid()`

- **`CaseType` enum** - legal case classification
  - Valid types: civil, criminal, bankruptcy, traffic, appeal
  - Methods: `String()`, `IsValid()`

- **`CaseStatus` enum** - case lifecycle
  - Valid statuses: open, closed, suspended, adjudicated
  - Methods: `String()`, `IsValid()`

- **`Case` entity** (main aggregate)
  - `caseNumber`: CaseNumber
  - `caseName`: string
  - `caseType`: CaseType
  - `status`: CaseStatus
  - `openedAt`, `closedAt`: time.Time
  - `parties`: []Party

- **Methods:**
  - `NewCase()` - factory with validation
  - `AddParty(party)` - add party to case
  - `UpdateStatus(status)` - state transition
  - Accessors with validation

**Purpose**: Case aggregate enforcing legal case invariants

---

**`party.go` - Party Entity**
- **`PartyRole` enum** - role in case
  - Valid roles: defendant, plaintiff, appellant, appellee, etc.

- **`PartyType` enum** - entity type
  - Valid types: individual, corporation, government

- **`Party` entity**
  - `name`: string
  - `role`: PartyRole
  - `partyType`: PartyType
  - `joinedAt`: time.Time

- **Methods:**
  - `NewParty()` - factory
  - `IsDefendant()`, `IsPlaintiff()` - role predicates
  - Accessors

**Purpose**: Party aggregate for case participants

---

**`attorney.go` - Attorney Entity**
- **`BarNumber` value object**
  - Validates: pattern `^[A-Za-z0-9\-]+$`
  - Methods: `String()`, `IsValid()`

- **`AttorneyRole` enum** - attorney role
  - Valid: defense, prosecution, counsel, plaintiff, respondent, appellant, amicus

- **`Attorney` entity**
  - `name`: string
  - `barNumber`: BarNumber
  - `role`: AttorneyRole
  - `organization`: string (firm/agency)
  - `phone`, `email`: contact info

- **Methods:**
  - `NewAttorney()` - factory
  - `IsDefense()`, `IsProsecution()` - role predicates
  - Accessors

**Purpose**: Attorney aggregate for legal representation

---

**`court.go` - Court Entity**
- **`Jurisdiction` enum** - court jurisdiction
  - Valid: federal, state, local

- **`CourtLevel` enum** - court hierarchy
  - Valid: trial, appellate, supreme

- **`Court` entity**
  - `courtID`: string
  - `courtName`: string
  - `jurisdiction`: Jurisdiction
  - `level`: CourtLevel
  - `district`, `division`, `county`: location info

- **Methods:**
  - `NewCourt()` - factory
  - `IsFederal()`, `IsState()` - jurisdiction predicates
  - Accessors

**Purpose**: Court aggregate for judicial system representation

---

#### 3.4 Domain Errors (1 file in `errors/`)

**`errors.go` - Domain Error Definitions**

**Exported Errors** (sentinel values):
```
// Document errors
ErrDocumentNotFound, ErrEmptyDocumentID, ErrEmptyFileName
ErrInvalidFileName, ErrInvalidFilePath, ErrEmptyFilePath
ErrInvalidS3URI, ErrEmptyHash, ErrInvalidHash
ErrInvalidContentType, ErrInvalidFileSize
ErrCannotClassifyEmptyDocument

// Classification errors
ErrInvalidConfidence, ErrInvalidClassification
ErrClassifierNotConfigured, ErrUnsupportedClassification

// Legal entity errors
ErrInvalidCaseNumber, ErrInvalidPartyRole, ErrInvalidBarNumber

// Repository errors
ErrDuplicateDocument, ErrConcurrencyConflict
```

**`ValidationError` type** - Structured validation error
  - `Field`: field name
  - `Message`: error message
  - `Value`: invalid value
  - Methods: `Error()`, `NewValidationError()`

**Purpose**: Domain-specific error definitions for invariant violations

---

### Summary: `internal/domain/` Structure

| Package | Purpose | Pattern |
|---------|---------|---------|
| document/ | Document aggregate root | DDD Aggregate |
| classification/ | Classification domain service | Domain Service |
| legal/ | Case, party, attorney, court | DDD Entities |
| errors/ | Domain error definitions | Sentinel Errors |

**Key Features:**
- Invariant enforcement through value objects
- Event sourcing for audit trails
- No infrastructure dependencies
- Functional options pattern for construction
- Rich business logic in aggregates

---

## 4. `internal/application/dto/` - Application Layer DTOs (5 files)

### Location: `/home/okita/Scripts/Work/TJL/motion-api/internal/application/dto/`

**Purpose**: Transfer objects between layers, validation at application boundary

#### Files:

**4.1 `document.go` - Document Processing DTOs**
- **`ProcessDocumentRequest` DTO**
  - `ID`, `FileName`, `Content` (io.Reader), `ContentSize`
  - `ContentType`, `StoragePath`, `Text`
  - `HashValue`, `HashAlgorithm`
  - Feature flags: `StoreBinary`, `Classify`, `Index`
  - `Metadata`, `ProcessedAt`
  - `Validate()` method with domain validation

- **`ProcessDocumentResponse` DTO**
  - `DocumentID`, `Stored`, `StorageURL`
  - `Classified`, `Indexed`
  - `CreatedAt`, `ProcessedAt`

**Purpose**: Application-level document processing contract

---

**4.2 `classification.go` - Classification DTOs**
- **`ClassifyDocumentRequest` DTO**
  - `DocumentID`, `Force` flag
  - `Validate()` method

- **`ClassificationResponse` DTO**
  - `DocumentID`, `DocumentType`, `Category`
  - `Confidence`, `LegalTags`
  - `ClassifiedBy`, `ClassifiedAt`

- **`UpdateClassificationRequest` DTO**
  - `DocumentID`, `LegalTags`
  - `Validate()` method

**Purpose**: Classification use case contracts

---

**4.3 `search.go` - Search DTOs**
- **`SearchDocumentsRequest` DTO**
  - `Query`, `DocumentType`, `Category`
  - `LegalTags`, `FromDate`, `ToDate`, `MinConfidence`
  - `Page`, `PageSize`, `SortBy`, `SortOrder`
  - `Validate()` method

- **`SearchResultsResponse` DTO**
  - `Documents`: []DocumentSummary
  - `Total`, `Page`, `PageSize`, `TotalPages`

- **`DocumentSummary` DTO**
  - `ID`, `FileName`, `DocumentType`

**Purpose**: Search use case contracts

---

**4.4 `redaction.go` - Redaction DTOs**
(Need to read this file for details)

**4.5 `storage.go` - Storage DTOs**
(Need to read this file for details)

---

## 5. Model Dependencies Analysis

### Cross-Package Import Graph

```
pkg/models/
  ├── api.go (APIResponse, APIError)
  │   └── Used by: All HTTP handlers, internal/models
  ├── document.go (Document, DocumentMetadata)
  │   └── Used by: 28+ files (handlers, services, search, pipeline)
  ├── legal.go (CaseInfo, Party, Attorney, etc.)
  │   └── Used by: Document metadata, legal processing
  ├── search.go (SearchRequest, SearchResult, etc.)
  │   └── Used by: Search handlers, query builders
  ├── time.go (DateRange)
  │   └── Used by: Search, metadata queries
  └── document_part2.go (Mapping helpers)
      └── Used by: Index setup, search configuration

internal/models/
  ├── requests.go (ProcessDocumentRequest, etc.)
  │   └── Used by: HTTP handlers, middleware
  ├── responses.go (ProcessDocumentResponse, etc.)
  │   └── Used by: HTTP handlers
  ├── indexing.go (IndexDocumentRequest, etc.)
  │   └── Used by: Indexing handlers
  └── validation.go (Validation functions)
      └── Used by: All HTTP layer

internal/domain/
  ├── document/
  │   ├── aggregate.go (Document aggregate)
  │   ├── value_objects.go (DocumentID, FileName, etc.)
  │   ├── events.go (DomainEvent, DocumentCreatedEvent)
  │   ├── entity.go (Document entity)
  │   └── repository.go (DocumentRepository interface)
  ├── classification/
  │   ├── classifier.go (Classifier interface)
  │   └── result.go (Result value object)
  ├── legal/
  │   ├── case.go (Case, CaseNumber, CaseType)
  │   ├── party.go (Party, PartyRole)
  │   ├── attorney.go (Attorney, BarNumber)
  │   └── court.go (Court, Jurisdiction)
  └── errors/
      └── errors.go (Domain errors, ValidationError)

internal/application/dto/
  ├── document.go (ProcessDocumentRequest, ProcessDocumentResponse)
  ├── classification.go (ClassifyDocumentRequest, ClassificationResponse)
  ├── search.go (SearchDocumentsRequest, SearchResultsResponse)
  ├── redaction.go (TBD)
  └── storage.go (TBD)
```

### Layer Dependencies

```
HTTP Layer (handlers)
  ↓ depends on
internal/models + pkg/models
  ↓ depends on
internal/application/dto
  ↓ depends on
internal/domain (aggregates, value objects, services)
  ↓ depends on
internal/domain/errors
  ↓ (no external dependencies)
```

---

## 6. Consolidation Opportunity Analysis

### Current State
- **4 separate model locations** with overlapping concepts
- **Multiple definitions** of similar structures (Document appears in multiple layers)
- **Type aliases** used to bridge between layers (`type SearchDocumentsRequest = models.SearchRequest`)
- **Circular import risks** between pkg/models and internal/models

### Consolidation Strategy

**Recommended Structure**:
```
pkg/models/  (public API models)
  ├── core/
  │   ├── document.go (Document, DocumentMetadata)
  │   ├── legal.go (CaseInfo, Party, Attorney, etc.)
  │   └── types.go (DocumentType, LegalTag, etc.)
  ├── api/
  │   ├── request.go (all HTTP requests)
  │   ├── response.go (all HTTP responses)
  │   └── errors.go (HTTP error types)
  ├── search/
  │   ├── request.go (SearchRequest and variants)
  │   ├── result.go (SearchResult and variants)
  │   └── aggregation.go (Aggregation types)
  └── validation/
      └── validators.go (Validation functions, rules)

internal/domain/  (DDD layer - unchanged)
  ├── document/
  ├── classification/
  ├── legal/
  └── errors/

internal/application/  (Application layer - keep minimal)
  └── dto/
      ├── mappers.go (DTO ↔ Domain mappings)
      └── validators.go (Application validation)
```

### Benefits of Consolidation
1. **Single source of truth** for models
2. **Reduced duplication** of type definitions
3. **Clearer layer boundaries** between HTTP, application, and domain
4. **Easier maintenance** when adding new features
5. **Type safety improvements** with proper package organization

---

## 7. Type Count Summary

| Location | Count | Category |
|----------|-------|----------|
| `pkg/models/document.go` | 2 | Core models |
| `pkg/models/api.go` | 2 | HTTP envelope |
| `pkg/models/legal.go` | 8 | Legal types |
| `pkg/models/search.go` | 25+ | Search/aggregation |
| `pkg/models/time.go` | 1 | Date/time |
| `internal/models/requests.go` | 8 | HTTP requests |
| `internal/models/responses.go` | 7 | HTTP responses |
| `internal/models/indexing.go` | 2 | Indexing models |
| `internal/models/validation.go` | 2 | Validation |
| `internal/domain/document/` | 20+ | DDD aggregates & VOs |
| `internal/domain/classification/` | 3 | Domain services |
| `internal/domain/legal/` | 8 | Domain entities |
| `internal/domain/errors/` | 2 | Error types |
| `internal/application/dto/` | 10+ | Application DTOs |
| **TOTAL** | **~100+** | - |

---

## 8. Key Insights

### Layering
1. **pkg/models** - Public API, used across the system
2. **internal/models** - HTTP-specific wrappers, validation
3. **internal/domain** - Business logic, no infrastructure
4. **internal/application/dto** - Layer boundaries, mappers

### Validation Strategy
- **Value objects** validate in `internal/domain/`
- **Input validation** in `internal/models/validation.go`
- **Application validation** in `internal/application/dto/`
- **Search validation** in `pkg/models/search.go`

### Backward Compatibility
- Multiple field names (e.g., From/Start, To/End in DateRange)
- Type aliases bridge between layers
- Legacy fields in DocumentMetadata for compatibility

### Domain-Driven Design
- Rich aggregates in `internal/domain/`
- Value objects enforce invariants
- Event sourcing for audit trail
- Repository pattern for persistence

---

## 9. Files Requiring Consolidation Attention

**Priority 1 (High Duplication)**:
- `pkg/models/document.go` + `internal/models/requests.go`
- `pkg/models/legal.go` + `internal/domain/legal/`
- `pkg/models/search.go` (very large, might split)

**Priority 2 (Medium Duplication)**:
- API response types spread across pkg/models and internal/models
- Validation logic in multiple locations

**Priority 3 (Low Risk)**:
- Event definitions (domain layer only)
- Validation error types


---

## Appendix A: Quick Reference - Model Locations

### By File

| File Path | Structs/Types | Primary Purpose | Dependency Count |
|-----------|---------------|-----------------|-----------------|
| `pkg/models/api.go` | APIResponse, APIError | HTTP envelope | High |
| `pkg/models/document.go` | Document, DocumentMetadata | Core document model | High |
| `pkg/models/document_part2.go` | Mapping functions | Index schema | Low |
| `pkg/models/legal.go` | CaseInfo, Party, Attorney, Judge, Charge, Authority, DocumentType, LegalTag | Legal entities | High |
| `pkg/models/search.go` | SearchRequest, SearchResult, SearchDocument, TagCount, TypeCount, FieldValue, DocumentStats, FieldStat, FieldOptions, BulkResult, BulkResultItem, BulkItemResult, BulkError, BulkFailedDoc, SortOrder, Filters, SortOptions, PaginationOptions, HighlightOptions, AggregationBucket, AggregationResponse, SearchResponse, SearchError, IndexStats, ShardInfo, PerformanceStats | Search/aggregation API | High |
| `pkg/models/time.go` | DateRange | Date range filtering | Medium |
| **internal/models/requests.go** | ProcessDocumentRequest, ProcessOptions, BatchProcessRequest, UpdateMetadataRequest, DeleteDocumentRequest, AnalyzeRedactionsRequest, BulkUploadRequest, GetDocumentRequest, GetDocumentStatsRequest, HealthCheckRequest | HTTP request DTOs | High |
| **internal/models/responses.go** | ExtractionResult, ClassificationResult, ProcessDocumentResponse, BatchProcessResponse, BatchProcessError, IndexResult, ProcessingStep, SearchDocumentsResponse, DocumentStatsResponse | HTTP response DTOs | High |
| **internal/models/indexing.go** | IndexDocumentRequest, IndexDocumentResponse | Indexing DTOs | Medium |
| **internal/models/validation.go** | ValidationError, FileValidationRules | Validation utilities | High |
| **internal/domain/document/aggregate.go** | Document, DocumentOption | Document aggregate root | High |
| **internal/domain/document/value_objects.go** | DocumentID, FileName, FilePath, FileSize, ContentType, Hash, S3URI, Text, PageCount, WordCount, DocumentType, Category, Confidence | Value objects | High |
| **internal/domain/document/entity.go** | Document entity | Entity definition | Medium |
| **internal/domain/document/events.go** | DomainEvent, DocumentCreatedEvent, DocumentClassifiedEvent | Domain events | Medium |
| **internal/domain/document/repository.go** | DocumentRepository | Repository interface | Medium |
| **internal/domain/classification/classifier.go** | Classifier | Domain service interface | Medium |
| **internal/domain/classification/result.go** | Result | Classification value object | Medium |
| **internal/domain/legal/case.go** | CaseNumber, CaseType, CaseStatus, Case | Case aggregate | Medium |
| **internal/domain/legal/party.go** | PartyRole, PartyType, Party | Party entity | Medium |
| **internal/domain/legal/attorney.go** | BarNumber, AttorneyRole, Attorney | Attorney entity | Medium |
| **internal/domain/legal/court.go** | Jurisdiction, CourtLevel, Court | Court entity | Medium |
| **internal/domain/errors/errors.go** | ValidationError (domain), sentinel errors | Domain errors | High |
| **internal/application/dto/document.go** | ProcessDocumentRequest, ProcessDocumentResponse | Application DTOs | Medium |
| **internal/application/dto/classification.go** | ClassifyDocumentRequest, ClassificationResponse, UpdateClassificationRequest | Classification DTOs | Medium |
| **internal/application/dto/search.go** | SearchDocumentsRequest, SearchResultsResponse, DocumentSummary | Search DTOs | Medium |
| **internal/application/dto/redaction.go** | (to be detailed) | Redaction DTOs | TBD |
| **internal/application/dto/storage.go** | (to be detailed) | Storage DTOs | TBD |

---

## Appendix B: Type Categories

### Domain Value Objects (Internal Domain Layer)
- `DocumentID`, `FileName`, `FilePath`, `FileSize`, `ContentType`, `Hash`, `S3URI`
- `Text`, `PageCount`, `WordCount`
- `DocumentType`, `Category`, `Confidence`
- `CaseNumber`, `CaseType`, `CaseStatus`
- `BarNumber`, `AttorneyRole`, `PartyRole`, `PartyType`
- `Jurisdiction`, `CourtLevel`

### Domain Entities & Aggregates (Internal Domain Layer)
- `Document` (aggregate root)
- `Case` (aggregate)
- `Party` (entity)
- `Attorney` (entity)
- `Court` (entity)

### Business Models (pkg/models Layer)
- `Document`, `DocumentMetadata`
- `CaseInfo`, `CourtInfo`, `Party`, `Attorney`, `Judge`, `Charge`, `Authority`
- `DocumentType`, `LegalTag`

### Search & Query Models (pkg/models Layer)
- `SearchRequest`, `SearchResult`, `SearchDocument`
- `TagCount`, `TypeCount`, `FieldValue`, `DocumentStats`, `FieldStat`
- `FieldOptions`, `AggregationBucket`, `AggregationResponse`
- `Filters`, `SortOptions`, `PaginationOptions`, `HighlightOptions`
- `BulkResult`, `BulkResultItem`, `BulkItemResult`, `BulkError`, `BulkFailedDoc`
- `IndexStats`, `ShardInfo`, `PerformanceStats`

### HTTP Request Models (internal/models & internal/application/dto)
- `ProcessDocumentRequest`, `BatchProcessRequest`, `UpdateMetadataRequest`
- `DeleteDocumentRequest`, `AnalyzeRedactionsRequest`, `BulkUploadRequest`
- `GetDocumentRequest`, `GetDocumentStatsRequest`, `HealthCheckRequest`
- `ClassifyDocumentRequest`, `UpdateClassificationRequest`
- `SearchDocumentsRequest`, `IndexDocumentRequest`

### HTTP Response Models (internal/models & internal/application/dto)
- `ProcessDocumentResponse`, `BatchProcessResponse`, `BatchProcessError`
- `ExtractionResult`, `ClassificationResult`, `IndexResult`, `ProcessingStep`
- `SearchDocumentsResponse`, `DocumentStatsResponse`
- `ClassificationResponse`, `IndexDocumentResponse`
- `SearchResultsResponse`, `DocumentSummary`
- `APIResponse`, `APIError`

### Enumerations
- `DocumentType` (26+ document types)
- `SortOrder` (asc, desc)
- `CaseType` (civil, criminal, bankruptcy, traffic, appeal)
- `CaseStatus` (open, closed, suspended, adjudicated)
- `PartyRole` (defendant, plaintiff, appellant, etc.)
- `PartyType` (individual, corporation, government)
- `AttorneyRole` (defense, prosecution, counsel, etc.)
- `Jurisdiction` (federal, state, local)
- `CourtLevel` (trial, appellate, supreme)

### Validation & Configuration Types
- `ValidationError` (validation error structure)
- `FileValidationRules` (file validation configuration)
- `DateRange` (date filtering with backward compatibility)

---

## Appendix C: Direct Dependents by Model

### Highest Dependency Models
1. **Document** (28+ files)
   - Handlers: processing, search, storage, indexing
   - Services: search, pipeline, migration
   - Commands: setup-index, inspect-index

2. **SearchRequest** (12+ files)
   - Search handlers, query builders, services
   - Aggregation services

3. **APIResponse** (All handlers)
   - Every HTTP endpoint uses this for responses

4. **DocumentMetadata** (15+ files)
   - Document processing, search, storage
   - Handlers, services

### Medium Dependency Models
- `DateRange` (Search, metadata, statistics)
- `ProcessingStep`, `ProcessDocumentResponse` (Handlers, responses)
- `Classification Result`, `Classifier` (AI services, pipelines)
- Domain aggregates (Use cases, repositories)

### Low Dependency Models
- Document mapping helpers (Index setup only)
- Specific domain value objects (Limited to their aggregate)
- Health check models (Health check endpoint only)

---

## Appendix D: File Cross-References

### Which Package Imports What

**pkg/models** imported by:
```
internal/models/requests.go
internal/models/responses.go
internal/application/dto/*.go
internal/handlers/**/*.go (28+ files)
cmd/*/main.go
internal/infrastructure/**/*.go
pkg/processing/**/*.go
pkg/search/**/*.go
```

**internal/models** imported by:
```
internal/handlers/**/*.go
internal/infrastructure/http/**/*.go
cmd/api-classifier/main.go
(HTTP layer only)
```

**internal/domain** imported by:
```
internal/application/dto/*.go
internal/application/usecase/**/*.go
internal/application/ports/*.go
internal/application/validation/*.go
internal/domain/**/*.go (internal references)
internal/infrastructure/persistence/*.go
internal/infrastructure/ai/*.go
internal/infrastructure/events/*.go
```

**internal/application/dto** imported by:
```
internal/application/usecase/**/*.go (all use cases)
internal/handlers/**/*.go (modern refactored handlers)
internal/infrastructure/http/**/*.go
internal/infrastructure/http/presenter/*.go
```

---

## Appendix E: Model Size Statistics

### By File Size Estimate

| File | Approx Lines | Types/Structs |
|------|--------------|---------------|
| `pkg/models/search.go` | 400+ | 25+ |
| `pkg/models/document.go` | 360+ | 2 |
| `pkg/models/legal.go` | 214 | 8 |
| `pkg/models/api.go` | 122 | 2 |
| `internal/models/validation.go` | 400+ | 2 |
| `internal/models/requests.go` | 177+ | 10 |
| `internal/models/responses.go` | 100+ | 7 |
| `internal/domain/document/aggregate.go` | 300+ | 1 |
| `internal/domain/document/value_objects.go` | 400+ | 12 |
| `internal/domain/legal/case.go` | 150+ | 4 |
| `internal/domain/legal/attorney.go` | 150+ | 3 |

**Largest Files**: 
1. `pkg/models/search.go` (400+ lines)
2. `internal/models/validation.go` (400+ lines)
3. `internal/domain/document/value_objects.go` (400+ lines)

**Consolidation Candidates** (Large files that could be split):
- `pkg/models/search.go` → could split into request.go, result.go, aggregation.go
- `internal/models/validation.go` → could move sanitization to pkg/models/validation/
- `internal/domain/document/value_objects.go` → consider splitting into individual files per VO

---

## Appendix F: Import Cycles Risk Analysis

### Potential Circular Dependency Issues

**Currently Safe**:
- ✅ `pkg/models` → `internal/models` (one-way)
- ✅ `internal/domain` → `internal/domain/errors` (one-way)
- ✅ `internal/application/dto` → `internal/domain` (one-way)

**At Risk**:
- ⚠️ `pkg/models` ← → `internal/models` (type aliases used to bridge)
- ⚠️ `internal/domain` ← → `pkg/models` (no direct imports, but similar concepts)

**Recommendation**: 
- Never have `pkg/models` import from `internal/*`
- Never have `internal/models` import from `internal/domain` or `internal/application`
- Use DTOs and mappers for layer transitions

---

## Appendix G: Backward Compatibility Fields

### Fields Kept for Legacy Support

**DateRange**:
- `From`, `To` (primary)
- `Start`, `End` (legacy alternatives)

**DocumentMetadata**:
- `DocumentType` (primary)
- `CaseName`, `CaseNumber` (legacy from Case object)
- `Author` (legacy from Attorney)

**SearchRequest**:
- `Size`, `From` (primary)
- `Limit` (legacy alternative for Size)

**SearchResponse**:
- `Data` field (new structure)
- `Documents` field (legacy array)
- `Total` field (legacy hit count)

**ProcessingStep**:
- `StartTime`, `EndTime` (absolute times)
- `Duration` (computed from timestamps)

---

## Appendix H: Configuration & Constants

### Search Constants
```go
const (
    DefaultSearchSize = 20
    MaxSearchSize     = 100
)
```

### File Validation Defaults
```go
MaxSize             = 100 * 1024 * 1024  // 100MB
AllowedExtensions   = []string{"pdf", "doc", "docx", "txt", "rtf", "html", "htm"}
AllowedMIMETypes    = []string{
    "application/pdf",
    "application/msword",
    "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
    "text/plain",
    "application/rtf",
    "text/html",
}
MinSize             = 1  // 1 byte minimum
```

### Legal Category Constants
```go
LegalTagCategoryMotion       = "motion"
LegalTagCategoryEvidence     = "evidence"
LegalTagCategoryProcedural   = "procedural"
LegalTagCategorySubstantive  = "substantive"
LegalTagCategoryCriminal     = "criminal"
LegalTagCategoryCivil        = "civil"
```

### Document Type Constants
- 9 Motion types (suppress, dismiss, compel, in limine, summary judgment, etc.)
- 5 Order/Ruling types
- 5 Pleading types
- 7 Administrative types

### API Error Codes
```go
ErrorCodeValidation        = "validation_error"
ErrorCodeAuthentication    = "authentication_error"
ErrorCodeAuthorization     = "authorization_error"
ErrorCodeNotFound          = "not_found"
ErrorCodeConflict          = "conflict"
ErrorCodeRateLimit         = "rate_limit_exceeded"
ErrorCodeInternal          = "internal_error"
ErrorCodeBadRequest        = "bad_request"
ErrorCodeTimeout           = "timeout"
ErrorCodeServiceUnavailable = "service_unavailable"
```

---

