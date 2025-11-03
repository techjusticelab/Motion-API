# Phase 6: Dependency Injection with Wire

## Overview

Wire everything together using Google Wire for compile-time dependency injection. Eliminate manual wiring, reduce boilerplate, and ensure all dependencies are satisfied at compile time.

## Objectives

- [ ] Install and configure Google Wire
- [ ] Create provider functions for all components
- [ ] Create injection sets for grouped dependencies
- [ ] Generate wire_gen.go for main application
- [ ] Update main.go to use wire-generated code
- [ ] Create test container for integration tests
- [ ] Eliminate all manual dependency wiring
- [ ] Achieve compile-time dependency validation

## Prerequisites

- **Phase 1 Complete**: Domain layer
- **Phase 2 Complete**: Application layer with use cases
- **Phase 3 Complete**: Configuration management
- **Phase 4 Complete**: Infrastructure adapters
- **Phase 5 Complete**: HTTP handlers
- Google Wire knowledge
- Dependency injection patterns

## Current State

**Manual Wiring**: `cmd/server/main.go`, `internal/handlers/handlers.go`
- Manual creation of all services (~242 LOC in main.go)
- Hard to maintain (add new dependency = update multiple files)
- No compile-time validation
- Duplicated wiring in tests

**Example of Manual Wiring**:
```go
// cmd/server/main.go (current)
func main() {
    cfg := config.Load()

    // Manually create clients
    storageClient := spaces.NewClient(cfg.Spaces)
    searchClient := opensearch.NewClient(cfg.OpenSearch)

    // Manually create services
    storageService := storage.NewService(storageClient)
    searchService := search.NewService(searchClient)

    // Manually create use cases
    processUseCase := document.NewProcessDocumentUseCase(
        repo, storage, classifier, indexer, eventBus,
    )

    // Manually create handlers
    docHandler := handlers.NewDocumentHandler(processUseCase, updateUseCase)

    // ... 200+ lines of manual wiring
}
```

**Problems**:
- Error-prone (easy to forget dependencies)
- No compile-time safety
- Hard to maintain
- Duplicated in tests

## Target State

**Wire-Based DI**:
- Declarative provider functions
- Compile-time dependency graph validation
- Generated wiring code
- Single source of truth for dependencies

**Example with Wire**:
```go
// cmd/server/main.go (target)
func main() {
    app, cleanup, err := InitializeApplication()
    if err != nil {
        log.Fatal(err)
    }
    defer cleanup()

    app.Start()
}

// wire.go
//go:build wireinject
func InitializeApplication() (*Application, func(), error) {
    wire.Build(
        ConfigSet,
        InfrastructureSet,
        ApplicationSet,
        HTTPSet,
    )
    return nil, nil, nil
}
```

## Task Breakdown

### Task Group 1: Wire Setup

#### Task 1.1: Install Wire
```bash
go install github.com/google/wire/cmd/wire@latest
```

#### Task 1.2: Create Wire Configuration
**File**: `tools.go`

```go
//go:build tools

package main

import (
    _ "github.com/google/wire/cmd/wire"
)
```

**File**: `go.mod`
```go
require (
    github.com/google/wire v0.5.0
)
```

---

### Task Group 2: Provider Functions

#### Task 2.1: Configuration Providers
**File**: `internal/container/providers/config.go`

```go
package providers

import (
    "motion-index-fiber/internal/configuration"
    "motion-index-fiber/internal/configuration/server"
    "motion-index-fiber/internal/configuration/cloud"
    "motion-index-fiber/internal/configuration/ai"
)

// ProvideConfig loads and builds configuration
func ProvideConfig() (*configuration.Config, error) {
    builder := configuration.NewConfigBuilder()

    serverCfg, err := server.LoadFromEnvironment()
    if err != nil {
        return nil, err
    }

    cloudCfg, err := cloud.LoadFromEnvironment()
    if err != nil {
        return nil, err
    }

    aiCfg, err := ai.LoadFromEnvironment()
    if err != nil {
        return nil, err
    }

    return builder.
        WithServer(serverCfg).
        WithCloud(cloudCfg).
        WithAI(aiCfg).
        Build()
}

// ProvideServerConfig extracts server config
func ProvideServerConfig(cfg *configuration.Config) *server.Config {
    return cfg.Server
}

// ProvideCloudConfig extracts cloud config
func ProvideCloudConfig(cfg *configuration.Config) *cloud.Config {
    return cfg.Cloud
}

// ProvideAIConfig extracts AI config
func ProvideAIConfig(cfg *configuration.Config) *ai.Config {
    return cfg.AI
}
```

