# Architecture Description: Checkmarx One CLI (ast-cli)

**Version**: 1.0 | **Created**: 2026-03-19 | **Last Updated**: 2026-03-19  
**Architect**: AI (cx-spec) | **Status**: Draft

---

## 1. Introduction

### 1.1 Purpose

The Checkmarx One CLI (`ast-cli`) is a standalone command-line tool that provides unified access to the Checkmarx One application security testing platform. It enables developers, DevOps engineers, and security teams to initiate security scans (SAST, SCA, IaC/KICS, DAST, Secrets, Containers), retrieve results, manage projects, and integrate with CI/CD pipelines and source code management platforms — all from a single binary.

### 1.2 Scope

**In Scope:**

- CLI command interface for all Checkmarx One scan engines (SAST, SCA, IaC, DAST, Secrets, Containers)
- Real-time scanning engines for IDE and pre-commit workflows (ASCA, KICS, OSS, Secrets, Containers)
- Result retrieval and export in multiple formats (JSON, SARIF, SonarQube, GitLab SAST/SCA, PDF)
- PR decoration for GitHub, GitLab, Azure DevOps, Bitbucket Cloud, and Bitbucket Server
- Project and application lifecycle management
- Authentication via OAuth2 client credentials and API keys
- Policy evaluation and compliance gating
- AI-assisted chat for SAST and KICS findings
- Git pre-receive and pre-commit hook integration

**Out of Scope:**

- The Checkmarx One platform backend (APIs, scan engines, databases)
- Web UI or graphical interfaces
- Plugin implementations for specific CI/CD tools (Jenkins, GitHub Actions, etc.) — those consume this CLI
- License server or billing infrastructure

### 1.3 Definitions & Acronyms

| Term | Definition |
|------|------------|
| AST | Application Security Testing |
| SAST | Static Application Security Testing |
| SCA | Software Composition Analysis |
| DAST | Dynamic Application Security Testing |
| IaC | Infrastructure as Code |
| KICS | Keeping Infrastructure as Code Secure (Checkmarx IaC scanner) |
| ASCA | AI-Secured Code Assistant (real-time SAST via gRPC) |
| BFL | Best Fix Location |
| BYOR | Bring Your Own Runner |
| PR | Pull Request |
| SCM | Source Code Management |

---

## 2. Stakeholders & Concerns

| Stakeholder | Role | Key Concerns | Priority |
|-------------|------|--------------|----------|
| Developers | End users running scans locally or in CI | Fast feedback, clear results, IDE integration, pre-commit hooks | High |
| DevOps / CI Engineers | Pipeline integrators | Reliable exit codes, automation-friendly output, proxy support, non-interactive execution | High |
| Security Teams | Governance and compliance | Policy enforcement, comprehensive scan coverage, result accuracy, audit trails | Critical |
| Plugin Maintainers | CI/CD plugin developers (Jenkins, GH Actions, etc.) | Stable CLI interface, predictable behavior, version compatibility | High |
| Checkmarx Platform Team | Backend API providers | API contract stability, authentication flows, feature flag coordination | High |
| Open Source Community | Contributors and users | Clear documentation, build reproducibility, Apache 2.0 license compliance | Medium |

---

## 3. Architectural Views (Rozanski & Woods)

### 3.1 Context View

**Purpose**: Define system scope and external interactions

#### 3.1.1 System Scope

The CLI is a stateless, single-binary command-line application that acts as a client to the Checkmarx One platform REST APIs. It orchestrates scan workflows, retrieves and formats results, and integrates with SCM platforms for PR decoration. It also manages local real-time scanning engines (gRPC-based ASCA, embedded KICS, SCA, Secrets, Containers) for IDE-like feedback.

#### 3.1.2 External Entities

| Entity | Type | Interaction Type | Data Exchanged | Protocols |
|--------|------|------------------|----------------|-----------|
| Checkmarx One Platform | External API | REST API calls | Scans, results, projects, policies, feature flags | HTTPS |
| ASCA Engine | Local gRPC service | gRPC calls | Real-time SAST scan requests/results | gRPC (localhost) |
| GitHub | SCM Platform | REST API | PR comments, user counts, repository metadata | HTTPS |
| GitLab | SCM Platform | REST API | PR comments, user counts, repository metadata | HTTPS |
| Azure DevOps | SCM Platform | REST API | PR comments, user counts | HTTPS |
| Bitbucket Cloud | SCM Platform | REST API | PR comments, user counts | HTTPS |
| Bitbucket Server | SCM Platform | REST API | PR comments, user counts | HTTPS |
| Docker Engine | Container runtime | Docker CLI / API | Container image resolution, KICS container lifecycle | Unix socket / TCP |
| Local Filesystem | Storage | File I/O | Source code, scan results, configuration files | OS filesystem |
| CI/CD Pipelines | Orchestrator | Process invocation | CLI arguments, exit codes, stdout/stderr | Process |
| Proxy Servers | Network intermediary | HTTP/HTTPS proxy | All outbound traffic (supports NTLM, Kerberos) | HTTP(S) |

