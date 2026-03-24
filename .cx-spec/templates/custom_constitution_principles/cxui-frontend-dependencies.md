### [CP:cxui_frontend_dependencies] CXUI Frontend Dependencies (NON-NEGOTIABLE)

For every new frontend React feature, these repositories and packages are mandatory context:
- [checkmarx-shared-ui](https://github.com/Checkmarx-CxUI/checkmarx-shared-ui) -> `@cxui/shared-ui`
- [cxui-react-scripts](https://github.com/Checkmarx-CxUI/cxui-react-scripts) -> `@cxui/react-scripts`
- [cxui-mfe-util](https://github.com/Checkmarx-CxUI/cxui-mfe-util) -> `@cxui/mfe-util`
- [cxui-api-generator](https://github.com/Checkmarx-CxUI/cxui-api-generator) -> `@cxui/api-generator`
- [cxui-cypress-util](https://github.com/Checkmarx-CxUI/cxui-cypress-util) -> `@cxui/cypress-util`

All repositories above MUST exist locally.

Before running `/cx-spec.clarify` or `/cx-spec.plan`, the AI MUST:
- Collect and validate absolute local paths for all five repositories.
- Read one architecture file from each repository (prefer `.cx-spec/memory/architecture.md`, fallback `architecture.md`).
- Use that architectural context during feature design and implementation.
- Record evidence of those five reads (repo path + file path used) in generated feature docs.

If any required repository/path/file is missing, the command MUST fail immediately and MUST NOT continue generation.
