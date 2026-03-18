---
name: Configuration and Parameter Management
description: How configuration, environment variables, and parameters are handled
type: project
---

## Configuration System Overview

The CLI uses a three-tier configuration approach:

```
Environment Variables → Flags/CLI Args → Config File → Defaults
```

**Priority** (highest to lowest):
1. Environment variables (via `viper.BindEnv()`)
2. Command-line flags
3. Configuration file (YAML/TOML)
4. Default values

## Viper Configuration Framework

Uses `github.com/spf13/viper` for configuration management:

```go
// In main.go
bindProxy()
bindKeysToEnvAndDefault()
err := configuration.LoadConfiguration()
if err != nil {
    exitIfError(err)
}

// Later access
scans := viper.GetString(params.ScansPathKey)
```

## Parameters Package (`internal/params/`)

Defines all configuration keys and environment variable mappings:

### Key Structure
```go
// Define parameter key
const ScansPathKey = "scans-path"

// Define environment binding
type EnvVarBind struct {
    Key     string  // Viper key
    Env     string  // Environment variable name
    Default string  // Default value
}

var EnvVarsBinds = []EnvVarBind{
    {ScansPathKey, "CX_SCANS_PATH", "/api/scans"},
    {GroupsPathKey, "CX_GROUPS_PATH", "/api/groups"},
    // ... many more
}
```

### Common Configuration Keys

#### API Endpoints
- `ScansPathKey` - Scans API endpoint
- `ResultsPathKey` - Results API endpoint
- `ProjectsPathKey` - Projects API endpoint
- `GroupsPathKey` - Groups API endpoint
- `ApplicationsPathKey` - Applications API endpoint

#### Feature Paths
- `FeatureFlagsKey` - Feature flags API endpoint
- `PolicyEvaluationPathKey` - Policy evaluation endpoint
- `SastMetadataPathKey` - SAST metadata endpoint
- `CodeBashingPathKey` - Code training materials
- `BflPathKey` - Best fix location API

#### Authentication
- Usually configured via:
  - `CX_AST_USERNAME` / `CX_AST_PASSWORD` - Username/password auth
  - `CX_CLIENT_ID` / `CX_CLIENT_SECRET` - OAuth credentials
  - `CX_BASE_URI` - API base URL
  - `CX_BASE_AUTH_URI` - Auth endpoint URL

#### Network/Proxy
- `ProxyKey` - Proxy configuration
- Environment bindings: `CX_PROXY`, `PROXY`, `proxy`

#### Special Features
- `KicsContainerNameKey` - KICS container name for IaC scanning
- `RealtimeScannerPathKey` - Real-time scanner endpoint
- `DastEnvironmentsPathKey` - DAST environments

## Configuration Loading (`wrappers/configuration/`)

```go
// configuration.LoadConfiguration()
// 1. Loads from environment
// 2. Validates required fields
// 3. Sets up Viper bindings
// 4. Loads config file (if exists)
```

### Configuration File Paths
- User home directory: `~/.checkmarx/`
- Current directory: `.checkmarx/`
- Environment variable: `CX_CONFIG_PATH`

Supports formats:
- YAML (.yaml, .yml)
- TOML (.toml)
- JSON (.json)

### Example Configuration File
```yaml
# ~/.checkmarx/config.yaml
scans-path: "/api/scans"
results-path: "/api/results"
projects-path: "/api/projects"
base-uri: "https://ast.checkmarx.net"
base-auth-uri: "https://auth.checkmarx.net"
```

## Environment Variable Binding Pattern

### In Bind Function
```go
func bindKeysToEnvAndDefault() {
    for _, b := range params.EnvVarsBinds {
        // Bind env var to viper key
        err := viper.BindEnv(b.Key, b.Env)
        if err != nil {
            exitIfError(err)
        }
        // Set default value
        viper.SetDefault(b.Key, b.Default)
    }
}
```

### Proxy Configuration (Special Case)
```go
func bindProxy() {
    // Bind multiple env var names to one key
    err := viper.BindEnv(
        params.ProxyKey,
        params.CxProxyEnv,    // CX_PROXY
        params.ProxyEnv,      // PROXY
        params.ProxyLowerCaseEnv, // proxy
    )

    // Set as environment variable for HTTP client
    os.Setenv(params.ProxyEnv, viper.GetString(params.ProxyKey))
}
```

