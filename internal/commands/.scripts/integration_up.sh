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

gotestsum --rerun-fails=2 --format=short \
--packages github.com/checkmarx/ast-cli/test/integration -- \
-tags=integration -v -timeout 210m \
-coverpkg github.com/checkmarx/ast-cli/internal/commands,github.com/checkmarx/ast-cli/internal/services,github.com/checkmarx/ast-cli/internal/wrappers \
-coverprofile cover.out



status=$?
echo "status value after tests $status"
if [ $status -ne 0 ]; then
    echo "Integration tests failed"
    rm cover.out
fi

go tool cover -html=cover.out -o coverage.html
