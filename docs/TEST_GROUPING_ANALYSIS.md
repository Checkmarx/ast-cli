# Integration Test Grouping Analysis

## Overview

This document provides a detailed analysis of how the 337 integration tests are grouped for parallel execution.

## Test Groups

Based on analysis of test names and patterns, the 337 tests are distributed as follows:

### Group 1: fast-validation (Estimated: 40-50 tests, 3-5 minutes)

**Pattern:** `^Test(Auth|Configuration|Tenant|FeatureFlags|Predicate|Logs)`

**Test Examples:**
- `TestAuthValidate`
- `TestAuthValidateClientAndSecret`
- `TestAuthValidateMissingFlagsTogether`
- `TestAuthValidateEmptyFlags`
- `TestAuthValidateWithBaseAuthURI`
- `TestAuthValidateWrongTenantWithBaseAuth`
- `TestAuthValidateWrongAPIKey`
- `TestAuthValidateWithEmptyAuthenticationPath`
- `TestAuthValidateOnlyAPIKey`
- `TestAuthRegisterWithEmptyParameters`
- `TestAuthRegister`
- `TestLoadConfiguration_EnvVarConfigFilePath`
- `TestLoadConfiguration_FileNotFound`
- `TestLoadConfiguration_ValidDirectory`
- `TestLoadConfiguration_FileWithoutPermission_UsingConfigFile`
- `TestSetConfigProperty_EnvVarConfigFilePath`
- `TestLoadConfiguration_ConfigFilePathFlag`
- `TestLoadConfiguration_ConfigFilePathFlagValidDirectory`
- `TestLoadConfiguration_ConfigFilePathFlagFileNotFound`
- `TestSetConfigProperty_ConfigFilePathFlag`
- `TestLoadConfiguration_ConfigFilePathFlagFileWithoutPermission`
- `TestTenantConfigurationSuccessCaseJson`
- `TestTenantConfigurationSuccessCaseYaml`
- `TestFeatureFlagsSuccessCaseJson`
- `TestFeatureFlagsSuccessCaseYaml`
- `TestLogsSuccessCase`

**Characteristics:**
- Fast validation tests (no actual scans)
- Authentication and configuration tests
- Minimal external dependencies
- Quick execution (seconds per test)

---

### Group 2: scan-core (Estimated: 70-80 tests, 20-25 minutes)

**Pattern:** `^TestScan(Create|List|Show|Delete|Workflow|Logs|Filter|Threshold|Resubmit|Types)`

**Exclude Pattern:** `Timeout|Cancel|SlowRepo|Incremental|E2E`

**Test Examples:**
- `TestScanCreate`
- `TestScanCreateWithTags`
- `TestScanCreateWithBranch`
- `TestScanList`
- `TestScanListWithFilter`
- `TestScanShow`
- `TestScanShowWithFormat`
- `TestScanDelete`
- `TestScanWorkflow`
- `TestScanLogs`
- `TestScanFilter`
- `TestScanThreshold`
- `TestScanResubmit`
- `TestScanTypes`

**Characteristics:**
- Core scan CRUD operations
- Excludes slow/timeout tests
- Medium execution time
- Most critical scan functionality

---

### Group 3: scan-engines (Estimated: 60-70 tests, 25-30 minutes)

**Pattern:** `^Test(Container|Scs|CreateScan_With.*Engine|.*ApiSecurity|.*ExploitablePath)`

**Test Examples:**
- `TestContainerScan_EmptyFolderWithExternalImages`
- `TestContainerScan_EmptyFolderWithMultipleExternalImages`
- `TestContainerScan_EmptyFolderWithExternalImagesAndDebug`
- `TestContainerScan_EmptyFolderWithComplexImageNames`
- `TestContainerScan_EmptyFolderWithRegistryImages`
- `TestContainerScan_EmptyFolderInvalidImageShouldFail`
- `TestContainerScan_EmptyFolderMixedValidInvalidImages`
- `TestContainerImageValidation_ValidFormats`
- `TestContainerImageValidation_InvalidFormats`
- `TestContainerImageValidation_MultipleImagesValidation`
- `TestContainerImageValidation_TarFiles`
- `TestContainerImageValidation_MixedTarAndRegularImages`
- `TestScsScan`
- `TestScsScanWithFilter`
- `TestCreateScan_WithSastEngine`
- `TestCreateScan_WithScaEngine`
- `TestCreateScan_WithKicsEngine`
- `TestCreateScan_WithAllEngines`
- `TestScanApiSecurity`
- `TestScanExploitablePath`

**Characteristics:**
- Multi-engine scan tests
- Container scanning tests
- API Security and Exploitable Path tests
- Longer execution time due to multiple engines

---

### Group 4: scm-integration (Estimated: 40-50 tests, 15-20 minutes)

**Pattern:** `^Test(PR|UserCount)`

**Test Examples:**
- `TestPRDecorationGithub`
- `TestPRDecorationGitlab`
- `TestPRDecorationAzure`
- `TestPRDecorationBitbucket`
- `TestPRGithubWithComments`
- `TestPRGitlabWithComments`
- `TestPRAzureWithComments`
- `TestPRBitbucketWithComments`
- `TestUserCountGithub`
- `TestUserCountGitlab`
- `TestUserCountAzure`
- `TestUserCountBitbucket`
- `TestUserCountGithubEnterprise`
- `TestUserCountGitlabSelfManaged`

