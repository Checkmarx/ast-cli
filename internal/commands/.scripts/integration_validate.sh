#!/bin/bash
# =============================================================================
# Integration Test Validation & Aggregation Script
# =============================================================================
# This script runs after all shards complete to:
# 1. Verify ALL tests were executed across all shards
# 2. Merge coverage reports from all shards
# 3. Generate final summary report
# 4. Fail if any tests were missed or failed
#
# Usage: ./integration_validate.sh <total_shards>
# Example: ./integration_validate.sh 4
#
# Environment Variables:
#   TOTAL_SHARDS           - Total number of shards
#   EXPECTED_COVERAGE      - Minimum coverage percentage (default: 75)
#   ARTIFACTS_DIR          - Directory containing shard artifacts (default: .)
# =============================================================================

set -euo pipefail

# Configuration
TOTAL_SHARDS="${TOTAL_SHARDS:-${1:-4}}"
EXPECTED_COVERAGE="${EXPECTED_COVERAGE:-75}"
ARTIFACTS_DIR="${ARTIFACTS_DIR:-.}"
TEST_PACKAGE="github.com/checkmarx/ast-cli/test/integration"

# Output files
VALIDATION_REPORT="validation_report.json"
MERGED_COVERAGE="merged_coverage.out"
FINAL_COVERAGE_HTML="coverage.html"
SUMMARY_FILE="test_summary.md"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# =============================================================================
# Step 1: Discover All Expected Tests
# =============================================================================
discover_all_tests() {
    log_info "Discovering all expected integration tests..."

    ALL_TESTS=$(go test -tags integration -list "Test.*" "$TEST_PACKAGE" 2>/dev/null | grep "^Test" | sort)

    if [ -z "$ALL_TESTS" ]; then
        log_error "No tests discovered!"
        exit 1
    fi

    TOTAL_EXPECTED=$(echo "$ALL_TESTS" | wc -l | tr -d ' ')
    log_info "Expected total tests: ${TOTAL_EXPECTED}"

    echo "$ALL_TESTS"
}

# =============================================================================
# Step 2: Collect Results from All Shards
# =============================================================================
collect_shard_results() {
    log_info "Collecting results from ${TOTAL_SHARDS} shards..."

    local all_executed=""
    local all_failed=""
    local all_assigned=""
    local total_passed=0
    local total_failed=0
    local total_assigned=0
    local shards_found=0
    local shards_passed=0

    for shard in $(seq 1 "$TOTAL_SHARDS"); do
        local shard_dir="${ARTIFACTS_DIR}/shard_${shard}_of_${TOTAL_SHARDS}"
        local manifest="${shard_dir}/manifest.json"

        if [ ! -f "$manifest" ]; then
            log_error "Manifest not found for shard ${shard}: ${manifest}"
            continue
        fi

        shards_found=$((shards_found + 1))

        # Parse manifest
        local shard_assigned=$(jq -r '.tests_assigned' "$manifest")
        local shard_passed=$(jq -r '.tests_passed' "$manifest")
        local shard_failed=$(jq -r '.tests_failed' "$manifest")
        local shard_status=$(jq -r '.status' "$manifest")

        log_info "Shard ${shard}: assigned=${shard_assigned}, passed=${shard_passed}, failed=${shard_failed}, status=${shard_status}"

        # Aggregate counts
        total_assigned=$((total_assigned + shard_assigned))
        total_passed=$((total_passed + shard_passed))
        total_failed=$((total_failed + shard_failed))

        if [ "$shard_status" = "passed" ]; then
            shards_passed=$((shards_passed + 1))
        fi

        # Collect executed tests
        local executed_file="${shard_dir}/executed_tests.txt"
        if [ -f "$executed_file" ]; then
            if [ -n "$all_executed" ]; then
                all_executed="${all_executed}"$'\n'"$(cat "$executed_file")"
            else
                all_executed=$(cat "$executed_file")
            fi
        fi

        # Collect failed tests
        local failed_file="${shard_dir}/failed_tests.txt"
        if [ -f "$failed_file" ] && [ -s "$failed_file" ]; then
            if [ -n "$all_failed" ]; then
                all_failed="${all_failed}"$'\n'"$(cat "$failed_file")"
            else
                all_failed=$(cat "$failed_file")
            fi
        fi

        # Collect assigned tests
        local assigned_file="${shard_dir}/test_list.txt"
        if [ -f "$assigned_file" ]; then
            if [ -n "$all_assigned" ]; then
                all_assigned="${all_assigned}"$'\n'"$(cat "$assigned_file")"
            else
                all_assigned=$(cat "$assigned_file")
            fi
        fi
    done

    # Return results via global variables
    SHARDS_FOUND=$shards_found
    SHARDS_PASSED=$shards_passed
    TOTAL_ASSIGNED=$total_assigned
    TOTAL_PASSED=$total_passed
    TOTAL_FAILED=$total_failed
    ALL_EXECUTED_TESTS="$all_executed"
    ALL_FAILED_TESTS="$all_failed"
    ALL_ASSIGNED_TESTS="$all_assigned"
}

