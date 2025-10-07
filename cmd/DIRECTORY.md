# Command Directory (`/cmd`)

This directory contains all executable applications and command-line tools for the Motion-Index Fiber project. Each subdirectory represents a separate executable program with its own `main.go` file.

## Applications

### `/server` - Main API Server
**Purpose**: Primary HTTP API server using Fiber framework
**Entry Point**: `main.go`
**Description**: The main web server that handles document processing, search, and management operations. Provides REST API endpoints for the Motion-Index system.

### `/api-batch-classifier` - Batch Document Classification
**Purpose**: Batch processing tool for document classification
**Entry Point**: `main.go`
**Description**: Command-line utility for processing multiple documents through AI classification in batch mode. Useful for bulk document processing and system migrations.

### `/batch-processor` - General Batch Processing
**Purpose**: General-purpose batch processing utility
**Entry Point**: `main.go`
**Description**: Handles bulk document processing operations including text extraction, metadata updates, and search index population.

### `/high-performance-batch` - High-Performance Batch Processing
**Purpose**: Optimized batch processing for large-scale operations
**Entry Point**: `main.go`
**Contains**: 
- `bin/` - Compiled binaries for the high-performance processor
**Description**: High-throughput batch processing system designed for processing thousands of documents efficiently with GPU acceleration and parallel processing.

### `/inspect-index` - Search Index Inspector
**Purpose**: Search index debugging and inspection tool
**Entry Point**: `main.go`
**Description**: Command-line utility for inspecting OpenSearch indices, debugging search functionality, and validating document indexing.

### `/setup-index` - Index Setup and Configuration
**Purpose**: Search index initialization and setup
**Entry Point**: `main.go`
**Description**: Utility for creating and configuring OpenSearch indices, setting up mappings, and initializing the search infrastructure.

## Usage Patterns

### Development
```bash
# Start the main API server
go run cmd/server/main.go

# Run batch classification
go run cmd/api-batch-classifier/main.go [options]

# Setup search indices
go run cmd/setup-index/main.go
```

### Production
```bash
# Build all commands
go build -o bin/server cmd/server/main.go
go build -o bin/batch-classifier cmd/api-batch-classifier/main.go
go build -o bin/batch-processor cmd/batch-processor/main.go
```

## Command Design Principles

1. **Single Responsibility**: Each command has a focused, specific purpose
2. **UNIX Philosophy**: Tools that do one thing well and can be composed
3. **Configuration**: All commands use environment-based configuration
4. **Error Handling**: Comprehensive error reporting and graceful failure
5. **Logging**: Structured logging for monitoring and debugging