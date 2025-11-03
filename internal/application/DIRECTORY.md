# Application Layer

```
application/
├── dto/          # Data transfer objects (requests, responses)
├── ports/        # Interfaces for repositories/services/event bus
├── usecase/      # Use case interactors orchestrating domain objects
└── validation/   # DTO validation helpers
```

Each use case lives under `usecase/` in a package named for its concern (e.g., `usecase/document`). Ports are defined once under `ports/` and shared across use cases to keep infrastructure details out of the application core.