---

#### Task 2.2: Infrastructure Providers
**File**: `internal/container/providers/infrastructure.go`

```go
package providers

import (
    "motion-index-fiber/internal/configuration/cloud"
    "motion-index-fiber/internal/infrastructure/persistence/opensearch"
    "motion-index-fiber/internal/infrastructure/persistence/spaces"
    "motion-index-fiber/internal/application/ports"
)

// ProvideDocumentRepository provides document repository
func ProvideDocumentRepository(cfg *cloud.Config) (ports.DocumentRepository, error) {
    client, err := opensearch.NewClient(cfg.OpenSearch)
    if err != nil {
        return nil, err
    }

    return opensearch.NewDocumentRepository(client, cfg.OpenSearch.Index), nil
}

// ProvideStorageService provides storage service
func ProvideStorageService(cfg *cloud.Config) (ports.StorageService, error) {
    client, err := spaces.NewS3Client(cfg.Spaces)
    if err != nil {
        return nil, err
    }

    return spaces.NewStorageAdapter(client, cfg.Spaces), nil
}

// ProvideSearchService provides search service
func ProvideSearchService(cfg *cloud.Config) (ports.SearchService, error) {
    client, err := opensearch.NewClient(cfg.OpenSearch)
    if err != nil {
        return nil, err
    }

    return opensearch.NewSearchService(client, cfg.OpenSearch.Index), nil
}
```

---

#### Task 2.3: AI Provider Providers
**File**: `internal/container/providers/ai.go`

```go
package providers

import (
    "motion-index-fiber/internal/configuration/ai"
    "motion-index-fiber/internal/infrastructure/ai/registry"
    "motion-index-fiber/internal/infrastructure/ai/providers/openai"
    "motion-index-fiber/internal/infrastructure/ai/providers/claude"
    "motion-index-fiber/internal/infrastructure/ai/providers/ollama"
    "motion-index-fiber/internal/infrastructure/ai/fallback"
    "motion-index-fiber/internal/application/ports"
)

// ProvideAIRegistry provides AI provider registry
func ProvideAIRegistry(cfg *ai.Config) (*registry.AIProviderRegistry, error) {
    reg := registry.NewAIProviderRegistry()

    // Register OpenAI if configured
    if cfg.OpenAI.Enabled {
        openaiProvider, err := openai.NewProvider(cfg.OpenAI)
        if err != nil {
            return nil, err
        }
        reg.Register("openai", openaiProvider)
    }

    // Register Claude if configured
    if cfg.Claude.Enabled {
        claudeProvider, err := claude.NewProvider(cfg.Claude)
        if err != nil {
            return nil, err
        }
        reg.Register("claude", claudeProvider)
    }

    // Register Ollama if configured
    if cfg.Ollama.Enabled {
        ollamaProvider, err := ollama.NewProvider(cfg.Ollama)
        if err != nil {
            return nil, err
        }
        reg.Register("ollama", ollamaProvider)
    }

    return reg, nil
}

// ProvideClassificationService provides classification service with fallback
func ProvideClassificationService(
    registry *registry.AIProviderRegistry,
    cfg *ai.Config,
) (ports.ClassificationService, error) {
    // Get primary provider
    primary, err := registry.Get(cfg.PrimaryProvider)
    if err != nil {
        return nil, err
    }

    // Get fallback providers
    var fallbacks []ports.ClassificationService
    for _, name := range cfg.FallbackProviders {
        provider, err := registry.Get(name)
        if err == nil {
            fallbacks = append(fallbacks, provider)
        }
    }

    // Create fallback classifier
    return fallback.NewFallbackClassifier(primary, fallbacks), nil
}
```

---

#### Task 2.4: Use Case Providers
**File**: `internal/container/providers/usecases.go`

