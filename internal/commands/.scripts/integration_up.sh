# Accept optional test filter as first argument (regex pattern for -run flag)
TEST_FILTER=${1:-""}

# Start the Squid proxy in a Docker container
docker run \
  --name squid \
  -d \
  -p $PROXY_PORT:3128 \
  -v $(pwd)/internal/commands/.scripts/squid/squid.conf:/etc/squid/squid.conf \
  -v $(pwd)/internal/commands/.scripts/squid/passwords:/etc/squid/passwords \
  ubuntu/squid:5.2-22.04_beta

# Download and extract the ScaResolver tool
wget https://sca-downloads.s3.amazonaws.com/cli/latest/ScaResolver-linux64.tar.gz
tar -xzvf ScaResolver-linux64.tar.gz -C /tmp
rm -rf ScaResolver-linux64.tar.gz

# Build the test filter argument if provided
RUN_ARG=""
if [ -n "$TEST_FILTER" ]; then
    RUN_ARG="-run $TEST_FILTER"
    echo "Running tests matching filter: $TEST_FILTER"
fi

# Step 1: Check if the failedTests file exists
FAILED_TESTS_FILE="failedTests"

# Step 2: Create the failedTests file
echo "Creating $FAILED_TESTS_FILE..."
touch "$FAILED_TESTS_FILE"

# Step 3: Run all tests and write failed test names to failedTests file
echo "Running all tests..."
go test \
    -tags integration \
    -v \
    -timeout 210m \
    $RUN_ARG \
    -coverpkg github.com/checkmarx/ast-cli/internal/commands,github.com/checkmarx/ast-cli/internal/services,github.com/checkmarx/ast-cli/internal/wrappers \
    -coverprofile cover.out \
    github.com/checkmarx/ast-cli/test/integration 2>&1 | tee test_output.log

# Generate the initial HTML coverage report
go tool cover -html=cover.out -o coverage.html

# Extract names of failed tests and save them in the failedTests file
grep -E "^--- FAIL: " test_output.log | awk '{print $3}' > "$FAILED_TESTS_FILE"

# Capture the exit status of the tests
status=$?
echo "status value after tests $status"
if [ $status -ne 0 ]; then
    echo "Integration tests failed"
fi

# Step 4: Check if failedTests file is empty
if [ ! -s "$FAILED_TESTS_FILE" ]; then
    # If the file is empty, all tests passed
    echo "All tests passed."
    rm -f "$FAILED_TESTS_FILE" test_output.log
else
    # If the file is not empty, rerun the failed tests
    echo "Rerunning failed tests..."
    rerun_status=0
    while IFS= read -r testName; do
        go test \
            -tags integration \
            -v \
            -timeout 30m \
            -coverpkg github.com/checkmarx/ast-cli/internal/commands,github.com/checkmarx/ast-cli/internal/services,github.com/checkmarx/ast-cli/internal/wrappers \
            -coverprofile cover_rerun.out \
            -run "^$testName$" \
            github.com/checkmarx/ast-cli/test/integration || rerun_status=1
    done < "$FAILED_TESTS_FILE"

    # Step 5: Merge the original and rerun coverage profiles
    if [ -f cover_rerun.out ]; then
        echo "Merging coverage profiles..."
        gocovmerge cover.out cover_rerun.out > merged_coverage.out
        mv merged_coverage.out cover.out
        go tool cover -html=cover.out -o coverage.html
        rm -f cover_rerun.out
    fi

    # Step 6: Check if any tests failed again
    if [ $rerun_status -eq 1 ]; then
        echo "Some tests are still failing."
    else
        echo "All failed tests passed on rerun."
    fi
fi

# Step 7: Generate test summary table
echo ""
echo "=============================================="
echo "           TEST EXECUTION SUMMARY             "
echo "=============================================="

# Parse test results from log
TOTAL_PASSED=$(grep -c "^--- PASS:" test_output.log 2>/dev/null || echo "0")
TOTAL_FAILED=$(grep -c "^--- FAIL:" test_output.log 2>/dev/null || echo "0")
TOTAL_SKIPPED=$(grep -c "^--- SKIP:" test_output.log 2>/dev/null || echo "0")
TOTAL_TESTS=$((TOTAL_PASSED + TOTAL_FAILED + TOTAL_SKIPPED))

# Calculate pass rate
if [ "$TOTAL_TESTS" -gt 0 ]; then
    PASS_RATE=$(awk "BEGIN {printf \"%.1f\", ($TOTAL_PASSED/$TOTAL_TESTS)*100}")
else
    PASS_RATE="0.0"
fi

# Extract duration from test output
DURATION=$(grep -E "^(ok|FAIL)\s+github.com/checkmarx/ast-cli/test/integration" test_output.log | awk '{print $NF}' | head -1)
if [ -z "$DURATION" ]; then
    DURATION="N/A"
fi

# Get test filter info
if [ -n "$TEST_FILTER" ]; then
    FILTER_INFO="$TEST_FILTER"
else
    FILTER_INFO="All tests"
fi

# Print summary table
printf "\n"
printf "┌─────────────────────┬─────────────────────┐\n"
printf "│ %-19s │ %-19s │\n" "Metric" "Value"
printf "├─────────────────────┼─────────────────────┤\n"
printf "│ %-19s │ %-19s │\n" "Test Filter" "$FILTER_INFO"
printf "│ %-19s │ %-19s │\n" "Total Tests" "$TOTAL_TESTS"
printf "│ %-19s │ \033[32m%-19s\033[0m │\n" "Passed" "$TOTAL_PASSED"
printf "│ %-19s │ \033[31m%-19s\033[0m │\n" "Failed" "$TOTAL_FAILED"
printf "│ %-19s │ \033[33m%-19s\033[0m │\n" "Skipped" "$TOTAL_SKIPPED"
printf "│ %-19s │ %-19s │\n" "Pass Rate" "${PASS_RATE}%"
printf "│ %-19s │ %-19s │\n" "Duration" "$DURATION"
printf "└─────────────────────┴─────────────────────┘\n"

# Print failed test names if any
if [ "$TOTAL_FAILED" -gt 0 ]; then
    echo ""
    echo "Failed Tests:"
    echo "─────────────"
    grep "^--- FAIL:" test_output.log | awk '{print "  ✗ " $3}' | head -20
    if [ "$TOTAL_FAILED" -gt 20 ]; then
        echo "  ... and $((TOTAL_FAILED - 20)) more"
    fi
fi

echo ""
echo "=============================================="

# Step 8: Run the cleandata package to delete projects
echo "Running cleandata to clean up projects..."
go test -v github.com/checkmarx/ast-cli/test/cleandata

# Step 9: Final cleanup and exit
rm -f "$FAILED_TESTS_FILE" test_output.log

if [ $status -ne 0 ] || [ $rerun_status -eq 1 ]; then
    exit 1
else
    exit 0
fi
