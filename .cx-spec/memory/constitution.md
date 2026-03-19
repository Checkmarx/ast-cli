# Checkmarx AST CLI Constitution

## Core Principles

### I. CLI-First Interface

Every feature must be exposed as a Cobra command or subcommand with consistent flag conventions. Text in/out protocol: arguments and flags as input, structured output to stdout (JSON, table, list formats), errors to stderr. All commands must support `--format` for output flexibility and integrate with Viper for configuration binding.

### II. Wrapper Abstraction Layer

All external API interactions go through typed wrapper interfaces in `internal/wrappers/`. Each wrapper defines a Go interface, an HTTP implementation, and a mock implementation for testing. New integrations must follow this pattern — no direct HTTP calls from command handlers.

### III. Test Coverage (NON-NEGOTIABLE)

Integration tests in `test/integration/` validate end-to-end CLI behavior. Unit tests co-locate with source files using `_test.go` conventions. Mock wrappers in `internal/wrappers/mock/` enable isolated testing without external dependencies. All new commands and wrappers must include corresponding test coverage.

### IV. Security-Aware Development

As a security tooling product, the CLI must never log or expose credentials, tokens, or sensitive scan data in plain text. Authentication flows use JWT with proper token lifecycle management. Proxy and NTLM support must be maintained for enterprise environments.

### V. Modular Package Architecture

The codebase follows a strict internal package structure: `commands/` for CLI handlers, `wrappers/` for API abstractions, `params/` for configuration keys, `services/` for business logic, `logger/` for output control. Cross-package dependencies must flow downward — commands depend on wrappers, never the reverse.

## Additional Constraints

- **Go version**: Track the version specified in `go.mod` (currently Go 1.25.8)
- **Dependency management**: All dependencies managed via Go modules; new dependencies require justification and security review
- **Backward compatibility**: CLI flag names and output formats are public API; breaking changes require major version bumps
- **Platform support**: Must build and run on Linux, macOS, and Windows; platform-specific code must be clearly isolated
- **Error handling**: Use structured error types (`wrappers.AstError`) with meaningful exit codes; never panic in production paths

## Development Workflow

- **Branch naming**: Feature branches follow the pattern `ast-{ticket}-{description}`
- **Code review**: All changes require PR review before merge; wrapper interface changes require additional scrutiny
- **Integration testing**: Integration tests run against live Checkmarx One environments; mock-based tests for CI pipelines
- **Commit standards**: Conventional commits preferred (`feat:`, `fix:`, `docs:`, `refactor:`, `test:`)
- **Configuration**: Runtime configuration via Viper with environment variable bindings defined in `params/`

## Governance

This constitution supersedes all other development practices for the Checkmarx AST CLI project. Amendments require documented justification, team review, and an updated version entry below. All PRs must verify compliance with these principles. Complexity in wrapper patterns or command structures must be justified by corresponding test coverage and documentation.

**Version**: 1.0.0 | **Ratified**: 2026-03-19 | **Last Amended**: 2026-03-19