#### 3.1.3 Context Diagram

```text
                          ┌─────────────────────────────┐
                          │      CI/CD Pipelines         │
                          │  (Jenkins, GH Actions, etc.) │
                          └──────────┬──────────────────┘
                                     │ invokes
                                     ▼
┌──────────────┐          ┌─────────────────────────────┐          ┌──────────────────────┐
│  Developers  │─────────▶│     Checkmarx One CLI       │─────────▶│  Checkmarx One       │
│  (Terminal)  │  runs    │        (ast-cli)             │  REST    │  Platform APIs       │
└──────────────┘          │                             │  HTTPS   │  (Scans, Results,    │
                          │  ┌───────────────────────┐  │          │   Projects, Policies)│
                          │  │ Real-Time Engines     │  │          └──────────────────────┘
                          │  │ (ASCA/KICS/SCA/       │  │
                          │  │  Secrets/Containers)  │  │          ┌──────────────────────┐
                          │  └───────────────────────┘  │─────────▶│  SCM Platforms       │
                          └──────────┬──────────────────┘  REST    │  (GitHub, GitLab,    │
                                     │                     HTTPS   │   Azure, Bitbucket)  │
                                     │ reads                       └──────────────────────┘
                                     ▼
                          ┌─────────────────────────────┐
                          │    Local Filesystem          │
                          │  (Source code, configs,      │
                          │   scan results)              │
                          └─────────────────────────────┘
```

#### 3.1.4 External Dependencies

| Dependency | Purpose | SLA Expectations | Fallback Strategy |
|------------|---------|------------------|-------------------|
| Checkmarx One Platform | All scan operations, result retrieval, project management | Platform SLA (typically 99.9%) | CLI exits with error code; retries with rate limiting |
| SCM Platform APIs | PR decoration, user count | Varies by provider | Graceful degradation; PR decoration skipped on failure |
| Docker Engine | Container scanning, KICS container mode | Local availability | Falls back to non-container mode where possible |
| ASCA gRPC Engine | Real-time SAST scanning | Local process | Auto-install and restart; error reported to user |

---

### 3.2 Functional View

**Purpose**: Describe functional elements, their responsibilities, and interactions

#### 3.2.1 Functional Elements

| Element | Responsibility | Interfaces Provided | Dependencies |
|---------|----------------|---------------------|--------------|
| **Command Layer** (`internal/commands/`) | CLI command parsing, user interaction, output formatting | Cobra CLI commands: `scan`, `project`, `result`, `utils`, `chat`, etc. | Services, Wrappers, Params |
| **Service Layer** (`internal/services/`) | Business logic orchestration, multi-step workflows | Internal Go interfaces | Wrappers |
| **Wrapper Layer** (`internal/wrappers/`) | HTTP/gRPC API clients with interface abstraction | Go interfaces per API domain (e.g., `ScansWrapper`, `ProjectsWrapper`) | Checkmarx One APIs, SCM APIs |
| **Params** (`internal/params/`) | CLI flags, environment variables, config key definitions | Constants and binding definitions | Viper config |
| **Configuration** (`internal/wrappers/configuration/`) | Config file loading and management | `LoadConfiguration()` | Viper, local filesystem |
| **Logger** (`internal/logger/`) | Verbose/debug logging | `PrintIfVerbose()`, `PrintfIfVerbose()` | stdout/stderr |
| **Real-Time Engines** (`internal/services/realtimeengine/`) | Local real-time scanning (ASCA, KICS, OSS, Secrets, Containers) | Engine-specific command handlers | gRPC (ASCA), Docker (KICS), embedded libs |
| **Proxy Auth** (`internal/wrappers/kerberos/`, `ntlm/`) | Corporate proxy authentication | Transparent HTTP transport wrapping | OS Kerberos/NTLM libraries |

#### 3.2.2 Element Interactions

