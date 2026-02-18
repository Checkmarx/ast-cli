# CLAUDE.md ΓÇö Checkmarx One CLI (ast-cli)

This file provides Claude with essential context for working on the `ast-cli` codebase.
It is read automatically at the start of every Claude Code session.

---

## Project Overview

**Checkmarx One CLI** (`cx`) is a Go-based command-line tool for interacting with the Checkmarx One application security platform. It supports SAST, SCA, IaC Security, Container Security, Secret Detection, and API Security scans, along with project management, PR decoration (GitHub, GitLab, Bitbucket, Azure DevOps), real-time scanning, and Gen AI features (chat, remediation).

**Module:** `github.com/checkmarx/ast-cli`
**Go version:** 1.24.11
**Binary output:** `bin/cx.exe`

---

## Architecture

```
ast-cli/
Γö£ΓöÇΓöÇ cmd/main.go                  # Entry point ΓÇö wires all wrappers, creates root Cobra command
Γö£ΓöÇΓöÇ internal/
Γöé   Γö£ΓöÇΓöÇ commands/                # Cobra command definitions (scan, project, configure, etc.)
Γöé   Γöé   Γö£ΓöÇΓöÇ asca/               # ASCA engine subcommand
Γöé   Γöé   Γö£ΓöÇΓöÇ util/               # Printer, user count, remediation helpers
Γöé   Γöé   ΓööΓöÇΓöÇ .scripts/           # Test runner scripts (up.sh, integration_up.sh)
Γöé   Γö£ΓöÇΓöÇ constants/              # Errors, exit codes, feature flags, file extensions
Γöé   Γö£ΓöÇΓöÇ logger/                 # Logging utilities
Γöé   Γö£ΓöÇΓöÇ params/                 # CLI flags, env variable binds, parameter keys, filters
Γöé   Γö£ΓöÇΓöÇ services/               # Business logic layer
Γöé   Γöé   Γö£ΓöÇΓöÇ realtimeengine/     # IaC, OSS, containers, secrets real-time scanners
Γöé   Γöé   ΓööΓöÇΓöÇ osinstaller/        # Platform-specific installation utilities
Γöé   ΓööΓöÇΓöÇ wrappers/               # HTTP client integrations (API wrappers)
Γöé       Γö£ΓöÇΓöÇ mock/               # Mock implementations for testing
Γöé       Γö£ΓöÇΓöÇ grpcs/              # gRPC/Proto definitions (ASCA)
Γöé       Γö£ΓöÇΓöÇ kerberos/           # Kerberos authentication
Γöé       Γö£ΓöÇΓöÇ ntlm/               # NTLM authentication
Γöé       Γö£ΓöÇΓöÇ bitbucketserver/    # Bitbucket Server integration
Γöé       ΓööΓöÇΓöÇ configuration/      # Config loading and management
Γö£ΓöÇΓöÇ test/integration/           # Integration tests (build tag: integration)
Γö£ΓöÇΓöÇ docs/                       # contributing.md, code_of_conduct.md
Γö£ΓöÇΓöÇ .github/workflows/          # CI pipelines
Γö£ΓöÇΓöÇ .golangci.yml               # Linter configuration
Γö£ΓöÇΓöÇ Makefile                    # Build targets
ΓööΓöÇΓöÇ go.mod
```

### Key architectural patterns

- **Wrapper pattern**: All external API communication goes through `internal/wrappers/` interfaces. Each wrapper has an HTTP implementation and a mock implementation for testing.
- **Cobra + Viper**: CLI structure uses `spf13/cobra` for command trees and `spf13/viper` for configuration, env vars, and flag binding.
- **Dependency injection via constructor**: `cmd/main.go` wires all wrapper instances and passes them to `commands.NewAstCLI(...)`. Never use global state; always inject dependencies.
- **No `pkg/` directory**: All application code lives under `internal/` (not exported).

---

## Build, Test, and Lint

### Build

```bash
make build        # runs: go fmt ΓåÆ go vet ΓåÆ go build -o bin/cx.exe ./cmd
```

### Test

```bash
# Unit tests (excludes mock/wrappers directories)
make vet          # runs: go fmt ΓåÆ go vet (default target)
go test ./...     # all unit tests

# Integration tests (requires Checkmarx backend + env vars)
go test -tags integration ./test/integration
```

- **Unit test coverage threshold:** 77.7%
- **Integration test coverage threshold:** 75%
- Unit tests use mock wrappers from `internal/wrappers/mock/`
- Integration tests use the `//go:build integration` build tag

### Lint

```bash
make lint         # runs: golangci-lint run -c .golangci.yml
```

**Linter version:** golangci-lint 1.64.2

---

## Code Style and Conventions

### Go standards enforced by linter

The `.golangci.yml` enables a strict set of linters. Key constraints:

