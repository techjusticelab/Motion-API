# DigitalOcean Cloud Services Package

This package provides comprehensive integration with DigitalOcean cloud services for the Motion-Index system, following UNIX philosophy principles of small, composable, well-tested components.

## Overview

The package supports three environments:
- **Local**: Uses mock services for development
- **Staging**: Uses real DigitalOcean services with staging configuration  
- **Production**: Uses real DigitalOcean services with production configuration

## Architecture

### Key Components

1. **Configuration Management** (`config/`)
   - Environment-based configuration loading
   - Comprehensive validation with custom rules
   - Support for environment variables and defaults

2. **Service Factory** (`factory.go`)
   - Creates appropriate services based on environment
   - Handles service lifecycle and health monitoring
   - Supports dependency injection and testing

3. **Provider Interface** (`digitalocean.go`)
   - Main entry point for all DigitalOcean services
   - Unified interface for configuration, services, and health checks
   - Graceful shutdown and resource management

### Services (Planned Implementation)

- **Spaces Storage** (Phase 4.2): Document storage with CDN integration
- **OpenSearch** (Phase 4.3): Full-text search and document indexing
- **Health Monitoring** (Phase 4.4): Service health checks and metrics

## Usage

### Basic Usage

```go
// Create provider from environment variables
provider, err := digitalocean.NewProviderFromEnvironment()
if err != nil {
    return fmt.Errorf("failed to create provider: %w", err)
}
defer provider.Shutdown()

// Initialize services
err = provider.Initialize()
if err != nil {
    return fmt.Errorf("failed to initialize services: %w", err)
}

// Get services
services := provider.GetServices()
storageService := services.Storage
searchService := services.Search
```

### Configuration

#### Environment Variables

**Required for all environments:**
```bash
ENVIRONMENT=local|staging|production
```

**Required for staging/production:**
```bash
# DigitalOcean Spaces
DO_SPACES_ACCESS_KEY=your-access-key
DO_SPACES_SECRET_KEY=your-secret-key
DO_SPACES_BUCKET=your-bucket-name
DO_SPACES_REGION=nyc3

# DigitalOcean OpenSearch
DO_OPENSEARCH_HOST=your-cluster.db.ondigitalocean.com
DO_OPENSEARCH_PORT=25060
DO_OPENSEARCH_USERNAME=doadmin
DO_OPENSEARCH_PASSWORD=your-password
DO_OPENSEARCH_USE_SSL=true
DO_OPENSEARCH_INDEX=documents
```

**Optional configuration:**
```bash
# CDN and custom endpoints
DO_SPACES_CDN_ENDPOINT=https://your-cdn.example.com
DO_SPACES_ENDPOINT=https://custom.endpoint.com

# Health monitoring
HEALTH_CHECK_INTERVAL=30
HEALTH_TIMEOUT_SECONDS=10
HEALTH_MAX_RETRIES=3
HEALTH_CIRCUIT_BREAKER=true

# Performance tuning
PERF_MAX_CONCURRENT_UPLOADS=10
PERF_MAX_CONCURRENT_DOWNLOADS=20
PERF_CHUNK_SIZE_BYTES=8388608  # 8MB
PERF_ENABLE_CACHING=true
PERF_CACHE_TTL_SECONDS=3600
```

### Advanced Usage

#### Custom Configuration

```go
// Create custom configuration
cfg := &config.Config{
    Environment: config.EnvStaging,
    DigitalOcean: struct{...}{
        Spaces: struct{...}{
            AccessKey: "your-key",
            SecretKey: "your-secret",
            Bucket:    "your-bucket",
            Region:    "nyc3",
        },
        OpenSearch: struct{...}{
            Host:     "your-host.db.ondigitalocean.com",
            Port:     25060,
            Username: "doadmin",
            Password: "your-password",
            UseSSL:   true,
            Index:    "documents",
        },
    },
}

// Validate configuration
if err := cfg.Validate(); err != nil {
    return fmt.Errorf("invalid configuration: %w", err)
}

// Create provider with custom config
provider, err := digitalocean.NewProvider(cfg)
```

#### Health Monitoring

```go
// Check overall health
if !provider.IsHealthy() {
    log.Warn("DigitalOcean services are unhealthy")
}

// Get detailed metrics
metrics := provider.GetMetrics()
log.Info("Service metrics", "metrics", metrics)

// Validate configuration and services
ctx := context.Background()
if err := provider.ValidateConfiguration(ctx); err != nil {
    log.Error("Configuration validation failed", "error", err)
}
```

## Testing

### Unit Tests

Run unit tests for all components:

```bash
# Run all tests
go test ./pkg/cloud/digitalocean/... -v

# Run tests with coverage
go test ./pkg/cloud/digitalocean/... -v -coverprofile=coverage.out
go tool cover -html=coverage.out

# Run specific test package
go test ./pkg/cloud/digitalocean/config/... -v
```

