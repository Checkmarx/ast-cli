docker run \
  --name squid \
  -d \
  -p $PROXY_PORT:3128 \
  -v $(pwd)/internal/commands/.scripts/squid.conf:/etc/squid/squid.conf \
  -v $(pwd)/internal/commands/.scripts/passwords:/etc/squid/passwords \
  datadog/squid

go test \
  -tags integration \
  -v \
  -timeout 30m \
  -coverpkg github.com/checkmarxDev/ast-cli/internal/commands,github.com/checkmarxDev/ast-cli/internal/wrappers \
  -coverprofile cover.out \
  github.com/checkmarxDev/ast-cli/test/integration
