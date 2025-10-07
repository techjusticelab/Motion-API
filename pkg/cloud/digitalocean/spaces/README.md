# DigitalOcean Spaces Storage Service

This package implements production-ready DigitalOcean Spaces storage service with MCP integration, following UNIX philosophy principles.

## Architecture

### Hybrid Approach
The implementation uses a hybrid architecture combining the best of both worlds:

1. **MCP Tools**: For DigitalOcean-specific management operations
   - CDN creation, management, and cache invalidation
   - Access key lifecycle management
   - DigitalOcean-specific configurations

2. **S3-Compatible API**: For high-performance file operations
   - Upload/download operations with streaming
   - Batch operations and concurrent handling
   - Proven S3 performance and reliability

### Design Principles

#### UNIX Philosophy
- **Do One Thing Well**: Each component has a single, focused responsibility
- **Composable**: Services can be combined and used independently  
- **Testable**: Comprehensive test coverage with unit, integration, and benchmark tests
- **Observable**: Health checks, metrics, and logging throughout

#### Performance First
- Streaming uploads/downloads for memory efficiency
- Connection pooling and reuse
- CDN optimization for global delivery
- Concurrent operations with proper limits

#### Production Ready
- Comprehensive error handling with retry logic
- Circuit breaker for fault tolerance
- Health monitoring and metrics collection
- Security best practices

## Components

### Core Implementation
- `spaces.go` - Main SpacesClient implementing storage.Service interface
- `mcp_client.go` - MCP tool wrapper for DigitalOcean operations
- `s3_client.go` - S3-compatible client for file operations

### Specialized Operations
- `upload.go` - Upload operations with multipart support
- `download.go` - Download operations with streaming
- `operations.go` - Delete, Exists, List operations
- `urls.go` - URL generation (public and signed)

### Optimization and Management
- `cdn.go` - CDN management and optimization
- `cache.go` - Cache strategy and invalidation
- `batch.go` - Batch operations for efficiency
- `performance.go` - Performance optimizations

### Reliability and Monitoring
- `errors.go` - Error types and handling
- `retry.go` - Retry logic with exponential backoff
- `circuit_breaker.go` - Circuit breaker implementation
- `health.go` - Health checks and monitoring
- `metrics.go` - Metrics collection and reporting

## Current Infrastructure

Based on MCP discovery, the following infrastructure is already configured:

### Spaces Configuration
- **Bucket**: `motion-index-docs`
- **Region**: `nyc3` 
- **Origin**: `motion-index-docs.nyc3.digitaloceanspaces.com`

### CDN Configuration
- **ID**: `e7187b3e-9956-46a0-80aa-67cc63c2d110`
- **Endpoint**: `motion-index-docs.nyc3.cdn.digitaloceanspaces.com`
- **TTL**: 86400 seconds (24 hours)

### Access Key
- **Name**: `motion-index-upload-key`
- **Access Key**: `<your-spaces-access-key>`
- **Permissions**: Full access to all buckets (store the real key in `.env` only)

## Usage

### Basic Operations
```go
// Create Spaces client
client, err := spaces.NewSpacesClient(config)
if err != nil {
    return err
}

// Upload a file
result, err := client.Upload(ctx, "documents/file.pdf", reader, metadata)

// Download a file  
reader, err := client.Download(ctx, "documents/file.pdf")

// Get CDN-optimized public URL
url := client.GetURL("documents/file.pdf") // Returns CDN URL

// Get signed URL for temporary access
signedURL, err := client.GetSignedURL("documents/file.pdf", time.Hour)
```

### Advanced Features
```go
// Batch upload with progress tracking
results, err := client.BatchUpload(ctx, files, progressCallback)

// CDN cache invalidation
err = client.InvalidateCache([]string{"documents/*"})

// Health check
healthy := client.IsHealthy()
metrics := client.GetMetrics()
```

## Performance Targets

### Throughput
- **Upload**: 50MB/s for large files (>10MB)
- **Download**: 100MB/s for CDN-cached content
- **Batch Operations**: 100 files/second

### Latency
- **Small Files** (<1MB): <200ms upload/download
- **URL Generation**: <1ms
- **Health Checks**: <100ms

### Reliability
- **Availability**: 99.9% uptime
- **Error Rate**: <0.1% for normal operations
- **Recovery**: <30s for transient failures

## Testing

### Unit Tests
```bash
go test ./pkg/cloud/digitalocean/spaces/... -v
```

### Integration Tests (requires real credentials)
```bash
RUN_INTEGRATION_TESTS=true go test ./pkg/cloud/digitalocean/spaces/... -v -tags=integration
```

### Benchmark Tests
```bash
go test ./pkg/cloud/digitalocean/spaces/... -bench=. -benchmem
```

## Security

### Authentication
- Secure credential management via configuration
- Access key rotation support through MCP tools
- Signed URL expiration enforcement

### Data Protection
- In-transit encryption (TLS) for all operations
- Content validation and integrity checks
- Optional client-side encryption support

### Access Control
- Bucket policy compliance
- Least privilege access patterns
- Audit logging for all operations
