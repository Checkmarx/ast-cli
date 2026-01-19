#!/bin/bash
# =============================================================================
# Production-Ready Integration Test Sharding Script
# =============================================================================
# This script dynamically discovers all integration tests and runs a specific
# shard. It ensures all tests are accounted for and provides validation.
#
# Usage: ./integration_shard.sh <shard_index> <total_shards>
# Example: ./integration_shard.sh 1 4  (runs shard 1 of 4)
#
# Environment Variables:
#   SHARD_INDEX     - Current shard (1-based), overrides first argument
#   TOTAL_SHARDS    - Total number of shards, overrides second argument
#   TEST_TIMEOUT    - Timeout per shard (default: 45m)
#   RERUN_TIMEOUT   - Timeout for rerunning failed tests (default: 15m)
# =============================================================================

set -euo pipefail

# Configuration
SHARD_INDEX="${SHARD_INDEX:-${1:-1}}"
TOTAL_SHARDS="${TOTAL_SHARDS:-${2:-4}}"
TEST_TIMEOUT="${TEST_TIMEOUT:-45m}"
RERUN_TIMEOUT="${RERUN_TIMEOUT:-15m}"
TEST_PACKAGE="github.com/checkmarx/ast-cli/test/integration"
COVERAGE_PACKAGES="github.com/checkmarx/ast-cli/internal/commands,github.com/checkmarx/ast-cli/internal/services,github.com/checkmarx/ast-cli/internal/wrappers"

# Output files
SHARD_DIR="./shard_${SHARD_INDEX}_of_${TOTAL_SHARDS}"
TEST_LIST_FILE="${SHARD_DIR}/test_list.txt"
EXECUTED_TESTS_FILE="${SHARD_DIR}/executed_tests.txt"
FAILED_TESTS_FILE="${SHARD_DIR}/failed_tests.txt"
COVERAGE_FILE="${SHARD_DIR}/cover.out"
TEST_OUTPUT_FILE="${SHARD_DIR}/test_output.log"
MANIFEST_FILE="${SHARD_DIR}/manifest.json"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

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
# Step 1: Setup and Validation
# =============================================================================
setup() {
    log_info "Setting up shard ${SHARD_INDEX} of ${TOTAL_SHARDS}"

    # Validate inputs
    if [[ ! "$SHARD_INDEX" =~ ^[0-9]+$ ]] || [[ ! "$TOTAL_SHARDS" =~ ^[0-9]+$ ]]; then
        log_error "SHARD_INDEX and TOTAL_SHARDS must be positive integers"
        exit 1
    fi

    if [ "$SHARD_INDEX" -lt 1 ] || [ "$SHARD_INDEX" -gt "$TOTAL_SHARDS" ]; then
        log_error "SHARD_INDEX must be between 1 and TOTAL_SHARDS"
        exit 1
    fi

    # Create shard directory
    mkdir -p "$SHARD_DIR"

    # Initialize files
    > "$TEST_LIST_FILE"
    > "$EXECUTED_TESTS_FILE"
    > "$FAILED_TESTS_FILE"
    > "$TEST_OUTPUT_FILE"
}

# =============================================================================
# Step 2: Discover All Tests
# =============================================================================
discover_tests() {
    log_info "Discovering all integration tests..."

    # Get all test function names (sorted for deterministic sharding)
    ALL_TESTS=$(go test -tags integration -list "Test.*" "$TEST_PACKAGE" 2>/dev/null | grep "^Test" | sort)

    if [ -z "$ALL_TESTS" ]; then
        log_error "No tests discovered! Check if test package exists."
        exit 1
    fi

    TOTAL_TESTS=$(echo "$ALL_TESTS" | wc -l | tr -d ' ')
    log_info "Discovered ${TOTAL_TESTS} total tests"

    echo "$ALL_TESTS"
}

# =============================================================================
# Step 3: Calculate Shard Assignment
# =============================================================================
get_shard_tests() {
    local all_tests="$1"
    local shard_tests=""
    local test_index=0

    # Distribute tests using modulo for even distribution
    # This ensures new tests are automatically distributed across shards
    while IFS= read -r test_name; do
        test_index=$((test_index + 1))
        # Assign test to shard using: (test_index - 1) % total_shards + 1
        assigned_shard=$(( ((test_index - 1) % TOTAL_SHARDS) + 1 ))

        if [ "$assigned_shard" -eq "$SHARD_INDEX" ]; then
            if [ -n "$shard_tests" ]; then
                shard_tests="${shard_tests}"$'\n'"${test_name}"
            else
                shard_tests="${test_name}"
            fi
        fi
    done <<< "$all_tests"

    echo "$shard_tests"
}

