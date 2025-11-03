# Storage Handler Refactored - Usage Guide

## Quick Start

### 1. Using the Refactored Handler

```go
package main

import (
    "motion-index-fiber/internal/handlers"
    "motion-index-fiber/pkg/storage"
    "github.com/gofiber/fiber/v2"
)

func main() {
    app := fiber.New()

    // Initialize storage service
    storageService := storage.NewDigitalOceanSpacesService(config)

    // Create refactored handler (automatically initializes use cases)
    storageHandler := handlers.NewStorageHandlerRefactored(storageService)

    // Register routes
    app.Get("/api/storage/documents", storageHandler.ListDocuments)
    app.Get("/api/storage/documents/count", storageHandler.GetDocumentsCount)
    app.Get("/api/v1/files/search", storageHandler.FindDocumentsByName)
    app.Get("/api/v1/files/*", storageHandler.ServeDocument)

    app.Listen(":8003")
}
```

### 2. Using Use Cases Directly

```go
import (
    "motion-index-fiber/internal/application/usecase/storage"
    "motion-index-fiber/internal/application/dto"
)

// Initialize use cases
listUC := storage.NewListStorageObjectsUseCase(storageService)

// Use in any context (handler, CLI, background job)
req := &dto.ListStorageObjectsRequest{
    Prefix:   "documents/",
    Limit:    50,
    FileType: ".pdf",
    MinSize:  1024,
    MaxSize:  -1,
}

result, err := listUC.Execute(ctx, req)
if err != nil {
    // Handle error
}

// Use result
for _, doc := range result.Documents {
    fmt.Printf("File: %s, Size: %d\n", doc.Filename, doc.Size)
}
```

## API Endpoints

### List Documents
**GET** `/api/storage/documents`

Query Parameters:
- `prefix` - Directory prefix (default: "documents/")
- `limit` - Max results per page (default: 50, max: 500)
- `cursor` - Pagination cursor
- `file_type` - Filter by extension (e.g., ".pdf")
- `min_size` - Minimum file size in bytes
- `max_size` - Maximum file size in bytes

Example:
```bash
curl "http://localhost:8003/api/storage/documents?limit=10&file_type=.pdf&min_size=1024"
```

Response:
```json
{
  "status": "success",
  "data": {
    "documents": [
      {
        "path": "documents/file.pdf",
        "size": 2048,
        "last_modified": "2025-01-01T00:00:00Z",
        "file_type": ".pdf",
        "filename": "file.pdf"
      }
    ],
    "next_cursor": "eyJpbmRleCI6MTB9",
    "has_more": true,
    "total_returned": 10,
    "total_estimated": 50
  }
}
```

### Count Documents
**GET** `/api/storage/documents/count`

Query Parameters:
- `prefix` - Directory prefix (default: "documents/")
- `file_type` - Filter by extension
- `min_size` - Minimum file size
- `max_size` - Maximum file size

Example:
```bash
curl "http://localhost:8003/api/storage/documents/count?file_type=.pdf"
```

Response:
```json
{
  "status": "success",
  "data": {
    "total_count": 42,
    "prefix": "documents/",
    "applied_filters": {
      "file_type": ".pdf",
      "min_size": 0,
      "max_size": -1
    }
  }
}
```

### Search Documents
**GET** `/api/v1/files/search`

Query Parameters:
- `name` - Search pattern (required)
- `prefix` - Directory prefix (default: "documents/")
- `limit` - Max results (default: 20, max: 100)
- `exact` - Exact match (true/false, default: false)

Example:
```bash
curl "http://localhost:8003/api/v1/files/search?name=motion&limit=10"
```

Response:
```json
{
  "status": "success",
  "data": {
    "documents": [
      {
        "path": "documents/motion_to_suppress.pdf",
        "filename": "motion_to_suppress.pdf",
        "size": 2048,
        "last_modified": "2025-01-01T00:00:00Z",
        "file_type": ".pdf",
        "direct_url": "https://cdn.example.com/documents/motion_to_suppress.pdf",
        "signed_url": "https://cdn.example.com/documents/motion_to_suppress.pdf?signed=true",
        "api_url": "/api/v1/files/motion_to_suppress.pdf"
      }
    ],
    "total_found": 1,
    "search_pattern": "motion",
    "exact_match": false,
    "limit": 10
  }
}
```

### Serve Document
**GET** `/api/v1/files/{path}`

Query Parameters:
- `signed` - Use signed URL (true/false, default: true)
- `expires` - Expiration duration (default: "1h", max: "24h")
- `download` - Force download (true/false, default: false)

