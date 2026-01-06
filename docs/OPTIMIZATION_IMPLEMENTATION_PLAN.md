# Integration Test Optimization - Implementation Plan

## Phase 1: Preparation (Day 1)

### 1.1 Validate Test Grouping

```bash
# Make analysis script executable
chmod +x scripts/analyze-test-groups.sh

# Run analysis to validate grouping
./scripts/analyze-test-groups.sh

# Expected output:
# - Total tests: 337
# - All tests matched to groups
# - No duplicate assignments
# - Coverage: 100%
```

**Success Criteria:**
- ✅ All 337 tests assigned to groups
- ✅ No tests assigned to multiple groups
- ✅ No unmatched tests

### 1.2 Local Testing

Test each group individually on your local machine:

```bash
# Set required environment variables
export CX_APIKEY="your-api-key"
export CX_TENANT="your-tenant"
export CX_BASE_URI="https://your-instance.checkmarx.net"
# ... (set all required vars from ci-tests.yml)

# Test fast-validation group (should complete in 3-5 minutes)
export TEST_GROUP="fast-validation"
chmod +x internal/commands/.scripts/integration_up_parallel.sh
./internal/commands/.scripts/integration_up_parallel.sh

# If successful, test other groups
export TEST_GROUP="scan-core"
./internal/commands/.scripts/integration_up_parallel.sh
```

**Success Criteria:**
- ✅ Each group completes within expected time
- ✅ Coverage files generated for each group
- ✅ Retry mechanism works correctly
- ✅ No test failures (or expected failures only)

### 1.3 Coverage Validation

Test coverage merging locally:

```bash
# Run multiple groups and collect coverage
mkdir -p coverage-reports

for group in fast-validation scan-core; do
  export TEST_GROUP=$group
  ./internal/commands/.scripts/integration_up_parallel.sh
  cp cover.out coverage-reports/cover-${group}.out
done

# Merge coverage
go install github.com/wadey/gocovmerge@latest
gocovmerge coverage-reports/*.out > merged-cover.out

# Check total coverage
go tool cover -func merged-cover.out | grep total

# Expected: ≥75%
```

**Success Criteria:**
- ✅ Coverage merging works without errors
- ✅ Total coverage ≥75%
- ✅ Coverage HTML report generates correctly

---

## Phase 2: Feature Branch Deployment (Day 2-3)

### 2.1 Create Feature Branch

```bash
git checkout -b optimize-integration-tests
git pull origin main
```

### 2.2 Deploy Optimized Files

```bash
# Backup current configuration
cp .github/workflows/ci-tests.yml .github/workflows/ci-tests-backup.yml

# Deploy optimized workflow
cp .github/workflows/ci-tests-optimized.yml .github/workflows/ci-tests.yml

# Ensure scripts are executable
chmod +x internal/commands/.scripts/integration_up_parallel.sh
chmod +x scripts/analyze-test-groups.sh

# Add all changes
git add .github/workflows/ci-tests.yml
git add internal/commands/.scripts/integration_up_parallel.sh
git add scripts/analyze-test-groups.sh
git add docs/INTEGRATION_TEST_OPTIMIZATION_GUIDE.md
git add docs/OPTIMIZATION_IMPLEMENTATION_PLAN.md

# Commit
git commit -m "Optimize integration tests: 210min → 30min

- Split 337 tests into 6 parallel groups
- Add matrix strategy for parallel execution
- Implement caching for ScaResolver and Go modules
- Add coverage merging job
- Maintain 75% coverage requirement
- Preserve retry mechanism per group

Expected execution time: 25-35 minutes (85% reduction)"

# Push to remote
git push origin optimize-integration-tests
```

### 2.3 Create Pull Request

Create PR with this description:

