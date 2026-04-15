# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Checkmarx One CLI (`cx`) — a standalone command-line interface for the Checkmarx One application security platform. It orchestrates SAST, SCA, KICS (IaC), API Security, Container Security, Software Supply Chain Security, and DAST scans, manages projects/applications, retrieves results in multiple formats (JSON, SARIF, PDF, SBOM), and decorates pull requests across GitHub, GitLab, Azure DevOps, and Bitbucket. Licensed under Apache 2.0.

**The CLI is the backbone for all Checkmarx One plugins.** All plugins (IDE extensions, CI/CD integrations) wrap this CLI to initiate scans rather than calling APIs directly. This means changes here propagate to the entire plugin ecosystem — minimizing per-plugin updates but requiring careful backward compatibility.

### Key Capabilities

- **Project management:** Create, delete, list, show projects; manage tags
- **Scanning:** SAST, SCA, IaC Security, Container Security, API Security, Supply Chain Security
- **Results:** Retrieve, filter, export in JSON/SARIF/PDF/SBOM; CodeBashing training links; engine-specific exit codes
- **Utilities:** Shell auto-completion (bash/zsh/fish/PowerShell), environment variable display, contributor counting across SCMs (90-day window), PR decoration, SCA remediation (npm)
- **Integration:** CI/CD pipelines (Jenkins, Azure DevOps, GitHub Actions), IDEs, ServiceNow, Jira, Slack, Teams


### Plugin Ecosystem Context

The CLI sits at the center of the Checkmarx One plugin architecture. Downstream consumers that wrap this CLI:
- **Language wrappers:** ast-cli-java-wrapper, ast-cli-javascript-wrapper
- **IDE plugins:** VS Code, JetBrains (IntelliJ), Eclipse, Visual Studio extensions
- **CI/CD plugins:** Jenkins, Azure DevOps, TeamCity, GitHub Action, Maven plugin

All plugins invoke the `cx` binary rather than calling Checkmarx APIs directly. This centralizes API interaction logic but means CLI flag changes, exit code changes, or output format changes can break downstream consumers.

### Layered Design: Commands → Services → Wrappers → HTTP Client

1. **`cmd/main.go`** — Entry point. Instantiates all 30+ wrapper implementations via viper config keys and injects them into `commands.NewAstCLI()`.
2. **`internal/commands/`** — Cobra command definitions. Each command constructor accepts wrapper interfaces as parameters (e.g., `NewScanCommand(...wrappers...)`). Commands are registered in `root.go` via `rootCmd.AddCommand(...)`.
3. **`internal/services/`** — Business logic layer for multi-step operations (e.g., export, application management).
4. **`internal/wrappers/`** — HTTP abstractions for Checkmarx One APIs. Each service follows a triple-file pattern:
   - Interface definition (e.g., `scans.go`)
   - HTTP implementation (e.g., `scans-http.go`)
   - Mock implementation in `mock/` subdirectory (e.g., `mock/scans-mock.go`)
5. **`internal/wrappers/client.go`** — Central HTTP client with OAuth2 auth, token caching, retry with exponential backoff, proxy support (Basic/NTLM/Kerberos/SSPI), TLS config, and request/response logging.
6. **`internal/wrappers/grpcs/`** — gRPC client for the ASCA (Abstract Syntax Code Analysis) engine running on localhost.

### Adding a New API Integration

- Define the wrapper interface in `internal/wrappers/`
- Create HTTP implementation in a `-http.go` file
- Create mock in `internal/wrappers/mock/`
- Add the wrapper parameter to `NewAstCLI()` in `internal/commands/root.go` and instantiate it in `cmd/main.go`
- Command constructors follow the pattern: `func NewXyzCommand(wrapper wrappers.XyzWrapper, ...) *cobra.Command`

## Common User Flow

