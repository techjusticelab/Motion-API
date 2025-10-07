# Architecture & Design Decisions

Detailed architecture documentation for Motion-Index Fiber, covering design decisions, patterns, and implementation details.

## System Architecture

### High-Level Overview
```
┌─────────────────────────────────────────────────────────────┐
│                    Client Layer                             │
├─────────────────┬─────────────────┬─────────────────────────┤
│   Web Client    │   Mobile App    │   API Integrations      │
└─────────────────┴─────────────────┴─────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                  API Gateway Layer                          │
├─────────────────┬─────────────────┬─────────────────────────┤
│   Rate Limiting │   Authentication│   Request Validation    │
│   Load Balancing│   CORS Handling │   Response Formatting   │
└─────────────────┴─────────────────┴─────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                Application Layer                            │
├─────────────────┬─────────────────┬─────────────────────────┤
│  HTTP Handlers  │   Middleware    │   Request Processing    │
│  Input Validation│  Error Handling │   Response Formation   │
└─────────────────┴─────────────────┴─────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                 Service Layer                               │
├─────────────────┬─────────────────┬─────────────────────────┤
│ Document Service│  Search Service │   Processing Service    │
│ Business Logic  │  Query Building │   Workflow Orchestration│
└─────────────────┴─────────────────┴─────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│               Infrastructure Layer                          │
├─────────────────┬─────────────────┬─────────────────────────┤
│ Storage Service │ Search Service  │  External APIs          │
│ (DigitalOcean)  │ (OpenSearch)    │  (OpenAI, Supabase)     │
└─────────────────┴─────────────────┴─────────────────────────┘
```

### Component Interaction Flow
```
1. Client Request → API Gateway
2. Authentication & Rate Limiting Check
3. Route to Handler → Middleware Chain
4. Handler → Service Layer (Business Logic)
5. Service → Infrastructure Services
6. Response Formation → Client
```

## Design Principles

### 1. UNIX Philosophy Implementation

#### Do One Thing Well
Each component has a single, focused responsibility:

```go
// Good: Single responsibility
type DocumentExtractor interface {
    ExtractText(ctx context.Context, file io.Reader, format string) (*ExtractedText, error)
}

type DocumentClassifier interface {
    ClassifyDocument(ctx context.Context, text string) (*Classification, error)
}

// Not good: Multiple responsibilities
type DocumentProcessor interface {
    ExtractText(ctx context.Context, file io.Reader) (*ExtractedText, error)
    ClassifyDocument(ctx context.Context, text string) (*Classification, error)
    StoreDocument(ctx context.Context, doc *Document) error
    IndexDocument(ctx context.Context, doc *Document) error
    SendNotification(ctx context.Context, user string, message string) error
}
```

#### Composability
Services can be combined and swapped:

```go
// Interfaces allow easy composition and testing
type DocumentService struct {
    extractor   DocumentExtractor
    classifier  DocumentClassifier
    storage     storage.Service
    search      search.Service
}

func NewDocumentService(
    extractor DocumentExtractor,
    classifier DocumentClassifier,
    storage storage.Service,
    search search.Service,
) *DocumentService {
    return &DocumentService{
        extractor:  extractor,
        classifier: classifier,
        storage:    storage,
        search:     search,
    }
}
```

#### Testability
Every component is designed for testing:

```go
// Mock implementations for testing
type MockDocumentExtractor struct {
    mock.Mock
}

func (m *MockDocumentExtractor) ExtractText(ctx context.Context, file io.Reader, format string) (*ExtractedText, error) {
    args := m.Called(ctx, file, format)
    return args.Get(0).(*ExtractedText), args.Error(1)
}

// Test using mock
func TestDocumentService_Process(t *testing.T) {
    mockExtractor := &MockDocumentExtractor{}
    mockClassifier := &MockDocumentClassifier{}
    mockStorage := &storage.MockService{}
    mockSearch := &search.MockService{}
    
    service := NewDocumentService(mockExtractor, mockClassifier, mockStorage, mockSearch)
    
    // Configure mocks
    mockExtractor.On("ExtractText", mock.Anything, mock.Anything, "pdf").Return(
        &ExtractedText{Content: "test content"}, nil)
    
    // Test
    result, err := service.Process(context.Background(), testFile, "pdf")
    
    // Verify
    require.NoError(t, err)
    mockExtractor.AssertExpectations(t)
}
```

