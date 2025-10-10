# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Motion-Index Fiber is a high-performance legal document processing API built with Go and Fiber, designed for California public defenders. The system features production-ready DigitalOcean integration with direct REST API calls and AWS S3 SDK, following strict TDD and UNIX philosophy principles.

### Core Technologies
- **Runtime**: Go 1.24+ with Fiber v2 web framework
- **Cloud Platform**: DigitalOcean (Spaces storage, Managed OpenSearch, App Platform)
- **Integration Pattern**: Direct DigitalOcean REST API + AWS S3 SDK for Spaces
- **Authentication**: JWT with Supabase integration
- **AI Processing**: Multi-model support (OpenAI GPT-4, Claude, Ollama) with unified prompts and enhanced date extraction
- **Testing**: Comprehensive TDD with testify framework
- **Document Processing**: PDF, DOCX, TXT extraction with OCR capabilities (Tesseract)

## Architecture

### High-Level Design
```
┌─────────────────┐
│   Web Client    │
└─────────┬───────┘
          │
    ┌─────▼─────┐
    │Motion Index│
    │    API     │
    │ (Fiber v2) │
    └─────┬─────┘
          │
┌─────────┼─────────┐
│    DigitalOcean   │    │   Supabase    │    │ AI Models   │
│ (REST API + S3)   │    │     (JWT)     │    │GPT-4/Claude │
└─────────┬─────────┘    └───────────────┘    └─────────────┘
          │
    ┌─────▼─────┐        ┌─────────────┐
    │  Spaces   │        │ OpenSearch  │
    │Storage+CDN│        │   Cluster   │
    └───────────┘        └─────────────┘
```

### Package Structure
- **`cmd/server/`**: Application entry point with graceful shutdown
- **`cmd/api-classifier/`**: Single-threaded document classification tool
- **`cmd/api-batch-classifier/`**: Multi-threaded batch classification tool
- **`cmd/setup-index/`**: OpenSearch index setup and management
- **`cmd/inspect-index/`**: Index inspection and debugging tools
- **`pkg/cloud/digitalocean/`**: Direct DigitalOcean API integration and service factory
- **`pkg/processing/`**: Document processing pipeline (extract, classify, process)
  - **`classifier/`**: Multi-model AI classification (OpenAI, Claude, Ollama)
  - **`extractor/`**: Multi-format text extraction (PDF, DOCX, TXT, OCR)
  - **`pipeline/`**: Document processing pipeline and workers
  - **`queue/`**: Priority queue and rate limiting for batch processing
  - **`redaction/`**: PDF redaction analysis and processing
  - **`migration/`**: Metadata migration utilities
- **`pkg/search/`**: Search interfaces and OpenSearch client
- **`pkg/storage/`**: Storage interfaces and utilities
- **`internal/config/`**: Application configuration with validation
- **`internal/handlers/`**: HTTP request handlers
- **`internal/middleware/`**: Custom middleware (auth, error handling)
- **`internal/models/`**: Data models and validation
- **`internal/processing/`**: Internal processing coordinators
- **`internal/hardware/`**: Hardware analysis utilities
- **`internal/testutil/`**: Test utilities following UNIX principles
- **`pkg/models/`**: Core data models (document, legal, search, time)
- **`pkg/monitoring/`**: Metrics collection and monitoring

## Enhanced Date Extraction and Classification System

### Multi-Model AI Classification
The system supports three AI models with unified prompt architecture:

#### Supported Models
- **OpenAI GPT-4**: Production-grade classification with comprehensive analysis
- **Claude 3.5 Sonnet**: Advanced legal reasoning and structured extraction
- **Ollama (Local)**: Privacy-focused local model support (GPT-OSS:20b)

#### Unified Prompt System (`pkg/processing/classifier/prompts.go`)
- **Centralized Prompts**: Single source of truth for all classification prompts
- **Model-Specific Optimization**: Tailored configurations for each AI model
- **Enhanced Instructions**: Comprehensive date extraction and legal entity guidelines
- **Consistent Results**: Same classification logic across all models

