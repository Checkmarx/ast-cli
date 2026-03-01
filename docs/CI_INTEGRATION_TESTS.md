# CI Integration Test Grouping Strategy

## Overview

The CI workflow uses a **parallelized test execution strategy** that divides integration tests into 12 logical groups, plus a catch-all group for uncategorized tests. This approach provides:

- **Faster CI execution**: Tests run in parallel across multiple runners
- **Better isolation**: Related tests run together, reducing conflicts
- **Easier debugging**: Failures are isolated to specific functional areas
- **Automatic validation**: A pre-flight job identifies which tests are covered by patterns
- **Safety net**: Uncovered tests still run in a catch-all group instead of being skipped

### How It Works

1. **Validation Job** (`validate-test-coverage`): Runs first to identify which tests match patterns and which don't
2. **Integration Tests Job** (`integration-tests`): Runs 13 parallel jobs (12 categorized groups + 1 catch-all)
3. **Merge Coverage Job** (`merge-coverage`): Combines coverage reports from all parallel jobs

If a test doesn't match any pattern, it will **run in the "Uncovered Tests" catch-all group** (Group 13). The CI will display a warning but will not fail. This ensures new tests are always executed, even if they haven't been properly categorized yet.

---

## Test Group Structure

### Group 1: Scan Creation (`scan-create`)

**Purpose**: Tests for creating scans, scan configuration, and scan type validation.

**Pattern**:
```regex
^Test(CreateScan|CreateAsyncScan|CreateQueryDescriptionLink|ScanCreate[^I]|ScanTypeApi|ScanTypesValidation|ValidateScanTypes|ScanGenerating|ScanWith|ScanList|ScanTimeout|ScanASCA|ExecuteASCAScan|ScansAPISecThreshold)
```

**Example Tests**:
- `TestCreateScan_WithValidClientCredentialsFlag_Success`
- `TestCreateAsyncScan_CallExportServiceBeforeScanFinishWithRetry_Success`
- `TestScanCreateEmptyProjectName`
- `TestScanTypesValidation`
- `TestScanTimeout`
- `TestExecuteASCAScan_ASCALatestVersionSetTrue_Success`

---

### Group 2: Scan Operations (`scan-ops`)

**Purpose**: Tests for scan workflows, E2E scenarios, incremental scans, and scan management operations.

**Pattern**:
```regex
^Test(ScansE2E|ScansUpdate|FastScan|LightQueries|RecommendedExclusions|IncrementalScan|BranchPrimary|CancelScan|ScanCreateInclude|ScanCreateIgnore|ScanWorkflow|ScanWorkFlow|ScanLogs|InvalidSource|ScanShow|RequiredScan|ScaResolver|BrokenLink|PartialScan|FailedScan|RunKics|RunSca|ScanGLReport|ContainerEngineScan)
```

**Example Tests**:
- `TestScansE2E`
- `TestIncrementalScan`
- `TestCancelScan`
- `TestScanWorkflow`
- `TestScanWorkFlowWithSastEngineFilter`
- `TestRunKicsScan`
- `TestContainerEngineScansE2E_ContainerImagesFlagAndScanType`

---

### Group 3: Results & Reports (`results`)

**Purpose**: Tests for scan results, reports generation, CodeBashing, and risk management.

**Pattern**:
```regex
^Test(Result|CodeBashing|RiskManagement)
```

**Example Tests**:
- `TestResultsExitCode_OnSuccessfulScan_ShouldReturnStatusCompleted`
- `TestResultsGeneratingPdfReportWithPdfOptions`
- `TestCodeBashingList`
- `TestRiskManagementResults_ReturnResults`

---

### Group 4: PR Decoration (`pr-decoration`)

**Purpose**: Tests for pull request decoration across different SCM platforms.

**Pattern**:
```regex
^TestPR
```

**Example Tests**:
- `TestPRGithubDecorationSuccessCase`
- `TestPRGitlabDecoration_WhenUseCodeRepositoryFlag_ShouldSuccess`
- `TestPRAzureDecorationFailure`
- `TestPRBBOnCloudDecorationSuccessCase`

