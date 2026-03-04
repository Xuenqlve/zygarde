# Repository Guidelines

## Project Structure & Module Organization
The Go module `github.com/xuenqlve/zygarde` keeps the CLI entrypoint in `cmd/main.go`, while feature code lives under `internal/` (e.g., `internal/config` for configuration loaders, `internal/data_source` for pluggable data sources, and `internal/log` for Zerolog-based logging helpers). Shared building blocks intended for reuse sit in `pkg/`, mirroring their internal counterparts when they need to be imported by external consumers. Place new runtime assets or templates beside the modules that consume them; keep experimental code in dedicated feature branches until it stabilizes.

## Build, Test, and Development Commands
Run `go build ./...` before submitting to confirm the entire module compiles. Use `go run ./cmd --config ./configs/dev.yaml` while iterating; adjust the config path to match your environment. Execute `go test ./...` for unit coverage and `go test -race ./...` when touching concurrency or shared state. `go fmt ./...` and `go vet ./...` should be part of your local pre-flight checks.

## Coding Style & Naming Conventions
Follow idiomatic Go: tabs for indentation, UpperCamelCase for exported types/functions, and lowerCamelCase for private identifiers. Always format code with `gofmt` or `goimports`, and keep imports grouped standard-library first, third-party second, local modules last. Configuration keys mirror their struct tags (see `DataSourceConfig`), so keep new keys lowercase with hyphen-separated words. Align log messages with the structured Zerolog pattern used in `internal/log`—verbs first, context via fields.

## Testing Guidelines
Add `_test.go` files alongside the code under test, using table-driven tests where inputs vary. Prefer the standard `testing` package and cover error paths, especially around plugin registration and configuration parsing. Include representative configuration fixtures under `testdata/` when needed, and ensure `go test ./...` passes without requiring external services. Aim for meaningful assertions rather than raw error checks to prevent regressions in orchestration flows.

## Commit & Pull Request Guidelines
Commits in this repository are short, lowercase summaries (e.g., "data source"). Keep each commit focused; amend instead of stacking fixups. Pull requests should explain the problem, highlight touched modules, and list manual and automated checks run (`go test`, `go build`). Link related issues, attach configuration samples or screenshots if UX is affected, and note any follow-up work so reviewers understand remaining risks.

## Configuration Tips
Command-line flags are parsed in `internal/config`; always document new flags and update sample configs. Avoid committing secrets—reference local `.env` paths instead—and prefer default-safe values that let `go run` succeed with minimal setup.
