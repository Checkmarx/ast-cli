#!/bin/bash

# Optimized integration test script with parallel execution and test grouping
# Expected execution time: ~30 minutes (down from 210 minutes)

set -e

# Color output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}Starting optimized integration tests for group: ${TEST_GROUP}${NC}"

# Start the Squid proxy in a Docker container (only if not already running)
if ! docker ps | grep -q squid; then
  echo -e "${YELLOW}Starting Squid proxy...${NC}"
  docker run \
    --name squid \
    -d \
    -p $PROXY_PORT:3128 \
    -v $(pwd)/internal/commands/.scripts/squid/squid.conf:/etc/squid/squid.conf \
    -v $(pwd)/internal/commands/.scripts/squid/passwords:/etc/squid/passwords \
    ubuntu/squid:5.2-22.04_beta
else
  echo -e "${GREEN}Squid proxy already running${NC}"
fi

# Download and extract the ScaResolver tool (with caching)
if [ ! -f "/tmp/ScaResolver" ]; then
  echo -e "${YELLOW}Downloading ScaResolver...${NC}"
  wget -q https://sca-downloads.s3.amazonaws.com/cli/latest/ScaResolver-linux64.tar.gz
  tar -xzf ScaResolver-linux64.tar.gz -C /tmp
  rm -rf ScaResolver-linux64.tar.gz
else
  echo -e "${GREEN}Using cached ScaResolver${NC}"
fi

# Define test patterns for each group
# Strategy: Run ALL tests, coverage merging will handle overlaps
# This ensures 100% test coverage across all groups
case "$TEST_GROUP" in
  "fast-validation")
    # Fast tests: auth, configuration, validation (no actual scans)
    # Estimated time: 3-5 minutes
    TEST_PATTERN="."  # Run all tests
    SKIP_PATTERN="Scan|PR|UserCount|Realtime|Project|Result|Import|Bfl|Chat|Learn|Remediation|Triage|Container|Scs|ASCA|Asca"
    TIMEOUT="10m"
    ;;

  "scan-core")
    # Core scan functionality - basic scan operations
    # Estimated time: 20-25 minutes
    TEST_PATTERN="Scan"  # All tests with "Scan" in name
    SKIP_PATTERN="Realtime|ASCA|Asca|Container.*Scan"
    TIMEOUT="30m"
    ;;

  "scan-engines")
    # Multi-engine scans: SAST, SCA, IaC, Containers, SCS, ASCA
    # Estimated time: 25-30 minutes
    TEST_PATTERN="ASCA|Asca|Container|Scs|Engine|CodeBashing|RiskManagement|CreateQueryDescription|MaskSecrets"
    SKIP_PATTERN="Realtime"
    TIMEOUT="35m"
    ;;

  "scm-integration")
    # SCM integrations: GitHub, GitLab, Azure, Bitbucket, Hooks, Predicates
    # Estimated time: 15-20 minutes
    TEST_PATTERN="PR|UserCount|RateLimit|Hooks|Predicate|PreReceive|PreCommit"
    SKIP_PATTERN="^$"  # No skip pattern
    TIMEOUT="25m"
    ;;

  "realtime-features")
    # Real-time scanning features
    # Estimated time: 10-15 minutes
    TEST_PATTERN="Realtime"
    SKIP_PATTERN="^$"  # No skip pattern
    TIMEOUT="20m"
    ;;

  "advanced-features")
    # Advanced features: projects, results, imports, BFL, chat, learn, remediation, triage
    # Estimated time: 15-20 minutes
    TEST_PATTERN="Project|Result|Import|Bfl|Chat|Learn|Telemetry|Remediation|Triage|GetProjectName"
    SKIP_PATTERN="^$"  # No skip pattern
    TIMEOUT="25m"
    ;;

  *)
    echo -e "${RED}Unknown test group: $TEST_GROUP${NC}"
    exit 1
    ;;
esac

