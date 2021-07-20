docker run --name squid -d -p 3128:3128 -v ./internal/commands/.scripts/squid.conf:/etc/squid/squid.conf datadog/squid

go test -tags integration -v github.com/checkmarxDev/ast-cli/test/integration
