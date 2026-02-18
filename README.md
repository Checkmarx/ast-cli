<img src="https://raw.githubusercontent.com/Checkmarx/ci-cd-integrations/main/.images/banner.png">
<br />
<div  align="center" >

[![Documentation][documentation-shield]][documentation-url]
[![Contributors][contributors-shield]][contributors-url]
[![Forks][forks-shield]][forks-url]
[![Docker Pulls][docker-shield]][docker-url]
[![Stargazers][stars-shield]][stars-url]
[![Issues][issues-shield]][issues-url]
[![MIT License][license-shield]][license-url]

</div>

<!-- PROJECT  LOGO -->
<br />
<p align="center">
  <a href="">
    <img src="https://raw.githubusercontent.com/Checkmarx/ci-cd-integrations/main/.images/logo.png" alt="Logo" width="80" height="80">
  </a>

<h3 align="center">Checkmarx One CLI</h3>

<p align="center">
    Checkmarx CLI is a standalone Checkmarx tool. 
<br />
    <a href="https://checkmarx.com/resource/documents/en/34965-68620-checkmarx-one-cli-tool.html"><strong>Explore the docs Â»</strong></a>
    <br />
    <br />
    <a href="https://github.com/Checkmarx/ast-cli/issues/new/choose">Report Bug</a>
    Â·
    <a href="https://github.com/Checkmarx/ast-cli/issues/new/choose">Request Feature</a>
</p>



<!-- TABLE OF CONTENTS -->
<details>
  <summary>Table of Contents</summary>
  <ol>
   <li><a href="#getting-started">Getting Started</a></li>
   <li><a href="#releases">Releases</a></li>
   <li><a href="#compile">Compile</a></li>
   <li><a href="#repository-context">Repository Context</a></li>
   <li><a href="#contribution">Contribution</a></li>
   <li><a href="#license">License</a></li>
   <li><a href="#cli-integrations">CLI Integrations</a></li>
   <li><a href="#contact">Contact</a></li>
  </ol>
</details>


## Getting Started