| Rule | Setting |
|------|---------|
| **Function length** | Max 200 lines / 100 statements (`funlen`) |
| **Cyclomatic complexity** | Max 15 (`gocyclo`) |
| **Code duplication** | Threshold 500 tokens (`dupl`) |
| **Magic numbers** | Flagged in arguments, cases, conditions, returns (`mnd`) ΓÇö exempt in `_test.go` |
| **Line length** | Max 185 characters (`lll`) |
| **Named returns** | Naked returns forbidden (`nakedret`) |
| **Error checking** | All errors must be checked (`errcheck`) |
| **Imports** | Must pass `depguard` ΓÇö only approved packages allowed |

### Import rules (depguard)

Only these external dependencies are approved for import in `main`:
- Go standard library (`$gostd`)
- `github.com/checkmarx/ast-cli/internal`
- Approved Checkmarx packages: `containers-resolver`, `manifest-parser`, `secret-detection`, `gen-ai-prompts`, `gen-ai-wrapper`, `2ms`, `containers-images-extractor`, `containers-types`
- Approved third-party: `cobra`, `viper`, `color`, `errors`, `google`, `heredoc`, `go-getport`, `testify/assert`, `flock`, `golang-jwt`

Do NOT add new external imports without explicitly checking this list.

### Formatting

- `go fmt` is always run before build and lint.
- `go vet` is always run before build.
- `goimports` is enabled ΓÇö imports are auto-organized.

### Naming and branching

- Branch naming: `feature/<issue#>-description` or `hotfix/<issue#>-description`
- Exit codes: `0` for success, `1` for general failure, custom codes via `internal/constants/exit-codes/`
- PRs must reference an accepted issue.

---

## Working Effectively in This Codebase

### When adding a new command

1. Create the command definition in `internal/commands/`
2. Define any new parameters in `internal/params/`
3. Create wrapper interfaces and HTTP implementations in `internal/wrappers/`
4. Add mock implementations in `internal/wrappers/mock/`
5. Wire the wrapper in `cmd/main.go` constructor chain
6. Add unit tests alongside the command file

### When modifying API integrations

- Always update both the wrapper interface and its mock
- HTTP wrappers live in `internal/wrappers/` with naming pattern `*HTTPWrapper`
- Mock wrappers live in `internal/wrappers/mock/`
- Test using mock wrappers ΓÇö never call real APIs in unit tests

### When working with real-time engines

- Real-time scanner implementations are in `internal/services/realtimeengine/`
- Each engine type (IaC, OSS, containers, secrets) has its own sub-package
- Container management logic handles engine lifecycle

---

## Prompting and Interaction Guidelines

*Synthesized from Anthropic's engineering best practices, context engineering research, and prompt evaluation frameworks.*

### Context management principles

- **Treat context as finite**: Every token in the context window competes for attention. Provide the smallest set of high-signal information needed for the task at hand.
- **Just-in-time retrieval**: Don't load entire files upfront. Use `grep`, `glob`, and targeted reads to find exactly what's needed before editing.
- **Progressive disclosure**: Start with file structure and signatures, then drill into specific implementations only when needed.
- **Structured note-taking**: For multi-step tasks, maintain a mental checklist of what's done and what remains. Use the todo tool for complex changes.

### Effective task decomposition

- **Start simple**: Attempt the simplest solution first. Only add complexity when it demonstrably improves the outcome.
- **Prompt chaining over monolithic prompts**: Break complex changes into sequential steps ΓÇö understand the code first, plan the change, implement, then verify.
- **Parallelization where safe**: Independent file reads, searches, and lint checks can run in parallel. Sequential operations (edit then test) must be ordered.
- **Evaluator-optimizer loop**: After implementing a change, review it critically. Check for linter errors, test failures, and edge cases before considering the task done.

### The 4D Framework for AI collaboration

When working on tasks in this codebase, apply:

1. **Delegation**: Assess whether a task is appropriate for AI assistance. Complex architectural decisions should be discussed with the team; routine implementation, refactoring, and test writing are ideal for AI.
2. **Description**: Provide clear, specific descriptions of what you want. Include file paths, function names, expected behavior, and constraints. The more precise the prompt, the better the output.
3. **Discernment**: Always review AI-generated code critically. Verify it follows the project's patterns (wrapper injection, Cobra command structure, linter compliance). Check that tests actually test meaningful behavior.
4. **Diligence**: Ensure changes are ethical, safe, and maintain code quality. Run `make lint` and `go test ./...` after changes. Don't skip error handling or testing.

### Writing effective prompts for this project

- **Be specific about scope**: "Add a new `list` subcommand under the `project` command that supports JSON and table output formats" is better than "add project listing".
- **Reference existing patterns**: "Follow the same pattern as `internal/commands/scan.go` for the new command structure" helps anchor to project conventions.
- **Include constraints**: "The function must stay under 200 lines to pass `funlen` and cyclomatic complexity must stay under 15" prevents linter failures.
- **Provide examples of expected output**: When asking for test generation, show an example of how existing tests in the project are structured.