### 2. Dependency Injection Pattern

#### Service Factory Pattern
```go
// Central factory for service creation
type ServiceFactory struct {
    config  *config.Config
    logger  *slog.Logger
    storage storage.Service
    search  search.Service
}

func NewServiceFactory(cfg *config.Config) (*ServiceFactory, error) {
    factory := &ServiceFactory{
        config: cfg,
        logger: slog.Default(),
    }
    
    if err := factory.initializeServices(); err != nil {
        return nil, fmt.Errorf("failed to initialize services: %w", err)
    }
    
    return factory, nil
}

func (f *ServiceFactory) CreateDocumentService() DocumentService {
    extractor := processing.NewTextExtractor()
    classifier := processing.NewAIClassifier(f.config.OpenAI.APIKey)
    
    return NewDocumentService(extractor, classifier, f.storage, f.search)
}
```

#### Configuration Injection
```go
// Configuration is injected, not accessed globally
type StorageService struct {
    config  StorageConfig
    client  *s3.Client
    logger  *slog.Logger
}

func NewStorageService(cfg StorageConfig) (*StorageService, error) {
    client, err := createS3Client(cfg)
    if err != nil {
        return nil, fmt.Errorf("failed to create S3 client: %w", err)
    }
    
    return &StorageService{
        config: cfg,
        client: client,
        logger: slog.Default().With("component", "storage"),
    }, nil
}
```

### 3. Error Handling Strategy

#### Error Wrapping and Context
```go
// Wrap errors with context at each layer
func (s *DocumentService) ProcessDocument(ctx context.Context, file io.Reader, metadata Metadata) (*Document, error) {
    // Extract text
    extracted, err := s.extractor.ExtractText(ctx, file, metadata.Format)
    if err != nil {
        return nil, fmt.Errorf("text extraction failed for document %s: %w", metadata.Filename, err)
    }
    
    // Classify document
    classification, err := s.classifier.ClassifyDocument(ctx, extracted.Content)
    if err != nil {
        return nil, fmt.Errorf("document classification failed for %s: %w", metadata.Filename, err)
    }
    
    // Store document
    doc := &Document{
        Content:        extracted.Content,
        Classification: classification,
        Metadata:       metadata,
    }
    
    if err := s.storage.Store(ctx, doc); err != nil {
        return nil, fmt.Errorf("storage failed for document %s: %w", metadata.Filename, err)
    }
    
    return doc, nil
}
```

#### Custom Error Types
```go
// Specific error types for different scenarios
type ValidationError struct {
    Field   string
    Value   interface{}
    Message string
}

func (e ValidationError) Error() string {
    return fmt.Sprintf("validation failed for field '%s': %s", e.Field, e.Message)
}

type ServiceUnavailableError struct {
    Service string
    Cause   error
}

func (e ServiceUnavailableError) Error() string {
    return fmt.Sprintf("service %s is unavailable: %v", e.Service, e.Cause)
}

// Usage in handlers
func (h *DocumentHandler) Upload(c *fiber.Ctx) error {
    doc, err := h.service.ProcessDocument(c.Context(), file, metadata)
    if err != nil {
        var validationErr ValidationError
        if errors.As(err, &validationErr) {
            return c.Status(400).JSON(models.NewErrorResponse("VALIDATION_ERROR", err.Error(), nil))
        }
        
        var serviceErr ServiceUnavailableError
        if errors.As(err, &serviceErr) {
            return c.Status(503).JSON(models.NewErrorResponse("SERVICE_UNAVAILABLE", err.Error(), nil))
        }
        
        return c.Status(500).JSON(models.NewErrorResponse("PROCESSING_ERROR", "Internal server error", nil))
    }
    
    return c.JSON(models.NewSuccessResponse(doc, "Document processed successfully"))
}
```

