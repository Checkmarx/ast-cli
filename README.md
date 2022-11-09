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

© 2022 Checkmarx Ltd. All Rights Reserved. 


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