```go
package providers

import (
    "motion-index-fiber/internal/application/usecases/document"
    "motion-index-fiber/internal/application/ports"
)

// ProvideProcessDocumentUseCase provides process document use case
func ProvideProcessDocumentUseCase(
    repo ports.DocumentRepository,
    storage ports.StorageService,
    classifier ports.ClassificationService,
    indexer ports.SearchService,
    eventBus ports.EventBus,
) *document.ProcessDocumentUseCase {
    return document.NewProcessDocumentUseCase(
        repo, storage, classifier, indexer, eventBus,
    )
}

// ProvideClassifyDocumentUseCase provides classify document use case
func ProvideClassifyDocumentUseCase(
    repo ports.DocumentRepository,
    classifier ports.ClassificationService,
) *document.ClassifyDocumentUseCase {
    return document.NewClassifyDocumentUseCase(repo, classifier)
}

// ProvideSearchDocumentsUseCase provides search documents use case
func ProvideSearchDocumentsUseCase(
    searcher ports.SearchService,
) *document.SearchDocumentsUseCase {
    return document.NewSearchDocumentsUseCase(searcher)
}

// ProvideUpdateMetadataUseCase provides update metadata use case
func ProvideUpdateMetadataUseCase(
    repo ports.DocumentRepository,
) *document.UpdateMetadataUseCase {
    return document.NewUpdateMetadataUseCase(repo)
}
```

---

#### Task 2.5: HTTP Handler Providers
**File**: `internal/container/providers/handlers.go`

```go
package providers

import (
    "motion-index-fiber/internal/application/usecases/document"
    "motion-index-fiber/internal/infrastructure/http/handlers"
)

// ProvideDocumentHandler provides document handler
func ProvideDocumentHandler(
    processUseCase *document.ProcessDocumentUseCase,
    updateUseCase *document.UpdateMetadataUseCase,
) *handlers.DocumentHandler {
    return handlers.NewDocumentHandler(processUseCase, updateUseCase)
}

// ProvideSearchHandler provides search handler
func ProvideSearchHandler(
    searchUseCase *document.SearchDocumentsUseCase,
) *handlers.SearchHandler {
    return handlers.NewSearchHandler(searchUseCase)
}

// ProvideClassificationHandler provides classification handler
func ProvideClassificationHandler(
    classifyUseCase *document.ClassifyDocumentUseCase,
) *handlers.ClassificationHandler {
    return handlers.NewClassificationHandler(classifyUseCase)
}

// ProvideHealthHandler provides health handler
func ProvideHealthHandler(
    storage ports.StorageService,
    search ports.SearchService,
) *handlers.HealthHandler {
    return handlers.NewHealthHandler(storage, search)
}
```

---

### Task Group 3: Wire Injection Sets

#### Task 3.1: Injection Sets
**File**: `internal/container/wire_sets.go`

```go
package container

import (
    "github.com/google/wire"
    "motion-index-fiber/internal/container/providers"
)

// ConfigSet provides configuration dependencies
var ConfigSet = wire.NewSet(
    providers.ProvideConfig,
    providers.ProvideServerConfig,
    providers.ProvideCloudConfig,
    providers.ProvideAIConfig,
)

// InfrastructureSet provides infrastructure dependencies
var InfrastructureSet = wire.NewSet(
    providers.ProvideDocumentRepository,
    providers.ProvideStorageService,
    providers.ProvideSearchService,
    providers.ProvideEventBus,
)

// AISet provides AI dependencies
var AISet = wire.NewSet(
    providers.ProvideAIRegistry,
    providers.ProvideClassificationService,
)

// UseCaseSet provides use case dependencies
var UseCaseSet = wire.NewSet(
    providers.ProvideProcessDocumentUseCase,
    providers.ProvideClassifyDocumentUseCase,
    providers.ProvideSearchDocumentsUseCase,
    providers.ProvideUpdateMetadataUseCase,
)

// HandlerSet provides HTTP handler dependencies
var HandlerSet = wire.NewSet(
    providers.ProvideDocumentHandler,
    providers.ProvideSearchHandler,
    providers.ProvideClassificationHandler,
    providers.ProvideHealthHandler,
)

// ApplicationSet combines all use case and handler sets
var ApplicationSet = wire.NewSet(
    UseCaseSet,
    HandlerSet,
)
```

---

### Task Group 4: Wire Initialization

#### Task 4.1: Wire Injector
**File**: `cmd/server/wire.go`

```go
//go:build wireinject
// +build wireinject

package main

import (
    "github.com/google/wire"
    "motion-index-fiber/internal/container"
)

// InitializeApplication creates and wires the entire application
func InitializeApplication() (*Application, func(), error) {
    wire.Build(
        container.ConfigSet,
        container.InfrastructureSet,
        container.AISet,
        container.ApplicationSet,
        NewApplication,
    )
    return nil, nil, nil
}
```

---