1. **Configure:** `cx configure --apikey <key>` or set `CX_CLIENT_ID` + `CX_CLIENT_SECRET` env vars
2. **Create project:** `cx project create --name "MyProject"`
3. **Initiate scan:** `cx scan create --project-name "MyProject" --branch "main" --file-source "path/to/code" --scan-types "sast,sca,iac-security"`
4. **Retrieve results:** `cx results show --scan-id "<id>" --report-format "json"` (supports json, sarif, pdf, sbom)
5. **Integrate findings:** PR decoration, IDE feedback, CI/CD pipeline exit codes

Scans can target local directories (`--file-source`), Git repos (`--repo-url`), or container images (`--container-image`). Multiple engines can run simultaneously via `--scan-types`.

## Repository Structure

```
cmd/                          Entry point (main.go) — wrapper instantiation & DI
internal/
  commands/                   Cobra command definitions (~20 top-level commands)
    asca/                     AI-powered code analysis subcommand
    dast/                     DAST environment management
    util/                     Shared utilities, printer, user count
    .scripts/                 CI test runner scripts (up.sh, integration_up.sh)
  services/                   Business logic (projects, applications, groups, export)
  wrappers/                   HTTP API abstractions (30+ wrapper interfaces)
    mock/                     Mock implementations for unit testing
    grpcs/                    gRPC clients (ASCA engine)
    configuration/            Config file loading (~/.checkmarx/checkmarxcli.yaml)
    bitbucketserver/          Bitbucket Server-specific wrapper
  params/                     CLI flags, env vars, viper keys, bindings
    flags.go                  Flag name constants
    envs.go                   Environment variable names (CX_* prefix)
    keys.go                   Viper configuration keys
    binds.go                  Flag-to-env-var bindings
  constants/
    errors/                   Domain-specific error constants
    exit-codes/               Engine-specific exit codes for CI/CD
  logger/                     Logging with sensitive data sanitization
test/
  integration/                Integration tests (//go:build integration)
  cleandata/                  Post-test cleanup utilities
```

## Technology Stack

- **Language:** Go 1.25.8
- **CLI Framework:** Cobra v1.10.1 + Viper v1.20.1
- **gRPC:** google.golang.org/grpc v1.79.3 + protobuf v1.36.10
- **Auth:** golang-jwt/jwt/v5 v5.2.2, gokrb5/v8 (Kerberos), alexbrainman/sspi (Windows NTLM)
- **Container Analysis:** containers-resolver, containers-images-extractor, anchore/syft (SBOM)
- **Testing:** stretchr/testify v1.11.1, gotest.tools
- **Linting:** golangci-lint v2 with 19 enabled linters
- **Release:** GoReleaser, Cosign (image signing), gon (macOS notarization)
- **No database** — the CLI is stateless; all persistence is server-side


### Build

```bash
# Build (runs fmt → vet → build)
make build

# Individual steps
go fmt ./...
go vet ./...
go build -o bin/cx.exe ./cmd    # Windows
go build -o bin/cx ./cmd        # Linux/macOS

# Initial CLI configuration
cx configure   # Interactive prompt for base-uri, tenant, credentials
```

**Config file location:** `~/.checkmarx/checkmarxcli.yaml` (override with `--config-file-path` or `CX_CONFIG_FILE_PATH`). Uses file locking (`gofrs/flock`) for thread-safe writes.

### Running Tests

```bash
# Run all unit tests (excludes mock, wrappers, bitbucketserver, logger packages)
# Add -v for verbose output
go test $(go list ./... | grep -v "mock" | grep -v "wrappers" | grep -v "bitbucketserver" | grep -v "logger") -timeout 25m

# Run tests for a specific package
go test ./internal/commands/ -v
go test ./internal/services/ -v

# Run a specific test by name
go test ./internal/commands/ -run TestAuthValidate -v

# Run tests matching a pattern
go test ./internal/commands/ -run TestScan -v    # Runs all tests starting with "TestScan"

# Run integration tests (requires env vars — see "Integration Test Configuration" below)
go test -tags integration -v -timeout 210m github.com/checkmarx/ast-cli/test/integration

# Run a specific integration test
go test -tags integration -run TestScanCreate -v -timeout 210m github.com/checkmarx/ast-cli/test/integration
```