### Integration Tests

Integration tests require real DigitalOcean credentials and are skipped by default:

```bash
# Run integration tests (requires real credentials)
RUN_INTEGRATION_TESTS=true go test ./pkg/cloud/digitalocean/... -v -tags=integration

# Run integration tests with specific environment
INTEGRATION_TEST_ENV=staging RUN_INTEGRATION_TESTS=true go test ./pkg/cloud/digitalocean/... -v -tags=integration
```

#### Integration Test Setup

1. Set up real DigitalOcean resources (Spaces bucket, OpenSearch cluster)
2. Configure environment variables with real credentials
3. Run tests with integration tag

```bash
export INTEGRATION_TEST_ENV=staging
export DO_SPACES_ACCESS_KEY=real-access-key
export DO_SPACES_SECRET_KEY=real-secret-key
export DO_SPACES_BUCKET=test-bucket
export DO_SPACES_REGION=nyc3
export DO_OPENSEARCH_HOST=real-host.db.ondigitalocean.com
export DO_OPENSEARCH_USERNAME=doadmin
export DO_OPENSEARCH_PASSWORD=real-password
export RUN_INTEGRATION_TESTS=true

go test ./pkg/cloud/digitalocean/... -v -tags=integration
```

### Benchmark Tests

Run performance benchmarks:

```bash
# Run all benchmarks
go test ./pkg/cloud/digitalocean/... -bench=. -benchmem

# Run specific benchmarks
go test ./pkg/cloud/digitalocean/... -bench=BenchmarkProviderCreation -benchmem

# Compare performance
go test ./pkg/cloud/digitalocean/... -bench=. -count=5 | tee bench.txt
benchstat bench.txt
```

### Test Categories

1. **Unit Tests** (`*_test.go`): Fast, isolated tests for individual components
2. **Integration Tests** (`integration_test.go`): Tests with real DigitalOcean services
3. **Benchmark Tests**: Performance validation and optimization

## Development

### UNIX Philosophy Principles

This package follows UNIX philosophy:

1. **Do One Thing Well**: Each component has a single, focused responsibility
2. **Composable**: Services can be combined and used independently
3. **Testable**: Comprehensive test coverage with unit, integration, and benchmark tests
4. **Configurable**: Environment-based configuration with sensible defaults
5. **Observable**: Health checks, metrics, and logging throughout

### Adding New Services

1. Define service interface in appropriate package (`pkg/storage/`, `pkg/search/`)
2. Implement service in this package (e.g., `spaces/`, `opensearch/`)
3. Add service creation to `factory.go`
4. Update `Services` struct and related methods
5. Add comprehensive tests (unit, integration, benchmarks)
6. Update documentation

### Code Organization

```
pkg/cloud/digitalocean/
├── README.md                 # This file
├── digitalocean.go          # Main provider interface
├── digitalocean_test.go     # Provider tests
├── factory.go               # Service factory
├── factory_test.go          # Factory tests
├── integration_test.go      # Integration tests
├── config/                  # Configuration management
│   ├── config.go
│   └── config_test.go
├── spaces/                  # Spaces storage service (Phase 4.2)
└── opensearch/              # OpenSearch service (Phase 4.3)
```

## Roadmap

### Phase 4.1: Foundation Setup ✅
- [x] Configuration management with validation
- [x] Environment detection and service factory
- [x] Comprehensive test framework

### Phase 4.2: Spaces Storage Integration (Next)
- [ ] MCP-powered Spaces client implementation
- [ ] Upload/download with progress tracking
- [ ] CDN integration and caching
- [ ] Batch operations and optimization

### Phase 4.3: OpenSearch Service Integration
- [ ] MCP-powered OpenSearch client
- [ ] Search query building and execution
- [ ] Index management and health monitoring
- [ ] Aggregations and analytics

### Phase 4.4: Health Monitoring & Observability
- [ ] Real-time health checks
- [ ] Metrics collection and reporting
- [ ] Circuit breaker implementation
- [ ] Alerting and notification

### Phase 4.5: Production Deployment & Documentation
- [ ] Production deployment guides
- [ ] Performance optimization
- [ ] Security best practices
- [ ] Monitoring and troubleshooting

## Contributing

1. Follow UNIX philosophy principles
2. Write comprehensive tests (unit + integration + benchmarks)
3. Update documentation
4. Validate with real DigitalOcean services when possible
5. Monitor performance and optimize for scale

## Security

- Never commit credentials to source control
- Use environment variables for sensitive configuration
- Validate all inputs and configurations
- Implement proper error handling without exposing secrets
- Follow DigitalOcean security best practices