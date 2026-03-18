---
name: Wrapper Pattern for API Abstraction
description: How HTTP API clients are abstracted through interfaces
type: project
---

## Wrapper Pattern Overview

The wrapper pattern abstracts external service communication through Go interfaces:

```
Interface (contract) → HTTP Implementation → External API/Service
```

This enables:
- Testing with mock implementations
- Easy API swaps
- Centralized error handling
- Retry logic and resilience

## Standard Implementation Structure

For each external service, there are typically 3 files:

### 1. Interface Definition (e.g., `scans.go`)
```go
type ScansWrapper interface {
    GetScanByID(ctx context.Context, scanID string) (*Scan, error)
    CreateScan(ctx context.Context, req *ScanRequest) (*Scan, error)
    UpdateScan(ctx context.Context, scanID string, req *ScanRequest) error
}
```

### 2. HTTP Implementation (e.g., `scans-http.go`)
```go
type httpScansWrapper struct {
    client *http.Client
    baseURL string
}

func NewScansWrapper(baseURL string) ScansWrapper {
    return &httpScansWrapper{
        client: createHTTPClient(),
        baseURL: baseURL,
    }
}

func (w *httpScansWrapper) GetScanByID(ctx context.Context, scanID string) (*Scan, error) {
    // Make HTTP request
    // Parse response
    // Handle errors
    return scan, nil
}
```

### 3. Alternative Implementation (Mock/Test)
- Can be mocked with test double
- Used in unit tests
- Implements same interface

## Major Wrappers

### Core Platform Wrappers
- `ScansWrapper` - Scan operations
- `ResultsWrapper` - Result querying and filtering
- `ProjectsWrapper` - Project management
- `ApplicationsWrapper` - Application management
- `AuthWrapper` - Authentication
- `GroupsWrapper` - User/group management

### Integration Wrappers
- `GitHubWrapper` - GitHub PR decoration
- `GitLabWrapper` - GitLab MR decoration
- `AzureWrapper` - Azure DevOps decoration
- `BitBucketWrapper` - Bitbucket Cloud
- `BitbucketServerWrapper` - Bitbucket Server (in separate package)

### Feature Wrappers
- `ExportWrapper` - Result export (JSON, SARIF, PDF)
- `CodeBashingWrapper` - Training materials
- `ChatWrapper` - AI chat features
- `FeatureFlagsWrapper` - Feature flag management
- `PolicyWrapper` - Policy evaluation
- `ContainerResolverWrapper` - Container image resolution

### Real-time Wrappers
- `ScaRealTimeWrapper` - Real-time SCA scanning
- `RealtimeScannerWrapper` - General real-time scanning

### Specialized Wrappers
- `JwtWrapper` - JWT token handling
- `TenantConfigurationWrapper` - Multi-tenant config
- `LearnMoreWrapper` - Help and documentation
- `BflWrapper` - Best fix location
- `AccessManagementWrapper` - Access control
- `ByorWrapper` - Bring Your Own Runner
- `DastEnvironmentsWrapper` - DAST environment config

## Client Wrapper Base

All HTTP wrappers typically use the base client from `client.go`:

```go
type HttpClient struct {
    baseURL string
    client *http.Client
    // ... other fields
}

// Common methods for all wrappers:
// - SendRequest()
// - HandleResponse()
// - RetryOnError()
```

## Error Handling

Wrappers use custom `AstError` type:

```go
type AstError struct {
    Err  string // Error message
    Code int    // Exit code (0 for success, non-zero for failure)
}
```

Commands check for `AstError` type for structured error handling.

## Configuration in main.go

Each wrapper is instantiated and injected in `cmd/main.go`:

```go
scansWrapper := wrappers.NewHTTPScansWrapper(scans)
resultsWrapper := wrappers.NewHTTPResultsWrapper(results, scanSummary)
authWrapper := wrappers.NewAuthHTTPWrapper()
// ... many more
```

This enables:
- Easy dependency management
- Single point to swap implementations
- Clear visibility of all integrations

## Key Pattern Points

1. **Interface First** - Define interface before implementation
2. **Constructor Pattern** - `NewHTTPXyzWrapper()` creates HTTP implementation
3. **Dependency Injection** - Pass to commands via constructor
4. **Error Wrapping** - Use `AstError` for CLI compatibility
5. **Client Reuse** - Share base HTTP client for connection pooling
6. **Configuration via Constructor** - Pass URLs and config at creation time
