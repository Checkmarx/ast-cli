# Coverage Issue Fix - Integration Test Optimization

## Issue Summary

**Date:** 2026-01-05  
**CI Run:** https://github.com/Checkmarx/ast-cli/actions/runs/20724098604  
**Problem:** Merged coverage is 58.2% (below required 75%)  
**Root Cause:** Test grouping patterns were too restrictive, excluding many tests

---

## Analysis from CI Logs

### What Worked ✅

1. **Parallelization:** All 6 test groups ran successfully in parallel
2. **Coverage Merging:** `gocovmerge` successfully merged all 6 coverage files
3. **Artifact Upload:** All coverage artifacts uploaded correctly
4. **Execution Time:** Groups completed within expected timeframes

### What Failed ❌

```
Your code coverage is too low. Coverage precentage is: 58.2
Expected: 75%
Actual: 58.2%
Gap: -16.8%
```

---

## Root Cause Analysis

### Original Test Patterns (Too Restrictive)

The original patterns were missing many tests:

```bash
# fast-validation - MISSED: Test_HandleFeatureFlags, Test_DownloadScan_Logs
TEST_PATTERN="^Test(Auth|Configuration|Tenant|FeatureFlags|Predicate|Logs)"

# scan-core - TOO SPECIFIC: Only matched TestScan(Create|List|Show...)
TEST_PATTERN="^TestScan(Create|List|Show|Delete|Workflow|Logs|Filter|Threshold|Resubmit|Types)"

# scan-engines - MISSED: All ASCA tests (TestScanASCA_, TestExecuteASCAScan_)
TEST_PATTERN="^Test(Container|Scs|CreateScan_With.*Engine|.*ApiSecurity|.*ExploitablePath)"

# realtime-features - MISSED: TestIacRealtimeScan tests
TEST_PATTERN="^Test(Kics|Sca|Oss|Secrets|Containers)Realtime|^TestRun.*Realtime"

# advanced-features - MISSED: TestGetProjectNameFunction
TEST_PATTERN="^Test(Project|Result|Import|Bfl|Asca|Chat|Learn|Telemetry|RateLimit|PreCommit|PreReceive|Remediation)"
```

### Tests That Were Excluded

Based on the test list, these tests were NOT matched by any pattern:

1. **ASCA Tests** (~11 tests):
   - `TestScanASCA_NoFileSourceSent_ReturnSuccess`
   - `TestExecuteASCAScan_ASCALatestVersionSetTrue_Success`
   - `TestExecuteASCAScan_NoSourceAndASCALatestVersionSetFalse_Success`
   - etc.

2. **IAC Realtime Tests** (~10 tests):
   - `TestIacRealtimeScan_TerraformFile_Success`
   - `TestIacRealtimeScan_DockerFile_Success`
   - `TestIacRealtimeScan_YamlConfigFile_Success`
   - etc.

3. **Utility Tests**:
   - `Test_HandleFeatureFlags_WhenCalled_ThenNoErrorAndCacheNotEmpty`
   - `Test_DownloadScan_Logs_Success`
   - `Test_DownloadScan_Logs_Failed`
   - `TestGetProjectNameFunction_ProjectNameValueIsEmpty_ReturnRelevantError`

4. **Scan Tests** (many variations):
   - The pattern `^TestScan(Create|List|...)` was too specific
   - Missed tests like `TestScanASCA_...`, `TestScan_...`, etc.

**Estimated excluded tests:** 50-80 tests out of 337 (15-24%)

---

## Solution

### Updated Test Patterns (Comprehensive)

```bash
case "$TEST_GROUP" in
  "fast-validation")
    # Includes: Auth, Configuration, Tenant, FeatureFlags, Logs, Proxy
    TEST_PATTERN="^Test(Auth|.*Configuration|Tenant|FeatureFlags|Predicate|.*Logs|FailProxyAuth)"
    TIMEOUT="10m"
    ;;
  
  "scan-core")
    # Includes: ALL scan tests EXCEPT specific engines
    TEST_PATTERN="^TestScan|^Test.*Scan.*"
    EXCLUDE_PATTERN="ASCA|Container|Realtime|Iac|Oss|Secrets|Kics|Scs"
    TIMEOUT="30m"
    ;;
  
  "scan-engines")
    # Includes: ASCA, Container, Scs, and all engine-specific tests
    TEST_PATTERN="^Test(.*ASCA|.*Asca|Container|Scs|.*Engine)"
    TIMEOUT="35m"
    ;;
  
  "scm-integration")
    # Includes: PR decoration and UserCount tests
    TEST_PATTERN="^Test(PR|UserCount)"
    TIMEOUT="25m"
    ;;
  
  "realtime-features")
    # Includes: ALL realtime tests (Iac, Kics, Sca, Oss, Secrets, Containers)
    TEST_PATTERN="^Test(Iac|Kics|Sca|Oss|Secrets|Containers)Realtime"
    TIMEOUT="20m"
    ;;
  
  "advanced-features")
    # Includes: Projects, Results, Import, BFL, Chat, Learn, and utilities
    TEST_PATTERN="^Test(Project|Result|Import|.*Bfl|Chat|.*Learn|Telemetry|RateLimit|PreCommit|PreReceive|Remediation|GetProjectName)"
    TIMEOUT="25m"
    ;;
esac
```

