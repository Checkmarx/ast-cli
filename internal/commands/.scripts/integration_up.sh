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

INCLUDE_PACKAGES=(
  "github.com/checkmarx/ast-cli/internal/commands"
  "github.com/checkmarx/ast-cli/internal/services"
  "github.com/checkmarx/ast-cli/internal/wrappers"
)

# Define the packages to exclude
EXCLUDE_PACKAGES=(
  "github.com/checkmarx/ast-cli/internal/wrappers/microsastengine"
  "github.com/checkmarx/ast-cli/internal/commands/util/help.go"
  "github.com/checkmarx/ast-cli/internal/services/microsast.go"
  "github.com/checkmarx/ast-cli/internal/wrappers/bitbucketserver"
  "github.com/checkmarx/ast-cli/internal/wrappers/ntlm"
)

# Convert the list of exclude packages to a pattern
EXCLUDE_PATTERN=$(printf "|%s" "${EXCLUDE_PACKAGES[@]}")
EXCLUDE_PATTERN=${EXCLUDE_PATTERN:1}

COVERPKG=""
for pkg in "${INCLUDE_PACKAGES[@]}"; do
  for subpkg in $(go list ${pkg}/...); do
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