## Data Flow Architecture

### 1. Document Processing Pipeline

```
Input File → Validation → Text Extraction → AI Classification → Entity Extraction → Storage → Indexing → Response
```

#### Detailed Flow
```go
func (p *ProcessingPipeline) ProcessDocument(ctx context.Context, file io.Reader, metadata Metadata) (*ProcessedDocument, error) {
    // Stage 1: Validation
    if err := p.validator.ValidateFile(file, metadata); err != nil {
        return nil, fmt.Errorf("validation stage failed: %w", err)
    }
    
    // Stage 2: Text Extraction
    extracted, err := p.extractor.ExtractText(ctx, file, metadata.Format)
    if err != nil {
        return nil, fmt.Errorf("extraction stage failed: %w", err)
    }
    
    // Stage 3: AI Classification
    classification, err := p.classifier.ClassifyDocument(ctx, extracted.Content)
    if err != nil {
        return nil, fmt.Errorf("classification stage failed: %w", err)
    }
    
    // Stage 4: Entity Extraction
    entities, err := p.entityExtractor.ExtractEntities(ctx, extracted.Content)
    if err != nil {
        // Non-critical failure, log and continue
        p.logger.Warn("entity extraction failed", "error", err)
        entities = &EntityResults{}
    }
    
    // Stage 5: Document Assembly
    doc := &ProcessedDocument{
        ID:             generateID(),
        Content:        extracted.Content,
        Classification: classification,
        Entities:       entities,
        Metadata:       metadata,
        ProcessedAt:    time.Now(),
    }
    
    // Stage 6: Storage (parallel with indexing)
    var storageErr, indexErr error
    var wg sync.WaitGroup
    
    wg.Add(2)
    go func() {
        defer wg.Done()
        storageErr = p.storage.Store(ctx, doc)
    }()
    
    go func() {
        defer wg.Done()
        indexErr = p.search.Index(ctx, doc)
    }()
    
    wg.Wait()
    
    if storageErr != nil {
        return nil, fmt.Errorf("storage stage failed: %w", storageErr)
    }
    
    if indexErr != nil {
        // Storage succeeded but indexing failed - document is uploaded but not searchable
        p.logger.Error("indexing failed", "document_id", doc.ID, "error", indexErr)
        // Could trigger retry mechanism here
    }
    
    return doc, nil
}
```

### 2. Search Query Flow

```
Query Input → Validation → Query Building → OpenSearch → Result Processing → Response Formatting → Client
```

#### Search Architecture
```go
type SearchService struct {
    client     *opensearch.Client
    queryBuilder QueryBuilder
    aggregator   ResultAggregator
    cache        Cache
}

func (s *SearchService) Search(ctx context.Context, query SearchQuery) (*SearchResults, error) {
    // Stage 1: Input validation
    if err := query.Validate(); err != nil {
        return nil, fmt.Errorf("invalid search query: %w", err)
    }
    
    // Stage 2: Check cache
    if cached, found := s.cache.Get(query.CacheKey()); found {
        return cached.(*SearchResults), nil
    }
    
    // Stage 3: Build OpenSearch query
    osQuery, err := s.queryBuilder.BuildQuery(query)
    if err != nil {
        return nil, fmt.Errorf("query building failed: %w", err)
    }
    
    // Stage 4: Execute search
    response, err := s.client.Search(ctx, osQuery)
    if err != nil {
        return nil, fmt.Errorf("search execution failed: %w", err)
    }
    
    // Stage 5: Process results
    results, err := s.aggregator.ProcessResults(response)
    if err != nil {
        return nil, fmt.Errorf("result processing failed: %w", err)
    }
    
    // Stage 6: Cache results
    s.cache.Set(query.CacheKey(), results, time.Minute*5)
    
    return results, nil
}
```

## Service Layer Design

### 1. Service Interfaces

