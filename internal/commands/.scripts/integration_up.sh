#!/bin/bash

# Step 1: Check if the failedTests file exists
FAILED_TESTS_FILE="failedTests"

if [ -f "$FAILED_TESTS_FILE" ]; then
    # Step 2.1: If it exists, run all the tests listed in this file
    echo "Running tests from $FAILED_TESTS_FILE..."
    while IFS= read -r testName; do
        go test \
            -tags integration \
            -v \
            -timeout 210m \
            -coverpkg github.com/checkmarx/ast-cli/internal/commands,github.com/checkmarx/ast-cli/internal/services,github.com/checkmarx/ast-cli/internal/wrappers \
            -coverprofile cover.out \
            -run "^$testName$" \
            github.com/checkmarx/ast-cli/test/integration
    done < "$FAILED_TESTS_FILE"
else
    # Step 2.2: If not, create the failedTests file
    echo "Creating $FAILED_TESTS_FILE..."
    touch "$FAILED_TESTS_FILE"
fi

# Step 3: Run all tests and write failed test names to failedTests file
echo "Running all tests..."
go test \
    -tags integration \
    -v \
    -timeout 210m \
    -coverpkg github.com/checkmarx/ast-cli/internal/commands,github.com/checkmarx/ast-cli/internal/services,github.com/checkmarx/ast-cli/internal/wrappers \
    -coverprofile cover.out \
    github.com/checkmarx/ast-cli/test/integration 2>&1 | tee test_output.log

grep -E "^--- FAIL: " test_output.log | awk '{print $3}' > "$FAILED_TESTS_FILE"

status=$?
echo "status value after tests $status"
if [ $status -ne 0 ]; then
    echo "Integration tests failed"
fi

# Step 4: Check if failedTests file is empty
if [ ! -s "$FAILED_TESTS_FILE" ]; then
    # If empty, exit with success
    echo "All tests passed."
    rm -f "$FAILED_TESTS_FILE" test_output.log
    exit 0
else
    # If not empty, rerun the failed tests
    echo "Rerunning failed tests..."
    while IFS= read -r testName; do
        go test \
            -tags integration \
            -v \
            -timeout 210m \
            -coverpkg github.com/checkmarx/ast-cli/internal/commands,github.com/checkmarx/ast-cli/internal/services,github.com/checkmarx/ast-cli/internal/wrappers \
            -coverprofile cover.out \
            -run "^$testName$" \
            github.com/checkmarx/ast-cli/test/integration
    done < "$FAILED_TESTS_FILE"

    # Check if any tests failed again
    if [ -s "$FAILED_TESTS_FILE" ]; then
        echo "Some tests are still failing."
        exit 1
    else
        echo "All failed tests passed on rerun."
        rm -f "$FAILED_TESTS_FILE" test_output.log
        exit 0
    fi
fi
