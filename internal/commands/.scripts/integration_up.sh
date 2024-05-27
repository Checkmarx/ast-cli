docker run \
  --name squid \
  -d \
  -p $PROXY_PORT:3128 \
  -v $(pwd)/internal/commands/.scripts/squid/squid.conf:/etc/squid/squid.conf \
  -v $(pwd)/internal/commands/.scripts/squid/passwords:/etc/squid/passwords \
  ubuntu/squid:5.2-22.04_beta

wget https://sca-downloads.s3.amazonaws.com/cli/latest/ScaResolver-linux64.tar.gz
tar -xzvf ScaResolver-linux64.tar.gz -C /tmp
rm -rf ScaResolver-linux64.tar.gz

# Define the packages to include
INCLUDE_PACKAGES=(
  "github.com/checkmarx/ast-cli/internal/commands"
  "github.com/checkmarx/ast-cli/internal/services"
  "github.com/checkmarx/ast-cli/internal/wrappers"
)

# Define the files and folders to exclude
EXCLUDE_PATHS=(
  "github.com/checkmarx/ast-cli/internal/wrappers/microsastengine"
  "github.com/checkmarx/ast-cli/internal/commands/util"
  "github.com/checkmarx/ast-cli/internal/services/microsast"
  "github.com/checkmarx/ast-cli/internal/wrappers/bitbucketserver"
  "github.com/checkmarx/ast-cli/internal/wrappers/ntlm"
)

# Convert the list of exclude paths to a pattern
EXCLUDE_PATTERN=$(IFS="|"; echo "${EXCLUDE_PATHS[*]}")
EXCLUDE_PATTERN=$(echo "${EXCLUDE_PATTERN}" | sed 's/\//\\\//g') # Escape slashes for regex

COVERPKG=""
for pkg in "${INCLUDE_PACKAGES[@]}"; do
  for subpkg in $(go list ${pkg}/...); do
    # Check if the subpackage matches the exclude pattern
    if ! [[ $subpkg =~ $EXCLUDE_PATTERN ]]; then
      COVERPKG="${COVERPKG},${subpkg}"
    fi
  done
done

# Remove leading comma
COVERPKG=${COVERPKG:1}

go test \
  -tags integration \
  -v \
  -timeout 210m \
  -coverpkg "$COVERPKG" \
  -coverprofile cover.out \
  github.com/checkmarx/ast-cli/test/integration

status=$?
echo "status value after tests $status"
if [ $status -ne 0 ]; then
    echo "Integration tests failed"
    rm cover.out
fi

go tool cover -html=cover.out -o coverage.html
