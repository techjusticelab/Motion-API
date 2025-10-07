# Package Directory (`/pkg`)

This directory contains reusable library code that can be imported by external applications. These packages provide well-defined interfaces and implementations for cloud services, document processing, search functionality, and storage operations.

## Structure

```
pkg/
├── api/                 # API utilities and helpers
├── cloud/               # Cloud service integrations
├── monitoring/          # Monitoring and metrics
├── processing/          # Document processing pipeline
├── search/              # Search functionality
└── storage/             # Storage interfaces and implementations
```

## Package Descriptions

### `/api` - API Utilities
**Purpose**: Reusable API utilities and helpers
**Files**:
- `health.go` - Health check utilities

**Responsibilities**:
- Common API response utilities
- Health check implementations
- API versioning support
- Request/response helpers

### `/cloud` - Cloud Service Integrations
**Purpose**: Cloud platform integrations and abstractions

#### `/cloud/digitalocean` - DigitalOcean Integration
**Files**:
- `README.md` - DigitalOcean integration documentation
- `digitalocean.go` - Main DigitalOcean service implementation
- `digitalocean_test.go` - Unit tests
- `factory.go` - Service factory pattern
- `factory_test.go` - Factory testing
- `integration_test.go` - Integration testing
- `benchmark_test.go` - Performance benchmarks

**Subdirectories**:
- `/config` - DigitalOcean configuration management
- `/spaces` - DigitalOcean Spaces storage integration

**Responsibilities**:
- DigitalOcean API integration
- Spaces storage operations
- OpenSearch cluster management
- Service discovery and configuration
- Authentication and authorization

### `/monitoring` - Monitoring and Metrics
**Purpose**: Application monitoring, metrics collection, and observability
**Files**:
- `metrics.go` - Metrics collection and reporting

**Responsibilities**:
- Performance metrics collection
- System health monitoring
- Custom metric definitions
- Integration with monitoring systems
- Alert configuration

### `/processing` - Document Processing Pipeline
**Purpose**: Comprehensive document processing capabilities

#### Subpackages:
- `/classifier` - AI-powered document classification
- `/extractor` - Text extraction from various document formats
- `/gpu` - GPU acceleration for processing
- `/migration` - Data migration utilities
- `/pipeline` - Processing pipeline coordination
- `/queue` - Queue management and job processing

**Key Files**:
- Interface definitions for each processing stage
- Service implementations with dependency injection
- Mock implementations for testing
- Performance optimization utilities

**Responsibilities**:
- Document text extraction (PDF, DOCX, TXT, RTF)
- AI-powered document classification
- OCR processing for scanned documents
- Processing pipeline orchestration
- Queue management and job scheduling
- GPU acceleration where available

### `/search` - Search Functionality
**Purpose**: Search interfaces and OpenSearch integration
**Files**:
- `interfaces.go` - Search interface definitions
- `service.go` - Search service implementation
- `service_test.go` - Service testing
- `aggregations.go` - Search aggregations
- `aggregations_test.go` - Aggregation testing

**Subdirectories**:
- `/client` - OpenSearch client implementations
- `/models` - Search data models
- `/query` - Query building utilities

**Responsibilities**:
- Full-text search implementation
- Document indexing and mapping
- Search aggregations and faceting
- Query building and optimization
- Result ranking and scoring

### `/storage` - Storage Interfaces
**Purpose**: Storage abstractions and implementations
**Files**:
- `interface.go` - Storage interface definitions
- `spaces.go` - DigitalOcean Spaces implementation
- `spaces_test.go` - Spaces testing
- `storage_test.go` - General storage testing
- `utils.go` - Storage utilities
- `utils_test.go` - Utility testing

**Responsibilities**:
- Storage interface abstractions
- DigitalOcean Spaces integration
- File upload and download operations
- CDN integration and optimization
- Storage utility functions

## Design Principles

### Interface-First Design
All packages define clear interfaces before implementations, enabling:
- Easy testing with mocks
- Multiple implementation strategies
- Dependency injection patterns
- Future extensibility

### UNIX Philosophy
- Each package does one thing well
- Packages are composable and independent
- Clear separation of concerns
- Minimal dependencies between packages

### Performance Optimization
- Benchmark tests for critical paths
- Memory allocation optimization
- Concurrent processing where appropriate
- GPU acceleration for intensive operations

### Error Handling
- Comprehensive error definitions
- Error wrapping with context
- Graceful degradation strategies
- Detailed error logging

## Package Dependencies

### External Dependencies
- Cloud provider SDKs (AWS S3 for Spaces)
- OpenSearch/Elasticsearch clients
- Document processing libraries
- AI/ML service clients

### Internal Dependencies
```
processing → search, storage
cloud → search, storage
monitoring → all packages (for metrics)
api → core utilities only
```

## Testing Strategy

### Unit Tests
- Interface compliance testing
- Mock-based isolated testing
- Error condition testing
- Edge case validation

### Integration Tests
- Real service integration testing
- End-to-end workflow testing
- Performance benchmarking
- Load testing scenarios

### Benchmark Tests
- Performance regression testing
- Memory allocation tracking
- Concurrent operation testing
- Resource utilization optimization

## Usage Examples

### Basic Service Initialization
```go
// Initialize DigitalOcean services
provider, err := digitalocean.NewProviderFromEnvironment()
services := provider.GetServices()

// Use storage service
storage := services.Storage
document, err := storage.GetDocument(ctx, "document-id")

// Use search service
search := services.Search
results, err := search.Search(ctx, query)
```

### Processing Pipeline
```go
// Initialize processing pipeline
pipeline := processing.NewPipeline(
    extractor.NewService(),
    classifier.NewService(),
    search.NewIndexer(),
)

// Process document
result, err := pipeline.ProcessDocument(ctx, document)
```

## Best Practices

1. **Interface Compliance**: All implementations must satisfy defined interfaces
2. **Context Propagation**: Use context.Context for cancellation and timeouts
3. **Error Wrapping**: Provide context when wrapping errors
4. **Testing**: Maintain high test coverage with meaningful tests
5. **Performance**: Include benchmark tests for performance-critical code
6. **Documentation**: Document all public interfaces and complex logic