### Coverage

```bash
# Generate coverage report (console summary)
go test $(go list ./... | grep -v "mock" | grep -v "wrappers" | grep -v "bitbucketserver" | grep -v "logger") -timeout 25m -coverprofile cover.out
go tool cover -func cover.out                          # Show per-function coverage
go tool cover -func cover.out | grep total             # Show total coverage percentage

# Generate detailed HTML coverage report
go tool cover -html=cover.out -o coverage.html

# View coverage in browser
go tool cover -html=cover.out                          # Opens browser automatically

# Integration test coverage (specific packages only)
go test -tags integration -v -timeout 210m \
  -coverpkg github.com/checkmarx/ast-cli/internal/commands,github.com/checkmarx/ast-cli/internal/services,github.com/checkmarx/ast-cli/internal/wrappers \
  -coverprofile cover-integration.out \
  github.com/checkmarx/ast-cli/test/integration
```

### Linting & Static Analysis

```bash
# Run full linter suite (19 linters, see .golangci.yml)
golangci-lint run -c .golangci.yml
# or
make lint

# Run Go vet only
go vet ./...

# Run vulnerability scanner
govulncheck ./...
```

### Pre-Commit Checklist

Always run before committing:

```bash
go mod tidy
go vet ./...
go test -v $(go list ./... | grep -v "mock" | grep -v "wrappers" | grep -v "bitbucketserver" | grep -v "logger") -timeout 25m
golangci-lint run -c .golangci.yml
```

### Integration Test Configuration

Integration tests require these environment variables (set via `.env` file or IDE run configuration):

| Variable | Purpose |
|---|---|
| `CX_BASE_URI` | Checkmarx One API base URL |
| `CX_BASE_AUTH_URI` | Keycloak authentication URL |
| `CX_CLIENT_ID` | OAuth2 client ID |
| `CX_CLIENT_SECRET` | OAuth2 client secret |
| `CX_APIKEY` | API key (for API key auth tests) |
| `CX_TENANT` | Tenant name |
| `CX_AST_USERNAME` | Username (for basic auth tests) |
| `CX_AST_PASSWORD` | Password (for basic auth tests) |
| `CX_SCAN_SSH_KEY` | SSH key (for SSH scan tests) |
| `PERSONAL_ACCESS_TOKEN` | GitHub/GitLab token (for SCM tests) |
| `PR_GITHUB_TOKEN`, `PR_GITLAB_TOKEN`, `AZURE_TOKEN` | PR decoration tests |
| `PROXY_HOST`, `PROXY_PORT`, `PROXY_USERNAME`, `PROXY_PASSWORD` | Proxy tests |

No `.env.example` file exists — refer to `.github/workflows/ci-tests.yml` (lines 54-96) for the full list of required secrets.

## Coding Standards

- **Flag naming:** kebab-case (e.g., `--scan-id`, `--client-secret`)
- **Environment variables:** `CX_` prefix in SCREAMING_SNAKE_CASE (e.g., `CX_BASE_URI`, `CX_CLIENT_ID`)
- **Max function length:** 200 lines / 100 statements (`funlen`)
- **Max cyclomatic complexity:** 15 (`gocyclo`)
- **Max line length:** 185 characters (`lll`)
- **Magic numbers:** Flagged by `mnd` linter (except in test files)
- **Duplicate code threshold:** 500 tokens (`dupl`)
- **Import restrictions:** `depguard` limits imports to stdlib + explicitly whitelisted packages in `.golangci.yml`
- **Formatters:** `gofmt` and `goimports` enforced

## Project Rules

