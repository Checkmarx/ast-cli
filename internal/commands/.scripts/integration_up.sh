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
    exit 0
else
    # If the file is not empty, rerun the failed tests
    echo "Rerunning failed tests..."
    rerun_status=0
    while IFS= read -r testName; do
        go test \
            -tags integration \
            -v \
            -timeout 210m \
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
        rm -f "$FAILED_TESTS_FILE" test_output.log
        exit 1
    else
        echo "All failed tests passed on rerun."
        rm -f "$FAILED_TESTS_FILE" test_output.log
        exit 0
    fi
fi