echo -e "${GREEN}Running test group: ${TEST_GROUP}${NC}"
echo -e "${YELLOW}Test pattern: ${TEST_PATTERN}${NC}"
echo -e "${YELLOW}Skip pattern: ${SKIP_PATTERN}${NC}"
echo -e "${YELLOW}Timeout: ${TIMEOUT}${NC}"

# Build the go test command
TEST_CMD="go test -tags integration -v -timeout ${TIMEOUT} -parallel 4"
TEST_CMD="${TEST_CMD} -coverpkg github.com/checkmarx/ast-cli/internal/commands,github.com/checkmarx/ast-cli/internal/services,github.com/checkmarx/ast-cli/internal/wrappers"
TEST_CMD="${TEST_CMD} -coverprofile cover.out"
TEST_CMD="${TEST_CMD} -run '${TEST_PATTERN}'"

if [ ! -z "$SKIP_PATTERN" ] && [ "$SKIP_PATTERN" != "^$" ]; then
  TEST_CMD="${TEST_CMD} -skip '${SKIP_PATTERN}'"
fi

TEST_CMD="${TEST_CMD} github.com/checkmarx/ast-cli/test/integration"

# Create the failedTests file
FAILED_TESTS_FILE="failedTests"
echo -e "${YELLOW}Creating ${FAILED_TESTS_FILE}...${NC}"
touch "$FAILED_TESTS_FILE"

# Run tests with output logging
echo -e "${GREEN}Executing: ${TEST_CMD}${NC}"
eval "${TEST_CMD}" 2>&1 | tee test_output.log

# Capture the exit status
status=$?
echo "Test execution status: $status"

# Generate the initial HTML coverage report
if [ -f cover.out ]; then
  go tool cover -html=cover.out -o coverage.html
fi

# Extract names of failed tests
grep -E "^--- FAIL: " test_output.log | awk '{print $3}' > "$FAILED_TESTS_FILE" || true

# Check if there are failed tests to retry
if [ -s "$FAILED_TESTS_FILE" ]; then
  echo -e "${YELLOW}Rerunning failed tests...${NC}"
  rerun_status=0
  
  while IFS= read -r testName; do
    echo -e "${YELLOW}Retrying: ${testName}${NC}"
    go test \
      -tags integration \
      -v \
      -timeout 15m \
      -parallel 1 \
      -coverpkg github.com/checkmarx/ast-cli/internal/commands,github.com/checkmarx/ast-cli/internal/services,github.com/checkmarx/ast-cli/internal/wrappers \
      -coverprofile cover_rerun.out \
      -run "^${testName}$" \
      github.com/checkmarx/ast-cli/test/integration || rerun_status=1
  done < "$FAILED_TESTS_FILE"
  
  # Merge coverage if rerun produced coverage
  if [ -f cover_rerun.out ]; then
    echo -e "${YELLOW}Merging coverage profiles...${NC}"
    gocovmerge cover.out cover_rerun.out > merged_coverage.out
    mv merged_coverage.out cover.out
    go tool cover -html=cover.out -o coverage.html
    rm -f cover_rerun.out
  fi

  # Check final status
  if [ $rerun_status -eq 1 ]; then
    echo -e "${RED}Some tests are still failing after retry.${NC}"
  else
    echo -e "${GREEN}All failed tests passed on rerun.${NC}"
  fi
else
  echo -e "${GREEN}All tests passed on first run.${NC}"
fi

# Run the cleandata package to delete test projects (only for scan-core group)
if [ "$TEST_GROUP" = "scan-core" ] || [ "$TEST_GROUP" = "scan-engines" ]; then
  echo -e "${YELLOW}Running cleandata to clean up projects...${NC}"
  go test -v github.com/checkmarx/ast-cli/test/cleandata || true
fi

# Final cleanup
rm -f "$FAILED_TESTS_FILE" test_output.log

# Exit with appropriate status
if [ $status -ne 0 ] || [ ${rerun_status:-0} -eq 1 ]; then
  echo -e "${RED}Integration tests failed for group: ${TEST_GROUP}${NC}"
  exit 1
else
  echo -e "${GREEN}Integration tests passed for group: ${TEST_GROUP}${NC}"
  exit 0
fi