### Enhanced Date Extraction

#### Five Date Types Extracted
1. **`filing_date`**: When document was filed with court
2. **`event_date`**: Key event or action date referenced in document
3. **`hearing_date`**: Scheduled court hearing or proceeding date
4. **`decision_date`**: When court decision, ruling, or order was made
5. **`served_date`**: When documents were served to parties

#### Date Processing Features (`pkg/processing/classifier/date_extraction.go`)
- **ISO Format Standardization**: All dates converted to YYYY-MM-DD
- **Context-Aware Validation**: Legal document date range validation (1950-present)
- **Multiple Format Support**: MM/DD/YYYY, Month DD YYYY, YYYY-MM-DD
- **Error Handling**: Invalid dates safely handled and logged
- **Relative Date Parsing**: "tomorrow", "next Monday" calculated from context

#### Search Integration
All extracted dates are indexed in OpenSearch as searchable fields:
```json
{
  "metadata": {
    "filing_date": "2024-03-15",
    "event_date": "2024-03-10", 
    "hearing_date": "2024-04-20",
    "decision_date": "2024-03-25",
    "served_date": "2024-03-12"
  }
}
```

### Classification Commands

#### Single-Threaded Processing (Debugging)
```bash
# Test API connectivity
go run cmd/api-classifier/main.go test-connection

# Classify specific number of documents
go run cmd/api-classifier/main.go classify-count 10

# Classify all unindexed documents (sequential)
go run cmd/api-classifier/main.go classify-all
```

#### Index Management
```bash
# Setup OpenSearch index with proper mappings
go run cmd/setup-index/main.go

# Inspect index structure and sample documents
go run cmd/inspect-index/main.go
```

## Development Commands

### Core Development Workflow
```bash
# Initial setup
go mod tidy
cp .env.example .env  # Edit with your configuration

# Development server (runs on port 8003)
go run cmd/server/main.go

# Or with auto-reload (install air first)
go install github.com/cosmtrek/air@latest
air

# Build for development
go build -o bin/server cmd/server/main.go

# Build for production
go build -ldflags="-s -w" -o bin/server cmd/server/main.go
```

### Testing Commands (Critical)
This project follows strict TDD principles. ALWAYS run tests before and after code changes:

```bash
# Run all tests with coverage (primary command)
go test ./... -v -coverprofile=coverage.out
go tool cover -html=coverage.out

# Unit tests only (fast feedback loop)
go test ./... -short -v

# Integration tests (requires real DigitalOcean credentials)
RUN_INTEGRATION_TESTS=true go test ./... -v -tags=integration

# Test specific packages
go test ./internal/config/... -v
go test ./pkg/cloud/digitalocean/... -v
go test ./internal/handlers/... -v

# Benchmark tests
go test ./... -bench=. -benchmem

# Race condition detection
go test ./... -race -v

# Test coverage by package
go test ./... -coverprofile=coverage.out
go tool cover -func=coverage.out
```

### Code Quality Commands
```bash
# Format code
go fmt ./...

# Vet for common issues
go vet ./...

# Lint (if golangci-lint is installed)
golangci-lint run

# Dependency verification
go mod verify
go mod download
```

## Configuration Management

### Environment Files
- **`.env.example`**: Template with all required variables
- **`.env.local.example`**: Local development configuration
- **`.env.production.example`**: Production deployment configuration

### Key Configuration Sections

#### Core Application
```bash
PORT=8003
ENVIRONMENT=local  # local, staging, production
PRODUCTION=false   # Enable production optimizations
JWT_SECRET=your-jwt-secret
ALLOWED_ORIGINS=http://localhost:3000,http://localhost:5173
```

