# Single Action Approach - Integration Test Optimization

## 🎯 **Overview**

This document explains the **single action** approach for running integration tests. Instead of showing 6 separate jobs in the GitHub Actions UI, all test groups run as **background processes within a single job**.

---

## 📊 **Comparison: Matrix vs Grouped Approach**

### **Matrix Approach** (Previous)
```yaml
strategy:
  matrix:
    test-group: [group1, group2, group3, group4, group5, group6]
```

**GitHub Actions UI:**
```
✓ integration-tests (fast-validation)
✓ integration-tests (scan-core)
✓ integration-tests (scan-engines)
✓ integration-tests (scm-integration)
✓ integration-tests (realtime-features)
✓ integration-tests (advanced-features)
✓ merge-coverage
```
**Result:** 7 separate entries in Actions tab

---

### **Grouped Approach** (New)
```yaml
- name: Run All Integration Tests (Grouped)
  run: ./internal/commands/.scripts/integration_up_grouped.sh
```

**GitHub Actions UI:**
```
✓ integration-tests
```
**Result:** 1 single entry in Actions tab ✨

---

## 🔧 **How It Works**

### **1. Single Job Execution**
The workflow runs a single job that:
1. Sets up the environment (Go, ScaResolver, Squid proxy)
2. Launches all 6 test groups as **background processes**
3. Waits for all groups to complete
4. Merges coverage reports
5. Checks coverage threshold

### **2. Parallel Background Execution**
```bash
# Start all test groups in parallel
for group in "${TEST_GROUPS[@]}"; do
  run_test_group "$group" &  # & runs in background
  PIDS[$group]=$!
done

# Wait for all to complete
for group in "${TEST_GROUPS[@]}"; do
  wait ${PIDS[$group]}
done
```

### **3. Individual Group Logs**
Each test group writes to its own log file:
```
test-results/
├── fast-validation/
│   ├── output.log
│   └── cover.out
├── scan-core/
│   ├── output.log
│   └── cover.out
├── scan-engines/
│   ├── output.log
│   └── cover.out
├── scm-integration/
│   ├── output.log
│   └── cover.out
├── realtime-features/
│   ├── output.log
│   └── cover.out
└── advanced-features/
    ├── output.log
    └── cover.out
```

---

## ⚡ **Performance**

| Metric | Matrix Approach | Grouped Approach | Difference |
|--------|----------------|------------------|------------|
| **Execution Time** | ~30 min | ~30 min | Same |
| **Actions UI Entries** | 7 jobs | 1 job | -6 entries |
| **Parallel Execution** | ✅ Yes | ✅ Yes | Same |
| **Coverage Merging** | Separate job | Same job | Simpler |
| **Resource Usage** | 6 runners | 1 runner | More efficient |

---

## 📁 **Key Files**

### **1. Workflow File**
**Path:** `.github/workflows/ci-tests-optimized.yml`

**Key Changes:**
- Removed `strategy.matrix` configuration
- Single `integration-tests` job
- Calls `integration_up_grouped.sh` instead of `integration_up_parallel.sh`
- No separate `merge-coverage` job needed

### **2. Grouped Execution Script**
**Path:** `internal/commands/.scripts/integration_up_grouped.sh`

**Features:**
- Runs all 6 test groups as background processes
- Collects individual results
- Merges coverage automatically
- Provides detailed summary output

---

## 📝 **Console Output Example**

```bash
========================================
Starting Grouped Integration Tests
========================================

✓ Squid proxy started
✓ Using cached ScaResolver

========================================
Running Test Groups in Parallel
========================================

Started fast-validation (PID: 12345)
Started scan-core (PID: 12346)
Started scan-engines (PID: 12347)
Started scm-integration (PID: 12348)
Started realtime-features (PID: 12349)
Started advanced-features (PID: 12350)

Waiting for all test groups to complete...

[fast-validation] ✓ PASSED
[realtime-features] ✓ PASSED
[scm-integration] ✓ PASSED
[advanced-features] ✓ PASSED
[scan-core] ✓ PASSED
[scan-engines] ✓ PASSED

========================================
Test Results Summary
========================================

✓ fast-validation: PASSED
✓ scan-core: PASSED
✓ scan-engines: PASSED
✓ scm-integration: PASSED
✓ realtime-features: PASSED
✓ advanced-features: PASSED

Merging coverage reports...
✓ Coverage reports merged
✓ HTML coverage report generated

========================================
Coverage Summary
========================================
github.com/checkmarx/ast-cli/internal/commands/util.go:123:  someFunc  75.0%
...
total:                                                        76.2%

========================================
All test groups PASSED! ✓
========================================
```

---

## ✅ **Advantages**

1. **Cleaner UI** - Single entry in GitHub Actions tab
2. **Simpler Workflow** - No matrix strategy, no merge job
3. **Same Performance** - Still runs in parallel (~30 min)
4. **Better Resource Usage** - Uses 1 runner instead of 6
5. **Easier Debugging** - All logs in one place
6. **Automatic Merging** - Coverage merged in same job

---

## ⚠️ **Trade-offs**

| Aspect | Matrix Approach | Grouped Approach |
|--------|----------------|------------------|
| **Visibility** | See each group status separately | See only overall status |
| **Retry** | Can retry individual groups | Must retry entire job |
| **Debugging** | Click on specific group | Check logs in artifacts |
| **Resource Limits** | 6 runners (more capacity) | 1 runner (may hit limits) |

---

## 🚀 **Deployment**

### **Option 1: Replace Existing Workflow**
```bash
# Backup current workflow
cp .github/workflows/ci-tests.yml .github/workflows/ci-tests-backup.yml

# Replace with optimized version
cp .github/workflows/ci-tests-optimized.yml .github/workflows/ci-tests.yml

# Commit
git add .github/workflows/ci-tests.yml
git add internal/commands/.scripts/integration_up_grouped.sh
git commit -m "Optimize: Single action for integration tests"
git push
```

### **Option 2: Run Both (A/B Testing)**
Keep both workflows and compare:
- `ci-tests.yml` - Original (210 min)
- `ci-tests-optimized.yml` - New single action (~30 min)

---

## 🔍 **Monitoring**

### **Check Test Group Status**
```bash
# Download artifacts from GitHub Actions
# Extract test-results/ directory
# Check individual group logs

cat test-results/scan-core/output.log
cat test-results/scan-engines/output.log
```

### **Verify Coverage**
```bash
# Check merged coverage
go tool cover -func cover.out | grep total

# View HTML report
open coverage.html
```

---

## 🎓 **When to Use Each Approach**

### **Use Matrix Approach When:**
- You need to see individual group status in UI
- You want to retry specific groups
- You have complex dependencies between groups
- You need maximum visibility for debugging

### **Use Grouped Approach When:**
- You want a cleaner Actions UI
- You prefer single-entry execution
- You want to save runner resources
- You're confident in the test stability

---

## 📞 **Summary**

**The grouped approach provides:**
- ✅ Same 30-minute execution time
- ✅ Same parallel execution
- ✅ Same test coverage (100%)
- ✅ Same code coverage (≥75%)
- ✅ Cleaner GitHub Actions UI (1 entry instead of 7)
- ✅ More efficient resource usage (1 runner instead of 6)

**Perfect for teams that want fast tests with a clean UI!** 🚀
