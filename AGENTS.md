# Repository Guidelines

## Project Structure & Module Organization
- Source: package `shttp` at repo root (`server.go`, `router.go`, `middleware.go`, `error.go`).
- Tests: root-level `*_test.go` (e.g., `server_test.go`, `router_test.go`).
- Examples: `examples/<topic>/main.go` (see `examples/README.md`).
- Docs: `README.md` (usage), `SERVER.md` (architecture).

## Build, Test, and Development Commands
- `go build ./...`: builds all packages.
- `go test ./...`: runs tests.
- `go test -race -cover ./...`: race checks and coverage summary.
- `go vet ./...`: static checks.
- `go fmt ./...`: format code before committing.
- Run examples: `cd examples/quickstart && go run main.go`.

## Coding Style & Naming Conventions
- Formatting: enforce `go fmt` (standard Go formatting).
- Docs: add package and exported symbol comments (`// Foo ...`).
- Names: exported APIs in PascalCase; unexported in lowerCamelCase.
- Errors: return `error` (don’t panic); use `HTTPError` via `NewHTTPError` when mapping to status codes.
- Handlers: prefer the project signature `func(ctx context.Context, w http.ResponseWriter, r *http.Request) error` for new endpoints/middleware.

## Testing Guidelines
- Framework: standard `testing` + `net/http/httptest`.
- Conventions: files end with `_test.go`; tests named `TestXxx`; favor table-driven tests.
- Coverage: aim to cover new logic (happy-path and failure cases).
- Running: `go test -race -cover ./...`; keep tests hermetic (avoid real network where possible).

## Commit & Pull Request Guidelines
- Commits: concise, imperative subject (e.g., `add router param parsing`), include rationale in body when needed.
- Scope: keep changes focused; format and refactors in separate commits.
- PRs: clear description, linked issues, test updates, and example or docs updates when public APIs change. Include before/after snippets or curl examples if relevant.

## Security & Configuration Tips
- Avoid logging secrets or request bodies containing credentials.
- Validate and bound timeouts; keep recovery middleware enabled in examples.
- Prefer context-aware operations and propagate `context.Context` correctly.

## Architecture Overview
- See `SERVER.md` for layers (Server, Router, Handler/Middleware) and composition patterns. Use existing patterns when extending routing or middleware.

