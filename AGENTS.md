# Repository Guidelines

## Project Structure & Module Organization
`ast-cli` is a Go CLI (`module github.com/checkmarx/ast-cli`) with entrypoint at `cmd/main.go`.

```text
cmd/main.go                         # bootstrap + dependency wiring
internal/
  commands/                         # Cobra commands (+ .scripts test runners)
  commands/asca/                    # ASCA-specific commands
  commands/util/                    # shared command helpers
  services/                         # business logic layer
  services/realtimeengine/          # iac, oss, containers, secrets engines
  wrappers/                         # API integrations (HTTP + interfaces)
  wrappers/mock/                    # test doubles
  wrappers/grpcs/                   # ASCA protobuf/gRPC clients
  params/                           # flags, keys, env bindings
  constants/                        # errors, exit codes, feature flags
  logger/                           # logging utilities
test/
  integration/                      # integration suite (`-tags integration`)
  cleandata/                        # cleanup tests/helpers
docs/                               # contribution + code of conduct
```

Architecture pattern: `cmd/main.go` wires wrappers/services into Cobra commands (dependency injection). Keep external/API calls behind `internal/wrappers` and avoid global state.

## Extended Directory Index
- `cmd/`: executable entry (`main.go`) and startup wiring.
- `internal/commands/`: top-level Cobra commands (`scan`, `project`, `result`, `auth`, `chat`, etc.).
- `internal/commands/util/`: shared CLI helpers (env/config, completion, import, tenant, version, remediation, usercount).
- `internal/commands/asca/`: ASCA command logic and engine bootstrap.
- `internal/commands/asca/ascaconfig/`: OS/arch-specific ASCA runtime config.
- `internal/commands/scarealtime/`: SCA realtime command flow, structs, and helpers.
- `internal/commands/data/`: embedded sample payloads/test assets used by command tests.
- `internal/services/`: core domain services (projects, policy, tags, metadata, export, applications).
- `internal/services/realtimeengine/iacrealtime/`: IaC realtime scanner and parsers.
- `internal/services/realtimeengine/ossrealtime/`: OSS realtime scanner and cache layer.
- `internal/services/realtimeengine/containersrealtime/`: container realtime scan orchestration.
- `internal/services/realtimeengine/secretsrealtime/`: secret-detection realtime scanner.
- `internal/wrappers/`: API integration layer (HTTP wrappers + interfaces per domain).
- `internal/wrappers/mock/`: mock wrappers used by unit tests.
- `internal/wrappers/grpcs/` and `internal/wrappers/grpcs/protos/`: gRPC clients and ASCA protobuf definitions.
- `internal/wrappers/kerberos/` and `internal/wrappers/ntlm/`: enterprise proxy auth support.
- `internal/params/`: canonical CLI keys, flags, filters, and env binding constants.
- `internal/constants/`: shared constants (errors, feature flags, exit codes, file extensions).
- `test/integration/`: integration tests and fixture-driven end-to-end coverage.
- `test/integration/data/`: integration test fixtures (archives, SARIF, manifests, Dockerfiles).
- `test/cleandata/`: cleanup tests/helpers for integration environments.
- `docs/`: contributor and conduct documentation.
- `.github/workflows/`: CI checks, release automation, security and PR policy workflows.
- `repocontext/`: auxiliary AI context docs (`CLAUDE.md`).

## Build, Test, and Development Commands
- `make` or `make vet`: `go fmt ./...` then `go vet ./...` (default target).
- `make build`: vet pipeline + binary build (`bin/cx.exe`).
- `make lint`: run `golangci-lint -c .golangci.yml`.
- `go build -o ./bin/cx ./cmd`: local build.
- `go test ./...`: unit tests.
- `bash internal/commands/.scripts/up.sh`: CI-style unit run with coverage file.
- `bash internal/commands/.scripts/integration_up.sh`: integration run (Docker + env/secrets required).

## Coding Style & Naming Conventions
Use Go `1.24.11` (`go.mod`). Always run `gofmt` and `goimports`. Follow Go idioms: lowercase package names, `CamelCase` exported symbols, descriptive file names (existing command files include hyphenated names such as `pre-receive.go`).

Lint rules are strict (`.golangci.yml`): `errcheck`, `staticcheck`, `revive`, `gocyclo` (15), `funlen` (200 lines/100 statements), `mnd`, `goimports`, `dupl`, `depguard`, and more.

## Testing Guidelines
Place tests near source as `*_test.go`; prefer table-driven tests. Use wrapper mocks for unit tests; do not call real APIs in unit tests. Integration tests live under `test/integration` and use the `integration` build tag.

CI coverage gates:
- Unit coverage: `>= 77.7%`
- Integration coverage: `>= 75%`

## Commit & Pull Request Guidelines
Use concise, imperative commit subjects and include ticket context when relevant (example: `Support for SCA Delta Scans (AST-130110) (#1417)`).

PR requirements from repo workflows/templates:
- Link related issue/ticket.
- Add tests/docs/help updates for behavior changes.
- Pass all checks (tests, lint, security scans).
- PR title format: CamelCase and ends with `(AST-12345)`.
- Branch prefix: `bug/`, `fix/`, `feature/`, or `other/`.

## Agent-Specific Notes
For quickest orientation, start with:
1. `cmd/main.go` (wiring and dependency graph)
2. `internal/commands/root.go` (command tree entry)
3. `internal/params/envs.go` and `internal/params/keys.go` (config surface)
4. `.golangci.yml` and `.github/workflows/ci-tests.yml` (quality gates)
