# Repository Guidelines

## Project Structure & Module Organization
- `motion-index-fiber/` holds the Go Fiber API; start points sit under `cmd/` (`cmd/server/main.go` is the HTTP bootstrap).
- Business logic and handlers live in `internal/` (`internal/http` for route groups, `internal/service` for domain logic). Keep handler files aligned to their route group name.
- Shared clients, logging, and utilities belong in `pkg/`.
- Integration harnesses and fixtures reside in `test/`; documentation and deployment helpers live in `docs/` and `deployments/`.
- Produced binaries land in `bin/`; never commit them.

## Build, Test, and Development Commands
- `go run cmd/server/main.go` — boot the API using the current `.env`.
- `go build -o bin/server cmd/server/main.go` — compile the production binary.
- `go test ./...` — execute the full unit and integration suite.
- `GO_ENV=test go test ./internal/...` — scope testing to core services with the test environment.
- Optional: from `cmd/server`, run `air` for live reload once dependencies are tidy.

## Coding Style & Naming Conventions
- Format Go code with `gofmt` or `goimports` before pushing; keep diffs clean.
- Prefer camelCase for locals and exported identifiers; reserve ALL_CAPS for immutable configuration.
- Name HTTP handler files after their route group (e.g., `internal/http/search_handler.go`).
- Use structured logging via shared utilities in `pkg/`; avoid ad-hoc `fmt.Println`.

## Testing Guidelines
- Mirror packages with `_test.go` files; rely on fixtures in `test/` to mock external providers.
- Seed integration scenarios through scripts in `deployments/`; document any manual prerequisites in `deployments/README.md`.
- Validate search and indexing paths with relevance assertions and fallback checks.
- Run `go test ./...` locally before opening a PR; investigate flaky tests before retrying.

## Commit & Pull Request Guidelines
- Commit subjects stay concise, present tense (e.g., `add asset sync fallback`).
- PRs should summarize the change, link issues, attach `go test` output, and include API response screenshots when payloads shift.
- Call out new env vars, feature flags, or migrations so Ops can adjust deployment scripts.

## Security & Configuration Tips
- Start with `cp .env.example .env`; never commit credentials. Rotate shared keys when touched.
- Use `deployments/` scripts to verify Elasticsearch indices, Spaces buckets, and Supabase links before merging infra changes.

## Agent-Specific Instructions
- Keep edits narrow and avoid unnecessary renames or structural moves.
- Follow repository conventions above, add succinct comments only when essential, and update docs when behavior changes.
