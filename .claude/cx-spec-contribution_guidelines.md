---
name: Contribution Guidelines and PR Process
description: Guidelines for contributing, branch naming, and PR process
type: feedback
---

**Why:** Clear contribution process ensures code quality and maintainability. Following these patterns keeps the codebase consistent and reviewable.

**How to apply:** Use these patterns when creating PRs, implementing features, or fixing bugs.

## PR Requirements

### Before Creating PR
1. **Search for existing issues** - Avoid duplicate work
2. **Open issue first** - Issues must be opened and accepted by Checkmarx team before PR
3. **Link to issue** - PR must reference associated issue number
4. **One concern per PR** - Fix bugs OR add features, not both
5. **Minimal changes** - Only modify what's necessary

### PR Content Standards
- **Only fix/add the stated functionality** - Don't refactor unrelated code
- **Minimal changed lines** - Address single concern in least possible changes
- **Include tests** - Unit tests or integration tests for all new/changed functionality
- **Update docs** - If behavior changes, update CLAUDE.md or README
- **Follow existing patterns** - Match code style of nearby code

## Branch Naming Convention

### Feature Branches
```
feature/<issue-number>-descriptive-name
```

Examples:
```
feature/AST-123-add-scan-filtering
feature/456-improve-error-handling
```

### Hotfix Branches
```
hotfix/<issue-number>-descriptive-name
```

Examples:
```
hotfix/789-fix-auth-bug
hotfix/101-security-update
```

## Git Workflow

### Standard Fork-and-Pull Workflow
1. Fork repository to your GitHub account
2. Clone to local machine: `git clone https://github.com/yourname/ast-cli.git`
3. Add upstream: `git remote add upstream https://github.com/Checkmarx/ast-cli.git`
4. Create feature branch: `git checkout -b feature/123-description`
5. Commit changes: `git commit -m "descriptive message"`
6. Push to fork: `git push origin feature/123-description`
7. Open PR in main repository
8. Link to accepted issue in PR description
9. Address review feedback
10. Merge when approved

### Keeping Fork Updated
```bash
# Fetch from upstream
git fetch upstream

# Rebase your branch
git rebase upstream/main

# Push updated branch
git push origin feature/123-description --force-with-lease
```

## Commit Message Standards

### Format
```
<type>(<scope>): <subject>

<body>

<footer>
```

### Types
- `feat:` - New feature
- `fix:` - Bug fix
- `refactor:` - Code refactoring (no functional change)
- `test:` - Test additions or updates
- `docs:` - Documentation changes
- `style:` - Code style changes (formatting, unused imports)
- `chore:` - Build, CI/CD, dependency updates

### Example
```
feat(scan): add result filtering by severity

Implement filtering mechanism to show only high/critical vulnerabilities.
Adds new --filter-severity flag to scan results command.

Closes #123
```

### Guidelines
- Use imperative mood: "add feature" not "added feature"
- First line max 50 characters
- Reference issues: "Closes #123" or "Fixes #456"
- Provide context in body for non-obvious changes

## PR Template

Use the standard PR template (`.github/PULL_REQUEST_TEMPLATE.md`):

```markdown
## Description
Brief description of what the PR does

## Related Issue
Fixes #123 (must link to accepted issue)

## Type of Change
- [ ] Bug fix
- [ ] New feature
- [ ] Enhancement
- [ ] Breaking change

## Testing
- [ ] Unit tests added
- [ ] Integration tests added
- [ ] Tested locally
- [ ] All tests pass

## Checklist
- [ ] Code follows style guidelines
- [ ] Comments/docs updated
- [ ] No new warnings
- [ ] Dependent changes merged and published
```

## Code Review Standards

### Before Requesting Review
1. **Run all checks**: `make fmt && make vet && make lint`
2. **Run all tests**: `go test ./...` and `go test -tags integration ./...`
3. **Build locally**: `make build` succeeds
4. **No secrets**: Ensure no API keys, passwords in code

### Review Expectations
- Checkmarx team reviews for:
  - Code quality and patterns
  - Test coverage
  - Security implications
  - Documentation accuracy
  - Adherence to contribution guidelines

### Addressing Feedback
- Address each comment
- Commit new changes (don't amend existing commits)
- Re-request review
- Respond to comments to acknowledge changes

## Testing Requirements

### Unit Tests Required
- For all new functions/methods
- For bug fixes (ensure bug is caught by test)
- For refactoring (no test changes unless fixing)

### Integration Tests
- For new commands
- For API integrations
- Optional for pure utility functions

### Test Quality
- Use table-driven tests for multiple scenarios
- Test both success and error cases
- Clean up resources (temp files, etc.)
- No flaky tests (must be reliable)

## Documentation Requirements

### Code Comments
- Exported functions must have comment
- Complex logic should be explained
- No obvious comments ("i = 0 // set i to zero")

### Commit Messages
- Explain "why", not just "what"
- Include issue references
- Provide context for understanding change

### PR Description
- Link to issue
- Describe what changed
- Explain why (motivation)
- How to test it

## PR Labeling

Automatic labeling via `.github/pr-labeler.yml`:
- Labels applied based on file patterns
- Review suggested labels for correctness

Common labels:
- `type/bug` - Bug fixes
- `type/feature` - New features
- `type/enhancement` - Enhancements
- `status/review-needed` - Awaiting review
- `status/blocked` - Blocked on something

## Common Pitfalls to Avoid

### ❌ Don't:
- Create PR without linked issue
- Mix multiple concerns in one PR
- Refactor unrelated code
- Skip tests
- Hardcode secrets or credentials
- Force-push after PR created (use rebase carefully)
- Ignore code review feedback

### ✓ Do:
- Link to issue in PR
- Keep changes focused
- Follow existing patterns
- Add tests
- Use environment variables for secrets
- Respond to all review comments
- Run full test suite before requesting review

## Release Process

### Releases Use
- Semantic versioning: `v1.2.3`
- Release notes generated from commits
- Multi-platform builds via goreleaser
- Published to:
  - GitHub releases
  - Docker Hub
  - Download center

### Your PR in Release
- Link to issue in description
- Clear commit message
- Proper type prefix
- Included in release notes if merged

## Getting Help

- Comment on issue for clarification
- Ask in PR for code review help
- Check documentation: CLAUDE.md, README.md
- Look at similar PRs for pattern examples