```text
┌─────────────────────────────────────────────────────────────────────────┐
│                           cmd/main.go                                   │
│  Entry point: loads config, constructs wrappers, builds CLI tree        │
└───────────────────────────────┬─────────────────────────────────────────┘
                                │ injects ~38 wrappers
                                ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                     Command Layer (Cobra)                                │
│                                                                         │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────────┐ │
│  │  scan     │ │ project  │ │ result   │ │  utils   │ │ chat/realtime│ │
│  └────┬─────┘ └────┬─────┘ └────┬─────┘ └────┬─────┘ └──────┬───────┘ │
└───────┼────────────┼────────────┼────────────┼───────────────┼─────────┘
        │            │            │            │               │
        ▼            ▼            ▼            ▼               ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                       Service Layer                                      │
│  projects.go, applications.go, export.go, policy-management.go, etc.    │
└───────────────────────────────┬─────────────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                       Wrapper Layer                                      │
│                                                                         │
│  ┌────────────┐ ┌────────────┐ ┌────────────┐ ┌────────────────────┐   │
│  │ scans-http │ │results-http│ │projects-http│ │ github/gitlab/     │   │
│  │            │ │            │ │             │ │ azure/bitbucket    │   │
│  └─────┬──────┘ └─────┬──────┘ └──────┬──────┘ └────────┬──────────┘   │
│        │              │               │                  │              │
│  ┌─────┴──────────────┴───────────────┴──────────────────┴───────────┐  │
│  │                     client.go (HTTP client)                       │  │
│  │  Auth, retries, rate limiting, proxy, NTLM/Kerberos               │  │
│  └───────────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────────┘
                                │
                                ▼
                    Checkmarx One Platform APIs
                    SCM Platform APIs
```

#### 3.2.3 Functional Boundaries

**What this system DOES:**

- Provides a CLI interface to all Checkmarx One scanning capabilities
- Formats and exports scan results in multiple industry-standard formats
- Decorates pull requests with security findings across 5 SCM platforms
- Runs local real-time scanning engines for fast developer feedback
- Manages projects, applications, groups, and tenant configuration
- Evaluates policies and gates CI/CD pipelines based on results
- Supports corporate proxy environments (NTLM, Kerberos)

**What this system does NOT do:**

- Does not perform actual security analysis (delegated to Checkmarx One platform engines)
- Does not store scan results persistently (stateless client)
- Does not provide a web interface or REST API
- Does not manage user accounts or RBAC (delegated to platform)

---

### 3.3 Information View

**Purpose**: Describe data storage, management, and flow

#### 3.3.1 Data Entities

| Entity | Storage Location | Owner Component | Lifecycle | Access Pattern |
|--------|------------------|-----------------|-----------|----------------|
| CLI Configuration | `~/.checkmarx/` (Viper config file) | Configuration wrapper | Persistent across sessions | Read on startup, write on `utils config` |
| Source Code (scan input) | Local filesystem / zip upload | Uploads wrapper | Ephemeral (zipped and uploaded) | Read-once, upload to platform |
| Scan Results | Checkmarx One Platform (remote) | Results wrapper | Retrieved on demand | Read-heavy, formatted for output |
| JWT Access Token | In-memory | Auth wrapper | Short-lived (1 hour), auto-refreshed | Read per API call |
| Feature Flags | Checkmarx One Platform (remote) | Feature flags wrapper | Cached per session | Read on first use, cached |
| PR Decoration Data | SCM Platform (remote) | PR wrapper | Created per scan | Write-once per PR |
| Real-Time Engine Binaries | OS-specific install path | OS installer service | Installed/updated on demand | Read on engine start |

#### 3.3.2 Data Flow

**Key Data Flows:**

1. **Scan Creation**: User invokes `cx scan create` → CLI zips source code → uploads to platform → creates scan → polls for completion → retrieves results → formats output
2. **Result Export**: User invokes `cx result show` → CLI fetches results from platform → transforms to requested format (SARIF, JSON, PDF, SonarQube, GL-SAST, GL-SCA) → writes to file or stdout
3. **PR Decoration**: Scan completes → CLI fetches results → formats findings as PR comments → posts to SCM platform API
4. **Real-Time Scan**: IDE/hook triggers real-time scan → CLI starts local engine (gRPC/Docker) → sends file content → receives findings → returns results
5. **Authentication**: CLI reads credentials from env vars or config → exchanges for JWT via OAuth2 client credentials flow → caches token in memory → attaches to all API calls

#### 3.3.3 Data Quality & Integrity

- **Consistency Model**: Stateless client; all persistent state lives on the Checkmarx One platform (strong consistency via platform APIs)
- **Validation Rules**: CLI validates input parameters before API calls; JSON schema validation for policy rules
- **Retention Policy**: No local data retention; results are transient and written to user-specified output paths
- **Backup Strategy**: N/A — CLI is stateless; configuration is user-managed

---

### 3.4 Concurrency View

**Purpose**: Describe runtime processes, threads, and coordination

#### 3.4.1 Process Structure

| Process | Purpose | Scaling Model | State Management | Resource Limits |
|---------|---------|---------------|------------------|-----------------|
| CLI main process | Command execution, API orchestration | Single process per invocation | Stateless | OS process limits |
| ASCA gRPC engine | Real-time SAST scanning | Single instance per machine | Stateful (loaded rules) | Managed by engine binary |
| KICS Docker container | Real-time IaC scanning | Single container per scan | Stateless | Docker resource limits |
| Scan polling loop | Waits for scan completion | Single goroutine | In-memory scan ID | Configurable timeout |

#### 3.4.2 Thread Model

