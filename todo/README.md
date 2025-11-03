# SOLID Redesign Implementation Plan

## Project Overview

**Motion-Index-Fiber** is a high-performance legal document processing API for California public defenders. This redesign transforms the architecture from transaction script to clean, layered SOLID architecture using Domain-Driven Design and Hexagonal Architecture patterns.

## Current State Analysis

### Codebase Metrics
- **Files**: 123 Go files
- **Lines of Code**: 38,228 LOC
- **Handler Code**: 4,180 LOC in `/internal/handlers/`
- **Test Coverage**: ~18.8%
- **Architecture**: Mixed concerns, tightly coupled

### Key Issues
1. **Business logic in HTTP handlers** - 4,180 LOC mixing HTTP and domain logic
2. **Tight coupling** - Direct dependencies on concrete implementations
3. **Monolithic configuration** - Single massive config struct
4. **Hard-coded AI providers** - No plugin system
5. **Poor testability** - Hard to mock, low coverage
6. **No clear domain model** - Anemic models with getters/setters

## Target State

### Architecture Goals
- **Hexagonal Architecture** (Ports & Adapters)
- **Domain-Driven Design** with rich domain models
- **CQRS** for read/write separation
- **Dependency Injection** using Wire
- **Plugin System** for AI providers

### Success Metrics
| Metric | Current | Target | Improvement |
|--------|---------|--------|-------------|
| Handler LOC | 4,180 | <1,000 | 76% reduction |
| Test Coverage | 18.8% | >80% | 4.4x increase |
| Files | 123 | ~200 | +77 (clean separation) |
| Avg Handler LOC | ~700 | <100 | 86% reduction |
| Business Logic in Domain | 0% | 100% | Complete separation |

## SOLID Principles Applied

### Single Responsibility Principle (SRP)
- **Before**: Handlers do everything (HTTP, validation, business logic, persistence)
- **After**: Separated layers - HTTP handlers only handle HTTP, use cases handle orchestration, domain handles business rules

### Open/Closed Principle (OCP)
- **Before**: Hard-coded AI providers, can't add new ones without modifying code
- **After**: Plugin system for AI providers, extend by adding new plugins

### Liskov Substitution Principle (LSP)
- **Before**: Concrete implementations everywhere
- **After**: Interface-based design, any implementation can substitute another

### Interface Segregation Principle (ISP)
- **Before**: Large, monolithic interfaces
- **After**: Small, focused interfaces (one per use case)

### Dependency Inversion Principle (DIP)
- **Before**: High-level modules depend on low-level modules
- **After**: Both depend on abstractions (ports/interfaces)

## Implementation Phases

### Phase 1: Domain Layer Foundation
**Status**: 🟡 In Progress
**File**: [`phase-01-domain-layer.md`](./phase-01-domain-layer.md)

Create pure domain model with aggregates, entities, and value objects.

**Key Deliverables**:
- Document aggregate with business logic
- Legal domain entities (Case, Court, Party, Attorney)
- Classification domain service
- Domain events
- 100% test coverage

### Phase 2: Application Layer (Use Cases)
**Status**: 🟢 Completed
**File**: [`phase-02-application-layer.md`](./phase-02-application-layer.md)

Create use cases that orchestrate domain objects.

**Key Deliverables**:
- Application ports (interfaces)
- DTOs for all operations
- 8 core use cases
- Application-level validation
- Use case tests with mocks

### Phase 3: Configuration Management
**Status**: 🟢 Completed
**File**: [`phase-03-configuration.md`](./phase-03-configuration.md)

Split monolithic config into cohesive modules using builder pattern.

**Key Deliverables**:
- Config builder pattern
- Config modules by concern
- Environment loader
- Validation
- Backward compatibility

### Phase 4: Infrastructure Adapters
**Status**: 🟢 Completed
**File**: [`phase-04-infrastructure.md`](./phase-04-infrastructure.md)

Implement adapters for external systems (databases, AI, storage).

**Key Deliverables**:
- ✅ Search service adapter (wraps pkg/search)
- ✅ Storage service adapter (wraps pkg/storage)
- ✅ AI classification adapter (wraps pkg/processing/classifier)
- ✅ Error translation layer
- ✅ Event bus implementation
- ✅ Comprehensive adapter tests (40/40 passing)

### Phase 5: HTTP Layer Refactor
**Status**: ⚪ Not Started
**File**: [`phase-05-http-layer.md`](./phase-05-http-layer.md)

Refactor handlers from 4,180 LOC to <1,000 LOC.

**Key Deliverables**:
- Thin handlers using use cases
- Error translator middleware
- Response presenters
- <100 LOC per handler
- HTTP integration tests

### Phase 6: Dependency Injection
**Status**: ⚪ Not Started
**File**: [`phase-06-dependency-injection.md`](./phase-06-dependency-injection.md)

Wire everything together with compile-time DI using Google Wire.

**Key Deliverables**:
- Wire setup
- Provider functions
- Injection sets
- Updated main.go
- Test container