#### Task 4.2: Application Container
**File**: `cmd/server/application.go`

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"
    "os/signal"
    "syscall"

    "github.com/gofiber/fiber/v2"
    "motion-index-fiber/internal/configuration"
    "motion-index-fiber/internal/infrastructure/http/handlers"
    "motion-index-fiber/internal/infrastructure/http/router"
)

// Application represents the entire application
type Application struct {
    config   *configuration.Config
    handlers *handlers.Handlers
    app      *fiber.App
}

// NewApplication creates a new application instance
func NewApplication(
    config *configuration.Config,
    docHandler *handlers.DocumentHandler,
    searchHandler *handlers.SearchHandler,
    classificationHandler *handlers.ClassificationHandler,
    healthHandler *handlers.HealthHandler,
) *Application {
    // Create Fiber app
    app := fiber.New(fiber.Config{
        AppName:      "Motion-Index-Fiber",
        ServerHeader: "Fiber",
        ErrorHandler: customErrorHandler,
    })

    // Group handlers
    handlers := &handlers.Handlers{
        Document:       docHandler,
        Search:         searchHandler,
        Classification: classificationHandler,
        Health:         healthHandler,
    }

    // Setup routes
    router.SetupRoutes(app, handlers)

    return &Application{
        config:   config,
        handlers: handlers,
        app:      app,
    }
}

// Start starts the application
func (a *Application) Start() error {
    // Setup graceful shutdown
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

    // Start server in goroutine
    go func() {
        addr := fmt.Sprintf(":%s", a.config.Server.Port)
        log.Printf("Server starting on %s", addr)
        if err := a.app.Listen(addr); err != nil {
            log.Fatal(err)
        }
    }()

    // Wait for interrupt
    <-quit
    log.Println("Shutting down server...")

    // Graceful shutdown
    if err := a.app.Shutdown(); err != nil {
        return fmt.Errorf("server forced to shutdown: %w", err)
    }

    log.Println("Server exited")
    return nil
}

func customErrorHandler(c *fiber.Ctx, err error) error {
    // Error handling logic
    return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
        "error": err.Error(),
    })
}
```

---

#### Task 4.3: Updated Main
**File**: `cmd/server/main.go`

```go
package main

import (
    "log"
)

func main() {
    // Initialize application with Wire
    app, cleanup, err := InitializeApplication()
    if err != nil {
        log.Fatalf("Failed to initialize application: %v", err)
    }
    defer cleanup()

    // Start application
    if err := app.Start(); err != nil {
        log.Fatalf("Failed to start application: %v", err)
    }
}
```

**Result**: ~15 LOC (down from 242 LOC)

---

### Task Group 5: Wire Generation

#### Task 5.1: Generate Wire Code
```bash
cd cmd/server
wire
```

This generates `wire_gen.go` with all wiring logic.

#### Task 5.2: Makefile Target
**File**: `Makefile`

```makefile
.PHONY: wire
wire:
	@echo "Generating wire code..."
	cd cmd/server && wire
	@echo "Wire code generated successfully"

.PHONY: build
build: wire
	@echo "Building server..."
	go build -o bin/server cmd/server/main.go
	@echo "Build complete"
```

---

### Task Group 6: Test Container

#### Task 6.1: Test Wire Injector
**File**: `internal/container/wire_test.go`

```go
//go:build wireinject
// +build wireinject

package container

import (
    "github.com/google/wire"
    "motion-index-fiber/internal/application/usecases/document"
)

// InitializeTestContainer creates test container
func InitializeTestContainer() (*TestContainer, func(), error) {
    wire.Build(
        ConfigSet,
        InfrastructureSet,
        AISet,
        UseCaseSet,
        NewTestContainer,
    )
    return nil, nil, nil
}
```

---

#### Task 6.2: Test Container
**File**: `internal/container/test_container.go`

```go
package container

import (
    "motion-index-fiber/internal/application/usecases/document"
    "motion-index-fiber/internal/configuration"
)

// TestContainer provides dependencies for integration tests
type TestContainer struct {
    Config              *configuration.Config
    ProcessUseCase      *document.ProcessDocumentUseCase
    ClassifyUseCase     *document.ClassifyDocumentUseCase
    SearchUseCase       *document.SearchDocumentsUseCase
    UpdateMetadataUseCase *document.UpdateMetadataUseCase
}