---

### Group 5: Projects (`projects`)

**Purpose**: Tests for project creation, management, and configuration.

**Pattern**:
```regex
^Test(Project|CreateEmpty|CreateAlready|CreateWith|CreateProject|GetProjectByTagsFilter)
```

**Example Tests**:
- `TestProjectsE2E`
- `TestProjectCreate_ApplicationExists_CreateProjectSuccessfully`
- `TestCreateEmptyProjectName`
- `TestCreateAlreadyExistingProject`
- `TestGetProjectByTagsFilter_whenProjectHasNoneTags_shouldReturnProjectWithNoTags`

---

### Group 6: Predicates & BFL (`predicates`)

**Purpose**: Tests for predicates, Best Fix Location (BFL), triage, and result updates.

**Pattern**:
```regex
^Test(Predicate|Bfl|RunGetBfl|SastUpdate|GetAndUpdate|Triage|ScaUpdate)
```

**Example Tests**:
- `TestPredicateWithInvalidValues`
- `TestRunGetBflByScanIdAndQueryId`
- `TestSastUpdateAndGetPredicatesForSimilarityId`
- `TestTriageShowAndUpdateWithCustomStates`
- `TestScaUpdateWithVulnerabilityDetails`

---

### Group 7: Container Tests (`containers`)

**Purpose**: Tests for container scanning, image validation, and edge cases.

**Pattern**:
```regex
^Test(ContainerScan|ContainerImage|EmptyFolder)
```

**Example Tests**:
- `TestContainerScan_EmptyFolderWithExternalImages`
- `TestContainerScan_ErrorHandling_InvalidAndValidScenarios`
- `TestContainerImageValidation_ValidFormats`
- `TestContainerScan_TarFileValidation`

---

### Group 8: Realtime Scanning (`realtime`)

**Purpose**: Tests for realtime scanning engines (IaC, OSS, Secrets, Containers).

**Pattern**:
```regex
^Test(IacRealtime|OssRealtime|Secrets_Realtime|ContainersRealtime|EngineNameResolution|ScaRealtime)
```

**Example Tests**:
- `TestIacRealtimeScan_TerraformFile_Success`
- `TestOssRealtimeScan_PackageJsonFile_Success`
- `TestSecrets_RealtimeScan_TextFile_Success`
- `TestContainersRealtimeScan_PositiveDockerfile_Success`
- `TestEngineNameResolution_engine_NotFound`

---

### Group 9: Auth & Config (`auth-config`)

**Purpose**: Tests for authentication, configuration loading, and tenant settings.

**Pattern**:
```regex
^Test(Auth|LoadConfiguration|SetConfigProperty|GetTenant|FailProxy)
```

**Example Tests**:
- `TestAuthValidate`
- `TestAuthValidateClientAndSecret`
- `TestLoadConfiguration_EnvVarConfigFilePath`
- `TestSetConfigProperty_ConfigFilePathFlag`
- `TestGetTenantConfigurationSuccessCaseJson`
- `TestFailProxyAuth`

---

### Group 10: Root & Logs (`root-logs`)

**Purpose**: Tests for root commands, logging, feature flags, and download operations.

**Pattern**:
```regex
^Test(RootVersion|SetLogOutput|_DownloadScan|_HandleFeatureFlags|Main)
```

**Example Tests**:
- `TestRootVersion`
- `TestSetLogOutputFromFlag_DirPath_Success`
- `Test_DownloadScan_Logs_Success`
- `Test_HandleFeatureFlags_WhenCalled_ThenNoErrorAndCacheNotEmpty`
- `TestMain`

---

### Group 11: SCM Rate Limit & User Count (`scm-tests`)

**Purpose**: Tests for SCM platform rate limiting and user counting functionality.

**Pattern**:
```regex
^Test(GitHub|GitLab|Azure|Bitbucket|BitBucket)(RateLimit|UserCount|Count)
```

