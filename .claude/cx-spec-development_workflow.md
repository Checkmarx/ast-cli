---
name: Development Workflow - Build, Test, and Lint
description: Commands and workflows for building, testing, and code quality checks
type: project
---

## Build Process

### Using Makefile (Recommended)
```bash
# Full build (fmt → vet → build)
make build

# Just format
make fmt

# Just vet
make vet

# Just lint
make lint
```

**Default goal is `vet`** - Running `make` alone formats and vets code.

### Direct Go Commands
```bash
# Format code
go fmt ./...

# Vet code
go vet ./...

# Build executable
go build -o bin/cx.exe ./cmd

# Build for different OS
GOOS=linux GOARCH=amd64 go build -o bin/cx ./cmd
GOOS=darwin GOARCH=amd64 go build -o bin/cx-mac ./cmd
```

## Testing

### Unit Tests
```bash
# Run all unit tests
go test ./...

# Run with verbose output
go test -v ./...

# Run specific test file
go test -v ./internal/commands/

# Run specific test function
go test -v ./internal/commands/ -run TestAuthValidate

# Run with code coverage
go test -v -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Run specific package
go test ./internal/services
```

### Integration Tests
```bash
# Run integration tests (requires credentials)
go test -tags integration ./...

# Run with verbose output
go test -v -tags integration ./...

# Run only integration tests in test/integration
go test -v -tags integration ./test/integration/

# Run specific integration test
go test -v -tags integration ./test/integration/ -run TestAuthValidate
```

**Requirements for integration tests**:
- Environment variables: `CX_AST_USERNAME`, `CX_AST_PASSWORD`
- Valid Checkmarx One credentials
- Network access to Checkmarx platform
- Tests use `//go:build integration` tag

### Test Organization
- **Unit tests**: Named `*_test.go`, alongside source files
- **Integration tests**: In `test/integration/`, use `//go:build integration` tag
- Test helper: `executeCommand()` for CLI testing
- Assertions: Use `testify/assert` package

## Code Quality & Linting

### Using Makefile
```bash
# Run linting (depends on fmt)
make lint
```

### Direct Linting
```bash
# Run golangci-lint
golangci-lint run -c .golangci.yml

# Specific linters
golangci-lint run --enable=gocyclo --enable=funlen ./...

# Fix issues automatically
golangci-lint run -c .golangci.yml --fix
```

### Linting Rules
See `.golangci.yml` for configuration:
- **Line length**: 185 characters max
- **Function complexity**: Max 15 (Gocyclo)
- **Function length**: Max 200 lines, 100 statements
- **Duplication**: 500 token threshold
- **Enabled linters**: bodyclose, depguard, dogsled, dupl, errcheck, funlen, goconst, gocritic, gocyclo, ineffassign, mnd, nakedret, revive, rowserrcheck, staticcheck, unconvert, unparam, unused, whitespace

## Workflow: Making Changes

### 1. Format & Vet Code
```bash
make fmt
make vet
# Or just: make vet (includes fmt)
```

### 2. Run Tests
```bash
# Unit tests
go test ./...

# Or specific tests if changing a domain
go test -v ./internal/commands/ -run TestScanCreate

# Before committing, run ALL tests
go test -v ./...
go test -v -tags integration ./...
```

### 3. Check Linting
```bash
make lint
# Fix any issues
golangci-lint run -c .golangci.yml --fix
```

### 4. Build & Verify
```bash
make build
./bin/cx.exe --version
./bin/cx.exe auth validate --help
```

### 5. Commit
```bash
git add .
git commit -m "descriptive message"
```

## Makefile Details

```makefile
.DEFAULT_GOAL := vet

.PHONY: fmt vet build lint

fmt:
    go fmt ./...

vet: fmt
    go vet ./...

build: vet
    go build -o bin/cx.exe ./cmd

lint: fmt
    golangci-lint run -c .golangci.yml
```

**Note**: Each target depends on previous:
- `build` → `vet` → `fmt`
- `lint` → `fmt`

## Performance Tips

### Speed Up Tests
- Run only changed package: `go test ./internal/commands`
- Skip integration tests for quick feedback: `go test ./...` (no -tags integration)
- Cache test results: First test run slower, subsequent runs cached

### Speed Up Linting
- Linting is slower than tests (~5 minutes full run)
- Run only during final verification before commit
- Use `--fix` to auto-correct many issues

### Build Variations
- Quick rebuild after changes: `go build -o bin/cx.exe ./cmd` (skip fmt/vet)
- Final build: `make build` (full validation)

## Multi-Platform Builds

For release builds with goreleaser:
```bash
# Build for all platforms (requires goreleaser)
goreleaser build --clean

# Use dev config for local testing
goreleaser build -f .goreleaser-dev.yml --single-target
```

## Troubleshooting

### Test Failures
```bash
# Run with verbose and race detection
go test -v -race -run TestName

# Get full output
go test -v -run TestName
```

### Linting Issues
```bash
# See all lint errors
golangci-lint run -c .golangci.yml

# Auto-fix where possible
golangci-lint run -c .golangci.yml --fix

# Check specific linter
golangci-lint run -c .golangci.yml --enable=gocyclo
```

### Build Issues
```bash
# Clean build artifacts
rm -rf bin/

# Force rebuild
go clean -cache

# Verbose build
go build -v -o bin/cx.exe ./cmd
```
