docker run \
  --name squid \
  -d \
  -p 3128:3128 \
  -v $(pwd)/internal/commands/.scripts/squid.conf:/etc/squid/squid.conf \
  -v $(pwd)/internal/commands/.scripts/passwords:/etc/squid/passwords \
  datadog/squid

go test -tags integration -v github.com/checkmarxDev/ast-cli/test/integration
