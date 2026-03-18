---
name: Code Quality Standards and Linting Rules
description: Coding standards, linting configuration, and quality metrics
type: project
---

## Linting Configuration

### Configuration File: `.golangci.yml`
Defines all linting rules and tool settings.

### Enabled Linters

| Linter | Purpose |
|--------|---------|
| `bodyclose` | Ensures HTTP response bodies are closed |
| `depguard` | Restricts imports to allowed packages |
| `dogsled` | Finds blank assignments in function declarations |
| `dupl` | Detects code duplication |
| `errcheck` | Ensures errors are checked |
| `funlen` | Flags functions exceeding length limits |
| `gochecknoinits` | Warns on init function usage |
| `goconst` | Finds repeated strings that should be constants |
| `gocritic` | Code critique for bugs and style issues |
| `gocyclo` | Detects complex functions |
| `ineffassign` | Finds unused assignments |
| `mnd` | Magic number detector |
| `nakedret` | Warns on naked returns in functions > 5 lines |
| `revive` | Fast linter with many rules (replaces golint) |
| `rowserrcheck` | Checks sql.Rows errors |
| `staticcheck` | Static analysis tool |
| `unconvert` | Finds unnecessary type conversions |
| `unparam` | Finds unused function parameters |
| `unused` | Finds unused code (variables, functions, types) |
| `whitespace` | Finds leading/trailing whitespace issues |

### Disabled Checks
- `dupImport` - False positives
- `ifElseChain` - Code style choice
- `octalLiteral` - Potential false positives
- `whyNoLint` - Too strict
- `wrapperFunc` - Too noisy for legitimate wrapper functions

## Code Metrics

### Line Length
- **Limit**: 185 characters
- **Linter**: `lll`
- **Excludes**: Comments with URLs

### Function Complexity
- **Limit**: Max complexity of 15
- **Linter**: `gocyclo` (Cyclomatic complexity)
- **Large files**: scan.go, result.go, predicates.go justify high complexity due to domain

### Function Length
- **Lines limit**: 200 lines max
- **Statement limit**: 100 statements max
- **Linter**: `funlen`

### Code Duplication
- **Threshold**: 500 tokens
- **Linter**: `dupl`
- **Why 500**: Accounts for legitimate similar patterns

## Dependency Rules

### Allowed External Dependencies
```yaml
depguard:
  allowed:
    - $gostd                          # Standard library
    - github.com/checkmarx/ast-cli/internal  # Internal packages
    - github.com/gookit/color         # CLI colors
    - github.com/CheckmarxDev/containers-resolver
    - github.com/Checkmarx/manifest-parser
    - github.com/Checkmarx/secret-detection
    - github.com/Checkmarx/gen-ai-*   # AI integration
    - github.com/spf13/viper          # Config management
    - github.com/spf13/cobra          # CLI framework
    - github.com/pkg/errors           # Error wrapping
    - github.com/google/*             # Google packages (UUID, protobuf)
    - github.com/checkmarx/2ms/v3     # Secret scanning
    - github.com/MakeNowJust/heredoc  # Multi-line strings
    - github.com/jsumners/go-getport  # Port utilities
    - github.com/stretchr/testify/assert  # Testing
    - github.com/gofrs/flock          # File locking
    - github.com/golang-jwt/jwt/v5    # JWT handling
    # ... and others (see .golangci.yml for full list)
```

### Import Organization
- Standard library imports first
- Internal imports grouped together
- External imports grouped
- Use `goimports` for automatic organization

## Code Style Standards

### Naming Conventions
- **Variables**: camelCase (Go standard)
- **Constants**: MixedCase (Go standard)
- **Exported functions**: PascalCase
- **Private functions**: camelCase
- **Interfaces**: Short names ending in `-er` (e.g., `ScansWrapper`)

### Error Handling
- Always check errors
- Use custom `AstError` type for CLI errors
- Wrap errors with context using `pkg/errors`

Example:
```go
if err != nil {
    return errors.Wrap(err, "failed to create scan")
}
```

### Comments
- **Exported items**: Must have comment starting with name
- **Complex logic**: Add explanatory comments
- **Avoid**: Obvious comments ("i := 0 // set i to 0")

Example:
```go
// CreateScan initiates a new security scan
func (w *scansWrapper) CreateScan(ctx context.Context, req *ScanRequest) (*Scan, error) {
    // Implementation...
}
```

### Formatting
- Use `go fmt` (enforced by make)
- Import sorting via `goimports`
- Line breaks before long parameter lists
- Consistent brace placement

## Testing Standards

### Test Coverage
- Aim for >80% coverage
- Write tests for:
  - Happy path
  - Error cases
  - Edge cases (empty input, nil values, etc.)
- Use table-driven tests for multiple scenarios

### Test Naming
- Start with `Test`
- Descriptive names: `TestScanCreateWithValidProject`
- Use subtests for related cases: `t.Run("valid", func(t *testing.T) {})`

## Common Code Patterns

### Error Handling Pattern
```go
result, err := operation()
if err != nil {
    return errors.Wrap(err, "context about what failed")
}
```

### Interface Definition Pattern
```go
type SomethingWrapper interface {
    Method1(context.Context, *Request) (*Response, error)
    Method2(context.Context, string) error
}
```

### Dependency Injection Pattern
```go
type Service struct {
    wrapper1 Wrapper1
    wrapper2 Wrapper2
}

func NewService(w1 Wrapper1, w2 Wrapper2) *Service {
    return &Service{
        wrapper1: w1,
        wrapper2: w2,
    }
}
```

## Avoiding Common Issues

### Race Conditions
- Use `go test -race` to detect
- Avoid global mutable state
- Pass data through channels/parameters

### Resource Leaks
- Close HTTP response bodies: `defer resp.Body.Close()`
- Clean up temp files: `defer os.Remove(tempFile)`
- Use file locks properly: `defer lock.Unlock()`

### Error Masking
- Check all return values
- Don't silently ignore errors
- Use `errcheck` linter to catch

### Inefficient Patterns
- Prefer `strings.Builder` for string concatenation
- Cache expensive computations
- Reuse HTTP clients (connection pooling)

## Running Quality Checks

```bash
# Full lint (includes fmt first)
make lint

# Quick format check
make fmt

# Vet code
make vet

# Build with all checks
make build

# Just run linting
golangci-lint run -c .golangci.yml

# Fix issues automatically
golangci-lint run -c .golangci.yml --fix
```

## Git Hooks (Pre-commit)

While not enforced in settings, recommended to run before committing:
```bash
# Before commit
make fmt
make vet
go test ./...
make lint
```

## Performance Considerations

### For Large Files
- scan.go (~150KB) justifies size due to scan complexity
- result.go (~110KB) handles result parsing complexity
- Consider splitting only if new unrelated functionality added

### HTTP Client Optimization
- Reuse base HTTP client for connection pooling
- Set appropriate timeouts
- Handle retries at wrapper level

### Memory Usage
- Stream large files instead of loading fully
- Use iterators for large result sets
- Profile with pprof before optimizing
