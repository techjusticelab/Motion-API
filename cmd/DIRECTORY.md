# Command Directory (`/cmd`)

This directory contains all executable applications and command-line tools for the Motion-Index Fiber project. Each subdirectory represents a separate executable program with its own `main.go` file.

## Applications

### `/server` - Main API Server
**Purpose**: Primary HTTP API server using Fiber framework
**Entry Point**: `main.go`
**Description**: The main web server that handles document processing, search, and management operations. Provides REST API endpoints for the Motion-Index system.

<!-- Batch-oriented CLI tools have been removed to simplify the repository. -->

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

# Setup search indices
go run cmd/setup-index/main.go
```

### Production
```bash
# Build server and utilities
go build -o bin/server cmd/server/main.go
```

## Command Design Principles

1. **Single Responsibility**: Each command has a focused, specific purpose
2. **UNIX Philosophy**: Tools that do one thing well and can be composed
3. **Configuration**: All commands use environment-based configuration
4. **Error Handling**: Comprehensive error reporting and graceful failure
5. **Logging**: Structured logging for monitoring and debugging