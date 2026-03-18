---
name: Testing Patterns and Best Practices
description: Unit test and integration test patterns used in the codebase
type: project
---

## Test Organization

### Unit Tests
- **Location**: Alongside source files as `*_test.go`
- **Example**: `scan.go` → `scan_test.go`
- **No build tag required** - Run with standard `go test`
- **Mock dependencies** for isolated testing
- **Fast execution** - Should complete in milliseconds

### Integration Tests
- **Location**: `test/integration/` directory
- **Build tag**: `//go:build integration` at file top
- **Run command**: `go test -tags integration ./...`
- **Requirements**: Valid Checkmarx One credentials
- **Slow execution** - Makes real API calls
- **Environment variables**: `CX_AST_USERNAME`, `CX_AST_PASSWORD`

## Unit Test Pattern

### Basic Structure
```go
package commands

import (
    "testing"
    "github.com/stretchr/testify/assert"
)

func TestMyFunction(t *testing.T) {
    // Arrange - Set up test fixtures
    expectedValue := "expected"

    // Act - Execute the code being tested
    result, err := MyFunction()

    // Assert - Verify the result
    assert.NoError(t, err)
    assert.Equal(t, expectedValue, result)
}
```

### With Mocking
```go
func TestScanCreate(t *testing.T) {
    // Mock wrapper
    mockWrapper := &mockScansWrapper{
        createFunc: func(ctx context.Context, req *ScanRequest) (*Scan, error) {
            return &Scan{ID: "123", Name: req.Name}, nil
        },
    }

    // Test code using mock
    service := NewScanService(mockWrapper)
    result, err := service.CreateScan(context.Background(), &ScanRequest{Name: "test"})

    // Assertions
    assert.NoError(t, err)
    assert.Equal(t, "123", result.ID)
}
```

### Using testify/assert
```go
// Most common assertions
assert.NoError(t, err)
assert.Error(t, err)
assert.Equal(t, expected, actual)
assert.NotEqual(t, expected, actual)
assert.True(t, condition)
assert.False(t, condition)
assert.Nil(t, value)
assert.NotNil(t, value)
assert.Contains(t, slice, value)
assert.Empty(t, slice)
assert.NotEmpty(t, slice)
```

## Integration Test Pattern

### Basic Structure
```go
//go:build integration

package integration

import (
    "testing"
    "github.com/checkmarx/ast-cli/internal/commands"
    "gotest.tools/assert"
)

func TestAuthValidate(t *testing.T) {
    // Requires credentials from environment
    err, buffer := executeCommand(t, "auth", "validate")
    assertSuccessAuthentication(t, err, buffer, "Auth should succeed")
}
```

### Test with Flags
```go
func TestScanCreate(t *testing.T) {
    // Command with flags
    err, buffer := executeCommand(t,
        "scan", "create",
        "--project-name", "TestProject",
        "--branch", "main",
        "--debug",
    )

    if err != nil {
        t.Fatalf("Command failed: %v", err)
    }

    // Verify output contains expected text
    output := buffer.String()
    assert.Assert(t, strings.Contains(output, "scan created"))
}
```

### Helper: executeCommand()
```go
// Executes a CLI command and captures output
// Returns: error, output buffer
err, buffer := executeCommand(t, "auth", "validate", "--apikey", "test-key")

// Buffer contents
output := buffer.String()

// Check for specific messages
if strings.Contains(output, "error message") {
    t.Fail()
}
```

### Error Assertions
```go
// Expecting an error
err, _ := executeCommand(t, "auth", "validate", "--apikey", "")
assertError(t, err, commands.FailedAuthError)

// Expecting success
err, _ := executeCommand(t, "auth", "validate")
assertSuccessAuthentication(t, err, buffer, "Validation should pass")
```

## Mocking Patterns

### Mock Interface
```go
// Define mock that implements the interface
type mockScansWrapper struct {
    createFunc func(context.Context, *ScanRequest) (*Scan, error)
    getFunc    func(context.Context, string) (*Scan, error)
}

func (m *mockScansWrapper) Create(ctx context.Context, req *ScanRequest) (*Scan, error) {
    if m.createFunc != nil {
        return m.createFunc(ctx, req)
    }
    return nil, nil
}

func (m *mockScansWrapper) Get(ctx context.Context, id string) (*Scan, error) {
    if m.getFunc != nil {
        return m.getFunc(ctx, id)
    }
    return nil, nil
}
```

### Using bouk/monkey for Function Mocking
```go
import "github.com/bouk/monkey"

func TestWithFunctionMock(t *testing.T) {
    // Patch function globally
    guard := monkey.Patch(externalFunction, func() string {
        return "mocked result"
    })
    defer guard.Restore()

    // Run test
    result := functionThatCallsExternal()
    assert.Equal(t, "mocked result", result)
}
```

## Common Test Utilities

### Assert Helpers (in test/integration/)
```go
// Authentication assertions
assertSuccessAuthentication(t, err, buffer, "message")
assertError(t, err, expectedErrorString)

// Output verification
assert.Assert(t, strings.Contains(output, "expected text"))
assert.Assert(t, strings.HasPrefix(output, "prefix"))
```

### Table-Driven Tests
```go
func TestMultipleScenarios(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    string
        wantErr bool
    }{
        {"valid", "test", "result", false},
        {"invalid", "", "", true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := Process(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("unexpected error: %v", err)
            }
            assert.Equal(t, tt.want, result)
        })
    }
}
```

## Test Data

### Test Data Files
- Located in `test/integration/data/`
- Include: `.testdata`, `.zip`, `.json`, `.tf`, `.py`, etc.
- Used for testing parsing and scanning
- Examples: malformed SARIF, vulnerable code samples

### Creating Test Files
```go
// Create temporary file for test
tmpFile := createTempFile(t, "test-content")
defer os.Remove(tmpFile)

// Use in test
result, err := ProcessFile(tmpFile)
```

## Best Practices

1. **Test Names** - Descriptive: `TestScanCreateWithValidProject`
2. **Test Isolation** - Each test independent, can run in any order
3. **Cleanup** - Use `defer` for cleanup (files, temp data)
4. **No External Dependencies** - Mock all external calls in unit tests
5. **Arrange-Act-Assert** - Clear structure for readability
6. **Error Messages** - Include context: `t.Fatalf("failed: %v, got %v", expected, actual)`
7. **Focus Tests** - Use `t.Run()` for subtests
8. **Concurrency** - Tests run in parallel unless sequential needed

## Running Tests

```bash
# All unit tests
go test ./...

# Specific package
go test ./internal/commands

# Specific test
go test -run TestAuthValidate ./...

# With coverage
go test -coverprofile=coverage.out ./...

# Integration tests only
go test -tags integration ./test/integration/

# All tests (unit + integration)
go test ./...
go test -tags integration ./...
```

## Test Coverage

```bash
# Generate coverage report
go test -coverprofile=coverage.out ./...

# View in HTML
go tool cover -html=coverage.out

# See percentage
go tool cover -func=coverage.out | tail -1
```
