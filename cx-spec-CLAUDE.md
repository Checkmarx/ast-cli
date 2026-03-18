# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**Checkmarx AST CLI** is a standalone command-line tool for security scanning built in Go. It provides a comprehensive CLI for managing scans, authentication, results analysis, and various security operations (SAST, SCA, DAST, IaC, Containers, etc.) through the Checkmarx One platform.

## Key Commands

### Building
- **Full build**: `make build` - Runs fmt, vet, and builds executable to `bin/cx.exe`
- **Format only**: `make fmt` - Runs `go fmt ./...`
- **Vet only**: `make vet` - Runs `go vet ./...` (depends on fmt)
- **Build only**: Skip other checks with `go build -o bin/cx.exe ./cmd`

### Testing
- **Unit tests**: `go test ./...` - Runs all unit tests (tests without `//go:build integration` tag)
- **Integration tests**: `go test -tags integration ./...` - Requires environment credentials (CX_AST_USERNAME, CX_AST_PASSWORD)
- **Specific test file**: `go test -v ./internal/commands/ -run TestAuthValidate`
- **All tests with coverage**: `go test -v -tags integration -coverprofile=coverage.out ./...`

**Note**: Integration tests in `test/integration/` use `//go:build integration` build tag and require valid AST credentials.

### Linting
- **Lint**: `make lint` - Runs golangci-lint with `.golangci.yml` config (runs fmt first)
- **Manual lint**: `golangci-lint run -c .golangci.yml`

## Project Structure

```
cmd/
  └─ main.go                 # Entry point, initializes all wrappers and creates CLI
internal/
  ├─ commands/              # Command definitions and handlers (scan, auth, result, etc.)
  │  ├─ util/              # Utility functions (printer, helpers)
  │  ├─ dast/              # DAST-specific command handlers
  │  ├─ scarealtime/       # SCA real-time engine commands
  │  └─ policymanagement/  # Policy management commands
  ├─ services/             # Business logic layer (applications, projects, groups, export, etc.)
  ├─ wrappers/             # HTTP API wrappers and interfaces
  │  ├─ configuration/     # Configuration loading and management
  │  ├─ bitbucketserver/   # Bitbucket Server integration
  │  └─ grpcs/            # gRPC protocol buffer definitions
  ├─ params/               # Configuration parameters and environment variable bindings
  ├─ constants/            # Application constants
  ├─ logger/               # Logging utilities
  └─ wrappers/             # HTTP client wrappers for external services
test/
  └─ integration/          # Integration tests (require //go:build integration tag and credentials)
```

## Architecture Patterns

### Command Structure
Commands are defined using Cobra framework and organized hierarchically:
- Root command created in `NewAstCLI()` with dependency injection of all wrappers
- Subcommands (scan, auth, result, project, etc.) follow Cobra's structure
- Each command validates inputs and delegates to services/wrappers

### Wrapper Pattern
HTTP API interactions are abstracted through "wrappers":
- Each wrapper implements an interface (e.g., `ScansWrapper`, `ResultsWrapper`)
- Actual HTTP implementations have `-http` suffix (e.g., `scans-http.go`)
- Wrappers handle API communication, error handling, and response parsing

### Service Layer
Services in `internal/services/` contain business logic:
- `applications.go`, `projects.go`, `groups.go` - Core domain operations
- `export.go`, `policy-management.go` - Specialized functionality
- Services depend on wrappers for external communication

### Configuration
- Uses Viper for configuration management
- Environment variable bindings defined in `internal/params/`
- Configuration loaded in `main.go` via `configuration.LoadConfiguration()`
- Supports environment variables and config files

## Testing Patterns

### Unit Tests
- Located alongside source files with `_test.go` suffix
- Use `testing` package and `testify/assert` for assertions
- Mock dependencies with mocking frameworks (e.g., `bouk/monkey` for function mocking)
- No special build tags required

### Integration Tests
- Located in `test/integration/` directory
- Use `//go:build integration` build tag at file start
- Require environment credentials: `CX_AST_USERNAME`, `CX_AST_PASSWORD`
- Helper function `executeCommand()` for CLI testing
- Run with `-tags integration` flag

**Example test pattern**:
```go
func TestSomething(t *testing.T) {
    err, buffer := executeCommand(t, "command", "subcommand", "--flag", "value")
    assertSuccessfulExecution(t, err, buffer)
}
```

## Code Quality Standards

### Linting Rules (.golangci.yml)
- **Line length**: 185 characters max (see `lll` setting)
- **Function complexity**: Max 15 (Gocyclo)
- **Function length**: Max 200 lines, 100 statements
- **Duplication threshold**: 500 tokens
- **Enabled linters**: bodyclose, depguard, dogsled, dupl, errcheck, funlen, goconst, gocritic, gocyclo, ineffassign, mnd, nakedret, revive, rowserrcheck, staticcheck, unconvert, unparam, unused, whitespace

### Dependency Rules
- Limited external dependencies via depguard (see `.golangci.yml` for allowed imports)
- Internal imports must use `github.com/checkmarx/ast-cli/internal`
- Key allowed external: Cobra, Viper, Google protobuf, JWT, gRPC, testify

### Go Version
- Go 1.25.8 required
- Uses modern Go features with go modules

## Important Files & Patterns

### Configuration & Parameters
- `internal/params/` - Define all configuration keys and environment variable mappings
- `main.go` - Wraps all dependencies and passes to `NewAstCLI()`
- `wrappers/configuration/` - Handles loading and validation

### Common Command Patterns
- Scan operations: `internal/commands/scan.go` (largest file, ~150KB+)
- Result handling: `internal/commands/result.go`, `predicates.go`
- Pre-commit/pre-receive hooks: `internal/commands/pre_commit.go`, `pre-receive.go`
- Real-time engines: Multiple files (`*-realtime.go`, `*-realtime-engine.go`)

### Error Handling
- Uses custom `AstError` wrapper in `wrappers/` for structured error codes
- Exit codes: 0 for success, 1 for general failure
- Consider `AstError` when implementing new commands

## Contribution Guidelines

See `docs/contributing.md` for full guidelines:
- Fork-and-pull Git workflow
- Branch naming: `feature/<issue#>-descriptive` or `hotfix/<issue#>-descriptive`
- PRs require associated accepted GitHub issue
- Add unit/integration tests for all new functionality
- One concern per PR in minimal changed lines
- Follow existing code patterns and conventions

## PR Labeler
Automated PR labels via `.github/pr-labeler.yml` - check config for label patterns when creating PRs.

## Release & Deployment
- Multi-platform builds (Windows, Linux, macOS) via `.goreleaser.yml`
- Docker support for CI/CD integrations
- Releases published to GitHub releases and Docker Hub
