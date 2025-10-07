# Development Guide

Comprehensive guide for developing Motion-Index Fiber, following UNIX philosophy and Test-Driven Development principles.

## Getting Started

### Prerequisites
- **Go 1.21+** with modules support
- **Git** for version control
- **DigitalOcean Account** (for cloud services)

### Quick Setup
```bash
# Clone repository
git clone <repository-url>
cd motion-index-fiber

# Install dependencies
go mod tidy

# Copy environment template
cp .env.example .env
# Edit .env with your configuration

# Run tests to verify setup
go test ./... -short -v

# Start development server
go run cmd/server/main.go
```

## Development Workflow

### 1. Environment Setup

#### Local Development Configuration
```bash
# .env for local development
PORT=6000
ENVIRONMENT=local
PRODUCTION=false
JWT_SECRET=local-development-secret

# Mock services for local development
USE_MOCK_SERVICES=true

# Real services for integration testing
OPENSEARCH_HOST=your-cluster.k.db.ondigitalocean.com
OPENSEARCH_PASSWORD=your-password
DO_SPACES_KEY=your-spaces-key
DO_SPACES_SECRET=your-spaces-secret
```

### 2. Test-Driven Development (TDD)

#### TDD Cycle
1. **🔴 Red**: Write a failing test first
2. **🟢 Green**: Write minimal code to make test pass  
3. **🔵 Refactor**: Improve code while keeping tests green
4. **🔄 Repeat**: Continue with next requirement

#### Test Commands
```bash
# Primary development test command
go test ./... -v -coverprofile=coverage.out

# Fast unit tests only (for TDD cycle)
go test ./... -short -v

# Specific package tests
go test ./internal/config/... -v
go test ./pkg/cloud/digitalocean/... -v

# Test with race detection
go test ./... -race -v

# Benchmark tests
go test ./... -bench=. -benchmem

# Coverage analysis
go tool cover -html=coverage.out
```

#### Test Structure
```go
func TestSomething(t *testing.T) {
    // Arrange
    cfg := testutil.TestConfig()
    service := NewService(cfg)
    
    // Act
    result, err := service.DoSomething("input")
    
    // Assert
    require.NoError(t, err)
    assert.Equal(t, "expected", result)
}

func TestSomething_ErrorCase(t *testing.T) {
    // Test error conditions
    cfg := testutil.TestConfig()
    service := NewService(cfg)
    
    _, err := service.DoSomething("invalid")
    
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "expected error message")
}
```

### 3. Code Quality Standards

#### UNIX Philosophy Implementation
- **Do One Thing Well**: Each function/package has single responsibility
- **Composable**: Interfaces allow easy testing and swapping
- **Testable**: All code has comprehensive test coverage
- **Observable**: Logging and metrics throughout

#### Code Style
```bash
# Format code (run before commits)
go fmt ./...

# Vet for common issues
go vet ./...

# Lint (if golangci-lint installed)
golangci-lint run

# Import organization
goimports -w .
```

#### Package Design Principles
```go
// Good: Clear interface with single responsibility
type DocumentProcessor interface {
    ProcessDocument(ctx context.Context, doc *Document) (*ProcessedDocument, error)
}

// Good: Dependency injection for testability
func NewDocumentService(processor DocumentProcessor, storage Storage) *DocumentService {
    return &DocumentService{
        processor: processor,
        storage:   storage,
    }
}

// Good: Comprehensive error handling
func (s *DocumentService) Process(ctx context.Context, file io.Reader) error {
    doc, err := s.processor.ProcessDocument(ctx, file)
    if err != nil {
        return fmt.Errorf("failed to process document: %w", err)
    }
    
    if err := s.storage.Store(ctx, doc); err != nil {
        return fmt.Errorf("failed to store document: %w", err)
    }
    
    return nil
}
```

### 4. Development Patterns

#### Service Layer Pattern
```go
// interfaces.go
type DocumentService interface {
    Upload(ctx context.Context, file io.Reader, metadata Metadata) (*Document, error)
    Get(ctx context.Context, id string) (*Document, error)
    Search(ctx context.Context, query SearchQuery) (*SearchResults, error)
}

// service.go
type documentService struct {
    storage   storage.Service
    search    search.Service
    processor processing.Service
    logger    *slog.Logger
}

func NewDocumentService(storage storage.Service, search search.Service, processor processing.Service) DocumentService {
    return &documentService{
        storage:   storage,
        search:    search,
        processor: processor,
        logger:    slog.Default().With("component", "document_service"),
    }
}
```