// NewTestContainer creates a new test container
func NewTestContainer(
    config *configuration.Config,
    processUseCase *document.ProcessDocumentUseCase,
    classifyUseCase *document.ClassifyDocumentUseCase,
    searchUseCase *document.SearchDocumentsUseCase,
    updateMetadataUseCase *document.UpdateMetadataUseCase,
) *TestContainer {
    return &TestContainer{
        Config:              config,
        ProcessUseCase:      processUseCase,
        ClassifyUseCase:     classifyUseCase,
        SearchUseCase:       searchUseCase,
        UpdateMetadataUseCase: updateMetadataUseCase,
    }
}
```

---

## Files to Create

| File | Purpose | LOC |
|------|---------|-----|
| `container/providers/config.go` | Config providers | ~100 |
| `container/providers/infrastructure.go` | Infrastructure providers | ~150 |
| `container/providers/ai.go` | AI provider providers | ~120 |
| `container/providers/usecases.go` | Use case providers | ~100 |
| `container/providers/handlers.go` | Handler providers | ~80 |
| `container/wire_sets.go` | Wire injection sets | ~80 |
| `cmd/server/wire.go` | Wire injector | ~20 |
| `cmd/server/application.go` | Application container | ~150 |
| `cmd/server/main.go` | Main entry point | ~15 |
| `container/wire_test.go` | Test wire injector | ~20 |
| `container/test_container.go` | Test container | ~50 |
| `tools.go` | Build tools | ~10 |
| `Makefile` | Build automation | ~30 |

**Total**: ~13 files, ~925 LOC (new), ~242 LOC deleted (main.go wiring)

---

## Files to Delete/Refactor

### Files to Delete (~242 LOC)
- Most of `cmd/server/main.go` (242 LOC → 15 LOC)
- `internal/handlers/handlers.go` (manual handler initialization)

**Total Deleted**: ~242 LOC of manual wiring

---

## Testing Strategy

### Wire Validation
- Wire generates code at compile time
- Any missing dependencies = compile error
- Graph validation ensures all dependencies satisfied

### Integration Tests
- Use test container for integration tests
- Wire generates test dependencies
- No manual mock setup

### Example Test
```go
func TestIntegration_ProcessDocument(t *testing.T) {
    // Given: Test container with real dependencies
    container, cleanup, err := InitializeTestContainer()
    require.NoError(t, err)
    defer cleanup()

    // When: Use case is executed
    resp, err := container.ProcessUseCase.Execute(ctx, req)

    // Then: Result is as expected
    assert.NoError(t, err)
    assert.NotNil(t, resp)
}
```

---

## Acceptance Criteria

- [ ] Wire installed and configured
- [ ] All provider functions created
- [ ] Injection sets defined
- [ ] wire_gen.go generated successfully
- [ ] main.go reduced from 242 LOC to ~15 LOC
- [ ] Application starts with wire-generated code
- [ ] Test container working
- [ ] All integration tests passing
- [ ] No manual wiring remaining

---

## Success Metrics

- main.go LOC: 242 → 15 (94% reduction)
- Compile-time dependency validation: 100%
- Manual wiring: 0%
- Wire generation time: <1 second

---

## Migration Strategy

### Step 1: Setup Wire
1. Install Wire
2. Add tools.go
3. Create container package structure

### Step 2: Create Providers
1. Start with config providers
2. Add infrastructure providers
3. Add use case providers
4. Add handler providers

### Step 3: Create Injection Sets
1. Group providers by concern
2. Define wire sets
3. Test each set independently

### Step 4: Wire Application
1. Create wire.go injector
2. Run `wire` to generate code
3. Update main.go to use generated code
4. Test application startup

### Step 5: Create Test Container
1. Create test wire injector
2. Generate test container
3. Update integration tests

---

## Common Issues & Solutions

### Issue 1: Circular Dependency
**Error**: `wire: found circular dependency`
**Solution**: Refactor to break circular dependency (extract interface)

### Issue 2: Missing Provider
**Error**: `no provider found for X`
**Solution**: Add provider function for X

### Issue 3: Ambiguous Provider
**Error**: `multiple providers for X`
**Solution**: Use wire.Bind to disambiguate

---

## Dependencies

- Phase 1 complete (domain layer)
- Phase 2 complete (application layer)
- Phase 3 complete (configuration)
- Phase 4 complete (infrastructure)
- Phase 5 complete (HTTP handlers)
- Google Wire installed

---

## Rollback Strategy

If Wire causes issues:
1. Keep old main.go as main_manual.go
2. Revert to manual wiring if needed
3. Fix Wire issues and try again

---

**Phase Status**: ⚪ Not Started
**Dependencies**: Phases 1-5 complete
**Next Phase**: Phase 7 (Documentation & Testing)