# =============================================================================
# Step 3: Validate All Tests Were Executed
# =============================================================================
validate_test_coverage() {
    local expected_tests="$1"

    log_info "Validating all tests were executed..."

    local expected_sorted=$(echo "$expected_tests" | sort | uniq)
    local executed_sorted=$(echo "$ALL_EXECUTED_TESTS" | sort | uniq)
    local assigned_sorted=$(echo "$ALL_ASSIGNED_TESTS" | sort | uniq)

    local expected_count=$(echo "$expected_sorted" | grep -c "^Test" || echo "0")
    local executed_count=$(echo "$executed_sorted" | grep -c "^Test" || echo "0")
    local assigned_count=$(echo "$assigned_sorted" | grep -c "^Test" || echo "0")

    log_info "Expected: ${expected_count}, Assigned: ${assigned_count}, Executed: ${executed_count}"

    # Find missing tests (expected but not assigned)
    MISSING_FROM_ASSIGNMENT=$(comm -23 <(echo "$expected_sorted") <(echo "$assigned_sorted") | grep "^Test" || true)

    # Find tests not executed (assigned but not executed)
    NOT_EXECUTED=$(comm -23 <(echo "$assigned_sorted") <(echo "$executed_sorted") | grep "^Test" || true)

    # Find extra tests (executed but not expected - shouldn't happen)
    EXTRA_TESTS=$(comm -13 <(echo "$expected_sorted") <(echo "$executed_sorted") | grep "^Test" || true)

    local validation_passed=true

    if [ -n "$MISSING_FROM_ASSIGNMENT" ]; then
        log_error "Tests missing from shard assignment:"
        echo "$MISSING_FROM_ASSIGNMENT" | head -20
        validation_passed=false
    fi

    if [ -n "$NOT_EXECUTED" ]; then
        log_error "Tests assigned but not executed:"
        echo "$NOT_EXECUTED" | head -20
        validation_passed=false
    fi

    if [ -n "$EXTRA_TESTS" ]; then
        log_warning "Extra tests executed (not in expected list):"
        echo "$EXTRA_TESTS" | head -10
    fi

    if [ "$validation_passed" = true ]; then
        log_success "All expected tests were executed!"
        return 0
    else
        return 1
    fi
}

# =============================================================================
# Step 4: Merge Coverage Reports
# =============================================================================
merge_coverage() {
    log_info "Merging coverage reports..."

    local coverage_files=""

    for shard in $(seq 1 "$TOTAL_SHARDS"); do
        local coverage_file="${ARTIFACTS_DIR}/shard_${shard}_of_${TOTAL_SHARDS}/cover.out"
        if [ -f "$coverage_file" ]; then
            if [ -n "$coverage_files" ]; then
                coverage_files="${coverage_files} ${coverage_file}"
            else
                coverage_files="${coverage_file}"
            fi
        fi
    done

    if [ -z "$coverage_files" ]; then
        log_warning "No coverage files found to merge"
        return 0
    fi

    # Check if gocovmerge is available
    if command -v gocovmerge &> /dev/null; then
        log_info "Merging with gocovmerge: ${coverage_files}"
        gocovmerge $coverage_files > "$MERGED_COVERAGE"
    else
        log_warning "gocovmerge not found, using first coverage file"
        cp $(echo "$coverage_files" | awk '{print $1}') "$MERGED_COVERAGE"
    fi

    # Generate HTML report
    if [ -f "$MERGED_COVERAGE" ]; then
        go tool cover -html="$MERGED_COVERAGE" -o "$FINAL_COVERAGE_HTML" 2>/dev/null || true

        # Calculate coverage percentage
        COVERAGE_PCT=$(go tool cover -func "$MERGED_COVERAGE" | grep total | awk '{print substr($3, 1, length($3)-1)}')
        log_info "Total coverage: ${COVERAGE_PCT}%"
    fi
}

