#!/bin/bash

# Script to analyze and validate integration test grouping
# Usage: ./scripts/analyze-test-groups.sh

set -e

echo "=== Integration Test Grouping Analysis ==="
echo ""

# Get all integration test functions
echo "Extracting all integration test functions..."
ALL_TESTS=$(go test -tags integration -list . ./test/integration 2>/dev/null | grep "^Test" || true)
TOTAL_TESTS=$(echo "$ALL_TESTS" | wc -l)

echo "Total integration tests found: $TOTAL_TESTS"
echo ""

# Define test patterns for each group (same as in integration_up_parallel.sh)
declare -A TEST_GROUPS
TEST_GROUPS["fast-validation"]="^Test(Auth|Configuration|Tenant|FeatureFlags|Predicate|Logs)"
TEST_GROUPS["scan-core"]="^TestScan(Create|List|Show|Delete|Workflow|Logs|Filter|Threshold|Resubmit|Types)"
TEST_GROUPS["scan-engines"]="^Test(Container|Scs|CreateScan_With.*Engine|.*ApiSecurity|.*ExploitablePath)"
TEST_GROUPS["scm-integration"]="^Test(PR|UserCount)"
TEST_GROUPS["realtime-features"]="^Test(Kics|Sca|Oss|Secrets|Containers)Realtime|^TestRun.*Realtime"
TEST_GROUPS["advanced-features"]="^Test(Project|Result|Import|Bfl|Asca|Chat|Learn|Telemetry|RateLimit|PreCommit|PreReceive|Remediation)"

# Analyze each group
echo "=== Test Distribution by Group ==="
echo ""

declare -A GROUP_COUNTS
TOTAL_MATCHED=0

for group in "${!TEST_GROUPS[@]}"; do
  pattern="${TEST_GROUPS[$group]}"
  
  # Count tests matching this pattern
  count=$(echo "$ALL_TESTS" | grep -E "$pattern" | wc -l)
  GROUP_COUNTS[$group]=$count
  TOTAL_MATCHED=$((TOTAL_MATCHED + count))
  
  printf "%-25s: %3d tests\n" "$group" "$count"
done

echo ""
echo "Total tests matched: $TOTAL_MATCHED"
UNMATCHED=$((TOTAL_TESTS - TOTAL_MATCHED))
echo "Unmatched tests: $UNMATCHED"

if [ $UNMATCHED -gt 0 ]; then
  echo ""
  echo "=== Unmatched Tests ==="
  for test in $ALL_TESTS; do
    matched=false
    for pattern in "${TEST_GROUPS[@]}"; do
      if echo "$test" | grep -qE "$pattern"; then
        matched=true
        break
      fi
    done
    
    if [ "$matched" = false ]; then
      echo "  - $test"
    fi
  done
fi

echo ""
echo "=== Detailed Test Listing by Group ==="
echo ""

for group in fast-validation scan-core scan-engines scm-integration realtime-features advanced-features; do
  pattern="${TEST_GROUPS[$group]}"
  count="${GROUP_COUNTS[$group]}"
  
  echo "[$group] ($count tests)"
  echo "Pattern: $pattern"
  echo "Tests:"
  
  echo "$ALL_TESTS" | grep -E "$pattern" | sed 's/^/  - /' || echo "  (none)"
  echo ""
done

# Check for duplicate matches
echo "=== Checking for Duplicate Test Assignments ==="
echo ""

declare -A TEST_ASSIGNMENTS

for test in $ALL_TESTS; do
  groups_matched=""
  
  for group in "${!TEST_GROUPS[@]}"; do
    pattern="${TEST_GROUPS[$group]}"
    if echo "$test" | grep -qE "$pattern"; then
      if [ -z "$groups_matched" ]; then
        groups_matched="$group"
      else
        groups_matched="$groups_matched, $group"
      fi
    fi
  done
  
  if [ ! -z "$groups_matched" ]; then
    TEST_ASSIGNMENTS[$test]="$groups_matched"
  fi
done

DUPLICATES_FOUND=false
for test in "${!TEST_ASSIGNMENTS[@]}"; do
  groups="${TEST_ASSIGNMENTS[$test]}"
  if [[ "$groups" == *","* ]]; then
    echo "⚠️  $test matches multiple groups: $groups"
    DUPLICATES_FOUND=true
  fi
done

if [ "$DUPLICATES_FOUND" = false ]; then
  echo "✅ No duplicate test assignments found"
fi

echo ""
echo "=== Summary ==="
echo "Total tests: $TOTAL_TESTS"
echo "Matched tests: $TOTAL_MATCHED"
echo "Unmatched tests: $UNMATCHED"
echo "Coverage: $(awk "BEGIN {printf \"%.1f%%\", ($TOTAL_MATCHED/$TOTAL_TESTS)*100}")"

if [ $UNMATCHED -eq 0 ] && [ "$DUPLICATES_FOUND" = false ]; then
  echo ""
  echo "✅ Test grouping is valid and complete!"
  exit 0
else
  echo ""
  echo "⚠️  Test grouping needs adjustment"
  exit 1
fi
