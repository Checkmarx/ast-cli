[![Contributors][contributors-shield]][contributors-url]
[![Forks][forks-shield]][forks-url]
[![Stargazers][stars-shield]][stars-url]
[![Issues][issues-shield]][issues-url]
[![MIT License][license-shield]][license-url]



<!-- PROJECT LOGO -->
<br />
<p align="center">
  <a href="">
    <img src="./logo.png" alt="Logo" width="80" height="80">
  </a>

<h3 align="center">AST-CLI</h3>

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
</p>



<!-- TABLE OF CONTENTS -->
<details open="open">
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
****

<!-- GETTING STARTED -->
## Getting Started

Installing the CLI tool is very simple.

### Prerequisites

To be able to build the code you should have:
* Go
 ```
  You can download and install Go using this link: https://golang.org/doc/install
```

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

** **


## Usage

To see how you can use our tool, please refer to the [Documentation](https://checkmarx.atlassian.net/wiki/spaces/AST/pages/2445443121/CLI+Tool)


## Contribution

We appreciate feedback and contribution to the CLI! Before you get started, please see the following:

- [Checkmarx contribution guidelines](docs/contributing.md)
- [Checkmarx Code of Conduct](docs/code_of_conduct.md)

** **

<!-- LICENSE -->
## License
Distributed under the [Apache 2.0](LICENSE). See `LICENSE` for more information.



<!-- CONTACT -->
## Contact

Checkmarx - AST Integrations Team

Project Link: [https://github.com/Checkmarx/ast-cli](https://github.com/Checkmarx/ast-cli)


© 2021 Checkmarx Ltd. All Rights Reserved.

<!-- MARKDOWN LINKS & IMAGES -->
<!-- https://www.markdownguide.org/basic-syntax/#reference-style-links -->
[contributors-shield]: https://img.shields.io/github/contributors/Checkmarx/ast-cli.svg?style=flat-square
[contributors-url]: https://github.com/Checkmarx/ast-cli/graphs/contributors
[forks-shield]: https://img.shields.io/github/forks/Checkmarx/ast-cli.svg?style=flat-square
[forks-url]: https://github.com/Checkmarx/ast-cli/network/members
[stars-shield]: https://img.shields.io/github/stars/Checkmarx/ast-cli.svg?style=flat-square
[stars-url]: https://github.com/Checkmarx/ast-cli/stargazers
[issues-shield]: https://img.shields.io/github/issues/Checkmarx/ast-cli.svg?style=flat-square
[issues-url]: https://github.com/Checkmarx/ast-cli/issues
[license-shield]: https://img.shields.io/github/license/Checkmarx/ast-cli.svg?style=flat-square
[license-url]: https://github.com/Checkmarx/ast-cli/blob/master/LICENSE
[product-screenshot]: images/screenshot.png