# =============================================================================
# Step 5: Check Coverage Threshold
# =============================================================================
check_coverage_threshold() {
    if [ -z "${COVERAGE_PCT:-}" ]; then
        log_warning "Coverage percentage not available"
        return 0
    fi

    log_info "Checking coverage threshold: ${COVERAGE_PCT}% >= ${EXPECTED_COVERAGE}%"

    local result=$(awk "BEGIN {print ($COVERAGE_PCT < $EXPECTED_COVERAGE)}")
    if [ "$result" -eq 1 ]; then
        log_error "Coverage ${COVERAGE_PCT}% is below threshold ${EXPECTED_COVERAGE}%"
        return 1
    else
        log_success "Coverage ${COVERAGE_PCT}% meets threshold ${EXPECTED_COVERAGE}%"
        return 0
    fi
}

# =============================================================================
# Step 6: Generate Reports
# =============================================================================
generate_reports() {
    local expected_count="$1"
    local validation_status="$2"
    local coverage_status="$3"

    # JSON Report
    local missing_json="[]"
    local not_executed_json="[]"
    local failed_json="[]"

    if [ -n "$MISSING_FROM_ASSIGNMENT" ]; then
        missing_json=$(echo "$MISSING_FROM_ASSIGNMENT" | jq -R -s 'split("\n") | map(select(length > 0))')
    fi

    if [ -n "$NOT_EXECUTED" ]; then
        not_executed_json=$(echo "$NOT_EXECUTED" | jq -R -s 'split("\n") | map(select(length > 0))')
    fi

    if [ -n "$ALL_FAILED_TESTS" ]; then
        failed_json=$(echo "$ALL_FAILED_TESTS" | jq -R -s 'split("\n") | map(select(length > 0))')
    fi

    local final_status="passed"
    if [ "$validation_status" != "0" ] || [ "$coverage_status" != "0" ] || [ "$TOTAL_FAILED" -gt 0 ]; then
        final_status="failed"
    fi

    cat > "$VALIDATION_REPORT" << EOF
{
    "total_shards": ${TOTAL_SHARDS},
    "shards_found": ${SHARDS_FOUND},
    "shards_passed": ${SHARDS_PASSED},
    "expected_tests": ${expected_count},
    "assigned_tests": ${TOTAL_ASSIGNED},
    "passed_tests": ${TOTAL_PASSED},
    "failed_tests": ${TOTAL_FAILED},
    "coverage_percentage": ${COVERAGE_PCT:-0},
    "coverage_threshold": ${EXPECTED_COVERAGE},
    "validation_status": "${validation_status}",
    "coverage_status": "${coverage_status}",
    "final_status": "${final_status}",
    "missing_from_assignment": ${missing_json},
    "not_executed": ${not_executed_json},
    "failed_tests_list": ${failed_json},
    "timestamp": "$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
}
EOF

    # Markdown Summary
    cat > "$SUMMARY_FILE" << EOF
# Integration Test Summary

## Overview
| Metric | Value |
|--------|-------|
| Total Shards | ${TOTAL_SHARDS} |
| Shards Completed | ${SHARDS_FOUND} |
| Shards Passed | ${SHARDS_PASSED} |
| Expected Tests | ${expected_count} |
| Assigned Tests | ${TOTAL_ASSIGNED} |
| Passed Tests | ${TOTAL_PASSED} |
| Failed Tests | ${TOTAL_FAILED} |
| Coverage | ${COVERAGE_PCT:-N/A}% |
| Threshold | ${EXPECTED_COVERAGE}% |
| **Final Status** | **${final_status^^}** |

## Validation Results
- Test Assignment: $([ "$validation_status" = "0" ] && echo "PASS" || echo "FAIL")
- Coverage Threshold: $([ "$coverage_status" = "0" ] && echo "PASS" || echo "FAIL")
- All Tests Executed: $([ "$TOTAL_ASSIGNED" -eq "$TOTAL_PASSED" ] && echo "PASS" || echo "FAIL")
EOF

    if [ -n "$ALL_FAILED_TESTS" ]; then
        echo "" >> "$SUMMARY_FILE"
        echo "## Failed Tests" >> "$SUMMARY_FILE"
        echo '```' >> "$SUMMARY_FILE"
        echo "$ALL_FAILED_TESTS" >> "$SUMMARY_FILE"
        echo '```' >> "$SUMMARY_FILE"
    fi

    if [ -n "$MISSING_FROM_ASSIGNMENT" ]; then
        echo "" >> "$SUMMARY_FILE"
        echo "## Missing Tests (Not Assigned to Any Shard)" >> "$SUMMARY_FILE"
        echo '```' >> "$SUMMARY_FILE"
        echo "$MISSING_FROM_ASSIGNMENT" >> "$SUMMARY_FILE"
        echo '```' >> "$SUMMARY_FILE"
    fi

    log_info "Reports generated: ${VALIDATION_REPORT}, ${SUMMARY_FILE}"
}

