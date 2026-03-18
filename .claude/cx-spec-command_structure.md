---
name: Cobra Command Structure and Organization
description: How CLI commands are organized and structured
type: project
---

## Command Hierarchy

Commands are organized hierarchically with Cobra framework:

```
root command (NewAstCLI in root.go)
├── scan            (scan.go) - Initiate scans
├── auth            (auth.go) - Authentication and validation
├── result          (result.go) - Query and display scan results
├── project         (project.go) - Project management
├── application     (asca/) - Application management
├── group           (groups.go) - User group management
├── chat            (chat*.go) - AI-powered chat for results
├── pre-commit      (pre_commit.go) - Git pre-commit hook
├── pre-receive     (pre-receive.go) - Git pre-receive hook
├── dast            (dast/) - DAST-specific commands
├── export          (util/export.go) - Export scan results
├── policy          (policymanagement/) - Policy enforcement
├── version         (version.go) - Version info
├── completion      (util/completion.go) - Shell completion
└── ... (other commands)
```

## Command Implementation Pattern

Each command typically:

1. **Define** using Cobra's `&cobra.Command{}`:
   - `Use` - Command name and args
   - `Short` - Brief description
   - `Long` - Detailed help with examples
   - `RunE` - Handler function

2. **Validate Input**:
   - Check required flags
   - Validate parameter values
   - Return early with clear error messages

3. **Call Service Layer**:
   - Extract parameters from viper/flags
   - Call appropriate service method
   - Handle service errors

4. **Format Output**:
   - Use printer utilities from `util/printer/`
   - Handle different output formats (JSON, table, SARIF, etc.)
   - Write to provided io.Writer for testability

## Example Command Structure

```go
func NewScanCommand(wrapper ScanWrapper) *cobra.Command {
    cmd := &cobra.Command{
        Use:   "scan",
        Short: "Create and manage scans",
        RunE: func(cmd *cobra.Command, args []string) error {
            // Validate
            if err := validateFlags(); err != nil {
                return err
            }

            // Execute
            result, err := wrapper.ScanCreate(ctx, params)
            if err != nil {
                return err
            }

            // Format output
            return printer.PrintScan(result)
        },
    }

    cmd.Flags().StringVar(&flagVar, "flag-name", "default", "Description")
    cmd.MarkFlagRequired("flag-name")

    return cmd
}
```

## Key Command Files

### Core Commands
- `root.go` - Root command setup, command tree definition
- `auth.go` - Authentication validation
- `scan.go` - Scan creation and management (LARGEST file ~150KB+)
- `result.go` - Result querying and display (~110KB+)
- `project.go` - Project management

### Specialized Commands
- `pre_commit.go` - Pre-commit hook integration
- `pre-receive.go` - Pre-receive hook integration
- `chat-sast.go`, `chat-kics.go` - AI chat features
- `hooks.go` - Hook management

### Real-time Engines
- `containers-realtime-engine.go`
- `iac-realtime-engine.go`
- `oss-realtime-engine.go`
- `secrets-realtime.go`

### Utility Subdirectories
- `util/` - Helper functions and utilities
- `util/printer/` - Output formatting
- `dast/` - DAST-specific commands
- `scarealtime/` - SCA real-time features
- `policymanagement/` - Policy commands

## Flag Patterns

Commands commonly use:
- `--debug` - Enable debug logging
- `--format` - Output format (json, sarif, table, etc.)
- `--output-file` - Write results to file
- `--filter` - Filter results
- `--tenant` or `--cx-tenant` - Multi-tenant support

## Testing Commands

Commands are tested via integration test helper:
```go
err, buffer := executeCommand(t, "scan", "create", "--project-name", "test")
```

This allows testing command parsing, validation, and output without full integration.