- **Threading Strategy**: Go goroutines managed by the Go runtime scheduler
- **Async Patterns**: Goroutines for signal handling (SIGTERM), scan polling, and parallel container resolution
- **Resource Pools**: HTTP connection pooling via Go's `net/http` default transport; rate limiting via `rate-limit.go`

#### 3.4.3 Coordination Mechanisms

- **Synchronization**: File-based locking via `gofrs/flock` for concurrent CLI invocations; Go channels for signal handling
- **Communication**: gRPC for ASCA engine; REST/HTTPS for all platform and SCM APIs; Docker CLI for container operations
- **Deadlock Prevention**: Stateless design eliminates most deadlock scenarios; HTTP client timeouts prevent hanging

---

### 3.5 Development View

**Purpose**: Constraints for developers — code organization, dependencies, CI/CD

#### 3.5.1 Code Organization

```text
ast-cli/
├── cmd/
│   └── main.go                    # Entry point: config loading, wrapper construction, CLI execution
├── internal/
│   ├── commands/                  # Cobra command definitions
│   │   ├── root.go                # NewAstCLI() — wires all commands and wrappers
│   │   ├── scan.go                # scan create/list/show/delete/cancel
│   │   ├── project.go             # project CRUD
│   │   ├── result.go              # result show with multi-format output
│   │   ├── auth.go                # authentication commands
│   │   ├── chat.go                # AI chat (SAST, KICS)
│   │   ├── hooks.go               # Git hooks (pre-receive, pre-commit)
│   │   ├── asca/                  # ASCA engine management
│   │   ├── dast/                  # DAST environments
│   │   ├── scarealtime/           # SCA real-time config
│   │   ├── policymanagement/      # Policy evaluation
│   │   ├── util/                  # Utility subcommands (version, config, import, PR, etc.)
│   │   └── data/                  # Embedded data files
│   ├── services/                  # Business logic layer
│   │   ├── projects.go            # Project find/create/update orchestration
│   │   ├── applications.go        # Application management
│   │   ├── export.go              # Export orchestration
│   │   ├── policy-management.go   # Policy evaluation flow
│   │   ├── realtimeengine/        # Real-time engine orchestration
│   │   │   ├── secretsrealtime/   # Secrets detection engine
│   │   │   ├── ossrealtime/       # OSS vulnerability engine
│   │   │   ├── containersrealtime/# Container scanning engine
│   │   │   └── iacrealtime/       # IaC (KICS) engine
│   │   └── osinstaller/           # OS-specific binary installer
│   ├── wrappers/                  # API client layer (interface + HTTP impl)
│   │   ├── client.go              # Core HTTP client (auth, retries, proxy)
│   │   ├── scans.go / scans-http.go
│   │   ├── results.go / results-http.go
│   │   ├── projects.go / projects-http.go
│   │   ├── github.go / github-http.go
│   │   ├── gitlab.go / gitlab-http.go
│   │   ├── azure.go / azure-http.go
│   │   ├── bitbucket.go / bitbucket-http.go
│   │   ├── ... (~85 wrapper files)
│   │   ├── mock/                  # Mock implementations for testing
│   │   ├── grpcs/                 # gRPC client (ASCA engine)
│   │   ├── configuration/         # Config file management
│   │   ├── remediation/           # Package remediation
│   │   ├── kerberos/              # Kerberos proxy auth
│   │   ├── ntlm/                  # NTLM proxy auth
│   │   ├── bitbucketserver/       # Bitbucket Server client
│   │   └── utils/                 # Wrapper utilities
│   ├── params/                    # CLI parameters, env vars, config keys
│   │   ├── flags.go               # Flag name constants
│   │   ├── keys.go                # Viper config keys
│   │   ├── envs.go                # Environment variable names (CX_*)
│   │   ├── binds.go               # Env → config bindings with defaults
│   │   └── filters.go             # File extension filters
│   ├── constants/                 # Application constants
│   │   ├── errors/                # Error message constants
│   │   ├── exit-codes/            # Exit code definitions
│   │   └── feature-flags/         # Feature flag constants
│   └── logger/                    # Logging utilities
├── test/
│   ├── integration/               # Integration tests (real API calls)
│   │   ├── data/                  # Test fixtures and sample files
│   │   └── *_test.go              # 33 integration test files
│   └── cleandata/                 # Test artifact cleanup
├── docs/                          # Contributing guide, code of conduct
├── licenses/                      # Third-party license files
├── .github/
│   ├── workflows/                 # 11 CI/CD workflow files
│   └── scripts/                   # Release signing scripts
├── Dockerfile                     # Minimal container image
├── Makefile                       # Build targets (fmt, vet, build, lint)
├── .goreleaser.yml                # Production release config
├── .goreleaser-dev.yml            # Dev/prerelease config
├── .golangci.yml                  # Linter configuration
└── go.mod / go.sum                # Go module dependencies
```

#### 3.5.2 Module Dependencies

**Dependency Rules:**

