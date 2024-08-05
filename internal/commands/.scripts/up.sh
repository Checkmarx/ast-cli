#!/bin/bash

wget https://sca-downloads.s3.amazonaws.com/cli/latest/ScaResolver-linux64.tar.gz
tar -xzvf ScaResolver-linux64.tar.gz -C /tmp
rm -rf ScaResolver-linux64.tar.gz
# ignore mock and wrappers packages, as they checked by integration tests
go test $(go list ./... | grep -v "mock" | grep -v "wrappers" | grep -v "bitbucketserver" | grep -v "logger") -timeout 20m -coverprofile cover.out