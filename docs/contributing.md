## Contributing

Hi! Thank you for your interest in contributing to the CLI!

Besides code, we also welcome your help in:

- identifying new bugs
- improving the documentation

Please make sure you check the issues page to confirm the feature/bug you want to work on is not already in progress.

By participating and contributing to the project, you agree to uphold our [Code of Conduct](code_of_conduct.md).

## Getting started

1. Fork the CLI repo on GitHub
2. Create you branch with a descriptive name
3. Perform your changes
4. Validate your changes according to the validation section
5. Commit your changes
6. Submit a PR on GitHub

## Validation of changes

Before submitting a PR, please make sure you performed the following steps:

- linted the code and fixed any warning thrown

`golangci-lint run ./...`

- added/updated unit and integration tests for your changes
- updated the documentation to reflect any changes when necessary
- updated the cli help when changed the command structure