#### DigitalOcean Services (Production)
```bash
# DigitalOcean API Token for CDN/Management operations
DO_API_TOKEN=your-digitalocean-api-token

# Spaces Storage
DO_SPACES_ACCESS_KEY=your-access-key
DO_SPACES_SECRET_KEY=your-secret-key
DO_SPACES_BUCKET=motion-index-docs
DO_SPACES_REGION=nyc3

# Managed OpenSearch
DO_OPENSEARCH_HOST=your-cluster.k.db.ondigitalocean.com
DO_OPENSEARCH_PORT=25060
DO_OPENSEARCH_USERNAME=doadmin
DO_OPENSEARCH_PASSWORD=your-password
DO_OPENSEARCH_USE_SSL=true
```

#### External Services
```bash
# Supabase Authentication
SUPABASE_URL=https://your-project.supabase.co
SUPABASE_ANON_KEY=your-anon-key
SUPABASE_SERVICE_KEY=your-service-key

# AI Classification Services
# OpenAI GPT-4 (Primary)
OPENAI_API_KEY=your-api-key
OPENAI_MODEL=gpt-4

# Claude (Alternative)
CLAUDE_API_KEY=your-claude-api-key
CLAUDE_MODEL=claude-3-5-sonnet-20241022

# Ollama (Local/Privacy)
OLLAMA_BASE_URL=http://localhost:11434
OLLAMA_MODEL=llama3
```

## UNIX Philosophy Adherence

### Core Principles
1. **Do One Thing Well**: Each component has a single, focused responsibility
2. **Composable**: Services can be combined and used independently
3. **Testable**: Every component has comprehensive test coverage
4. **Observable**: Health checks, metrics, and logging throughout

### Implementation Guidelines

#### Package Design
- Each package should have a clear, single purpose
- Interfaces should be minimal and focused
- Dependencies should be explicit and injectable
- Avoid global state and singleton patterns

#### Function Design
- Functions should be pure when possible
- Single responsibility and clear naming
- Comprehensive error handling
- Input validation at boundaries

#### Testing Strategy
- Unit tests for isolated logic
- Integration tests for service interactions
- Benchmark tests for performance validation
- Mock external dependencies consistently

## Test-Driven Development (TDD)

### TDD Workflow
1. **Red**: Write a failing test first
2. **Green**: Write minimal code to make test pass
3. **Refactor**: Improve code while keeping tests green
4. **Repeat**: Continue with next requirement

### Test Categories

#### Unit Tests (`*_test.go`)
- Fast, isolated tests with no external dependencies
- Use mocks for external services
- Test edge cases and error conditions
- Example: `internal/config/config_test.go`

#### Integration Tests (`integration_test.go`)
- Test interactions between components
- Use real services when possible (with proper setup)
- Mark with `// +build integration` tag
- Example: `pkg/cloud/digitalocean/integration_test.go`

#### Benchmark Tests
- Performance validation and optimization
- Memory allocation tracking
- Comparison between implementations
- Run with `go test -bench=. -benchmem`

### Test Utilities (`internal/testutil/`)
Following UNIX principles, test utilities are composable and single-purpose:

```go
// Environment setup
cleanup := testutil.SetEnv(t, map[string]string{
    "PORT": "8080",
    "JWT_SECRET": "test-secret",
})
defer cleanup()

// Configuration testing
cfg := testutil.TestConfig()
cfg.DigitalOcean = testutil.TestDigitalOceanConfig()

// Temporary files and directories
tmpDir, cleanup := testutil.TempDir(t)
defer cleanup()
```

## Classification and Date Extraction Patterns

### Unified Prompt Architecture
```go
// Use centralized prompt builder for all models
config := DefaultPromptConfigs["openai"] // or "claude", "ollama"
builder := NewPromptBuilder(config)
prompt := builder.BuildClassificationPrompt(text, metadata)

// Model-specific configurations
type PromptConfig struct {
    Model           string  // "gpt-4", "claude-3-sonnet", "llama3"
    MaxTextLength   int     // Text truncation limit
    IncludeContext  bool    // Include document context analysis
    DetailLevel     string  // "minimal", "standard", "comprehensive"
}
```