# =============================================================================
# Step 7: Cleanup Test Data
# =============================================================================
run_cleanup() {
    log_info "Running test data cleanup..."

    set +e
    go test -v github.com/checkmarx/ast-cli/test/cleandata 2>&1 || true
    set -e
}

# =============================================================================
# Main Execution
# =============================================================================
main() {
    local exit_code=0

    log_info "========================================"
    log_info "Integration Test Validation"
    log_info "========================================"
    log_info "Total Shards: ${TOTAL_SHARDS}"
    log_info "Expected Coverage: ${EXPECTED_COVERAGE}%"

    # Discover expected tests
    EXPECTED_TESTS=$(discover_all_tests)
    EXPECTED_COUNT=$(echo "$EXPECTED_TESTS" | wc -l | tr -d ' ')

    # Collect shard results
    collect_shard_results

    # Validate test coverage
    set +e
    validate_test_coverage "$EXPECTED_TESTS"
    validation_status=$?
    set -e

    # Merge coverage
    merge_coverage

    # Check coverage threshold
    set +e
    check_coverage_threshold
    coverage_status=$?
    set -e

    # Generate reports
    generate_reports "$EXPECTED_COUNT" "$validation_status" "$coverage_status"

    # Run cleanup
    run_cleanup

    # Determine final status
    if [ "$validation_status" -ne 0 ]; then
        log_error "Validation failed - not all tests were executed!"
        exit_code=1
    fi

    if [ "$coverage_status" -ne 0 ]; then
        log_error "Coverage threshold not met!"
        exit_code=1
    fi

    if [ "$TOTAL_FAILED" -gt 0 ]; then
        log_error "${TOTAL_FAILED} tests failed!"
        exit_code=1
    fi

    if [ "$SHARDS_FOUND" -ne "$TOTAL_SHARDS" ]; then
        log_error "Not all shards completed! Found ${SHARDS_FOUND}/${TOTAL_SHARDS}"
        exit_code=1
    fi

    # Final summary
    log_info "========================================"
    log_info "Final Summary"
    log_info "========================================"
    log_info "Expected:  ${EXPECTED_COUNT} tests"
    log_info "Executed:  ${TOTAL_PASSED} passed, ${TOTAL_FAILED} failed"
    log_info "Coverage:  ${COVERAGE_PCT:-N/A}%"

    if [ "$exit_code" -eq 0 ]; then
        log_success "All validations passed!"
    else
        log_error "Validation failed!"
        cat "$SUMMARY_FILE"
    fi

    exit $exit_code
}

main "$@"