- `cmd/main.go` depends on `commands`, `wrappers`, `params`, `logger`, `configuration`
- `commands/` depends on `services/`, `wrappers/` (interfaces), `params/`
- `services/` depends on `wrappers/` (interfaces)
- `wrappers/` implementations depend only on external HTTP/gRPC libraries
- No circular dependencies: strict layering `commands → services → wrappers`
- All wrapper dependencies are injected via constructor (`NewAstCLI()` receives ~38 wrapper interfaces)

**Key External Dependencies:**

| Dependency | Purpose |
|------------|---------|
| `spf13/cobra` | CLI framework |
| `spf13/viper` | Configuration management |
| `golang-jwt/jwt/v5` | JWT token handling |
| `google.golang.org/grpc` | gRPC client for ASCA |
| `google.golang.org/protobuf` | Protobuf serialization |
| `stretchr/testify` | Test assertions |
| `checkmarx/2ms` | Secret detection engine |
| `Checkmarx/containers-resolver` | Container image resolution |
| `Checkmarx/gen-ai-wrapper` | AI chat integration |
| `gofrs/flock` | File-based locking |
| `jcmturner/gokrb5` | Kerberos authentication |
| `alexbrainman/sspi` | Windows SSPI/NTLM |

#### 3.5.3 Build & CI/CD

- **Build System**: Go toolchain + Makefile (`go build -o bin/cx.exe ./cmd`)
- **CI Pipeline Stages** (`.github/workflows/ci-tests.yml`):
  1. Lint (`golangci-lint` with `.golangci.yml`)
  2. Unit tests (`go test` excluding integration tag, coverage threshold 77.7%)
  3. Integration tests (`go test -tags integration`, coverage threshold 75%)
  4. Vulnerability check (`govulncheck`)
  5. Docker image scan (Trivy)
  6. Failed test rerun (retry once)
  7. Coverage merge and upload
- **Release Pipeline** (`.github/workflows/release.yml`):
  1. GoReleaser builds for Linux (amd64, arm, arm64), Windows (amd64), macOS (universal)
  2. Apple code signing (macOS)
  3. Windows binary signing via remote HSM
  4. Docker images pushed to `cxsdlc/ast-cli` and `checkmarx/ast-cli`
  5. Cosign signing and verification of Docker images
  6. S3 upload to `CxOne/CLI/{tag}` and `CxOne/CLI/latest`
  7. Homebrew formula update (`checkmarx/homebrew-ast-cli`)
  8. Downstream plugin notification via `plugins-release-workflow`
- **Artifact Management**: GitHub Releases, Docker Hub, AWS S3, Homebrew tap
- **Deployment Strategy**: Immutable binary releases; users download specific versions

#### 3.5.4 Development Standards

- **Coding Standards**: `golangci-lint` with 20+ linters (see `.golangci.yml`); `gofmt` + `goimports` formatting; max function length 200 lines; cyclomatic complexity limit 15; line length limit 185
- **Review Requirements**: Auto-assigned `cx-plugins-releases` reviewer; PR title must match `CamelCase (AST-XXXX)` format; branch naming convention: `bug/`, `feature/`, `other/`
- **Testing Requirements**: Unit test coverage >= 77.7%; integration test coverage >= 75%; all tests pass before merge
- **Documentation Requirements**: Contributing guide in `docs/`; README with build instructions
- **Security**: GitGuardian secret scanning on all commits; Checkmarx One self-scan on PRs and main; Dependabot for dependency updates with auto-merge

---

### 3.6 Deployment View

**Purpose**: Physical environment — nodes, networks, storage

#### 3.6.1 Runtime Environments

| Environment | Purpose | Infrastructure | Scale | Configuration |
|-------------|---------|----------------|-------|---------------|
| Developer Machine | Local development and testing | Any OS (Linux, macOS, Windows) | Single binary | Env vars or `~/.checkmarx/` config |
| CI/CD Runner | Automated scanning in pipelines | GitHub Actions, Jenkins, etc. | Ephemeral per job | Env vars (`CX_BASE_URI`, `CX_CLIENT_ID`, `CX_CLIENT_SECRET`) |
| Docker Container | Containerized CLI execution | `checkmarx/ast-cli` image (Alpine-based) | Single container | Env vars passed at runtime |
| Pre-receive Hook | Git server-side scanning | Git server (GitHub Enterprise, etc.) | Per push event | Server-side configuration |

#### 3.6.2 Network Topology

