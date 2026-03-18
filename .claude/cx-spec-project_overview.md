---
name: Checkmarx AST CLI Project Overview
description: Core project details, purpose, and scope
type: project
---

**Checkmarx AST CLI** is a standalone command-line tool for security scanning built in Go.

## Purpose
Provides a comprehensive CLI for managing security scans and operations through the Checkmarx One platform, supporting multiple scanning engines and integration scenarios.

## Tech Stack
- **Language**: Go 1.25.8
- **Framework**: Cobra (CLI framework)
- **Configuration**: Viper (config management)
- **Testing**: testify/assert, custom integration test harness
- **Build**: Makefile, goreleaser for multi-platform releases
- **Linting**: golangci-lint with strict rules

## Supported Scanning Types
- SAST (Static Application Security Testing)
- SCA (Software Composition Analysis)
- DAST (Dynamic Application Security Testing)
- IaC (Infrastructure as Code)
- Container scanning
- OSS (Open Source Software)
- Secrets detection

## Multi-Platform Support
- Windows (x64)
- Linux (x64, ARMv6, ARM64)
- macOS (x64)
- Docker container support

## Key Integrations
- GitHub, GitLab, Azure DevOps, Bitbucket (Cloud & Server)
- Policy evaluation and enforcement
- PR decoration for multiple platforms
- Real-time scanning engines
- AI-powered remediation and chat features