### Tool and API design principles

When building or modifying CLI commands and wrapper interfaces:

- **Design for the agent-computer interface (ACI)**: Tool names, parameter names, and descriptions should be immediately obvious. If a human developer can't tell which wrapper to use in a given situation, the API needs clarification.
- **Self-contained tools**: Each wrapper should be robust to errors, return clear error messages, and not depend on global state.
- **Poka-yoke (mistake-proofing)**: Design parameters to prevent misuse ΓÇö use absolute paths over relative, enums over free strings, required parameters over optional with footgun defaults.
- **Token-efficient responses**: Wrapper methods should return structured data, not raw HTML or bloated JSON. Keep API responses lean.

### Building reliable agent workflows

- **Augmented LLM pattern**: The CLI itself is an augmented tool ΓÇö retrieval (API calls), tools (scan engines), and memory (configuration/viper). When extending it, maintain this clean separation.
- **Orchestrator-workers for complex operations**: The multi-wrapper architecture in `cmd/main.go` already follows this pattern. The CLI orchestrates; wrappers do focused work.
- **Error recovery**: Always handle errors explicitly. Use the `wrappers.AstError` type for structured error codes. Never silently swallow errors.
- **Ground truth validation**: After making changes, verify against real behavior ΓÇö run the linter, run tests, check that the binary builds.

### Prompt evaluation and quality assurance

- **Define success criteria upfront**: Before implementing, be clear about what "done" looks like ΓÇö all tests pass, linter is clean, the command produces correct output.
- **Use code-graded evaluation**: The existing test infrastructure (`go test`, `golangci-lint`) serves as automated evaluation. Always run it.
- **Iterate on failure modes**: If a change breaks tests or linting, analyze the root cause. Don't patch symptoms ΓÇö fix the underlying issue.
- **Regression awareness**: When modifying shared code (wrappers, params, constants), check what other commands depend on it. Use `grep` to find all callers before changing interfaces.

---

## CI Pipeline

The GitHub Actions CI runs these checks:

1. **unit-tests** ΓÇö `go test` with coverage >= 77.7%
2. **integration-tests** ΓÇö `go test -tags integration` with coverage >= 75%
3. **lint** ΓÇö `golangci-lint` v1.64.2 with `.golangci.yml`
4. **govulncheck** ΓÇö Go module vulnerability scanning
5. **checkDockerImage** ΓÇö Trivy scan for container vulnerabilities

All checks must pass before merging.

---

## Quick Reference

| Task | Command |
|------|---------|
| Format code | `go fmt ./...` |
| Vet code | `go vet ./...` |
| Build binary | `make build` |
| Run all unit tests | `go test ./...` |
| Run integration tests | `go test -tags integration ./test/integration` |
| Lint | `make lint` |
| Default make target | `make` (runs fmt + vet) |

---

## Key Files to Know

| File | Purpose |
|------|---------|
| `cmd/main.go` | Entry point ΓÇö all wrapper wiring happens here |
| `internal/commands/root.go` | Root Cobra command definition |
| `internal/params/keys.go` | All CLI parameter key constants |
| `internal/params/envs.go` | Environment variable bindings |
| `internal/wrappers/interface.go` | Core wrapper interfaces |
| `.golangci.yml` | Linter configuration (source of truth for code style) |
| `Makefile` | Build and lint targets |
| `docs/contributing.md` | Contribution guidelines |

---

## Sources

This document synthesizes guidance from:
1. [Context Engineering for AI Agents](https://www.anthropic.com/engineering/effective-context-engineering-for-ai-agents) ΓÇö context is finite; curate minimal high-signal tokens
2. [Building Effective Agents](https://www.anthropic.com/engineering/building-effective-agents) ΓÇö simple composable patterns over complex frameworks
3. [Claude Code Best Practices](https://www.anthropic.com/engineering/claude-code-best-practices) ΓÇö CLAUDE.md, hooks, agentic coding workflows
4. [AI Fluency Framework](https://anthropic.skilljar.com/ai-fluency-framework-foundations) ΓÇö the 4D Framework: Delegation, Description, Discernment, Diligence
5. [Build with Claude](https://docs.anthropic.com/en/home) ΓÇö API reference, SDK, developer platform
6. [Claude for Work](https://www.anthropic.com/learn/claude-for-work) ΓÇö team productivity, projects, skills
7. [Anthropic API Fundamentals](https://github.com/anthropics/courses/blob/master/anthropic_api_fundamentals/README.md) ΓÇö messages format, parameters, streaming
8. [Real-World Prompting](https://github.com/anthropics/courses/blob/master/real_world_prompting/README.md) ΓÇö practical prompting for complex tasks
9. [Prompt Evaluations](https://github.com/anthropics/courses/blob/master/prompt_evaluations/README.md) ΓÇö code-graded and model-graded evaluation strategies