Refer to the [Documentation](https://checkmarx.com/resource/documents/en/34965-68620-checkmarx-one-cli-tool.html) for CLI commands and usage.

## Releases
For the latest CLI release, please locate your platform download [here](https://github.com/Checkmarx/ast-cli/releases).

You can also use the relevant link to download the latest version of the CLI:
* [Windows x64](https://download.checkmarx.com/CxOne/CLI/latest/ast-cli_windows_x64.zip)
* [MacOS x64](https://download.checkmarx.com/CxOne/CLI/latest/ast-cli_darwin_x64.tar.gz)
* [Linux x64](https://download.checkmarx.com/CxOne/CLI/latest/ast-cli_linux_x64.tar.gz)
* [Linux ARMv6](https://download.checkmarx.com/CxOne/CLI/latest/ast-cli_linux_armv6.tar.gz)
* [Linux ARM64](https://download.checkmarx.com/CxOne/CLI/latest/ast-cli_linux_arm64.tar.gz)

## Compile

To be able to build the code you should have:
* Go - You can download and install Go using this [link](https://golang.org/doc/install).

#### Windows
``` powershell
setx GOOS=windows 
setx GOARCH=amd64
go build -o ./bin/cx.exe ./cmd
```

#### Linux

``` bash
export GOARCH=amd64
export GOOS=linux
go build -o ./bin/cx ./cmd
```

#### Macintosh

``` bash
export GOOS=darwin 
export GOARCH=amd64
go build -o ./bin/cx-mac ./cmd
```
### Makefile
For ease of use, a Makefile is provided to build the project for all platforms.

Install Make for Mac: https://formulae.brew.sh/formula/make

Install Make for Windows: https://sourceforge.net/projects/gnuwin32/files/make/3.81/make-3.81.exe/download

Run the following command to build the project:
``` make build ``` 

## Repository Context

The following repository context is distilled from `AGENTS.md` and `CLAUDE.md`.

### Project Structure
- `src/agent/` - Workflow orchestration, state machine graph, and node implementations
- `src/tools/` - LLM, repository, git, and Go tool wrappers used by the agent
- `src/config/` - Runtime configuration and environment-based settings
- `src/observability/` - LangSmith tracing integration
- `tests/` - Test suite with mocked subprocess, file I/O, and LLM interactions
- `repocontext/` - Local context files for the target repo (not checked in)

### Architecture Overview
- Core runtime is a LangGraph `StateGraph` with node flow: `interpret -> load_context -> plan -> [approval_node] -> implement -> validate -> git_ops -> summary`.
- State is managed in `AgentState` (`TypedDict`) with plan/build/test/risk Pydantic models.
- `LLMClient` prefers Codex CLI subprocess execution and falls back to API providers.
- State is persisted to `.agent_state.json` after node execution to support resume.

### Build / Test / Lint Commands
``` bash
pytest
pytest tests/test_llm_client.py
pytest --cov=src --cov-report=term-missing
black src tests
ruff check src tests
python run_agent.py suggest "Add feature X"
python run_agent.py approve "Fix bug Y" --branch feature/fix-y
python run_agent.py auto "Implement Z" --config config.json
```

### Coding Conventions
- Load configuration from `.env` (or `--config`) using `AgentConfig.from_env()`.
- Route tool operations through `ToolExecutor` for retries, failure classification, and tracing.
- Parse JSON-oriented LLM responses with `_parse_llm_json_response()` helpers.
- Keep node behavior deterministic and trace node entry/exit consistently.
- In tests, mock subprocess/LLM/file operations; avoid real external API or CLI calls.
- Follow `AGENTS.md` skill workflow: load only the needed skill docs and references.

### CI Checks
- Primary CI validation runs `pytest` and coverage reporting.
- Current baseline in context docs: ~588 tests with ~89% coverage.
- Formatting/lint checks run `black` and `ruff check`.

### Key Files
- `src/agent/orchestrator.py`
- `src/agent/graph.py`
- `src/agent/nodes.py`
- `src/agent/state.py`
- `src/agent/models.py`
- `src/tools/llm_client.py`
- `src/tools/executor.py`
- `src/tools/repository.py`
- `src/tools/git_tools.py`
- `src/tools/go_tools.py`
- `src/cli.py`

## Contribution
We appreciate feedback and contribution to the CLI! Before you get started, please see the following:

- [Checkmarx contribution guidelines](docs/contributing.md)
- [Checkmarx Code of Conduct](docs/code_of_conduct.md)


## License
Distributed under the [Apache 2.0](LICENSE). See `LICENSE` for more information.

## CLI Integrations
Find all Checkmarx One CLI integrations [here](https://github.com/Checkmarx/ci-cd-integrations#checkmarx-ast-integrations).


## Contact
Checkmarx One Integrations Team

Project Link: [https://github.com/Checkmarx/ast-cli](https://github.com/Checkmarx/ast-cli).

Â© 2025 Checkmarx Ltd. All Rights Reserved.


[docker-shield]: https://img.shields.io/docker/pulls/checkmarx/ast-cli
[docker-url]:https://hub.docker.com/r/checkmarx/ast-cli
[documentation-shield]: https://img.shields.io/badge/docs-viewdocs-blue.svg
[documentation-url]:https://checkmarx.com/resource/documents/en/34965-68620-checkmarx-one-cli-tool.html
[contributors-shield]: https://img.shields.io/github/contributors/Checkmarx/ast-cli.svg
[contributors-url]: https://github.com/Checkmarx/ast-cli/graphs/contributors
[forks-shield]: https://img.shields.io/github/forks/Checkmarx/ast-cli.svg
[forks-url]: https://github.com/Checkmarx/ast-cli/network/members
[stars-shield]: https://img.shields.io/github/stars/Checkmarx/ast-cli.svg
[stars-url]: https://github.com/Checkmarx/ast-cli/stargazers
[issues-shield]: https://img.shields.io/github/issues/Checkmarx/ast-cli.svg
[issues-url]: https://github.com/Checkmarx/ast-cli/issues
[license-shield]: https://img.shields.io/github/license/Checkmarx/ast-cli.svg
[license-url]: https://github.com/Checkmarx/ast-cli/blob/main/LICENSE