```text
┌──────────────────────────────────────────────────────────────────────┐
│                     User Environment                                  │
│                                                                      │
│  ┌────────────────┐     ┌────────────────┐     ┌─────────────────┐  │
│  │  Developer      │     │  CI/CD Runner   │     │  Docker Host    │  │
│  │  Workstation    │     │  (ephemeral)    │     │                 │  │
│  │                 │     │                 │     │  ┌───────────┐  │  │
│  │  ┌───────────┐ │     │  ┌───────────┐  │     │  │ ast-cli   │  │  │
│  │  │  cx CLI   │ │     │  │  cx CLI   │  │     │  │ container │  │  │
│  │  └─────┬─────┘ │     │  └─────┬─────┘  │     │  └─────┬─────┘  │  │
│  └────────┼───────┘     └────────┼────────┘     └────────┼────────┘  │
│           │                      │                       │           │
│           └──────────────────────┼───────────────────────┘           │
│                                  │                                    │
│                    ┌─────────────┴──────────────┐                    │
│                    │  Corporate Proxy (optional) │                    │
│                    │  (NTLM / Kerberos)          │                    │
│                    └─────────────┬──────────────┘                    │
└──────────────────────────────────┼───────────────────────────────────┘
                                   │ HTTPS
                                   ▼
                    ┌──────────────────────────────┐
                    │   Checkmarx One Platform      │
                    │   (Multi-tenant SaaS)         │
                    │                               │
                    │   ┌─────────┐  ┌──────────┐  │
                    │   │ Scan API│  │Result API │  │
                    │   └─────────┘  └──────────┘  │
                    │   ┌─────────┐  ┌──────────┐  │
                    │   │Auth API │  │Policy API │  │
                    │   └─────────┘  └──────────┘  │
                    └──────────────────────────────┘

                    ┌──────────────────────────────┐
                    │   SCM Platforms               │
                    │   (GitHub, GitLab, Azure,     │
                    │    Bitbucket)                  │
                    └──────────────────────────────┘
```

#### 3.6.3 Hardware Requirements

| Component | CPU | Memory | Storage | Network |
|-----------|-----|--------|---------|---------|
| CLI Binary | Minimal (single-threaded workload) | ~50-200MB during scan upload | ~50MB binary + temp space for zipping | Outbound HTTPS to platform |
| ASCA Engine | 1-2 cores | ~500MB-1GB | Engine binary (~100MB) | Localhost gRPC only |
| Docker (KICS) | 1-2 cores | ~512MB per container | Docker image (~200MB) | Localhost only |

#### 3.6.4 Third-Party Services

| Service | Provider | Purpose | SLA | Cost Model |
|---------|----------|---------|-----|------------|
| Checkmarx One | Checkmarx | Security scanning platform | Per customer SLA | Subscription |
| GitHub API | GitHub | PR decoration, user counts | 99.9% | Free tier / Enterprise |
| GitLab API | GitLab | PR decoration, user counts | 99.95% (SaaS) | Free tier / Premium |
| Azure DevOps API | Microsoft | PR decoration, user counts | 99.9% | Free tier / Enterprise |
| Docker Hub | Docker | Container image hosting | 99.9% | Free tier |
| AWS S3 | Amazon | CLI binary distribution | 99.99% | Pay per storage/transfer |
| Homebrew | Community | macOS package distribution | Best effort | Free |

---

### 3.7 Operational View

**Purpose**: Operations, support, and maintenance in production

#### 3.7.1 Operational Responsibilities

| Activity | Owner | Frequency | Automation Level | Tools |
|----------|-------|-----------|------------------|-------|
| CLI Release | Checkmarx Integrations Team | On-demand | Fully automated | GoReleaser, GitHub Actions |
| Dependency Updates | Dependabot + Team | Weekly | Semi-automated | Dependabot, auto-merge workflow |
| Security Scanning | Automated | Per PR + daily | Fully automated | Checkmarx One, GitGuardian, Trivy, govulncheck |
| Integration Testing | Automated | Per PR + nightly | Fully automated | GitHub Actions (13 parallel matrix groups) |
| Plugin Notification | Automated | Per release | Fully automated | `plugins-release-workflow` |
| Docker Image Signing | Automated | Per release | Fully automated | Cosign |

#### 3.7.2 Monitoring & Alerting

- **Key Metrics Collected**:
  - CI/CD pipeline pass/fail rates
  - Integration test success rates (nightly parallel runs)
  - Dependabot vulnerability alerts
  - Docker image vulnerability scans (Trivy, daily cache refresh)

- **Alerting Rules**:
  - CI test failures → PR blocked from merge
  - Nightly integration failures → Team notification
  - Critical vulnerability in dependencies → Dependabot PR auto-created

- **Logging Strategy**:
  - CLI supports `--verbose` flag for detailed operation logging
  - All output to stdout/stderr (consumed by CI/CD log aggregation)
  - No centralized log collection for CLI itself (client-side tool)

#### 3.7.3 Disaster Recovery

- **RTO**: N/A (stateless client; recovery = re-download binary)
- **RPO**: N/A (no persistent state)
- **Backup Strategy**: Release artifacts stored in GitHub Releases + AWS S3 + Docker Hub (triple redundancy)
- **Recovery Procedures**: Download latest or specific version from any distribution channel

