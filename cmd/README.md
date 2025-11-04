# Motion-Index Command Line Utilities

This directory contains command-line utilities for the Motion-Index legal document processing system.

## 🚀 Production Commands

### `server/`
**Main HTTP server** - The primary web application
```bash
go run cmd/server/main.go
```
- Starts Fiber web server on configured port (default: 6000)
- Handles document upload, processing, and search APIs
- Production-ready with graceful shutdown and health checks

<!-- Batch processors have been removed to simplify the repository. -->

## 🔧 Maintenance Commands

### `inspect-index/`
**Index inspection tool** - Examine OpenSearch index
```bash
go run cmd/inspect-index/main.go
```
- Shows index statistics and document counts
- Displays mapping information
- Useful for debugging search issues

### `setup-index/`
**Index setup tool** - Initialize OpenSearch index
```bash
go run cmd/setup-index/main.go
```
- Creates document index with proper legal metadata mapping
- Sets up field types for enhanced legal schema
- Required before first document indexing

## 🧪 Testing Commands

### `test-integration/`
**Consolidated integration testing** - Test all system components
```bash
go run cmd/test-integration/main.go <mode>
```
Modes:
- `json` - Test document JSON serialization/deserialization
- `extraction` - Test text extraction services
- `classification` - Test AI document classification (mock)
- `indexing` - Test document indexing to OpenSearch
- `pipeline` - Test full document processing pipeline
- `all` - Run all tests sequentially

### `test-model-json/`
**JSON model testing** - Test document models with OpenSearch
```bash
go run cmd/test-model-json/main.go
```
- Tests JSON document creation and indexing
- Verifies OpenSearch integration
- Useful for debugging model serialization issues

## 📋 Usage Examples

```bash
# Start the main server
go run cmd/server/main.go

# Test system integration
go run cmd/test-integration/main.go all

# Set up OpenSearch index
go run cmd/setup-index/main.go

# Inspect search index
go run cmd/inspect-index/main.go
```

## 🗂️ Previous Commands (Removed)

The following duplicate and debugging commands were removed to reduce clutter:

**Removed Duplicates:**
- `debug-model-json/` → Use `test-integration/` json mode
- `debug-opensearch/` → Use `test-integration/` indexing mode
- `debug-extraction/` → Use `test-integration/` extraction mode
- `debug-pipeline/` → Use `test-integration/` pipeline mode
- `test-direct-index/` → Use `test-integration/` indexing mode
- `test-search-service/` → Use `test-integration/` indexing mode
- `test-id-format/` → Use `test-integration/` indexing mode
- `test-raw-index/` → Use `test-integration/` indexing mode

All functionality is preserved in the consolidated commands.

## 💡 Development Tips

1. **Start with integration tests** - Run `test-integration` to verify system components
2. **Check index health** - Use `inspect-index` to verify OpenSearch state
4. **Development workflow**:
   ```bash
   # Setup (first time)
   go run cmd/setup-index/main.go
   
   # Test system
   go run cmd/test-integration/main.go all
   
   # Start server
   go run cmd/server/main.go
   ```