**Example Tests**:
- `TestGitHubRateLimit_SuccessAfterRetryOne`
- `TestGitLabRateLimit_SuccessAfterRetryOne`
- `TestBitBucketRateLimit_SuccessAfterRetryOne`
- `TestAzureRateLimit_SuccessAfterRetryOne`
- `TestGitHubUserCount`
- `TestBitbucketUserCountWorkspace`
- `TestAzureUserCountOrgs`

---

### Group 12: Miscellaneous (`misc`)

**Purpose**: Tests for chat, import, telemetry, remediation, pre-commit hooks, and other utilities.

**Pattern**:
```regex
^Test(Chat|Import|GetLearnMore|GetProjectName|Telemetry|Mask|FailedMask|ScaRemediation|KicsRemediation|HooksPreCommit|PreReceive|Pre_Receive)
```

**Example Tests**:
- `TestChatKicsInvalidAPIKey`
- `TestImport_ImportSarifFileWithCorrectFlags_CreateImportSuccessfully`
- `TestGetLearnMoreInformationSuccessCaseJson`
- `TestTelemetryAI`
- `TestMaskSecrets`
- `TestScaRemediation`
- `TestHooksPreCommitInstallAndUninstallPreCommitHook`
- `TestPreReceive_PushSecrets`

---

### Group 13: Uncovered Tests - Catch-All (`uncovered`)

**Purpose**: Safety net for tests that don't match any of the 12 categorized patterns. This group ensures new tests are always executed even if they haven't been properly categorized.

**Pattern**: Dynamically generated from the `validate-test-coverage` job output.

**Behavior**:
- If **all tests are covered** by Groups 1-12: This group is skipped with a success notice
- If **some tests are uncovered**: This group runs those tests and displays a warning

**Example Scenario**:
If you add a new test called `TestNewFeature_Success` that doesn't match any existing pattern, it will:
1. Be detected by the validation job
2. Run in the "Uncovered Tests (Catch-All)" group
3. Display a warning annotation asking you to categorize it

> ⚠️ **Note**: Tests running in this group should be properly categorized by updating the matrix patterns or renaming the test function. The catch-all is a safety net, not a permanent home for tests.

---

## Adding New Tests

### Step 1: Determine the Appropriate Group

Choose a group based on what your test is testing:

| If your test is about... | Use group |
|--------------------------|-----------|
| Creating scans, scan types, ASCA | `scan-create` |
| Scan workflows, E2E, incremental, cancel | `scan-ops` |
| Results, reports, CodeBashing | `results` |
| PR decoration (GitHub, GitLab, Azure, Bitbucket) | `pr-decoration` |
| Project creation and management | `projects` |
| Predicates, BFL, triage | `predicates` |
| Container scanning and validation | `containers` |
| Realtime scanning (IaC, OSS, Secrets) | `realtime` |
| Authentication and configuration | `auth-config` |
| Root commands, logging, feature flags | `root-logs` |
| SCM rate limiting or user counting | `scm-tests` |
| Everything else (chat, import, remediation) | `misc` |
| **Temporary/Uncategorized** | `uncovered` (automatic) |

### Step 2: Name Your Test to Match the Pattern

Use the appropriate prefix for your test function:

```go
// For scan-create group:
func TestCreateScan_MyNewFeature_Success(t *testing.T) { ... }
func TestScanCreate_MyNewScenario(t *testing.T) { ... }

// For results group:
func TestResults_MyNewReportType(t *testing.T) { ... }

// For projects group:
func TestProject_MyNewProjectFeature(t *testing.T) { ... }

// For realtime group:
func TestIacRealtimeScan_MyNewScenario(t *testing.T) { ... }
```

### Step 3: Verify Your Test Will Be Picked Up

Run this command locally to check if your test matches a pattern:

```bash
# On Linux/Mac:
echo "TestMyNewFunction" | grep -E "^Test(CreateScan|CreateAsyncScan|...)"

# Or use the validation script in CI - it will fail if your test is uncovered
```

### What If My Test Doesn't Fit Any Group?

1. **First, try to rename it** to match an existing pattern
2. **If it's a new category**, add a new pattern to the `misc` group
3. **If it needs its own group**, see "Adding a New Test Group" below