- All wrapper dependencies must be injected through constructor parameters — never instantiate wrappers inside commands directly
- Every new wrapper needs all three files: interface, HTTP implementation, and mock
- When adding a wrapper, update both `NewAstCLI()` signature in `internal/commands/root.go` AND instantiation in `cmd/main.go`
- Viper config precedence (highest to lowest): CLI flags (`BindPFlag` in `root.go`) > env vars (`BindEnv` in `binds.go`) > config file > defaults
- Build with `CGO_ENABLED=0` for static linking (no C dependencies)
- Exit codes are engine-specific (SAST=2, SCA=3, KICS=4, API Security=5, multiple=1) — do not change these as CI/CD pipelines and downstream plugins depend on them
- The `depguard` linter will reject any import not on the allowlist — add new external packages to `.golangci.yml` before using them
- **Do not break CLI flag names, output formats, or exit codes without coordinating with the plugin ecosystem** — Java wrapper, JavaScript wrapper, and all IDE/CI plugins parse CLI output and depend on stable interfaces
- Dependabot PRs are auto-merged after CI passes — do not add manual approval gates to dependency update workflows

## Testing Strategy

### Coverage Thresholds

- **Unit tests:** **85%** minimum (CI-enforced)
- **Integration tests:** **75%** minimum (CI-enforced)
- **CI pipeline order:** Unit tests → Integration tests → Lint (`golangci-lint`) → Vulnerability scan (`govulncheck`) → Docker image scan (Trivy)

### Test File Creation Rules

- Test files **must** end with `_test.go` and live in the same package as the code being tested
- **Unit tests in `internal/commands/`** must include `//go:build !integration` as the first line — this prevents them from running during integration test builds. Tests in other packages (`services/`, `wrappers/`) do not need this tag.
- **Integration tests must include** `//go:build integration` as the first line and live in `test/integration/`
- Each integration test file corresponds to a specific wrapper/service (e.g., `scan_test.go` for scanning)
- Test data files go in `test/integration/data/` (contains Dockerfiles, source files, ZIP archives, sample results)

### Unit Test Patterns

Unit tests use mock wrappers from `internal/wrappers/mock/`. The key setup function is `createASTTestCommand()` in `internal/commands/root_test.go`, which instantiates all mock wrappers and returns a fully-wired Cobra command:

```go
//go:build !integration

func TestMyFeature(t *testing.T) {
    err := executeTestCommand(createASTTestCommand(), "scan", "create", "--project-name", "test")
    assert.NilError(t, err)
}
```

**Helper functions** (defined in `internal/commands/root_test.go`):
- `createASTTestCommand()` — creates a CLI command with all mock wrappers injected
- `executeTestCommand(cmd, args...)` — executes command, resets viper after
- `execCmdNilAssertion(t, args...)` — execute and assert no error
- `execCmdNotNilAssertion(t, args...)` — execute and assert error returned
- `executeRedirectedTestCommand(args...)` — capture stdout to buffer

**Assertion libraries:** `gotest.tools/assert` (primary: `assert.NilError()`, `assert.Assert()`, `assert.ErrorContains()`) and `stretchr/testify/assert`

Both packages have a `TestMain(m *testing.M)` function for setup/teardown in `internal/commands/root_test.go` and `test/integration/root_test.go`.

### Integration Test Patterns

Integration tests use real HTTP wrappers (not mocks) and hit the **Canary environment** (`deu.ast.checkmarx.net`). The key setup function is `createASTIntegrationTestCommand(t)` in `test/integration/util_command.go`:

```go
//go:build integration

func TestScanCreate(t *testing.T) {
    cmd := createASTIntegrationTestCommand(t)
    err := execute(cmd, "scan", "create", "--project-name", projectName, "-b", "main", "-s", ".")
    assert.NilError(t, err)
}
```

**Helper functions** (defined in `test/integration/util_command.go`):
- `createASTIntegrationTestCommand(t)` — creates CLI command with real HTTP wrappers
- `executeWithTimeout(cmd, timeout, args...)` — execute with context deadline and automatic retry/proxy flags
- `flag(f)` — adds `--` prefix to flag name

**Test utilities** (defined in `test/integration/util.go`):
- `getProjectNameForTest()` / `GenerateRandomProjectNameForScan()` — unique project names
- `WriteProjectNameToFile(name)` — persist names for cleanup
- `formatTags(tags)` / `formatGroups(groups)` — format test data