#### 3.7.4 Support Model

- **Tier 1**: Checkmarx documentation and community forums
- **Tier 2**: Checkmarx support team (customer issues)
- **Tier 3**: Checkmarx Integrations engineering team (code bugs, architectural issues)
- **Escalation Path**: GitHub Issues → Internal JIRA (auto-synced via `issue_automation.yml`)
- **Code Owners**: `@cx-anurag-dalke`, `@cx-anjali-deore`, `@cx-umesh-waghode`

---

## 4. Architectural Perspectives (Cross-Cutting Concerns)

### 4.1 Security Perspective

**Applies to**: All views

#### 4.1.1 Authentication & Authorization

- **Identity Provider**: OAuth2 client credentials flow against Checkmarx One IAM; API key authentication as alternative
- **Authorization Model**: Delegated to Checkmarx One platform RBAC; CLI passes JWT tokens that carry tenant/role claims
- **Session Management**: JWT access tokens cached in-memory per CLI invocation; auto-refreshed on expiry; no persistent token storage on disk

#### 4.1.2 Data Protection

- **Encryption in Transit**: All API communication over TLS/HTTPS; gRPC with TLS for ASCA engine
- **Encryption at Rest**: No sensitive data stored at rest by CLI; configuration file may contain base URI (not secrets)
- **Secrets Management**: Credentials provided via environment variables (`CX_CLIENT_ID`, `CX_CLIENT_SECRET`, `CX_APIKEY`); never written to disk or logs
- **PII Handling**: CLI processes source code transiently; no PII stored; source code uploaded encrypted to platform

#### 4.1.3 Threat Model

| Threat | View Affected | Likelihood | Impact | Mitigation |
|--------|---------------|------------|--------|------------|
| Credential exposure in CI logs | Operational | Medium | High | Secrets masked in output; GitGuardian scanning; env var-based credential passing |
| Man-in-the-middle on API calls | Context | Low | High | TLS enforcement; certificate validation; proxy auth (NTLM/Kerberos) |
| Malicious dependency injection | Development | Low | Critical | Dependabot monitoring; `go.sum` verification; govulncheck in CI; depguard linter |
| Binary tampering | Deployment | Low | Critical | Cosign signing for Docker images; Apple code signing for macOS; Windows HSM signing |
| Source code exposure during upload | Information | Low | High | Encrypted upload to platform; temporary zip files cleaned up |
| Unauthorized scan initiation | Functional | Low | Medium | OAuth2 authentication required; tenant-scoped API keys |

---

### 4.2 Performance & Scalability Perspective

**Applies to**: Functional, Concurrency, Deployment views

#### 4.2.1 Performance Requirements

| Metric | Target | Measurement Method | Notes |
|--------|--------|--------------------|-------|
| CLI startup time | < 1 second | Manual testing | Single binary, no JVM warmup |
| Source zip + upload | Proportional to codebase size | Integration tests | Parallel file processing |
| API response handling | Bounded by platform API latency | Integration tests | Rate limiting with backoff |
| Real-time scan response | < 5 seconds for single file | Engine benchmarks | ASCA gRPC, local execution |

#### 4.2.2 Scalability Model

- **Horizontal Scaling**: Each CLI invocation is independent and stateless; CI/CD pipelines naturally parallelize across runners
- **Vertical Scaling**: N/A — single binary with minimal resource requirements
- **Rate Limiting**: Built-in rate limiter (`rate-limit.go`) respects platform API rate limits with exponential backoff and retry

#### 4.2.3 Capacity Planning

- **Current Capacity**: Unlimited concurrent CLI invocations (bounded only by platform API capacity)
- **Bottlenecks Identified**: Platform API rate limits; source code upload bandwidth; scan queue depth on platform side
- **Mitigation**: Rate limiting, retry logic, configurable timeouts

---

## 5. Global Constraints & Principles

### 5.1 Technical Constraints

- Must compile to a single static binary per platform (no runtime dependencies)
- Must support Linux (amd64, arm, arm64), macOS (universal), and Windows (amd64)
- Must work behind corporate proxies with NTLM and Kerberos authentication
- Must be embeddable in any CI/CD system via process invocation and exit codes
- Go 1.25.8+ required for compilation
- All API endpoints configurable via environment variables (`CX_*` prefix)

### 5.2 Architectural Principles

- **Stateless Client**: CLI stores no persistent state; all state lives on the Checkmarx One platform
- **Interface-Driven Design**: Every API client is defined as a Go interface with HTTP implementation, enabling testability via mocks
- **Dependency Injection**: All wrappers injected via constructor; no global state or singletons for API clients
- **Layered Architecture**: Strict `Commands → Services → Wrappers` dependency flow; no upward dependencies
- **Fail-Fast with Clear Exit Codes**: CLI returns specific exit codes for different failure modes; errors propagated to caller
- **Configuration Hierarchy**: CLI flags > environment variables > config file > defaults (Viper precedence)