### Phase 7: Documentation & Testing
**Status**: ⚪ Not Started
**File**: [`phase-07-documentation-testing.md`](./phase-07-documentation-testing.md)

Complete documentation and achieve 80%+ test coverage.

**Key Deliverables**:
- 4 Architecture Decision Records
- Architecture diagrams
- Migration guide
- 80%+ test coverage
- Performance benchmarks

### Phase 8: Migration & Cleanup
**Status**: ⚪ Not Started
**File**: [`phase-08-migration-cleanup.md`](./phase-08-migration-cleanup.md)

Complete migration and remove old code.

**Key Deliverables**:
- Feature flags
- A/B testing
- Deprecation
- Old code removal
- Final regression tests

## Phase Dependencies

```
Phase 1 (Domain Layer)
    ↓
Phase 2 (Application Layer)
    ↓
Phase 3 (Configuration) ──→ Phase 4 (Infrastructure)
    ↓                             ↓
    └──────────→ Phase 5 (HTTP Layer)
                      ↓
                Phase 6 (DI)
                      ↓
                Phase 7 (Docs & Testing)
                      ↓
                Phase 8 (Migration)
```

## Quick Start

1. **Read this README** to understand the overall plan
2. **Start with Phase 1** - Domain layer foundation
3. **Follow phases in order** - Each builds on the previous
4. **Check off tasks** as you complete them
5. **Update phase status** in this README

## Progress Tracking

Use checkboxes in each phase file to track progress:
- [ ] Task not started
- [x] Task completed

Update phase status in this README:
- ⚪ Not Started
- 🟡 In Progress
- 🟢 Completed

## Code Organization

### New Directory Structure

```
Motion-API/
├── internal/
│   ├── domain/              # Pure business logic (Phase 1)
│   │   ├── document/
│   │   ├── legal/
│   │   ├── classification/
│   │   └── errors/
│   ├── application/         # Use cases (Phase 2)
│   │   ├── usecases/
│   │   ├── dto/
│   │   ├── ports/
│   │   └── validation/
│   ├── configuration/       # Config management (Phase 3)
│   │   ├── server/
│   │   ├── cloud/
│   │   ├── ai/
│   │   └── processing/
│   ├── infrastructure/      # External adapters (Phase 4)
│   │   ├── persistence/
│   │   ├── ai/
│   │   ├── extraction/
│   │   └── http/
│   └── container/           # DI container (Phase 6)
└── docs/                    # Documentation (Phase 7)
    ├── architecture/
    └── migration/
```

## Testing Strategy

### Test Coverage Goals
- **Domain Layer**: 100% coverage (pure business logic)
- **Application Layer**: >90% coverage (use cases)
- **Infrastructure Layer**: >70% coverage (adapters)
- **Overall**: >80% coverage

### Test Types
1. **Unit Tests**: Fast, isolated tests with mocks
2. **Integration Tests**: Test interactions between components
3. **E2E Tests**: Full API request → response
4. **Performance Tests**: Load testing, benchmarks

## Risk Management

### High-Risk Areas
1. **Data Migration**: Existing documents in OpenSearch
   - **Mitigation**: Backward compatibility, gradual rollout
2. **AI Provider Changes**: Switching from hard-coded to plugins
   - **Mitigation**: Feature flags, A/B testing
3. **Configuration Changes**: Breaking existing deployments
   - **Mitigation**: Backward compatibility layer

### Rollback Strategy
Each phase includes a rollback strategy. Feature flags allow instant rollback if issues arise.

## Getting Help

### Resources
- **CLAUDE.md files**: Detailed guidance in each package
- **Phase files**: Step-by-step implementation plans
- **Architecture diagrams**: Visual understanding (Phase 7)
- **ADRs**: Decision rationale (Phase 7)

### Team Communication
- **Daily standups**: Discuss blockers
- **Code reviews**: Every PR reviewed before merge
- **Pairing**: Complex refactors done in pairs

## Success Criteria

### Must Have (Definition of Done)
- [ ] All 8 phases completed
- [ ] Test coverage >80%
- [ ] All tests passing
- [ ] Documentation complete
- [ ] Performance benchmarks met
- [ ] Zero regressions

### Nice to Have
- [ ] Test coverage >90%
- [ ] Performance improvement over baseline
- [ ] Developer training completed
- [ ] Migration guide validated

## Notes

- **Incremental Migration**: Each phase can be merged independently
- **Feature Flags**: Use flags for gradual rollout
- **Backward Compatibility**: Maintain during migration
- **Documentation**: Update as you go, not at the end
- **Testing**: Write tests BEFORE implementation (TDD)

## Questions?

If you have questions about:
- **Architecture**: See `docs/architecture/` (after Phase 7)
- **Specific phase**: See that phase's markdown file
- **Implementation**: See CLAUDE.md in relevant package
- **Testing**: See phase files for testing strategies

---

**Last Updated**: 2025-10-28
**Status**: Phase 4 completed, ready for Phase 5
**Next Milestone**: HTTP Layer Refactor (Phase 5)