#### Storage Service
```go
type Service interface {
    // Core operations
    Store(ctx context.Context, path string, reader io.Reader, metadata map[string]string) (*StoreResult, error)
    Retrieve(ctx context.Context, path string) (io.ReadCloser, error)
    Delete(ctx context.Context, path string) error
    Exists(ctx context.Context, path string) (bool, error)
    
    // Batch operations
    StoreBatch(ctx context.Context, operations []StoreOperation) (*BatchResult, error)
    DeleteBatch(ctx context.Context, paths []string) (*BatchResult, error)
    
    // URL operations
    GetURL(path string) string
    GetSignedURL(path string, expires time.Duration) (string, error)
    
    // Management
    IsHealthy() bool
    GetMetrics() map[string]interface{}
}
```

#### Search Service  
```go
type Service interface {
    // Document operations
    Index(ctx context.Context, doc *Document) error
    Update(ctx context.Context, id string, doc *Document) error
    Delete(ctx context.Context, id string) error
    Get(ctx context.Context, id string) (*Document, error)
    
    // Search operations
    Search(ctx context.Context, query SearchQuery) (*SearchResults, error)
    Suggest(ctx context.Context, query string, field string) ([]string, error)
    
    // Aggregations
    Aggregate(ctx context.Context, aggs AggregationQuery) (*AggregationResults, error)
    
    // Management
    CreateIndex(ctx context.Context, index string, mapping IndexMapping) error
    DeleteIndex(ctx context.Context, index string) error
    IsHealthy() bool
    Health(ctx context.Context) (*HealthStatus, error)
}
```

### 2. Service Implementation Strategy

#### DigitalOcean Integration Pattern
```go
// Hybrid approach: MCP for management, S3 for performance
type SpacesClient struct {
    // MCP client for DigitalOcean-specific operations
    mcpClient   *digitalocean.Client
    
    // S3-compatible client for high-performance operations
    s3Client    *s3.Client
    
    // Configuration
    config      SpacesConfig
    
    // Utilities
    urlGenerator URLGenerator
    cache       Cache
    metrics     Metrics
}

func (c *SpacesClient) Store(ctx context.Context, path string, reader io.Reader, metadata map[string]string) (*StoreResult, error) {
    // Use S3 client for actual file operations (performance)
    result, err := c.s3Client.PutObject(ctx, &s3.PutObjectInput{
        Bucket: aws.String(c.config.Bucket),
        Key:    aws.String(path),
        Body:   reader,
        Metadata: metadata,
    })
    if err != nil {
        return nil, fmt.Errorf("S3 upload failed: %w", err)
    }
    
    // Use MCP for DigitalOcean-specific operations (CDN invalidation)
    if c.config.CDNEnabled {
        go func() {
            if err := c.invalidateCDNCache([]string{path}); err != nil {
                c.logger.Error("CDN cache invalidation failed", "path", path, "error", err)
            }
        }()
    }
    
    return &StoreResult{
        URL:      c.urlGenerator.GetCDNURL(path),
        ETag:     *result.ETag,
        Size:     result.ContentLength,
    }, nil
}

func (c *SpacesClient) invalidateCDNCache(paths []string) error {
    // Use MCP client for CDN operations
    return c.mcpClient.SpacesCDNFlushCache(c.config.CDNID, paths)
}
```

## Configuration Architecture

### 1. Environment-Based Configuration