# =============================================================================
# Step 4: Build Test Pattern
# =============================================================================
build_test_pattern() {
    local tests="$1"

    if [ -z "$tests" ]; then
        echo ""
        return
    fi

    # Build regex pattern: ^(Test1|Test2|Test3)$
    local pattern="^("
    local first=true

    while IFS= read -r test_name; do
        if [ "$first" = true ]; then
            pattern="${pattern}${test_name}"
            first=false
        else
            pattern="${pattern}|${test_name}"
        fi
    done <<< "$tests"

    pattern="${pattern})$"
    echo "$pattern"
}

# =============================================================================
# Step 5: Run Tests
# =============================================================================
run_tests() {
    local pattern="$1"
    local shard_test_count="$2"

    if [ -z "$pattern" ] || [ "$shard_test_count" -eq 0 ]; then
        log_warning "No tests assigned to this shard"
        echo '{"shard": '"$SHARD_INDEX"', "total_shards": '"$TOTAL_SHARDS"', "tests_assigned": 0, "tests_passed": 0, "tests_failed": 0, "status": "empty"}' > "$MANIFEST_FILE"
        return 0
    fi

    log_info "Running ${shard_test_count} tests with pattern: ${pattern:0:100}..."

    # Run tests and capture output
    set +e
    go test \
        -tags integration \
        -v \
        -timeout "$TEST_TIMEOUT" \
        -coverpkg "$COVERAGE_PACKAGES" \
        -coverprofile "$COVERAGE_FILE" \
        -run "$pattern" \
        "$TEST_PACKAGE" 2>&1 | tee "$TEST_OUTPUT_FILE"

    local test_exit_code=${PIPESTATUS[0]}
    set -e

    return $test_exit_code
}

# =============================================================================
# Step 6: Extract Results
# =============================================================================
extract_results() {
    # Extract passed tests
    grep -E "^--- PASS: " "$TEST_OUTPUT_FILE" | awk '{print $3}' > "$EXECUTED_TESTS_FILE" 2>/dev/null || true

    # Extract failed tests
    grep -E "^--- FAIL: " "$TEST_OUTPUT_FILE" | awk '{print $3}' > "$FAILED_TESTS_FILE" 2>/dev/null || true

    local passed_count=$(wc -l < "$EXECUTED_TESTS_FILE" 2>/dev/null | tr -d ' ' || echo "0")
    local failed_count=$(wc -l < "$FAILED_TESTS_FILE" 2>/dev/null | tr -d ' ' || echo "0")

    log_info "Passed: ${passed_count}, Failed: ${failed_count}"

    echo "$passed_count $failed_count"
}

# =============================================================================
# Step 7: Rerun Failed Tests
# =============================================================================
rerun_failed_tests() {
    if [ ! -s "$FAILED_TESTS_FILE" ]; then
        return 0
    fi

    log_warning "Rerunning failed tests..."

    local rerun_status=0
    local rerun_coverage="${SHARD_DIR}/cover_rerun.out"

    while IFS= read -r test_name; do
        log_info "Rerunning: ${test_name}"

        set +e
        go test \
            -tags integration \
            -v \
            -timeout "$RERUN_TIMEOUT" \
            -coverpkg "$COVERAGE_PACKAGES" \
            -coverprofile "$rerun_coverage" \
            -run "^${test_name}$" \
            "$TEST_PACKAGE" 2>&1 | tee -a "$TEST_OUTPUT_FILE"

        if [ ${PIPESTATUS[0]} -ne 0 ]; then
            rerun_status=1
        else
            # Remove from failed, add to passed
            sed -i "/${test_name}/d" "$FAILED_TESTS_FILE" 2>/dev/null || true
            echo "$test_name" >> "$EXECUTED_TESTS_FILE"
        fi
        set -e

        # Merge coverage if rerun coverage exists
        if [ -f "$rerun_coverage" ] && [ -f "$COVERAGE_FILE" ]; then
            if command -v gocovmerge &> /dev/null; then
                gocovmerge "$COVERAGE_FILE" "$rerun_coverage" > "${SHARD_DIR}/merged_coverage.out"
                mv "${SHARD_DIR}/merged_coverage.out" "$COVERAGE_FILE"
            fi
            rm -f "$rerun_coverage"
        fi
    done < "$FAILED_TESTS_FILE"

    return $rerun_status
}

