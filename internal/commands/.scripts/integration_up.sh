docker run \
  --name squid \
  -d \
  -p $PROXY_PORT:3128 \
  -v $(pwd)/internal/commands/.scripts/squid/squid.conf:/etc/squid/squid.conf \
  -v $(pwd)/internal/commands/.scripts/squid/passwords:/etc/squid/passwords \
  datadog/squid

wget https://sca-downloads.s3.amazonaws.com/cli/latest/ScaResolver-linux64.tar.gz
tar -xzvf ScaResolver-linux64.tar.gz
chmod +x ScaResolver
rm -rf ScaResolver-linux64.tar.gz

go test \
  -tags integration \
  -v \
  -timeout 60m \
  -coverpkg github.com/checkmarx/ast-cli/internal/commands,github.com/checkmarx/ast-cli/internal/wrappers \
  -coverprofile cover.out \
  github.com/checkmarx/ast-cli/test/integration