**Flaky test handling:** Integration test CI script (`internal/commands/.scripts/integration_up.sh`) includes automatic retry logic for flaky tests and uses `gocovmerge` to merge coverage profiles from retried runs.


## External Integrations

| Integration | Protocol | Purpose |
|---|---|---|
| Checkmarx One API | REST/HTTPS | Scans, results, projects, applications, groups, policies, feature flags |
| ASCA Engine | gRPC (localhost) | AI-powered real-time code analysis |
| GitHub API | REST v3 | PR decoration, org/repo/commit fetching |
| GitLab API | REST v4 | PR decoration, project/commit fetching |
| Azure DevOps API | REST v5.0 | PR decoration, repo/project fetching |
| Bitbucket Cloud | REST | PR decoration, workspace/repo fetching |
| Bitbucket Server | REST v1.0 | PR decoration, project/repo fetching |
| SCA Realtime | HTTPS (`api-sca.checkmarx.net`) | Public endpoint for package vulnerability checks |
| Checkmarx Gen-AI | REST | AI chat for remediation guidance |
| Keycloak | OAuth2/OIDC | Token-based authentication |

## Deployment

- **Platforms:** Linux (amd64, arm, arm64), Windows (amd64), macOS (universal binary)
- **Formats:** tar.gz (Linux), ZIP (Windows), DMG (macOS), Docker image, Homebrew tap
- **Docker image:** Based on `checkmarx/bash:5.3` (Alpine), runs as `nonroot` user, SHA256-pinned base
- **Docker registries:** `checkmarx/ast-cli` and `cxsdlc/ast-cli` on Docker Hub
- **Signing:** Windows via AWS CloudHSM + osslsigncode, macOS via Apple Developer ID + gon notarization, Docker via Cosign
- **Release tool:** GoReleaser (`.goreleaser.yml` for production, `.goreleaser-dev.yml` for pre-releases)
- **Artifacts stored in:** GitHub Releases + AWS S3 (`CxOne/CLI/{tag}` and `CxOne/CLI/latest`)
- **Release workflow:** `.github/workflows/release.yml` — triggered via `workflow_dispatch` (manual) or `workflow_call` (from upstream). Runs on macOS 15 Intel.

### Release Process

1. Build binaries for all platforms (CGO_ENABLED=0, static linking)
2. Sign Windows binary (remote HSM signing), notarize macOS binary (gon)
3. Build and push Docker images (production only)
4. Sign Docker images with Cosign and verify signatures (production only)
5. Publish to GitHub Releases, S3, and Homebrew tap
6. **Notify** Microsoft Teams and JIRA with release info (production only)
7. **Dispatch auto-release** to downstream plugin repositories to update their CLI version

### CI/CD Workflows (`.github/workflows/`)

| Workflow | Purpose |
|---|---|
| `ci-tests.yml` | Unit tests, integration tests, lint, govulncheck, Trivy |
| `release.yml` | Full build, sign, publish, notify pipeline |
| `issue_automation.yml` | Auto-label and assign issues |
| `pr-automation.yml` | Auto-assign reviewers, enforce PR guidelines |
| `checkmarx-one-scan.yml` | Security scan on PRs |
| `update-trivy.yml` | Keep Trivy vulnerability definitions current |

### JIRA Integration

- JIRA tasks are automatically tagged with the release version upon deployment
- JIRA ticket numbers in PR titles (e.g., `AST-3432`) enable automation — see Contributing section for PR requirements

## Performance Considerations

- **HTTP timeout:** Default 5 seconds (configurable via `--timeout` / `CX_TIMEOUT`)
- **Connection pooling:** Persistent `KeepAlive` at 30 seconds via `http.Transport`
- **Token caching:** In-memory JWT cache with 10-second grace period before expiry; protected by `sync.Mutex`
- **Retry with exponential backoff:** Default 3 retries, 20s base delay. IAM retries: 4 attempts, 500ms base. Polling delay: 60s.
- **Rate limiting:** SCM-specific rate limit handling (GitHub, GitLab, Bitbucket, Azure) with 60s default wait and up to 3 retries
- **HTTP body logging cap:** Request/response bodies >1MB are not logged to prevent memory issues
- **Static binary:** `CGO_ENABLED=0` with `-s -w` flags strips symbols for smaller binaries

