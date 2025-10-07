# Repository Guidelines

## Project Structure & Module Organization
- `motion-index-fiber/` contains the Go Fiber API. Entrypoints live under `cmd/` (notably `cmd/server`), runtime config is in `.env`, and compiled binaries go to `bin/`.
- Business logic and HTTP handlers sit in `internal/` (`internal/http`, `internal/service`, etc.). Name handler files by route group (e.g., `internal/http/search_handler.go`).
- Reusable clients and utilities stay in `pkg/`. Integration harnesses live under `test/`. Supporting docs and deployment scripts are in `docs/` and `deployments/`.

## Build, Test, and Development Commands
- `go run cmd/server/main.go` — start the API using your local `.env`.
- `go build -o bin/server cmd/server/main.go` — build the production binary.
- `go test ./...` — run unit and integration tests.
- `GO_ENV=test go test ./internal/...` — focus tests on core services.
- Optional: use `air` in `cmd/server` for live reload after `go mod tidy`.

## Coding Style & Naming Conventions
- Format code with `gofmt` or `goimports` before committing. Prefer camelCase; export only necessary identifiers. Use ALL_CAPS only for immutable configuration constants.
- Keep HTTP handler files under `internal/http` and align names to routes. Use structured logging via shared logger utilities in `pkg/`.

## Testing Guidelines
- Mirror production packages with `_test.go` files. Mock external providers (Spaces, Elasticsearch, Supabase) using fixtures from `test/` to avoid remote calls.
- For integration, seed via scripts in `deployments/` and document manual prerequisites in `deployments/README.md`.
- Validate new search/indexing behavior with assertions on relevance ordering and failure fallbacks. Run `go test` locally before opening a PR.

## Commit & Pull Request Guidelines
- Commit subjects: concise, present tense (e.g., "add asset sync fallback").
- PRs: describe the change, link related issues, include API response screenshots when payloads change, and attach `go test` output.
- Call out migrations, new feature flags, or env vars so Ops can update deployment scripts.

## Security & Configuration Tips
- Initialize by copying `.env.example` to `.env`; never commit credentials. Rotate shared keys when touched and update `deployments/README.md`.
- Use `deployments/` helper scripts to verify Elasticsearch indices, Spaces buckets, and Supabase links before merging infrastructure-affecting changes.

## Agent-Specific Instructions
- Follow this guide for files within this repository. Keep changes minimal and focused; avoid renames or structural moves unless requested. Update docs when behavior changes.