Examples:
```bash
# Serve with signed URL (default)
curl "http://localhost:8003/api/v1/files/motion.pdf"

# Serve with public URL
curl "http://localhost:8003/api/v1/files/motion.pdf?signed=false"

# Force download
curl "http://localhost:8003/api/v1/files/motion.pdf?download=true"

# Custom expiration
curl "http://localhost:8003/api/v1/files/motion.pdf?expires=2h"
```

## Testing

### Run All Tests
```bash
# Use case tests
go test ./internal/application/usecase/storage/... -v -cover

# Handler tests
go test ./internal/handlers/... -run "TestStorageHandlerRefactored" -v -cover

# All tests together
go test ./internal/application/usecase/storage/... ./internal/handlers/... -run "TestStorageHandlerRefactored" -v
```

### Run Specific Test Suite
```bash
# List use case tests
go test ./internal/application/usecase/storage/... -run TestListStorageObjectsUseCase -v

# Handler integration tests
go test ./internal/handlers/... -run TestStorageHandlerRefactored_Integration -v
```

### Check Coverage
```bash
# Use case coverage
go test ./internal/application/usecase/storage/... -coverprofile=coverage.out
go tool cover -html=coverage.out

# Handler coverage
go test ./internal/handlers/storage_refactored_test.go ./internal/handlers/storage_refactored.go -coverprofile=coverage.out
go tool cover -func=coverage.out
```

## Architecture Benefits

### 1. **Separation of Concerns**
```
HTTP Layer (Handler) → Application Layer (Use Cases) → Domain Layer → Infrastructure
```

### 2. **Easy to Test**
- Use cases tested independently
- Handlers tested with mocks
- No external dependencies required

### 3. **Reusable Business Logic**
Use cases can be called from:
- HTTP handlers (current)
- gRPC services (future)
- CLI commands (e.g., batch operations)
- Background jobs (e.g., cleanup tasks)
- GraphQL resolvers (if needed)

### 4. **Easy to Extend**
Add new functionality by:
1. Create new use case
2. Add corresponding DTO
3. Wire into handler
4. Write tests

Example - Adding a "Move Document" feature:
```go
// 1. Create use case
type MoveStorageObjectUseCase struct { ... }

// 2. Add DTO
type MoveStorageObjectRequest struct { ... }

// 3. Wire into handler
func (h *StorageHandlerRefactored) MoveDocument(c *fiber.Ctx) error {
    req := &dto.MoveStorageObjectRequest{ ... }
    result, err := h.moveUC.Execute(c.Context(), req)
    // ...
}

// 4. Write tests
func TestMoveStorageObjectUseCase_Execute(t *testing.T) { ... }
```

## Migration Path

### Option 1: Side-by-Side (Recommended)
Keep both handlers during migration:
```go
// Old routes (deprecated)
app.Get("/api/storage/documents", oldHandler.ListDocuments)

// New routes (preferred)
app.Get("/api/v2/storage/documents", newHandler.ListDocuments)
```

### Option 2: Feature Flag
Use feature flags to gradually switch:
```go
if config.UseRefactoredHandler {
    app.Get("/api/storage/documents", newHandler.ListDocuments)
} else {
    app.Get("/api/storage/documents", oldHandler.ListDocuments)
}
```

### Option 3: Direct Replacement
Replace old handler with new one:
```go
// Replace in cmd/server/main.go
- storageHandler := handlers.NewStorageHandler(cfg, storageService)
+ storageHandler := handlers.NewStorageHandlerRefactored(storageService)
```

## Performance Considerations

### Pagination
- Use cursor-based pagination for large datasets
- Default limit: 50 items
- Maximum limit: 500 items

### Caching
Consider adding caching at use case level:
```go
type CachedListStorageObjectsUseCase struct {
    delegate *ListStorageObjectsUseCase
    cache    Cache
}
```

### Rate Limiting
Add rate limiting at handler level:
```go
app.Use(limiter.New(limiter.Config{
    Max: 100,
    Expiration: 1 * time.Minute,
}))
```

## Troubleshooting

### Issue: "Document not found"
- Check path encoding (should be URL-encoded)
- Verify document exists with List or Search endpoints
- Check prefix is correct (usually "documents/")

### Issue: High memory usage
- Reduce pagination limit
- Add streaming for large file downloads
- Implement result caching

### Issue: Slow search performance
- Add indexes in storage layer
- Implement prefix-based search
- Consider search service (Elasticsearch, etc.)

## Support

For issues or questions:
1. Check test files for usage examples
2. Review Phase 5 completion summary
3. Check CLAUDE.md for project guidelines
4. File issue in GitHub repository