## Accessing Configuration at Runtime

### In Commands
```go
// Access via viper
baseURI := viper.GetString(params.BaseURIKey)
apiKey := viper.GetString(params.APIKeyKey)

// With defaults
timeout := viper.GetInt(params.TimeoutKey) // Uses default if not set
verbose := viper.GetBool(params.VerboseKey)

// Check if set
if viper.IsSet(params.SomeKey) {
    value := viper.GetString(params.SomeKey)
}
```

### Common Pattern
```go
func (cmd *Command) RunE(cobraCmd *cobra.Command, args []string) error {
    // Get config
    username := viper.GetString(params.UsernameKey)
    password := viper.GetString(params.PasswordKey)

    // Validate
    if username == "" || password == "" {
        return errors.New("username and password required")
    }

    // Use
    result, err := wrapper.Authenticate(ctx, username, password)
    return err
}
```

## Multi-Tenant Configuration

Some wrappers support multi-tenant scenarios:

```go
// Tenant-specific endpoints
tenantConfig := viper.GetString(params.TenantConfigurationPathKey)

// Can be overridden per command
// --tenant myTenant or CX_TENANT env var
```

## Configuration Validation

### At Startup
```go
// configuration.LoadConfiguration() validates:
// - Required fields present
// - Valid URLs format
// - Required credentials available
// - File permissions for sensitive files
```

### Per Command
Commands often validate specific config:
```go
// In scan command
if baseURL := viper.GetString(params.BaseURIKey); baseURL == "" {
    return errors.New("base-uri is required")
}
```

## Common Environment Variables

| Variable | Purpose | Example |
|----------|---------|---------|
| `CX_AST_USERNAME` | Auth username | user@example.com |
| `CX_AST_PASSWORD` | Auth password | secret123 |
| `CX_CLIENT_ID` | OAuth client ID | uuid-value |
| `CX_CLIENT_SECRET` | OAuth secret | secret-value |
| `CX_BASE_URI` | API base URL | https://ast.checkmarx.net |
| `CX_BASE_AUTH_URI` | Auth endpoint | https://auth.checkmarx.net |
| `CX_TENANT` | Tenant name | my-tenant |
| `CX_PROXY` | HTTP proxy | http://proxy:8080 |
| `CX_DEBUG` | Debug mode | true/false |
| `CX_VERBOSE` | Verbose output | true/false |
| `CX_CONFIG_PATH` | Config file location | /etc/checkmarx/config.yaml |

## Flag Patterns

Commands use Cobra flags to accept configuration:

```go
cmd.Flags().StringVar(&username, "username", "", "Username for authentication")
cmd.Flags().StringVar(&password, "password", "", "Password for authentication")
cmd.Flags().BoolVar(&debug, "debug", false, "Enable debug output")
cmd.MarkFlagRequired("username")
cmd.MarkFlagRequired("password")
```

### Global Flags
Some flags are global across all commands:
- `--debug` - Debug logging
- `--verbose` - Verbose output
- `--base-uri` - Override API endpoint
- `--config` - Config file path

## Configuration Best Practices

1. **Never Hardcode Secrets** - Use environment variables
2. **Provide Defaults** - Make CLI usable without config file
3. **Validate Early** - Check config at startup
4. **Document Environment** - List all env vars users need
5. **Support Multiple Paths** - Config file + env vars + flags
6. **Mask Secrets** - Never log passwords or API keys
7. **Precedence Clear** - Document which takes priority

## Special Handling

### Signal Handling
Configuration for signal handling:
```go
// SIGTERM causes cleanup
signalChanel := make(chan os.Signal, 1)
signal.Notify(signalChanel, syscall.SIGTERM)

// Used to clean up KICS container on exit
kicsContainerName := viper.GetString(params.KicsContainerNameKey)
```

### Docker Integration
Some features detect Docker:
```go
// For container scanning
if canAccessDocker() {
    containerResolver := NewContainerResolver()
    // Use for scanning
}
```