```markdown
## Integration Test Optimization

### Summary
Reduces integration test execution time from 210 minutes to ~30 minutes (85% reduction) through parallel execution.

### Changes
- ✅ Split 337 tests into 6 logical groups
- ✅ Implement GitHub Actions matrix strategy
- ✅ Add caching for ScaResolver and Go modules
- ✅ Create coverage merging job
- ✅ Maintain 75% coverage requirement
- ✅ Preserve retry mechanism

### Test Groups
1. **fast-validation** (3-5 min): Auth, config, validation
2. **scan-core** (20-25 min): Core scan operations
3. **scan-engines** (25-30 min): Multi-engine scans
4. **scm-integration** (15-20 min): GitHub, GitLab, Azure, Bitbucket
5. **realtime-features** (10-15 min): Real-time scanning
6. **advanced-features** (15-20 min): Projects, results, imports

### Validation
- [ ] All 6 groups complete successfully
- [ ] Total execution time ≤35 minutes
- [ ] Coverage ≥75%
- [ ] No test failures (or expected failures only)
- [ ] Coverage merging works correctly

### Rollback Plan
If issues occur, revert to `ci-tests-backup.yml`

### Documentation
- See `docs/INTEGRATION_TEST_OPTIMIZATION_GUIDE.md` for details
- See `docs/OPTIMIZATION_IMPLEMENTATION_PLAN.md` for implementation steps
```

### 2.4 Monitor PR Build

Watch the GitHub Actions run:

1. Go to Actions tab
2. Find the PR build
3. Monitor all 6 parallel jobs
4. Check coverage merge job
5. Verify total execution time

**Expected Timeline:**
- Job start: 0-2 minutes (setup)
- Parallel execution: 25-35 minutes
- Coverage merge: 1-2 minutes
- **Total: 27-39 minutes**

**Success Criteria:**
- ✅ All 6 groups pass
- ✅ Total time ≤40 minutes
- ✅ Coverage ≥75%
- ✅ Coverage merge successful

---

## Phase 3: Validation & Adjustment (Day 4-5)

### 3.1 Analyze Results

After PR build completes, analyze:

```bash
# Download workflow logs
gh run view <run-id> --log > workflow-logs.txt

# Extract timing information
grep "Test execution status" workflow-logs.txt

# Extract coverage information
grep "Coverage precentage" workflow-logs.txt
```

### 3.2 Adjust if Needed

**If one group takes too long (>35 minutes):**

1. Identify slow tests in that group
2. Move some tests to a faster group
3. Update TEST_PATTERN in `integration_up_parallel.sh`
4. Push update and re-test

**If coverage is too low (<75%):**

1. Check which group has low coverage
2. Verify coverage files are being generated
3. Check gocovmerge is working correctly
4. Review test patterns to ensure all tests run

**If tests are flaky:**

1. Identify flaky tests from retry logs
2. Consider increasing timeout for that group
3. Or move flaky tests to separate group

### 3.3 Run Multiple PR Builds

To validate stability, trigger 3-5 PR builds:

```bash
# Make a trivial change
echo "# Test" >> README.md
git add README.md
git commit -m "Test: Validate optimization stability"
git push

# Repeat 3-5 times
```

**Success Criteria:**
- ✅ Consistent execution time (±5 minutes)
- ✅ Consistent coverage (±2%)
- ✅ No new test failures
- ✅ Retry rate similar to current

---

## Phase 4: Production Deployment (Day 6)

### 4.1 Final Validation

Before merging:

- [ ] All PR builds successful
- [ ] Execution time consistently ≤35 minutes
- [ ] Coverage consistently ≥75%
- [ ] No increase in test failures
- [ ] Team review completed
- [ ] Documentation reviewed

### 4.2 Merge to Main

```bash
# Squash and merge PR
gh pr merge <pr-number> --squash --delete-branch

# Or via GitHub UI:
# 1. Click "Squash and merge"
# 2. Confirm merge
# 3. Delete branch
```

### 4.3 Monitor Main Branch

After merge, monitor next 5-10 PR builds:

```bash
# Watch recent workflow runs
gh run list --workflow=ci-tests.yml --limit 10

# Check for any issues
gh run view <run-id>
```