## API / Endpoints / Interfaces

All API paths are configured via environment variables with `CX_` prefix. Key endpoints:

| Category | Env Var | Description |
|---|---|---|
| Core | `CX_BASE_URI` | Main Checkmarx One API URL |
| Auth | `CX_BASE_AUTH_URI` | Keycloak token endpoint |
| Scans | `CX_SCANS_PATH` | Scan CRUD operations |
| Results | `CX_RESULTS_PATH` | Scan result retrieval |
| Projects | `CX_PROJECTS_PATH` | Project management |
| Uploads | `CX_UPLOADS_PATH` | File upload (supports multipart via pre-signed URLs) |
| Feature Flags | `CX_FEATURE_FLAGS_PATH` | Feature gating per tenant |
| Export | `CX_EXPORT_PATH` | Bulk export operations |
| Reports | `CX_RESULTS_PDF_REPORT_PATH`, `CX_RESULTS_JSON_REPORT_PATH` | PDF/JSON report generation |

Full list of env vars in `internal/params/envs.go`. Viper bindings in `internal/params/binds.go`.

## Security & Access

- **Authentication methods:** OAuth2 client credentials (`CX_CLIENT_ID` + `CX_CLIENT_SECRET`) or API key (`CX_APIKEY` — a JWT refresh token)
- **Token endpoint:** `{base_auth_uri}/auth/realms/{tenant}/protocol/openid-connect/token`
- **Token refresh:** Automatic on 401 responses; cached in-memory with mutex protection
- **TLS:** `--insecure` flag disables certificate verification (for testing only)
- **Sensitive data sanitization:** Logger automatically replaces values of `CX_APIKEY`, `CX_CLIENT_ID`, `CX_CLIENT_SECRET`, `--username`, `--password`, `ast-token`, `--ssh-key`, `CX_HTTP_PROXY`, `--token`, and `SCS_REPO_TOKEN` with `***` in all output
- **Config obfuscation:** Interactive `cx configure` shows only last 4 characters of sensitive values

## Proxy Configuration

The CLI supports four methods to configure proxy settings (in order of typical usage):

### 1. Standard Environment Variables

```bash
export HTTP_PROXY=http://username:password@proxy.example.com:8080
export HTTPS_PROXY=http://username:password@proxy.example.com:8080
export NO_PROXY=localhost,127.0.0.1,*.example.com
```

### 2. CX-Specific Environment Variables

| Variable | Purpose |
|---|---|
| `CX_HTTP_PROXY` | Proxy server URL (overrides `HTTP_PROXY`) |
| `CX_PROXY_AUTH_TYPE` | Authentication type: `basic`, `ntlm`, `kerberos`, or `kerberos-native` |
| `CX_PROXY_NTLM_DOMAIN` | Windows domain for NTLM auth |
| `CX_PROXY_KERBEROS_SPN` | Kerberos SPN (e.g., `HTTP/proxy.example.com`) — required for Kerberos |
| `CX_PROXY_KERBEROS_KRB5_CONF` | Path to krb5.conf (default: `/etc/krb5.conf` or `C:\Windows\krb5.ini`) |
| `CX_PROXY_KERBEROS_CCACHE` | Path to Kerberos credential cache (optional) |
| `CX_IGNORE_PROXY` | Bypass proxy for current command |

### 3. CLI Global Flags

```bash
cx scan create --project-name "MyProject" \
  --proxy http://username:password@proxy.example.com:8080 \
  --proxy-auth-type ntlm \
  --proxy-ntlm-domain CORP
```

Flags: `--proxy`, `--proxy-auth-type` (basic/ntlm/kerberos/kerberos-native), `--proxy-ntlm-domain`, `--proxy-kerberos-spn`, `--ignore-proxy`