# =============================================================================
# Step 8: Generate Manifest
# =============================================================================
generate_manifest() {
    local assigned_count="$1"
    local passed_count="$2"
    local failed_count="$3"
    local status="$4"

    # Get list of executed tests
    local executed_tests_json="[]"
    if [ -s "$EXECUTED_TESTS_FILE" ]; then
        executed_tests_json=$(cat "$EXECUTED_TESTS_FILE" | jq -R -s 'split("\n") | map(select(length > 0))')
    fi

    # Get list of failed tests
    local failed_tests_json="[]"
    if [ -s "$FAILED_TESTS_FILE" ]; then
        failed_tests_json=$(cat "$FAILED_TESTS_FILE" | jq -R -s 'split("\n") | map(select(length > 0))')
    fi

    # Get list of assigned tests
    local assigned_tests_json="[]"
    if [ -s "$TEST_LIST_FILE" ]; then
        assigned_tests_json=$(cat "$TEST_LIST_FILE" | jq -R -s 'split("\n") | map(select(length > 0))')
    fi

    cat > "$MANIFEST_FILE" << EOF
{
    "shard_index": ${SHARD_INDEX},
    "total_shards": ${TOTAL_SHARDS},
    "tests_assigned": ${assigned_count},
    "tests_passed": ${passed_count},
    "tests_failed": ${failed_count},
    "status": "${status}",
    "assigned_tests": ${assigned_tests_json},
    "executed_tests": ${executed_tests_json},
    "failed_tests": ${failed_tests_json},
    "coverage_file": "${COVERAGE_FILE}",
    "timestamp": "$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
}
EOF

    log_info "Manifest generated: ${MANIFEST_FILE}"
}

# =============================================================================
# Main Execution
# =============================================================================
main() {
    local exit_code=0

    log_info "========================================"
    log_info "Integration Test Shard ${SHARD_INDEX}/${TOTAL_SHARDS}"
    log_info "========================================"

    # Setup
    setup

    # Discover all tests
    ALL_TESTS=$(discover_tests)
    TOTAL_TEST_COUNT=$(echo "$ALL_TESTS" | wc -l | tr -d ' ')

    # Get tests for this shard
    SHARD_TESTS=$(get_shard_tests "$ALL_TESTS")

    if [ -z "$SHARD_TESTS" ]; then
        SHARD_TEST_COUNT=0
    else
        SHARD_TEST_COUNT=$(echo "$SHARD_TESTS" | wc -l | tr -d ' ')
    fi

    # Save assigned tests to file
    echo "$SHARD_TESTS" > "$TEST_LIST_FILE"

    log_info "Total tests: ${TOTAL_TEST_COUNT}"
    log_info "Tests for shard ${SHARD_INDEX}: ${SHARD_TEST_COUNT}"

    # Build pattern and run tests
    TEST_PATTERN=$(build_test_pattern "$SHARD_TESTS")

    set +e
    run_tests "$TEST_PATTERN" "$SHARD_TEST_COUNT"
    test_exit_code=$?
    set -e

    # Extract results
    results=$(extract_results)
    passed_count=$(echo "$results" | awk '{print $1}')
    failed_count=$(echo "$results" | awk '{print $2}')

    # Rerun failed tests if any
    if [ "$failed_count" -gt 0 ]; then
        set +e
        rerun_failed_tests
        rerun_exit_code=$?
        set -e

        # Re-extract results after rerun
        results=$(extract_results)
        passed_count=$(echo "$results" | awk '{print $1}')
        failed_count=$(echo "$results" | awk '{print $2}')

        if [ "$rerun_exit_code" -ne 0 ]; then
            exit_code=1
        fi
    fi

    # Determine final status
    if [ "$failed_count" -gt 0 ]; then
        status="failed"
        exit_code=1
    elif [ "$passed_count" -eq "$SHARD_TEST_COUNT" ]; then
        status="passed"
    else
        status="incomplete"
        exit_code=1
    fi

    # Generate manifest
    generate_manifest "$SHARD_TEST_COUNT" "$passed_count" "$failed_count" "$status"

    # Summary
    log_info "========================================"
    log_info "Shard ${SHARD_INDEX}/${TOTAL_SHARDS} Summary"
    log_info "========================================"
    log_info "Assigned: ${SHARD_TEST_COUNT}"
    log_info "Passed:   ${passed_count}"
    log_info "Failed:   ${failed_count}"
    log_info "Status:   ${status}"

    if [ "$exit_code" -eq 0 ]; then
        log_success "Shard ${SHARD_INDEX} completed successfully!"
    else
        log_error "Shard ${SHARD_INDEX} failed!"
    fi

    exit $exit_code
}

main "$@"
