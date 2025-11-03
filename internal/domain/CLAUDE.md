# Domain Layer

The domain layer contains the core business logic and rules of the Motion-Index-Fiber application. This layer is completely independent of external concerns like databases, HTTP, or AI providers.

## Principles

- **Pure Business Logic**: No infrastructure dependencies
- **Rich Domain Models**: Behavior-focused entities and aggregates
- **Ubiquitous Language**: Use legal terminology consistently
- **Invariant Protection**: Aggregates enforce business rules
- **Domain Events**: Communicate state changes

## Structure

```
domain/
├── document/          # Document aggregate (primary entity)
├── legal/            # Legal domain concepts (cases, courts, parties)
├── classification/   # Classification domain service
└── errors/           # Domain-specific errors
```

## Aggregates

### Document Aggregate
- **Root Entity**: Document
- **Value Objects**: FilePath, Hash, Classification, S3URI
- **Invariants**:
  - Document must have unique ID
  - Classification confidence must be 0.0-1.0
  - File path must be valid storage path
- **Events**:
  - DocumentCreated
  - DocumentClassified
  - DocumentIndexed
  - MetadataUpdated

### Legal Aggregate
- **Entities**: Case, Party, Attorney
- **Value Objects**: Court, Charge, Authority
- **Invariants**:
  - Case must have valid case number format
  - Party roles must be valid legal roles
  - Attorneys must have valid bar numbers

## Domain Services

### Classifier Interface
Defines the contract for document classification without depending on specific AI providers.

```go
type Classifier interface {
    Classify(ctx context.Context, document *Document) (*Classification, error)
    GetSupportedCategories() []string
    GetProviderName() string
}
```

## Usage Guidelines

1. **Never import infrastructure packages** in domain layer
2. **Define interfaces here** for external dependencies
3. **Keep logic pure** - no side effects in entities
4. **Use value objects** for complex concepts (not primitives)
5. **Emit domain events** for important state changes

## Testing

- Unit test all domain logic
- No mocks needed (pure functions)
- Test invariant enforcement
- Test domain event emission

## Examples

### Creating a Document

```go
// Good: Using aggregate factory
doc := document.NewDocument(
    document.WithID(id),
    document.WithFileName(name),
    document.WithContent(content),
)

// Bad: Direct struct creation (bypasses invariants)
doc := &Document{ID: id, FileName: name} // ❌
```

### Applying Classification

```go
// Good: Through aggregate method
err := doc.ApplyClassification(classification)
if err != nil {
    return domainerrors.ErrInvalidClassification
}

// Bad: Direct field assignment
doc.Classification = classification // ❌ Bypasses validation
```

## Dependencies

This layer should have **ZERO** external dependencies except:
- Go standard library
- `context` package for context passing
- `time` package for timestamps

## Anti-Patterns to Avoid

1. **Anemic Domain Models**: Don't create models with only getters/setters
2. **Infrastructure Leakage**: Never import database, HTTP, or external service packages
3. **God Objects**: Keep aggregates focused and cohesive
4. **Primitive Obsession**: Use value objects instead of primitive types
5. **Missing Invariants**: Always validate business rules in aggregates

## Migration Notes

When migrating existing code:
1. Extract business logic from handlers/services
2. Create value objects for complex types
3. Add invariant checks to aggregates
4. Emit domain events for state changes
5. Write tests for all domain logic