Proxy URL format: `http://<proxy_ip>:<port>` or `http://<user>:<pass>@<proxy_ip>:<port>` (must include `http://` prefix).

### 4. CLI Config File

Settings in `~/.checkmarx/checkmarxcli.yaml`:

```yaml
cx_http_proxy: http://proxy.example.com:8080
cx_proxy_auth_type: ntlm
cx_proxy_ntlm_domain: dm.cx
```

## Logging

- **Debug mode:** `--debug` flag enables verbose output to stdout
- **File logging:** `--log-file <dir>` writes to `<dir>/ast-cli.log` (file only)
- **File + console:** `--log-file-console <dir>` writes to both file and stdout
- **HTTP tracing (debug mode):** Connection timing, DNS lookups, TLS handshake, full request/response dumps (sanitized, capped at 1MB, binary data excluded)
- **All log output** passes through `sanitizeLogs()` which replaces credential values with `***`

## Debugging Steps

### Runtime Debugging

1. **Enable debug output:** Add `--debug` to any command (e.g., `cx scan create --debug ...`)
2. **Check auth:** `cx auth validate --debug` — shows token fetch, credential type, tenant resolution
3. **Inspect HTTP traffic:** Debug mode logs full request/response including headers, status codes, and timing
4. **Proxy issues:** Debug mode logs proxy type detection ("Creating HTTP Client with Proxy", "Using NTLM Proxy", "Using Kerberos Proxy")
5. **Token problems:** Look for "Fetching API access token", "Using cached API access token", "API access token not found in cache" in debug output
6. **Log to file for analysis:** `cx <command> --log-file-console /tmp/logs` then inspect `/tmp/logs/ast-cli.log`
7. **Exit codes:** Non-zero exit identifies the failing engine (2=SAST, 3=SCA, 4=KICS, 5=API Security, 1=multiple)
8. **Config verification:** `cx utils env` shows current environment variable values
9. **Version check:** `cx version` to verify installed version and identify version-related issues
10. **Help:** `cx help` or `cx <command> --help` for usage and available flags

### Source Code Debugging

1. Set environment variables via `.env` file or IDE run configuration (`CX_BASE_URI`, `CX_CLIENT_ID`, `CX_CLIENT_SECRET`, etc.)
2. Set the CLI command as program arguments in your IDE (e.g., `scan create --project-name "Test" -b main -s <path>`)
3. Use **Delve** debugger to set breakpoints and step through code
4. Alternatively, add `fmt.Println` statements for quick variable inspection
5. Build and test: `go build -o bin/cx ./cmd` (Mac/Linux) or `go build -o bin/cx.exe ./cmd` (Windows)

### Debugging Integration Tests

1. Open `test/integration/` and select the test file for the functionality being debugged
2. Configure environment variables in your IDE run configuration
3. **Important:** Comment out `appendProxyArgs` call in `executeWithTimeout` function in `test/integration/util_command.go` before running locally — this avoids proxy-related failures in local debugging
4. Set breakpoints and run/debug the test; adjust test arguments to simulate scenarios
5. **Do NOT commit the `appendProxyArgs` change** — it is for local debugging only

## Contributing

- Branch naming: `feature/<issue#>-name` or `hotfix/<issue#>-name`
- PRs require associated issue marked "accepted"
- All changes need unit or integration tests meeting coverage thresholds
- Post-release, a dispatch triggers downstream plugin repositories to update their CLI version

### PR Requirements — Team Members

- Must have a valid JIRA ID in the PR title (e.g., `AST-3432`) — enables JIRA automation and traceability
- CI workflow must pass: **unit-tests**, **integration-tests**, **lint**, **cx-scan**
- Review by at least one team member
- Use the PR template (`.github/PULL_REQUEST_TEMPLATE.md`) — includes sections for changes, related issues, testing, and checklist

### PR Requirements — Dependabot

- CI workflow must pass: **unit-tests**, **integration-tests**, **lint**, **cx-scan**
- Auto-merged after CI passes (no manual review required)