#### Handler Pattern
```go
type DocumentHandler struct {
    service DocumentService
    logger  *slog.Logger
}

func NewDocumentHandler(service DocumentService) *DocumentHandler {
    return &DocumentHandler{
        service: service,
        logger:  slog.Default().With("component", "document_handler"),
    }
}

func (h *DocumentHandler) Upload(c *fiber.Ctx) error {
    // 1. Parse and validate input
    file, header, err := c.FormFile("file")
    if err != nil {
        return models.NewErrorResponse("VALIDATION_ERROR", "File is required", nil)
    }
    
    // 2. Call service layer
    doc, err := h.service.Upload(c.Context(), file, metadata)
    if err != nil {
        h.logger.Error("upload failed", "error", err)
        return models.NewErrorResponse("UPLOAD_ERROR", "Failed to upload document", nil)
    }
    
    // 3. Return response
    return c.JSON(models.NewSuccessResponse(doc, "Document uploaded successfully"))
}
```

#### Configuration Pattern
```go
// config.go - Environment-based configuration
type Config struct {
    Server    ServerConfig
    Database  DatabaseConfig
    Storage   StorageConfig
    // ... other configs
}

func Load() (*Config, error) {
    cfg := &Config{
        Server: ServerConfig{
            Port: getEnv("PORT", "6000"),
            Production: getEnvBool("PRODUCTION", false),
        },
        // ... load other configs
    }
    
    if err := cfg.validate(); err != nil {
        return nil, fmt.Errorf("invalid configuration: %w", err)
    }
    
    return cfg, nil
}

func (c *Config) validate() error {
    if c.Server.Port == "" {
        return errors.New("PORT is required")
    }
    // ... other validations
    return nil
}
```

### 5. Testing Strategies

#### Unit Testing
```go
// Use testutil helpers
func TestDocumentService_Upload(t *testing.T) {
    // Setup
    mockStorage := &storage.MockService{}
    mockSearch := &search.MockService{}
    mockProcessor := &processing.MockService{}
    
    service := NewDocumentService(mockStorage, mockSearch, mockProcessor)
    
    // Configure mocks
    mockProcessor.On("ProcessDocument", mock.Anything, mock.Anything).Return(
        &processing.ProcessedDocument{ID: "test-id"}, nil)
    mockStorage.On("Store", mock.Anything, mock.Anything).Return(nil)
    mockSearch.On("Index", mock.Anything, mock.Anything).Return(nil)
    
    // Test
    result, err := service.Upload(context.Background(), strings.NewReader("test"), metadata)
    
    // Verify
    require.NoError(t, err)
    assert.Equal(t, "test-id", result.ID)
    mockProcessor.AssertExpectations(t)
    mockStorage.AssertExpectations(t)
    mockSearch.AssertExpectations(t)
}
```

#### Integration Testing
```go
//go:build integration
// +build integration

func TestDocumentService_Integration(t *testing.T) {
    testutil.SkipIfShort(t, "integration test")
    
    // Use real services with test configuration
    cfg := testutil.TestDigitalOceanConfig()
    provider, err := digitalocean.NewProvider(cfg)
    require.NoError(t, err)
    
    err = provider.Initialize()
    require.NoError(t, err)
    
    services := provider.GetServices()
    service := NewDocumentService(services.Storage, services.Search, mockProcessor)
    
    // Test with real services
    result, err := service.Upload(context.Background(), testFile, metadata)
    require.NoError(t, err)
    assert.NotEmpty(t, result.URL)
    
    // Cleanup
    defer provider.Shutdown()
}
```

#### HTTP Testing
```go
func TestDocumentHandler_Upload(t *testing.T) {
    // Setup
    mockService := &DocumentServiceMock{}
    handler := NewDocumentHandler(mockService)
    
    app := fiber.New()
    app.Post("/upload", handler.Upload)
    
    // Create test request
    body := &bytes.Buffer{}
    writer := multipart.NewWriter(body)
    part, _ := writer.CreateFormFile("file", "test.pdf")
    part.Write([]byte("test content"))
    writer.Close()
    
    req := httptest.NewRequest("POST", "/upload", body)
    req.Header.Set("Content-Type", writer.FormDataContentType())
    
    // Configure mock
    mockService.On("Upload", mock.Anything, mock.Anything, mock.Anything).Return(
        &Document{ID: "test-id"}, nil)
    
    // Test
    resp, err := app.Test(req)
    require.NoError(t, err)
    assert.Equal(t, 200, resp.StatusCode)
    
    // Parse response
    var response models.SuccessResponse
    json.NewDecoder(resp.Body).Decode(&response)
    assert.True(t, response.Success)
}
```

### 6. Debugging & Troubleshooting

#### Logging
```go
// Structured logging with slog
logger := slog.Default().With(
    "component", "document_service",
    "operation", "upload",
    "user_id", userID,
)

logger.Info("processing document", 
    "filename", filename,
    "size", fileSize)

if err != nil {
    logger.Error("processing failed",
        "error", err,
        "filename", filename,
        "stage", "text_extraction")
    return fmt.Errorf("processing failed: %w", err)
}
```