```go
type Config struct {
    Environment   string                    // local, staging, production
    Server        ServerConfig
    Database      DatabaseConfig
    Storage       StorageConfig
    Auth          AuthConfig
    Processing    ProcessingConfig
    OpenSearch    OpenSearchConfig
    OpenAI        OpenAIConfig
    DigitalOcean  *digitalocean.Config     // Comprehensive DO configuration
}

func Load() (*Config, error) {
    environment := getEnv("ENVIRONMENT", "local")
    
    cfg := &Config{
        Environment: environment,
        Server:      loadServerConfig(environment),
        Database:    loadDatabaseConfig(environment),
        Storage:     loadStorageConfig(environment),
        Auth:        loadAuthConfig(environment),
        Processing:  loadProcessingConfig(environment),
        OpenSearch:  loadOpenSearchConfig(environment),
        OpenAI:      loadOpenAIConfig(environment),
    }
    
    // Load DigitalOcean configuration
    doConfig, err := digitalocean.LoadFromEnvironment()
    if err != nil {
        if environment == "local" {
            // Use default config for local development
            doConfig = digitalocean.DefaultConfig()
        } else {
            return nil, fmt.Errorf("DigitalOcean configuration required for %s environment: %w", environment, err)
        }
    }
    cfg.DigitalOcean = doConfig
    
    // Validate complete configuration
    if err := cfg.validate(); err != nil {
        return nil, fmt.Errorf("configuration validation failed: %w", err)
    }
    
    return cfg, nil
}
```

### 2. Configuration Validation

```go
func (c *Config) validate() error {
    validators := []func() error{
        c.validateServer,
        c.validateStorage,
        c.validateOpenSearch,
        c.validateAuth,
        c.validateProcessing,
    }
    
    for _, validate := range validators {
        if err := validate(); err != nil {
            return err
        }
    }
    
    return nil
}

func (c *Config) validateStorage() error {
    if c.Storage.Backend == "spaces" {
        if c.Storage.AccessKey == "" {
            return errors.New("STORAGE_ACCESS_KEY is required for spaces backend")
        }
        if c.Storage.SecretKey == "" {
            return errors.New("STORAGE_SECRET_KEY is required for spaces backend")
        }
        if c.Storage.Bucket == "" {
            return errors.New("STORAGE_BUCKET is required for spaces backend")
        }
    }
    return nil
}
```

## Testing Architecture

### 1. Test Structure

```
test/
├── unit/                    # Unit tests (package-level)
│   ├── config/
│   ├── handlers/
│   └── services/
├── integration/             # Integration tests (cross-package)
│   ├── api/
│   ├── storage/
│   └── search/
├── e2e/                     # End-to-end tests
│   ├── document_processing/
│   └── search_workflows/
├── testdata/                # Test fixtures and data
│   ├── documents/
│   └── configurations/
└── testutil/                # Test utilities and helpers
    ├── mocks/
    ├── fixtures/
    └── helpers/
```

### 2. Test Utilities Design

```go
// UNIX principle: Composable test utilities
package testutil

// Environment management
func SetEnv(t *testing.T, envVars map[string]string) func() {
    // Implementation that restores original environment
}

// Configuration helpers
func TestConfig() *config.Config {
    // Returns minimal valid configuration for testing
}

func TestDigitalOceanConfig() *digitalocean.Config {
    // Returns test DigitalOcean configuration
}

// Service mocking
func MockStorageService() *storage.MockService {
    // Returns configured mock storage service
}

func MockSearchService() *search.MockService {
    // Returns configured mock search service
}

// File helpers
func TempFile(t *testing.T, name, content string) (string, func()) {
    // Creates temporary file with cleanup
}

func LoadTestDocument(t *testing.T, filename string) io.Reader {
    // Loads document from testdata directory
}

// HTTP testing
func NewTestServer(t *testing.T, handlers *handlers.Handlers) *fiber.App {
    // Creates test Fiber application
}

func MakeRequest(t *testing.T, app *fiber.App, method, path string, body interface{}) *http.Response {
    // Makes HTTP request to test server
}
```

### 3. Test Categories

#### Unit Tests
```go
// Test individual functions/methods in isolation
func TestDocumentExtractor_ExtractTextFromPDF(t *testing.T) {
    extractor := processing.NewTextExtractor()
    pdfReader := testutil.LoadTestDocument(t, "sample.pdf")
    
    result, err := extractor.ExtractText(context.Background(), pdfReader, "pdf")
    
    require.NoError(t, err)
    assert.Contains(t, result.Content, "expected text")
    assert.Equal(t, "en", result.Language)
}
```

