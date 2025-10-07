# Repository Guidelines

## Project Structure & Module Organization
`motion-index-fiber/` encapsulates the Go Fiber API. Entry points live under `cmd/` (notably `cmd/server`), runtime configuration sits in `.env`, and compiled binaries belong in `bin/`. Business logic and HTTP handlers reside in `internal/` (`internal/http`, `internal/service`, etc.), reusable clients stay in `pkg/`, and integration harnesses are under `test/`. Supporting docs, samples, and diagrams land in `docs/` and `deployments/`.

## Build, Test, and Development Commands
- `go run cmd/server/main.go` starts the API with your local `.env`.
- `go build -o bin/server cmd/server/main.go` emits the production binary.
- `go test ./...` executes unit + integration suites (scope with `-run` as needed).
- `GO_ENV=test go test ./internal/...` narrows focus to core services.
Use `air` or your preferred watcher in `cmd/server` only after ensuring dependencies are tidy.

## Coding Style & Naming Conventions
Format Go code with `gofmt` or `goimports` before committing. Prefer camelCase for identifiers; export only what the package API demands. Configuration constants stay uppercase only when immutable. Keep HTTP handler files under `internal/http` named after their route group (e.g., `search_handler.go`). Maintain structured logging via the shared logger utilities in `pkg/`.

## Testing Guidelines
Mirror production packages with `_test.go` files. Mock external providers (Spaces, Elasticsearch, Supabase) using fixtures from `test/` to avoid remote calls. For integration scenarios, seed data through the scripts in `deployments/` and document any manual prerequisites. Validate new search or indexing behavior with assertions on relevance ordering and failure fallbacks.

## Commit & Pull Request Guidelines
Write concise, present-tense commit subjects ("add asset sync fallback"). Each PR should describe the change, reference relevant issues, attach screenshots for API responses when payloads change, and include test output (`go test`). Highlight migrations, new feature flags, or env vars so Ops can adjust deployment scripts.

## Security & Configuration Tips
Initialize environments by copying `.env.example` to `.env`, then populate secrets locally—never commit credentials. Rotate shared keys when touched and update `deployments/README.md`. Use the helper scripts in `deployments/` to verify Elasticsearch indices, Spaces buckets, and Supabase links before merging infrastructure-affecting changes.