### Date Extraction Implementation
```go
// Initialize date extractor with validation
dateExtractor := NewDateExtractor()

// Extract and validate dates from classification result
if result.FilingDate != nil {
    if !dateExtractor.validateDate(*result.FilingDate, "filing_date") {
        log.Printf("Invalid filing_date: %s, setting to nil", *result.FilingDate)
        result.FilingDate = nil
    }
}

// Parse dates in pipeline processing
if filingDateStr, exists := req.Metadata["filing_date"]; exists {
    if parsedDate, err := time.Parse("2006-01-02", filingDateStr); err == nil {
        doc.Metadata.FilingDate = &parsedDate
    }
}
```

### Multi-Model Classification Pattern
```go
// Interface-based classifier design
type Classifier interface {
    Classify(ctx context.Context, text string, metadata *DocumentMetadata) (*ClassificationResult, error)
    GetSupportedCategories() []string
    IsConfigured() bool
}

// Create classifiers for different models
openaiClassifier, _ := NewOpenAIClassifier(openaiConfig)
claudeClassifier, _ := NewClaudeClassifier(claudeConfig)
ollamaClassifier, _ := NewOllamaClassifier(ollamaConfig)

// Use same interface for all models
result, err := classifier.Classify(ctx, documentText, metadata)
```

### Enhanced Classification Result Structure
```go
type ClassificationResult struct {
    // Core Classification
    DocumentType  string  `json:"document_type"`
    LegalCategory string  `json:"legal_category"`
    Subject       string  `json:"subject"`
    Summary       string  `json:"summary"`
    Confidence    float64 `json:"confidence"`
    
    // Enhanced Date Fields (ISO format: YYYY-MM-DD)
    FilingDate   *string `json:"filing_date,omitempty"`
    EventDate    *string `json:"event_date,omitempty"`
    HearingDate  *string `json:"hearing_date,omitempty"`
    DecisionDate *string `json:"decision_date,omitempty"`
    ServedDate   *string `json:"served_date,omitempty"`
    
    // Legal Entity Extraction
    CaseInfo    *CaseInfo   `json:"case_info,omitempty"`
    CourtInfo   *CourtInfo  `json:"court_info,omitempty"`
    Parties     []Party     `json:"parties,omitempty"`
    Attorneys   []Attorney  `json:"attorneys,omitempty"`
    Authorities []Authority `json:"authorities,omitempty"`
}
```

## API Development Patterns

### Handler Structure
```go
type SomeHandler struct {
    service SomeService
    logger  Logger
}

func NewSomeHandler(service SomeService) *SomeHandler {
    return &SomeHandler{service: service}
}

func (h *SomeHandler) HandleRequest(c *fiber.Ctx) error {
    // 1. Parse and validate input
    // 2. Call business logic (service layer)
    // 3. Handle errors appropriately
    // 4. Return structured response
}
```

### Error Handling
```go
// Use structured errors with context
if err != nil {
    return fiber.NewError(fiber.StatusBadRequest, 
        fmt.Sprintf("failed to process document: %v", err))
}

// Or use custom error middleware
return models.NewErrorResponse("VALIDATION_ERROR", 
    "Invalid document format", details)
```

### Response Structure
```go
// Success responses
return c.JSON(models.NewSuccessResponse(data, "Operation completed"))

// Error responses  
return c.JSON(models.NewErrorResponse("ERROR_CODE", "Message", details))
```

## Service Integration Patterns

### DigitalOcean Services
```go
// Use service factory pattern
provider, err := digitalocean.NewProviderFromEnvironment()
if err != nil {
    return fmt.Errorf("failed to create provider: %w", err)
}

// Initialize services
err = provider.Initialize()
if err != nil {
    return fmt.Errorf("failed to initialize: %w", err)
}

// Get specific services
services := provider.GetServices()
storage := services.Storage
search := services.Search
```

