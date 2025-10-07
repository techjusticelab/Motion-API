# Motion-Index Fiber Directory Structure

This document provides a comprehensive overview of the directory structure for the Motion-Index Fiber project, a high-performance legal document processing API built with Go and Fiber.

## Root Directory Structure

```
motion-index-fiber/
├── cmd/                    # Command-line applications and entry points
├── deployments/            # Deployment configurations and scripts
├── docs/                   # Project documentation
├── internal/               # Private application code
├── pkg/                    # Public library code
├── test/                   # Test files and utilities
├── bin/                    # Compiled binaries
├── go.mod                  # Go module definition
├── go.sum                  # Go module checksums
├── CLAUDE.md              # AI assistant instructions
├── README.md              # Project overview
└── TODO.md                # Project todo items
```

## Directory Purposes

### `/cmd` - Command-Line Applications
Contains all executable commands and entry points for the application. Each subdirectory represents a different executable with its own main.go file.

### `/deployments` - Deployment Configurations
Houses deployment scripts, configurations, and platform-specific deployment files for DigitalOcean, Docker, and Kubernetes.

### `/docs` - Documentation
Comprehensive project documentation including API specifications, deployment guides, and development documentation.

### `/internal` - Private Application Code
Application-specific code that cannot be imported by external applications. Contains core business logic, handlers, middleware, and internal utilities.

### `/pkg` - Public Library Code
Reusable library code that can be imported by external applications. Contains well-defined interfaces and implementations for cloud services, processing, search, and storage.

### `/test` - Test Files
Testing utilities and integration tests that don't belong in specific package directories.

### `/bin` - Compiled Binaries
Output directory for compiled Go binaries during build processes.

## Architecture Overview

The project follows Go best practices with clear separation between:
- **Commands** (`cmd/`): Entry points and CLI tools
- **Internal Logic** (`internal/`): Business logic and HTTP handlers
- **Reusable Packages** (`pkg/`): Cloud integrations and processing pipelines
- **Configuration** (`internal/config/`): Environment-based configuration
- **Testing** (throughout): Comprehensive test coverage following TDD principles

This structure supports the project's goals of maintainability, testability, and adherence to UNIX philosophy principles.