**Characteristics:**
- SCM integration tests (GitHub, GitLab, Azure, Bitbucket)
- PR decoration tests
- User count tests
- External API dependencies

---

### Group 5: realtime-features (Estimated: 40-50 tests, 10-15 minutes)

**Pattern:** `^Test(Kics|Sca|Oss|Secrets|Containers)Realtime|^TestRun.*Realtime`

**Test Examples:**
- `TestKicsRealtime`
- `TestKicsRealtimeWithFilter`
- `TestKicsRealtimeWithThreshold`
- `TestScaRealtime`
- `TestScaRealtimeWithFilter`
- `TestScaRealtimeWithThreshold`
- `TestOssRealtime`
- `TestOssRealtimeWithFilter`
- `TestSecretsRealtime`
- `TestSecretsRealtimeWithFilter`
- `TestContainersRealtime`
- `TestContainersRealtimeWithFilter`
- `TestRunKicsRealtime`
- `TestRunScaRealtime`
- `TestRunOssRealtime`
- `TestRunSecretsRealtime`
- `TestRunContainersRealtime`

**Characteristics:**
- Real-time scanning features
- Multiple engine real-time tests
- Quick feedback scanning
- Medium execution time

---

### Group 6: advanced-features (Estimated: 50-60 tests, 15-20 minutes)

**Pattern:** `^Test(Project|Result|Import|Bfl|Asca|Chat|Learn|Telemetry|RateLimit|PreCommit|PreReceive|Remediation)`

**Test Examples:**
- `TestScanASCA_NoFileSourceSent_ReturnSuccess`
- `TestExecuteASCAScan_ASCALatestVersionSetTrue_Success`
- `TestExecuteASCAScan_NoSourceAndASCALatestVersionSetFalse_Success`
- `TestExecuteASCAScan_NotExistingFile_Success`
- `TestExecuteASCAScan_ASCALatestVersionSetFalse_Success`
- `TestExecuteASCAScan_NoEngineInstalledAndASCALatestVersionSetFalse_Success`
- `TestExecuteASCAScan_CorrectFlagsSent_SuccessfullyReturnMockData`
- `TestExecuteASCAScan_UnsupportedLanguage_Fail`
- `TestExecuteASCAScan_InitializeAndRunUpdateVersion_Success`
- `TestExecuteASCAScan_InitializeAndShutdown_Success`
- `TestExecuteASCAScan_EngineNotRunningWithLicense_Success`
- `TestRunGetBflByScanIdAndQueryId`
- `TestRunGetBflWithInvalidScanIDandQueryID`
- `TestChatKicsInvalidAPIKey`
- `TestChatSastInvalidAPIKey`
- `TestChatKicsAzureAIInvalidAPIKey`
- `TestProjectCreate`
- `TestProjectList`
- `TestProjectShow`
- `TestProjectDelete`
- `TestProjectUpdate`
- `TestResultList`
- `TestResultShow`
- `TestResultExport`
- `TestImportScan`
- `TestTelemetry`
- `TestRateLimit`
- `TestPreCommit`
- `TestPreReceive`
- `TestRemediation`

**Characteristics:**
- Advanced CLI features
- Project management
- Result handling
- BFL (Best Fix Location)
- ASCA (AI Security Code Analyzer)
- Chat features
- Import/export functionality

---

## Validation

### Test Coverage

```bash
# Run this command to verify all tests are covered
go test -tags integration -list . ./test/integration 2>&1 | grep "^Test" | wc -l
# Expected: 337
```

### Group Distribution

| Group | Estimated Tests | Estimated Time | Timeout |
|-------|----------------|----------------|---------|
| fast-validation | 40-50 | 3-5 min | 10 min |
| scan-core | 70-80 | 20-25 min | 30 min |
| scan-engines | 60-70 | 25-30 min | 35 min |
| scm-integration | 40-50 | 15-20 min | 25 min |
| realtime-features | 40-50 | 10-15 min | 20 min |
| advanced-features | 50-60 | 15-20 min | 25 min |
| **Total** | **337** | **~30 min** | - |

---

## Notes

1. **Parallel Execution:** Each group runs with `-parallel 4` flag, allowing up to 4 tests to run concurrently within the group.

2. **Retry Mechanism:** Failed tests are automatically retried within each group.

3. **Coverage:** Each group generates its own coverage file, which are merged at the end.

4. **Timeouts:** Each group has a specific timeout based on expected execution time + buffer.

5. **Exclusions:** The `scan-core` group excludes slow tests (Timeout, Cancel, SlowRepo, Incremental, E2E) to keep execution time reasonable.

6. **Dependencies:** Each group is independent and can run in parallel without conflicts.

---

## Maintenance

When adding new tests:

1. Identify the test category (auth, scan, container, PR, realtime, project, etc.)
2. Add the test to the appropriate group pattern
3. Verify the group timeout is sufficient
4. Run the group locally to validate

When rebalancing groups:

1. Monitor actual execution times from CI logs
2. Move tests between groups if one group consistently takes too long
3. Update patterns in `integration_up_parallel.sh`
4. Test the new grouping locally before deploying
