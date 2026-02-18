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
    <a href="https://checkmarx.com/resource/documents/en/34965-68620-checkmarx-one-cli-tool.html"><strong>Explore the docs »</strong></a>
    <br />
    <br />
    <a href="https://github.com/Checkmarx/ast-cli/issues/new/choose">Report Bug</a>
    ·
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

### Project Overview
- **Checkmarx One CLI** (`cx`) is a Go-based command-line tool for interacting with the Checkmarx One application security platform.
- Supports SAST, SCA, IaC Security, Container Security, Secret Detection, API Security scans, project management, PR decoration (GitHub, GitLab, Bitbucket, Azure DevOps), real-time scanning, and GenAI capabilities.
- Module: `github.com/checkmarx/ast-cli`
- Go version: `1.24.x` (project currently uses `1.24.11`)

### Architecture and Module Layout
- Entrypoint: `cmd/main.go` wires all wrappers and creates the root Cobra command.
- `internal/commands`: Cobra command definitions and command-level tests.
- `internal/services`: business logic and realtime engine implementations (including `internal/services/realtimeengine` and `osinstaller`).
- `internal/wrappers`: API/client wrappers, protocol adapters, and mocks (`internal/wrappers/mock`).
- `internal/params`, `internal/constants`, `internal/logger`: shared config and utilities.
- Integration tests are in `test/integration`, docs are in `docs/`, and CI rules are in `.github/workflows/`.

### Build, Test, and Lint Commands
``` bash
make fmt
make vet
make build
make lint
go build -o ./bin/cx ./cmd
go test ./...
go test -tags integration ./test/integration
bash internal/commands/.scripts/up.sh
bash internal/commands/.scripts/integration_up.sh
```

### Coding Conventions
- Use Go `1.24.x` and run `gofmt`/`goimports` before opening a PR.
- Follow Go naming idioms: lowercase package names, `CamelCase` exported symbols, and descriptive file names.
- Keep complexity manageable to satisfy enforced linter rules (`funlen`, `gocyclo`, `errcheck`, `staticcheck`, `revive`, and related checks).
- Prefer constructor-based dependency injection (as in `cmd/main.go`) and avoid global state.

### CI Requirements
- CI runs unit tests, integration tests, lint (`golangci-lint`), `govulncheck`, and container image vulnerability scanning.
- Coverage floors enforced by CI are `77.7%` for unit and `75%` for integration.
- Ensure branch/PR naming rules and required checks are satisfied before merge.

### Contribution and Security Guidance
- Open/link an issue for significant work and include relevant tests and docs updates.
- When modifying API integrations, update wrapper interfaces and their mock implementations together.
- Use environment variables for credentials (`CX_*`, proxy, SCM tokens) and never commit secrets.
- Validate release-related changes against CI and security workflows.

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

© 2025 Checkmarx Ltd. All Rights Reserved.


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