**Success Criteria:**
- ✅ All PR builds complete in ≤35 minutes
- ✅ Coverage remains ≥75%
- ✅ No increase in failure rate
- ✅ No developer complaints

---

## Phase 5: Monitoring & Optimization (Ongoing)

### 5.1 Weekly Monitoring

Track these metrics weekly:

```bash
# Average execution time
gh run list --workflow=ci-tests.yml --limit 50 --json conclusion,createdAt,updatedAt \
  | jq '.[] | select(.conclusion=="success") | (.updatedAt | fromdateiso8601) - (.createdAt | fromdateiso8601)' \
  | awk '{sum+=$1; count++} END {print "Average: " sum/count/60 " minutes"}'

# Success rate
gh run list --workflow=ci-tests.yml --limit 50 --json conclusion \
  | jq '[.[] | .conclusion] | group_by(.) | map({conclusion: .[0], count: length})'
```

### 5.2 Monthly Review

Every month:

1. Review test timing data
2. Rebalance groups if needed
3. Update timeouts if needed
4. Review and fix flaky tests
5. Update documentation

### 5.3 Continuous Improvement

**Future optimizations:**

1. **Dynamic test splitting** based on actual timing data
2. **Conditional execution** for tests affected by changes
3. **Test result caching** for unchanged code
4. **Increased parallelization** (12 groups for 15-minute target)

---

## Rollback Procedures

### Immediate Rollback (if critical issues)

```bash
# Revert the merge commit
git revert <merge-commit-sha>
git push origin main

# Or restore backup
cp .github/workflows/ci-tests-backup.yml .github/workflows/ci-tests.yml
git add .github/workflows/ci-tests.yml
git commit -m "Rollback: Restore sequential integration tests"
git push origin main
```

### Partial Rollback (if specific group has issues)

Disable problematic group temporarily:

```yaml
# In ci-tests.yml, comment out the problematic group
matrix:
  test-group: [
    "fast-validation",
    "scan-core", 
    # "scan-engines",  # Temporarily disabled due to timeout issues
    "scm-integration",
    "realtime-features",
    "advanced-features"
  ]
```

---

## Success Metrics

### Primary Metrics

| Metric | Current | Target | Actual |
|--------|---------|--------|--------|
| Execution Time | 210 min | 30 min | ___ min |
| Coverage | ≥75% | ≥75% | ___% |
| Success Rate | ___% | Same | ___% |
| Retry Rate | ___% | Same | ___% |

### Secondary Metrics

| Metric | Target |
|--------|--------|
| Developer Satisfaction | Positive feedback |
| PR Feedback Time | <40 minutes |
| Cost | No increase |
| Maintenance Effort | Minimal |

---

## Communication Plan

### Before Deployment

**To Development Team:**
```
Subject: Integration Test Optimization - Coming Soon

We're optimizing our integration tests to reduce execution time from 210 minutes to ~30 minutes.

What to expect:
- Faster PR feedback (30 min instead of 3.5 hours)
- Same coverage requirements (75%)
- Same test reliability
- No changes to local development workflow

Timeline:
- Testing: This week
- Deployment: Next week
- Monitoring: Ongoing

Questions? See docs/INTEGRATION_TEST_OPTIMIZATION_GUIDE.md
```

### After Deployment

**To Development Team:**
```
Subject: Integration Test Optimization - Now Live! 🚀

Great news! Integration tests now complete in ~30 minutes (down from 210 minutes).

Results:
- ✅ 85% time reduction
- ✅ Same 75% coverage requirement
- ✅ All 337 tests still running
- ✅ Retry mechanism preserved

What changed:
- Tests now run in 6 parallel groups
- Faster feedback on PRs
- Same reliability and coverage

Issues? Contact [team lead] or check docs/INTEGRATION_TEST_OPTIMIZATION_GUIDE.md
```

---

## Conclusion

This implementation plan provides a structured approach to deploying the integration test optimization with minimal risk and maximum benefit. Follow each phase carefully and validate thoroughly before proceeding to the next phase.
