# Repository Guidelines

## Project Structure & Module Organization
The CLI entrypoint is `cmd/main.go`. Core implementation lives under `internal/`:
- `internal/commands`: Cobra command definitions and command-level tests.
- `internal/services`: business logic and realtime engine implementations.
- `internal/wrappers`: API/client wrappers, protocol adapters, and mocks.
- `internal/params`, `internal/constants`, `internal/logger`: shared config and utilities.

Tests are split between package-level unit tests (`*_test.go` next to source) and integration suites in `test/integration`. Cleanup helpers live in `test/cleandata`. Contributor docs are under `docs/`, and CI rules are in `.github/workflows/`.

## Build, Test, and Development Commands
- `make fmt`: run `go fmt ./...`.
- `make vet`: format + `go vet ./...` (default `make` target).
- `make build`: vet + build binary at `bin/cx.exe`.
- `make lint`: run `golangci-lint` with `.golangci.yml`.
- `go build -o ./bin/cx ./cmd`: cross-platform local build.
- `go test ./...`: quick local test pass.
- `bash internal/commands/.scripts/up.sh`: CI-like unit run with coverage output (`cover.out`).
- `bash internal/commands/.scripts/integration_up.sh`: integration tests (requires Docker, secrets, and integration env vars).

## Coding Style & Naming Conventions
Use Go `1.24.x` (see `go.mod`). Always run `gofmt`/`goimports` before opening a PR. Follow Go naming idioms: lowercase package names, `CamelCase` exported symbols, descriptive file names (existing command files often use hyphenated names like `pre-receive.go`). Keep complexity manageable (`gocyclo`, `funlen`, `errcheck`, `staticcheck`, and `revive` are enforced).

## Testing Guidelines
Place tests close to implementation and name files `*_test.go`. Use table-driven tests where practical. Run unit tests locally before push; run integration tests when changing wrappers, auth flows, or network-dependent commands. CI enforces coverage floors: unit pipeline checks `77.7%`, integration pipeline checks `75%`.

## Commit & Pull Request Guidelines
Current history favors concise, imperative commit subjects with ticket/PR context, e.g. `Support for SCA Delta Scans (AST-130110) (#1417)`. Open an issue first for significant work, then link it in the PR. PR title and branch are linted: title must be CamelCase and end with `(AST-12345)`; branch must start with `bug/`, `fix/`, `feature/`, or `other/`. Complete the PR checklist, include tests/docs/help updates, and ensure all GitHub checks pass.

## Security & Configuration Tips
Use environment variables for credentials (`CX_*`, proxy, SCM tokens) and never commit secrets or test credentials. Validate changes against CI/security workflows before release-related work.