#### Profiling
```go
// Add to main.go for development
import _ "net/http/pprof"

func main() {
    if !cfg.IsProduction() {
        go func() {
            log.Println(http.ListenAndServe("localhost:6060", nil))
        }()
    }
    
    // ... rest of application
}
```

**Access profiling:**
```bash
# CPU profile
go tool pprof http://localhost:6060/debug/pprof/profile?seconds=30

# Memory profile
go tool pprof http://localhost:6060/debug/pprof/heap

# Goroutine profile
go tool pprof http://localhost:6060/debug/pprof/goroutine
```

#### Common Debug Commands
```bash
# Check service health
curl http://localhost:6000/health/detailed

# Test file upload
curl -X POST http://localhost:6000/api/v1/categorise \
  -F "file=@test.pdf"

# Check OpenSearch connection
curl -u admin:password https://your-cluster.k.db.ondigitalocean.com:25060

# Trace network calls
go run cmd/server/main.go 2>&1 | grep -E "(http|tcp|tls)"

# Memory usage
go tool pprof -top http://localhost:6060/debug/pprof/heap
```

### 7. Code Organization

#### Package Structure Rules
- **cmd/**: Application entry points
- **pkg/**: Public libraries (importable by external projects)
- **internal/**: Private application code (not importable)
- **test/**: Test-specific code and fixtures

#### Import Organization
```go
import (
    // Standard library
    "context"
    "fmt"
    "net/http"
    
    // Third-party dependencies
    "github.com/gofiber/fiber/v2"
    "github.com/stretchr/testify/assert"
    
    // Internal packages
    "motion-index-fiber/internal/config"
    "motion-index-fiber/pkg/storage"
)
```

#### Error Handling Patterns
```go
// Wrap errors with context
func (s *Service) ProcessDocument(ctx context.Context, doc *Document) error {
    if err := s.validate(doc); err != nil {
        return fmt.Errorf("validation failed: %w", err)
    }
    
    if err := s.extract(doc); err != nil {
        return fmt.Errorf("text extraction failed: %w", err)
    }
    
    return nil
}

// Custom error types for different error categories
type ValidationError struct {
    Field   string
    Value   interface{}
    Message string
}

func (e ValidationError) Error() string {
    return fmt.Sprintf("validation error for field %s: %s", e.Field, e.Message)
}
```

### 8. Performance Guidelines

#### Benchmarking
```go
func BenchmarkDocumentProcessing(b *testing.B) {
    service := setupBenchmarkService()
    testDoc := loadTestDocument()
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, err := service.Process(context.Background(), testDoc)
        if err != nil {
            b.Fatal(err)
        }
    }
}
```

#### Memory Management
```go
// Use streaming for large files
func (s *Service) ProcessLargeFile(ctx context.Context, reader io.Reader) error {
    // Process in chunks to avoid loading entire file into memory
    scanner := bufio.NewScanner(reader)
    for scanner.Scan() {
        if err := s.processChunk(scanner.Text()); err != nil {
            return err
        }
    }
    return scanner.Err()
}

// Close resources properly
func (s *Service) Upload(ctx context.Context, file io.Reader) error {
    if closer, ok := file.(io.Closer); ok {
        defer closer.Close()
    }
    
    // ... processing logic
}
```

### 9. Contributing Guidelines

#### Git Workflow
```bash
# Create feature branch
git checkout -b feature/document-processing

# Make changes following TDD
# Write test first
go test ./pkg/processing/... -v

# Implement feature
# Run tests again
go test ./pkg/processing/... -v

# Run full test suite
go test ./... -v

# Commit with descriptive message
git add .
git commit -m "Add document text extraction with OCR support

- Implement PDF text extraction using textract
- Add OCR fallback for scanned documents  
- Include confidence scoring for extracted text
- Add comprehensive test coverage

Fixes #123"

# Push and create PR
git push origin feature/document-processing
```

#### Pull Request Checklist
- [ ] All tests pass (`go test ./... -v`)
- [ ] Code coverage maintained or improved
- [ ] Documentation updated (API docs, README, etc.)
- [ ] CHANGELOG.md updated
- [ ] No breaking changes (or properly documented)
- [ ] Performance impact considered
- [ ] Security implications reviewed

#### Code Review Guidelines
- **Functionality**: Does it work as intended?
- **Tests**: Comprehensive test coverage?
- **Performance**: Any performance implications?
- **Security**: Any security vulnerabilities?
- **Maintainability**: Easy to understand and modify?
- **UNIX Philosophy**: Single responsibility, composable?

### 10. Resources

#### Go Resources
- [Effective Go](https://golang.org/doc/effective_go.html)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [Go Testing](https://golang.org/doc/tutorial/add-a-test)

#### Project Resources
- [CLAUDE.md](../../CLAUDE.md) - Detailed guidance for Claude Code
- [API Documentation](../api/) - Complete API reference
- [Deployment Guide](../deployment/) - Production deployment
- [Architecture Decisions](./architecture.md) - Design decisions and patterns