---

## 6. Architecture Decision Records (ADRs)

| ID | Decision | Status | Date | Owner |
|----|----------|--------|------|-------|
| ADR-001 | Use Cobra + Viper for CLI framework | Accepted | Legacy | Integrations Team |
| ADR-002 | Interface + HTTP implementation pattern for all API clients | Accepted | Legacy | Integrations Team |
| ADR-003 | gRPC for ASCA real-time engine communication | Accepted | Legacy | Integrations Team |
| ADR-004 | GoReleaser for cross-platform binary distribution | Accepted | Legacy | Integrations Team |
| ADR-005 | Cosign for Docker image signing | Accepted | Legacy | Integrations Team |

### ADR-001: Use Cobra + Viper for CLI Framework

**Status**: Accepted  
**Date**: Legacy (established at project inception)  
**Owner**: Checkmarx Integrations Team

**Context**: The CLI needed a mature, well-documented framework supporting subcommands, flags, environment variable binding, and configuration file management. The Go ecosystem offers several options.

**Decision**: Use `spf13/cobra` for command structure and `spf13/viper` for configuration management.

**Consequences**:

- **Positive**: Industry-standard Go CLI framework; excellent documentation; automatic help generation; shell completion support; seamless env var and config file binding
- **Negative**: Large dependency tree; Cobra's global state patterns require careful management
- **Risks**: Framework lock-in (mitigated by Cobra's stability and widespread adoption)

### ADR-002: Interface + HTTP Implementation Pattern

**Status**: Accepted  
**Date**: Legacy  
**Owner**: Checkmarx Integrations Team

**Context**: The CLI interacts with 20+ distinct API endpoints. Testing command logic requires isolating API calls.

**Decision**: Define a Go interface for each API domain (e.g., `ScansWrapper`, `ProjectsWrapper`) with a corresponding HTTP implementation. Use mock implementations from `wrappers/mock/` in unit tests.

**Consequences**:

- **Positive**: Complete testability without network calls; clear API contracts; easy to add new API clients
- **Negative**: ~85 wrapper files creates significant boilerplate; constructor (`NewAstCLI`) accepts ~38 parameters
- **Risks**: Interface drift if not kept synchronized with platform API changes

### ADR-003: gRPC for ASCA Real-Time Engine

**Status**: Accepted  
**Date**: Legacy  
**Owner**: Checkmarx Integrations Team

**Context**: Real-time SAST scanning requires low-latency communication with a locally running analysis engine.

**Decision**: Use gRPC with Protocol Buffers for communication between the CLI and the ASCA engine process.

**Consequences**:

- **Positive**: Low latency; strongly typed contracts; efficient binary serialization; streaming support
- **Negative**: Requires protobuf compilation step; adds gRPC dependency
- **Risks**: Engine binary distribution and version compatibility

---

## Appendix

### A. Glossary

| Term | Definition |
|------|------------|
| ast-cli | The Checkmarx One CLI tool (Application Security Testing CLI) |
| Wrapper | A Go interface + HTTP implementation pair that encapsulates API client logic |
| Engine | A local scanning process (ASCA, KICS, SCA, Secrets, Containers) for real-time results |
| PR Decoration | Posting scan findings as comments on pull requests in SCM platforms |
| Scan | A security analysis job submitted to the Checkmarx One platform |
| Policy | A set of rules that gate CI/CD pipelines based on scan findings |
| Feature Flag | A platform-controlled toggle that enables/disables CLI features |

### B. References

- [Checkmarx One CLI Documentation](https://checkmarx.com/resource/documents/en/34965-68620-checkmarx-one-cli-tool.html)
- [GitHub Repository](https://github.com/Checkmarx/ast-cli)
- [Rozanski & Woods — Software Systems Architecture](https://www.viewpoints-and-perspectives.info/)
- [Cobra CLI Framework](https://cobra.dev/)
- [GoReleaser](https://goreleaser.com/)

### C. Tech Stack Summary

**Languages**: Go 1.25.8  
**Frameworks**: Cobra (CLI), Viper (configuration), gRPC + Protobuf (ASCA engine)  
**Databases**: None (stateless client)  
**Infrastructure**: Docker (Alpine-based container image), GoReleaser (cross-compilation)  
**Cloud Platform**: AWS S3 (binary distribution), Docker Hub (container images)  
**CI/CD**: GitHub Actions (11 workflows), GoReleaser, Cosign  
**Monitoring**: Trivy (container scanning), govulncheck (Go vulnerability checking), GitGuardian (secret detection)  
**Testing**: testify, gotest.tools, golangci-lint (20+ linters)  
**Proxy Support**: NTLM (sspi), Kerberos (gokrb5)  
**SCM Integrations**: GitHub, GitLab, Azure DevOps, Bitbucket Cloud, Bitbucket Server
