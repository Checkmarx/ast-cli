---
name: Service Layer Business Logic
description: Service layer organization, responsibilities, and patterns
type: project
---

## Service Layer Purpose

The service layer in `internal/services/` contains pure business logic:
- Orchestrates multiple wrappers
- Implements domain logic
- No direct dependency on commands
- Independently testable

## Core Services

### Application Services (`applications.go`)
- List, get, create, delete applications
- Manage application metadata
- Application-to-project relationships

### Project Services (`projects.go`)
- Project CRUD operations
- Project filtering and search
- Project metadata management
- Integration with tenant/org structure

### Group Services (`groups.go`)
- User and team group management
- Group membership operations
- Group permission handling

### Export Services (`export.go`)
- Export scan results to multiple formats
- PDF report generation
- JSON export with configurable fields
- SARIF export for integration tools

### Policy Management Services (`policy-management.go`)
- Policy evaluation against scan results
- Policy rule application
- Compliance checking

### ASCA Services (`asca.go`)
- Application Software Composition Analysis
- Manages ASCA-specific scan operations
- Vulnerability aggregation

### SAST Metadata Services (`sast-metadata.go`)
- SAST incremental scan support
- Metadata tracking for incremental builds

### Tags Services (`tags.go`)
- Scan and result tagging
- Tag-based filtering and search

## Service Interface Pattern

Services typically implement interfaces to support mocking:

```go
type ApplicationService interface {
    GetApplications(ctx context.Context) ([]*Application, error)
    GetApplication(ctx context.Context, id string) (*Application, error)
    CreateApplication(ctx context.Context, req *CreateAppRequest) (*Application, error)
}

type ApplicationsService struct {
    wrapper ApplicationsWrapper
}

func NewApplicationsService(wrapper ApplicationsWrapper) ApplicationService {
    return &ApplicationsService{wrapper: wrapper}
}
```

## Service Responsibilities

Each service:

1. **Validates Input** - Ensures parameters are valid
2. **Coordinates Calls** - May call multiple wrappers
3. **Transforms Data** - Adapts wrapper responses
4. **Applies Business Rules** - Domain-specific logic
5. **Handles Errors** - Wraps and enriches errors
6. **No I/O** - Besides wrapper calls (no direct HTTP, file, etc.)

## Orchestration Pattern

Services often orchestrate multiple wrappers:

```go
func (s *ScanService) CreateAndWaitForScan(ctx context.Context, req *ScanRequest) (*Scan, error) {
    // 1. Create scan via ScansWrapper
    scan, err := s.scansWrapper.Create(ctx, req)
    if err != nil {
        return nil, err
    }

    // 2. Wait for completion with polling
    for {
        status, err := s.scansWrapper.GetStatus(ctx, scan.ID)
        if err != nil {
            return nil, err
        }
        if status.IsComplete() {
            break
        }
        time.Sleep(pollInterval)
    }

    // 3. Get final results via ResultsWrapper
    results, err := s.resultsWrapper.GetResults(ctx, scan.ID)
    if err != nil {
        return nil, err
    }

    return &Scan{...}, nil
}
```

## Dependency Injection

Services receive dependencies through constructors:

```go
// Multiple dependencies
func NewScanService(
    scansWrapper ScansWrapper,
    resultsWrapper ResultsWrapper,
    projectsWrapper ProjectsWrapper,
) *ScanService {
    return &ScanService{
        scansWrapper: scansWrapper,
        resultsWrapper: resultsWrapper,
        projectsWrapper: projectsWrapper,
    }
}
```

## Testing Services

Services are tested with mocked wrappers:

```go
func TestCreateApplication(t *testing.T) {
    mockWrapper := &MockApplicationsWrapper{
        CreateFunc: func(ctx context.Context, req *CreateAppRequest) (*Application, error) {
            return &Application{ID: "123", Name: req.Name}, nil
        },
    }

    service := NewApplicationsService(mockWrapper)
    result, err := service.CreateApplication(context.Background(), &CreateAppRequest{Name: "test"})

    assert.NoError(t, err)
    assert.Equal(t, "123", result.ID)
}
```

## Key Design Points

1. **Separation of Concerns** - Services don't know about CLI (commands)
2. **Reusability** - Services can be used by multiple commands
3. **Testability** - Mock wrappers for unit testing
4. **Composition** - Services can depend on other services
5. **No Global State** - All dependencies injected
6. **Context Awareness** - All methods take context for cancellation