---

## Pattern Matching Rules

### How Regex Patterns Work

- Patterns use **Go's `-run` flag** which accepts regex
- `^Test` anchors to the start of the function name
- `(A|B|C)` matches any of A, B, or C
- `[^I]` means "not followed by I" (used to exclude `ScanCreateInclude` from `scan-create`)

### Important Notes

1. **Case sensitivity matters**: `TestBitbucket` ≠ `TestBitBucket`
2. **Patterns are checked in order**: First match wins
3. **Underscores are significant**: `Test_DownloadScan` ≠ `TestDownloadScan`

### Validation Process

The `validate-test-coverage` job:
1. Extracts all `func Test*` functions from `test/integration/*_test.go`
2. Checks each against all patterns
3. Fails CI if any test is uncovered

---

## Troubleshooting

### Test Not Running in Expected Group

**Symptom**: Your test runs but in the wrong group.

**Solution**: Check which pattern matches first. Patterns are evaluated in order.

```bash
# Check which pattern your test matches
echo "TestYourFunctionName" | grep -E "^Test(Pattern1)"
echo "TestYourFunctionName" | grep -E "^Test(Pattern2)"
```

### Test Name Doesn't Match Any Pattern

**Symptom**: CI fails with "ERROR: The following tests are NOT covered by any CI matrix pattern"

**Solutions**:
1. **Rename your test** to match an existing pattern
2. **Add your prefix** to an existing group's pattern in `.github/workflows/ci-tests.yml`
3. **Update both places**: The matrix pattern AND the validation patterns array

### Adding a New Test Group

If you need a completely new group:

1. Add a new entry to the matrix in `.github/workflows/ci-tests.yml`:
```yaml
- group: my-new-group
  name: "My New Group"
  pattern: "^Test(MyPrefix|AnotherPrefix)"
```

2. Add the same pattern to the `patterns` array in the `validate-test-coverage` job

3. Consider if the new group needs:
   - Pre-test cleanup (like `projects`, `scan-create`, `scan-ops`)
   - Special dependencies (like `misc` needs `pre-commit`)

### Common Pattern Mistakes

| Mistake | Problem | Fix |
|---------|---------|-----|
| `TestBitbucket` vs `TestBitBucket` | Case mismatch | Add both variants to pattern |
| `TestScanWorkflow` vs `TestScanWorkFlow` | Case mismatch | Add both variants to pattern |
| Missing `^` anchor | Matches anywhere in name | Always start with `^Test` |
| Overlapping patterns | Test runs in wrong group | Make patterns more specific |

---

## Quick Reference

| Group | ID | Key Prefixes |
|-------|-----|--------------|
| Scan Creation | `scan-create` | `CreateScan`, `ScanCreate`, `ScanType`, `ExecuteASCAScan` |
| Scan Operations | `scan-ops` | `ScansE2E`, `IncrementalScan`, `ScanWorkflow`, `RunKics` |
| Results & Reports | `results` | `Result`, `CodeBashing`, `RiskManagement` |
| PR Decoration | `pr-decoration` | `PR` |
| Projects | `projects` | `Project`, `CreateEmpty`, `CreateAlready` |
| Predicates & BFL | `predicates` | `Predicate`, `Bfl`, `Triage`, `SastUpdate` |
| Container Tests | `containers` | `ContainerScan`, `ContainerImage`, `EmptyFolder` |
| Realtime Scanning | `realtime` | `IacRealtime`, `OssRealtime`, `Secrets_Realtime` |
| Auth & Config | `auth-config` | `Auth`, `LoadConfiguration`, `GetTenant` |
| Root & Logs | `root-logs` | `RootVersion`, `SetLogOutput`, `_DownloadScan` |
| SCM Tests | `scm-tests` | `GitHub*RateLimit`, `Bitbucket*UserCount` |
| Miscellaneous | `misc` | `Chat`, `Import`, `Telemetry`, `Remediation` |
| **Uncovered (Catch-All)** | `uncovered` | *(Dynamic - tests not matching any above)* |

