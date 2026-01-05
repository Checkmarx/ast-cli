#!/bin/bash

# Grouped integration test script - runs all test groups in parallel as background jobs
# Shows as single action in GitHub Actions UI
# Expected execution time: ~30 minutes (longest group)

set -e

# Color output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}Starting Grouped Integration Tests${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""

# Start the Squid proxy in a Docker container (shared across all groups)
if ! docker ps | grep -q squid; then
  echo -e "${YELLOW}Starting Squid proxy...${NC}"
  docker run \
    --name squid \
    -d \
    -p $PROXY_PORT:3128 \
    -v $(pwd)/internal/commands/.scripts/squid/squid.conf:/etc/squid/squid.conf \
    -v $(pwd)/internal/commands/.scripts/squid/passwords:/etc/squid/passwords \
    ubuntu/squid:5.2-22.04_beta
  echo -e "${GREEN}✓ Squid proxy started${NC}"
else
  echo -e "${GREEN}✓ Squid proxy already running${NC}"
fi

# Download and extract the ScaResolver tool (with caching)
if [ ! -f "/tmp/ScaResolver" ]; then
  echo -e "${YELLOW}Downloading ScaResolver...${NC}"
  wget -q https://sca-downloads.s3.amazonaws.com/cli/latest/ScaResolver-linux64.tar.gz
  tar -xzf ScaResolver-linux64.tar.gz -C /tmp
  rm -rf ScaResolver-linux64.tar.gz
  echo -e "${GREEN}✓ ScaResolver downloaded${NC}"
else
  echo -e "${GREEN}✓ Using cached ScaResolver${NC}"
fi

echo ""
echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}Running Test Groups in Parallel${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# Define test groups
TEST_GROUPS=("fast-validation" "scan-core" "scan-engines" "scm-integration" "realtime-features" "advanced-features")

# Create directories for each group's output
mkdir -p test-results
for group in "${TEST_GROUPS[@]}"; do
  mkdir -p "test-results/$group"
done

# Function to run a test group
run_test_group() {
  local group=$1
  local log_file="test-results/$group/output.log"
  local coverage_file="test-results/$group/cover.out"
  
  echo -e "${YELLOW}[${group}] Starting...${NC}" | tee -a "$log_file"
  
  # Source the test patterns from integration_up_parallel.sh logic
  case "$group" in
    "fast-validation")
      TEST_PATTERN="."
      SKIP_PATTERN="Scan|PR|UserCount|Realtime|Project|Result|Import|Bfl|Chat|Learn|Remediation|Triage|Container|Scs|ASCA|Asca"
      TIMEOUT="10m"
      ;;
    "scan-core")
      TEST_PATTERN="Scan"
      SKIP_PATTERN="Realtime|ASCA|Asca|Container.*Scan"
      TIMEOUT="30m"
      ;;
    "scan-engines")
      TEST_PATTERN="ASCA|Asca|Container|Scs|Engine|CodeBashing|RiskManagement|CreateQueryDescription|MaskSecrets"
      SKIP_PATTERN="Realtime"
      TIMEOUT="35m"
      ;;
    "scm-integration")
      TEST_PATTERN="PR|UserCount|RateLimit|Hooks|Predicate|PreReceive|PreCommit"
      SKIP_PATTERN="^$"
      TIMEOUT="25m"
      ;;
    "realtime-features")
      TEST_PATTERN="Realtime"
      SKIP_PATTERN="^$"
      TIMEOUT="20m"
      ;;
    "advanced-features")
      TEST_PATTERN="Project|Result|Import|Bfl|Chat|Learn|Telemetry|Remediation|Triage|GetProjectName"
      SKIP_PATTERN="^$"
      TIMEOUT="25m"
      ;;
  esac
  
  # Build test command
  TEST_CMD="go test -tags integration -v -timeout ${TIMEOUT} -parallel 4"
  TEST_CMD="${TEST_CMD} -coverpkg github.com/checkmarx/ast-cli/internal/commands,github.com/checkmarx/ast-cli/internal/services,github.com/checkmarx/ast-cli/internal/wrappers"
  TEST_CMD="${TEST_CMD} -coverprofile ${coverage_file}"
  TEST_CMD="${TEST_CMD} -run '${TEST_PATTERN}'"
  
  if [ ! -z "$SKIP_PATTERN" ] && [ "$SKIP_PATTERN" != "^$" ]; then
    TEST_CMD="${TEST_CMD} -skip '${SKIP_PATTERN}'"
  fi
  
  TEST_CMD="${TEST_CMD} github.com/checkmarx/ast-cli/test/integration"
  
  # Run tests
  echo -e "${BLUE}[${group}] Command: ${TEST_CMD}${NC}" >> "$log_file"
  eval "${TEST_CMD}" >> "$log_file" 2>&1
  local status=$?
  
  if [ $status -eq 0 ]; then
    echo -e "${GREEN}[${group}] ✓ PASSED${NC}" | tee -a "$log_file"
  else
    echo -e "${RED}[${group}] ✗ FAILED (exit code: $status)${NC}" | tee -a "$log_file"
  fi
  
  return $status
}

# Export function so it can be used by parallel
export -f run_test_group
export RED GREEN YELLOW BLUE NC

# Run all test groups in parallel using background jobs
declare -A PIDS
for group in "${TEST_GROUPS[@]}"; do
  run_test_group "$group" &
  PIDS[$group]=$!
  echo -e "${BLUE}Started ${group} (PID: ${PIDS[$group]})${NC}"
done

echo ""
echo -e "${YELLOW}Waiting for all test groups to complete...${NC}"
echo ""

# Wait for all background jobs and collect results
declare -A RESULTS
ALL_PASSED=true

for group in "${TEST_GROUPS[@]}"; do
  wait ${PIDS[$group]}
  RESULTS[$group]=$?
  
  if [ ${RESULTS[$group]} -ne 0 ]; then
    ALL_PASSED=false
  fi
done

echo ""
echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}Test Results Summary${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

for group in "${TEST_GROUPS[@]}"; do
  if [ ${RESULTS[$group]} -eq 0 ]; then
    echo -e "${GREEN}✓ ${group}: PASSED${NC}"
  else
    echo -e "${RED}✗ ${group}: FAILED${NC}"
  fi
done

echo ""

# Merge all coverage files
echo -e "${YELLOW}Merging coverage reports...${NC}"
COVERAGE_FILES=$(find test-results -name "cover.out" -type f)
if [ -z "$COVERAGE_FILES" ]; then
  echo -e "${RED}No coverage files found!${NC}"
  exit 1
fi

gocovmerge $COVERAGE_FILES > cover.out
echo -e "${GREEN}✓ Coverage reports merged${NC}"

# Generate HTML coverage report
go tool cover -html=cover.out -o coverage.html
echo -e "${GREEN}✓ HTML coverage report generated${NC}"

# Display coverage summary
echo ""
echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}Coverage Summary${NC}"
echo -e "${BLUE}========================================${NC}"
go tool cover -func cover.out | tail -10

echo ""
if [ "$ALL_PASSED" = true ]; then
  echo -e "${GREEN}========================================${NC}"
  echo -e "${GREEN}All test groups PASSED! ✓${NC}"
  echo -e "${GREEN}========================================${NC}"
  exit 0
else
  echo -e "${RED}========================================${NC}"
  echo -e "${RED}Some test groups FAILED! ✗${NC}"
  echo -e "${RED}========================================${NC}"
  echo ""
  echo -e "${YELLOW}Check individual group logs in test-results/ directory${NC}"
  exit 1
fi