#### Integration Tests
```go
// Test component interactions
//go:build integration
// +build integration

func TestDocumentService_EndToEndProcessing(t *testing.T) {
    testutil.SkipIfShort(t, "integration test")
    
    // Use real DigitalOcean services
    cfg := testutil.TestDigitalOceanConfig()
    provider, err := digitalocean.NewProvider(cfg)
    require.NoError(t, err)
    
    services := provider.GetServices()
    docService := services.NewDocumentService()
    
    // Test with real file
    testFile := testutil.LoadTestDocument(t, "motion.pdf")
    metadata := Metadata{Filename: "motion.pdf", Format: "pdf"}
    
    result, err := docService.ProcessDocument(context.Background(), testFile, metadata)
    
    require.NoError(t, err)
    assert.NotEmpty(t, result.ID)
    assert.NotEmpty(t, result.URL)
    
    // Cleanup
    defer provider.Shutdown()
}
```

#### End-to-End Tests
```go
// Test complete workflows
func TestDocumentUploadWorkflow(t *testing.T) {
    app := testutil.NewTestServer(t, handlers)
    
    // Upload document
    uploadResp := testutil.MakeRequest(t, app, "POST", "/api/v1/categorise", multipartBody)
    assert.Equal(t, 200, uploadResp.StatusCode)
    
    var uploadResult models.SuccessResponse
    json.NewDecoder(uploadResp.Body).Decode(&uploadResult)
    docID := uploadResult.Data.(map[string]interface{})["document_id"].(string)
    
    // Search for uploaded document
    searchBody := map[string]interface{}{
        "query": "test document",
        "limit": 10,
    }
    searchResp := testutil.MakeRequest(t, app, "POST", "/api/v1/search", searchBody)
    assert.Equal(t, 200, searchResp.StatusCode)
    
    var searchResult models.SuccessResponse
    json.NewDecoder(searchResp.Body).Decode(&searchResult)
    
    // Verify document appears in search results
    results := searchResult.Data.(map[string]interface{})["documents"].([]interface{})
    found := false
    for _, doc := range results {
        if doc.(map[string]interface{})["id"].(string) == docID {
            found = true
            break
        }
    }
    assert.True(t, found, "Uploaded document should appear in search results")
}
```

## Performance Considerations

### 1. Concurrency Patterns

#### Worker Pool for Processing
```go
type ProcessingWorkerPool struct {
    workers    int
    jobs       chan ProcessingJob
    results    chan ProcessingResult
    wg         sync.WaitGroup
    ctx        context.Context
    cancel     context.CancelFunc
}

func NewProcessingWorkerPool(workers int) *ProcessingWorkerPool {
    ctx, cancel := context.WithCancel(context.Background())
    
    pool := &ProcessingWorkerPool{
        workers: workers,
        jobs:    make(chan ProcessingJob, workers*2),
        results: make(chan ProcessingResult, workers*2),
        ctx:     ctx,
        cancel:  cancel,
    }
    
    pool.start()
    return pool
}

func (p *ProcessingWorkerPool) start() {
    for i := 0; i < p.workers; i++ {
        p.wg.Add(1)
        go p.worker(i)
    }
}

func (p *ProcessingWorkerPool) worker(id int) {
    defer p.wg.Done()
    
    for {
        select {
        case job := <-p.jobs:
            result := p.processJob(job)
            select {
            case p.results <- result:
            case <-p.ctx.Done():
                return
            }
        case <-p.ctx.Done():
            return
        }
    }
}
```

### 2. Caching Strategy

#### Multi-Level Caching
```go
type CacheManager struct {
    l1Cache cache.Cache        // In-memory (Redis/local)
    l2Cache cache.Cache        // CDN cache
    metrics CacheMetrics
}

func (c *CacheManager) Get(key string) (interface{}, bool) {
    // L1 cache (fastest)
    if value, found := c.l1Cache.Get(key); found {
        c.metrics.RecordHit("l1")
        return value, true
    }
    
    // L2 cache (CDN)
    if value, found := c.l2Cache.Get(key); found {
        c.metrics.RecordHit("l2")
        // Populate L1 for next time
        c.l1Cache.Set(key, value, time.Minute*5)
        return value, true
    }
    
    c.metrics.RecordMiss()
    return nil, false
}
```

