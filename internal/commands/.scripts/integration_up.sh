docker run \
  --name squid \
  -d \
  -p $PROXY_PORT:3128 \
  -v $(pwd)/internal/commands/.scripts/squid/squid.conf:/etc/squid/squid.conf \
  -v $(pwd)/internal/commands/.scripts/squid/passwords:/etc/squid/passwords \
  datadog/squid

go test \
  -tags integration \
  -v \
  -timeout 60m \
  -coverpkg github.com/checkmarx/ast-cli/internal/commands,github.com/checkmarx/ast-cli/internal/wrappers \
  -coverprofile cover.out \
  github.com/checkmarx/ast-cli/test/integration
