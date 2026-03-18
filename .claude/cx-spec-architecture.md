---
name: Checkmarx AST CLI Architecture
description: Core architectural patterns and design decisions
type: project
---

## Layered Architecture

The codebase follows a three-tier architecture:

```
cmd/main.go (Entry Point)
    ↓
internal/commands/ (Command Layer - Cobra CLI handlers)
    ↓
internal/services/ (Business Logic Layer)
    ↓
internal/wrappers/ (API Abstraction Layer - HTTP clients)
    ↓
External APIs (Checkmarx platform, GitHub, GitLab, Azure, etc.)
```

## Key Architectural Patterns

### Dependency Injection
- **main.go** creates all wrapper instances and injects them into `NewAstCLI()`
- Commands receive dependencies as constructor parameters
- Enables testing with mock implementations
- Reduces global state and makes dependencies explicit

### Wrapper Pattern (API Abstraction)
Each external service has:
1. **Interface** (e.g., `ScansWrapper`) - Defines contract
2. **HTTP Implementation** (e.g., `scans-http.go`) - Actual API calls
3. **Mock/Alternative** - For testing

This allows:
- Easy testing with mocks
- Swapping implementations
- Clear separation of concerns

### Service Layer
Located in `internal/services/`, contains pure business logic:
- `applications.go` - Application management
- `projects.go` - Project operations
- `groups.go` - User/group management
- `export.go` - Result export logic
- `policy-management.go` - Policy evaluation

Services depend on wrappers and each other, not on commands directly.

### Configuration Management
- **Viper** handles loading from environment, config files, defaults
- **params package** defines all configuration keys and env bindings
- **configuration wrapper** validates and loads on startup
- Supports multiple sources with precedence: env vars > flags > config file > defaults

## Directory Organization

```
internal/
├── commands/           # CLI command handlers (Cobra commands)
│   ├── util/          # Utility functions (printer, helpers)
│   ├── dast/          # DAST-specific commands
│   ├── scarealtime/   # SCA real-time scanning
│   └── policymanagement/
├── services/          # Business logic (scan logic, export, etc.)
├── wrappers/          # HTTP API abstraction layer
│   ├── configuration/ # Config loading
│   ├── grpcs/         # gRPC protocol definitions
│   └── bitbucketserver/
├── params/            # Configuration parameters
├── constants/         # Application constants
└── logger/            # Logging utilities
```

## Data Flow

1. **User Input** → CLI flags/args parsed by Cobra
2. **Command Handler** → Validates input, calls service method
3. **Service Layer** → Implements business logic, may call multiple wrappers
4. **Wrappers** → Make HTTP/gRPC calls to external services
5. **Response** → Formatted and printed by command (often using printer utilities)

## Key Design Decisions

### Why: Large monolithic files for complex domains
- **scan.go**: ~150KB+ - Complexity of scan operation justifies size
- **result.go**: Large size due to result parsing complexity
- **predicates.go**: Result filtering has many rules

**Why**: Keeps related logic together, easier to understand full domain

### Why: Separate wrappers for each service
- Clean separation between internal logic and external APIs
- Each wrapper can have its own error handling and retry logic
- Easier to test by mocking specific services