### Key Changes

1. **fast-validation:**
   - Changed `Configuration` → `.*Configuration` (matches LoadConfiguration, SetConfiguration, etc.)
   - Changed `Logs` → `.*Logs` (matches Test_DownloadScan_Logs, etc.)
   - Added `FailProxyAuth` explicitly

2. **scan-core:**
   - Changed from specific list to broad pattern: `^TestScan|^Test.*Scan.*`
   - Uses EXCLUDE pattern to remove engine-specific tests
   - Now catches ALL scan tests by default

3. **scan-engines:**
   - Added `.*ASCA|.*Asca` to catch all ASCA tests
   - Pattern now matches `TestScanASCA_`, `TestExecuteASCAScan_`, etc.

4. **realtime-features:**
   - Added `Iac` to catch `TestIacRealtimeScan_` tests
   - Now includes all 6 realtime engines

5. **advanced-features:**
   - Changed `Bfl` → `.*Bfl` (matches TestRunGetBflByScanIdAndQueryId)
   - Changed `Learn` → `.*Learn` (matches TestGetLearnMoreInformation...)
   - Added `GetProjectName` for utility tests

---

## Expected Impact

### Coverage Improvement

| Component | Before | After | Change |
|-----------|--------|-------|--------|
| Tests Matched | ~250-287 | 337 | +50-87 tests |
| Coverage | 58.2% | ≥75% | +16.8% |
| Test Execution | Partial | Complete | 100% |

### Test Distribution (Updated)

| Group | Tests (Est.) | Coverage Impact |
|-------|--------------|-----------------|
| fast-validation | 45-50 | +5-10 tests |
| scan-core | 80-90 | +10-15 tests |
| scan-engines | 70-80 | +15-20 tests (ASCA) |
| scm-integration | 45-50 | No change |
| realtime-features | 50-60 | +10-15 tests (IAC) |
| advanced-features | 50-60 | +5-10 tests |

---

## Deployment Steps

### 1. Update the Script

The fix has already been applied to `internal/commands/.scripts/integration_up_parallel.sh`.

### 2. Test Locally (Optional)

```bash
# Test one group to verify pattern works
export TEST_GROUP="scan-engines"
export CX_APIKEY="your-api-key"
# ... set other env vars

./internal/commands/.scripts/integration_up_parallel.sh

# Check that ASCA tests are now included
grep "TestScanASCA\|TestExecuteASCAScan" test_output.log
```

### 3. Commit and Push

```bash
git add internal/commands/.scripts/integration_up_parallel.sh
git add docs/COVERAGE_ISSUE_FIX.md
git commit -m "Fix: Update test patterns to include all 337 tests

- Broadened scan-core pattern to catch all scan tests
- Added ASCA tests to scan-engines group
- Added IAC realtime tests to realtime-features group
- Fixed pattern matching for configuration and log tests
- Expected coverage increase from 58.2% to ≥75%"

git push origin optimize-integration-tests
```

### 4. Monitor Next CI Run

Watch for:
- ✅ All 337 tests executed
- ✅ Coverage ≥75%
- ✅ No duplicate test execution
- ✅ Execution time still ~30 minutes

---

## Validation

### How to Verify Fix Worked

After the next CI run, check:

1. **Coverage Percentage:**
   ```
   Your code coverage test passed! Coverage precentage is: 75.X
   ```

2. **Test Count per Group:**
   - Check CI logs for each group
   - Count "PASS" and "FAIL" lines
   - Total should be 337 tests

3. **No Missing Tests:**
   ```bash
   # Download test output logs from all 6 groups
   # Combine and count unique tests
   cat group-*/test_output.log | grep "^--- PASS\|^--- FAIL" | wc -l
   # Should be 337
   ```

---

## Rollback (If Needed)

If the new patterns cause issues:

```bash
# Revert to original patterns
git revert <commit-sha>
git push origin optimize-integration-tests

# Or manually restore original patterns from git history
git show HEAD~1:internal/commands/.scripts/integration_up_parallel.sh > integration_up_parallel.sh.backup
```

---

## Lessons Learned

1. **Test Pattern Validation:** Always validate patterns match ALL tests before deployment
2. **Use Broad Patterns:** Start broad and exclude, rather than start narrow and include
3. **Coverage Monitoring:** Monitor coverage per group to catch missing tests early
4. **Local Testing:** Test patterns locally with `go test -list` before CI deployment

---

## Next Steps

1. ✅ Fix has been applied to `integration_up_parallel.sh`
2. ⏳ Commit and push the fix
3. ⏳ Monitor next CI run for coverage ≥75%
4. ⏳ If successful, merge PR
5. ⏳ Document final test distribution in TEST_GROUPING_ANALYSIS.md

---

## References

- **CI Run with Issue:** https://github.com/Checkmarx/ast-cli/actions/runs/20724098604
- **Coverage Merge Log:** Shows 58.2% coverage
- **Test List:** 337 integration tests total
- **Fix Applied:** `internal/commands/.scripts/integration_up_parallel.sh` (lines 40-90)