### Health Checks
```go
// Implement health check interface
func (s *SomeService) IsHealthy() bool {
    // Implement actual health check logic
    return s.client != nil && s.ping()
}

// Use in handlers
func (h *HealthHandler) DetailedStatus(c *fiber.Ctx) error {
    status := &models.SystemStatus{
        Service:   "motion-index-fiber",
        Status:    "healthy",
        Storage:   h.getStorageStatus(),
        Search:    h.getSearchStatus(),
    }
    return c.JSON(models.NewSuccessResponse(status, "System status"))
}
```

## Deployment & Production

### Local Development
1. Configure `.env` with local/staging credentials
2. Use `ENVIRONMENT=local` for mock services
3. Run with `go run cmd/server/main.go`
4. Access at `http://localhost:8003`

### DigitalOcean App Platform
1. Push to GitHub repository
2. Configure environment variables in dashboard
3. Deploy via App Platform interface or API
4. Monitor with health checks and logs


## Code Style & Conventions

### Go Conventions
- Follow standard Go formatting (`go fmt`)
- Use meaningful package and variable names
- Implement interfaces explicitly
- Handle errors explicitly (no silent failures)
- Use context.Context for cancellation and timeouts

### Project Conventions
- Configuration through environment variables
- Structured logging with context
- Comprehensive error handling with wrapped errors
- Health checks for all external dependencies
- Metrics collection for observability

## Troubleshooting Common Issues

### Configuration Issues
- Verify all required environment variables are set
- Check configuration validation in `internal/config/config_test.go`
- Use `go test ./internal/config/... -v` to test configuration

### Service Connection Issues
- Check DigitalOcean service status and credentials
- Verify network connectivity to external services
- Use health check endpoints to diagnose issues
- Check logs for specific error messages

### Test Failures
- Ensure test environment is clean (no conflicting env vars)
- For integration tests, verify real service credentials
- Use `-v` flag for detailed test output
- Check test coverage for missing test cases

### Performance Issues
- Use benchmark tests to identify bottlenecks
- Check memory allocation with `-benchmem`
- Profile with `go tool pprof` if needed
- Monitor health check metrics

## Important Implementation Notes

- **Never commit secrets**: Use environment variables for all sensitive data
- **Always test**: Follow TDD principles and maintain high test coverage
- **Handle errors**: Wrap errors with context and return meaningful messages
- **Document APIs**: Use clear endpoint documentation and examples
- **Monitor health**: Implement comprehensive health checks for all services
- **Follow UNIX**: Keep components small, focused, and composable
- **Validate input**: Sanitize and validate all user input at API boundaries
- **Use interfaces**: Design with interfaces for testability and flexibility

## Quick Reference Commands

```bash
# Start development server (runs on port 8003)
go run cmd/server/main.go

# Run tests with coverage
go test ./... -v -coverprofile=coverage.out

# Build production binary
go build -ldflags="-s -w" -o bin/server cmd/server/main.go

# Test specific package
go test ./internal/handlers/... -v

# Check configuration
go test ./internal/config/... -v

# Benchmark performance
go test ./... -bench=. -benchmem

# Integration tests (requires credentials)
RUN_INTEGRATION_TESTS=true go test ./... -v -tags=integration

# Classification Commands
go run cmd/api-classifier/main.go test-connection           # Test API connectivity
go run cmd/api-classifier/main.go classify-count 10        # Classify 10 documents
go run cmd/api-classifier/main.go classify-all             # Classify all documents
go run cmd/api-batch-classifier/main.go classify-all --limit=100  # Batch classification
go run cmd/setup-index/main.go                             # Setup OpenSearch index
go run cmd/inspect-index/main.go                           # Inspect index structure
```