### 3. Memory Management

#### Streaming for Large Files
```go
func (s *StorageService) UploadLargeFile(ctx context.Context, path string, reader io.Reader) error {
    // Use multipart upload for files > 100MB
    const chunkSize = 100 * 1024 * 1024 // 100MB chunks
    
    // Create multipart upload
    createResp, err := s.client.CreateMultipartUpload(ctx, &s3.CreateMultipartUploadInput{
        Bucket: aws.String(s.bucket),
        Key:    aws.String(path),
    })
    if err != nil {
        return fmt.Errorf("failed to create multipart upload: %w", err)
    }
    
    uploadID := *createResp.UploadId
    var parts []types.CompletedPart
    partNumber := int32(1)
    
    buffer := make([]byte, chunkSize)
    for {
        n, err := reader.Read(buffer)
        if err != nil && err != io.EOF {
            return fmt.Errorf("failed to read chunk: %w", err)
        }
        
        if n == 0 {
            break
        }
        
        // Upload part
        partResp, err := s.client.UploadPart(ctx, &s3.UploadPartInput{
            Bucket:     aws.String(s.bucket),
            Key:        aws.String(path),
            PartNumber: aws.Int32(partNumber),
            UploadId:   aws.String(uploadID),
            Body:       bytes.NewReader(buffer[:n]),
        })
        if err != nil {
            return fmt.Errorf("failed to upload part %d: %w", partNumber, err)
        }
        
        parts = append(parts, types.CompletedPart{
            ETag:       partResp.ETag,
            PartNumber: aws.Int32(partNumber),
        })
        
        partNumber++
    }
    
    // Complete multipart upload
    _, err = s.client.CompleteMultipartUpload(ctx, &s3.CompleteMultipartUploadInput{
        Bucket:   aws.String(s.bucket),
        Key:      aws.String(path),
        UploadId: aws.String(uploadID),
        MultipartUpload: &types.CompletedMultipartUpload{
            Parts: parts,
        },
    })
    
    return err
}
```

## Security Architecture

### 1. Authentication Flow
```
Client → JWT Token → Middleware Validation → Supabase Verification → Request Processing
```

### 2. Input Validation
```go
func (h *DocumentHandler) Upload(c *fiber.Ctx) error {
    // 1. File validation
    file, header, err := c.FormFile("file")
    if err != nil {
        return models.NewErrorResponse("VALIDATION_ERROR", "File is required", nil)
    }
    
    // 2. File size validation
    if header.Size > h.config.MaxFileSize {
        return models.NewErrorResponse("VALIDATION_ERROR", "File too large", map[string]interface{}{
            "max_size": h.config.MaxFileSize,
            "actual_size": header.Size,
        })
    }
    
    // 3. File type validation
    if !h.isAllowedFileType(header.Header.Get("Content-Type")) {
        return models.NewErrorResponse("VALIDATION_ERROR", "File type not allowed", nil)
    }
    
    // 4. Content scanning (virus scan, malicious content)
    if err := h.scanner.ScanFile(file); err != nil {
        return models.NewErrorResponse("SECURITY_ERROR", "File failed security scan", nil)
    }
    
    // Continue with processing...
}
```

### 3. Data Protection
```go
// Sanitize user input
func (s *SearchService) Search(ctx context.Context, query SearchQuery) (*SearchResults, error) {
    // Sanitize search query
    sanitizedQuery := s.sanitizer.SanitizeQuery(query.Query)
    
    // Validate search parameters
    if err := query.Validate(); err != nil {
        return nil, fmt.Errorf("invalid search query: %w", err)
    }
    
    // Apply user access controls
    filteredQuery := s.accessControl.ApplyUserFilters(ctx, sanitizedQuery)
    
    return s.client.Search(ctx, filteredQuery)
}
```

This architecture ensures scalable, maintainable, and secure legal document processing with comprehensive testing and observability.