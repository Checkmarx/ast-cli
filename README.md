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


<!-- PROJECT LOGO -->
<br />
<p align="center">
  <a href="">
    <img src="https://raw.githubusercontent.com/Checkmarx/ci-cd-integrations/main/.images/logo.png" alt="Logo" width="80" height="80">
  </a>

<h3 align="center">AST CLI</h3>

<p align="center">
    Checkmarx CLI is a standalone Checkmarx tool.
<br />
    <a href="https://checkmarx.atlassian.net/wiki/spaces/AST/pages/2445443121/CLI+Tool"><strong>Explore the docs »</strong></a>
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
    <li>
      <a href="#about-the-project">About The Project</a>
    </li>
    <li>
      <a href="#getting-started">Getting Started</a>
      <ul>
        <li><a href="#prerequisites">Prerequisites</a></li>
        <li><a href="#setting-up">Setting Up</a></li>
      </ul>
    </li>
    <li><a href="#usage">Usage</a></li>
    <li><a href="#contributing">Contributing</a></li>
    <li><a href="#license">License</a></li>
    <li><a href="#contact">Contact</a></li>
  </ol>
</details>



<!-- ABOUT THE PROJECT -->
## About The Project

The tool is a fully functional Command Line Interface (CLI) that interacts with the Checkmarx CxAST server.

The tool is able to perform all the functions that the REST APIs support, so the CLI users can perform all the tasks that are related to managing the Checkmarx CxAST server.

The CLI tool supports the following actions:

- Manage Checkmarx projects (create / delete / show)
- Manage Checkmarx scanning (create / cancel / delete / show)
  Display scan results

The CLI also acts as the backbone for all the supported plugins. All the plugins use the CLI tool to initiate scans. This allows minimal updates to the plugins which decreases the need for constant updates and feature requests.

The tool is universal that can manage all the CxAST scan types (CxSAST, CxSCA, KICS, etc.).


<!-- GETTING STARTED -->
## Getting Started

Installing the CLI tool is very simple.

### Prerequisites

To be able to build the code you should have:
* Go
You can download and install Go using this link: https://golang.org/doc/install

### Setting Up
### Windows
``` powershell
setx GOOS=windows 
setx GOARCH=am
go build -o ./bin/cx.exe ./cmd
```

### Linux

``` bash
export GOARCH=amd64
export GOOS=linux
go build -o ./bin/cx ./cmd
```

### Macintosh

``` bash
export GOOS=darwin 
export GOARCH=amd64
go build -o ./bin/cx-mac ./cmd
```


## Usage

To see how you can use our tool, please refer to the [Documentation](https://checkmarx.atlassian.net/wiki/spaces/AST/pages/2445443121/CLI+Tool)


## Contribution

We appreciate feedback and contribution to the CLI! Before you get started, please see the following:

- [Checkmarx contribution guidelines](docs/contributing.md)
- [Checkmarx Code of Conduct](docs/code_of_conduct.md)


<!-- LICENSE -->
## License
Distributed under the [Apache 2.0](LICENSE). See `LICENSE` for more information.


<!-- CONTACT -->
## Contact

Checkmarx - AST Integrations Team

Project Link: [https://github.com/Checkmarx/ast-cli](https://github.com/Checkmarx/ast-cli)

Find more integrations from our team [here](https://github.com/Checkmarx/ci-cd-integrations#checkmarx-ast-integrations)

© 2022 Checkmarx Ltd. All Rights Reserved. 


[docker-shield]: https://img.shields.io/docker/pulls/checkmarx/ast-cli
[docker-url]:https://hub.docker.com/r/checkmarx/ast-cli
[documentation-shield]: https://img.shields.io/badge/docs-viewdocs-blue.svg
[documentation-url]:https://checkmarx.atlassian.net/wiki/spaces/AST/pages/2967766116/CxAST+Plugins
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
