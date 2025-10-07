# Internal Directory (`/internal`)

This directory contains private application code that is specific to the Motion-Index Fiber application and cannot be imported by external applications. It follows Go's internal package convention to enforce encapsulation.

## Structure

```
internal/
├── config/               # Application configuration management
├── handlers/             # HTTP request handlers and routing
├── hardware/             # Hardware detection and optimization
├── middleware/           # HTTP middleware components
├── models/              # Data models and validation
├── processing/          # Internal processing coordination
└── testutil/            # Internal testing utilities
```

## Package Descriptions

### `/config` - Configuration Management
**Purpose**: Application configuration, validation, and environment management
**Files**:
- `config.go` - Main configuration structure and loading
- `config_test.go` - Configuration testing
- `performance.go` - Performance-related configuration
- `system.go` - System-level configuration

**Responsibilities**:
- Environment variable parsing
- Configuration validation
- Default value management
- Performance tuning parameters
- System resource configuration

### `/handlers` - HTTP Request Handlers
**Purpose**: HTTP request handling and business logic coordination
**Files**:
- `handlers.go` - Main handler setup and routing
- `handlers_test.go` - Handler unit tests
- `health.go` - Health check endpoints
- `health_test.go` - Health check testing
- `processing.go` - Document processing endpoints
- `processing_integration_test.go` - Integration tests
- `processing_test.go` - Processing unit tests
- `search.go` - Search endpoints
- `search_extended_test.go` - Extended search testing
- `search_test.go` - Search unit tests
- `storage.go` - Storage management endpoints
- `storage_test.go` - Storage testing
- `batch.go` - Batch processing endpoints

**Responsibilities**:
- HTTP request parsing and validation
- Business logic coordination
- Response formatting
- Error handling
- Authentication integration
- Rate limiting enforcement

### `/hardware` - Hardware Detection and Optimization
**Purpose**: Hardware capability detection and performance optimization
**Files**:
- `analyzer.go` - Hardware analysis and capability detection

**Responsibilities**:
- CPU architecture detection
- GPU availability and capability assessment
- Memory analysis and optimization
- Performance profiling and tuning
- Resource allocation optimization

### `/middleware` - HTTP Middleware
**Purpose**: Reusable HTTP middleware components for request processing
**Files**:
- `auth.go` - Authentication middleware
- `auth_test.go` - Authentication testing
- `error.go` - Error handling utilities
- `error_handler.go` - HTTP error handling middleware
- `error_handler_test.go` - Error handler testing
- `file_upload.go` - File upload handling middleware
- `file_upload_test.go` - File upload testing

**Responsibilities**:
- JWT authentication and authorization
- Request/response logging
- Error handling and formatting
- File upload processing
- CORS handling
- Security headers

### `/models` - Data Models and Validation
**Purpose**: Internal data structures, validation, and serialization
**Files**:
- `requests.go` - Request data models
- `responses.go` - Response data models
- `validation.go` - Data validation logic
- `models_test.go` - Model testing

**Responsibilities**:
- Request/response structure definitions
- Data validation rules
- JSON serialization/deserialization
- Input sanitization
- Business rule validation

### `/processing` - Internal Processing Coordination
**Purpose**: Internal processing workflow coordination and management
**Files**:
- `coordinator.go` - Processing workflow coordination
- `processors.go` - Internal processor definitions

**Responsibilities**:
- Processing pipeline coordination
- Internal workflow management
- Resource allocation for processing
- Error handling and recovery
- Progress tracking and reporting

### `/testutil` - Internal Testing Utilities
**Purpose**: Testing utilities and helpers specific to internal packages
**Files**:
- `testutil.go` - Internal testing utilities

**Responsibilities**:
- Test environment setup
- Mock data generation
- Test configuration management
- Common test assertions
- Integration test helpers

## Design Principles

### Encapsulation
- All code in `/internal` is private to the application
- No external dependencies can import internal packages
- Clear separation between public interfaces and private implementation

### UNIX Philosophy
- Each package has a single, focused responsibility
- Packages are composable and can be used independently
- Minimal dependencies between internal packages

### Testing Strategy
- Comprehensive unit tests for all packages
- Integration tests for handler interactions
- Mock-based testing for external dependencies
- Performance testing for critical paths

### Error Handling
- Structured error handling with context
- Error wrapping and propagation
- Graceful degradation patterns
- Comprehensive error logging

## Inter-Package Dependencies

### Configuration Flow
`config` → `handlers`, `middleware`, `processing`

### Request Processing Flow
`middleware` → `handlers` → `processing` → `models`

### Testing Dependencies
`testutil` → all other packages (for testing only)

## Best Practices

1. **Import Restrictions**: Internal packages should not import from `pkg/`
2. **Error Handling**: Use structured errors with proper context
3. **Testing**: Maintain high test coverage with meaningful tests
4. **Configuration**: Use environment-based configuration throughout
5. **Logging**: Implement structured logging for debugging and monitoring