#!/bin/bash

wget https://sca-downloads.s3.amazonaws.com/cli/latest/ScaResolver-linux64.tar.gz
tar -xzvf ScaResolver-linux64.tar.gz -C /tmp
rm -rf ScaResolver-linux64.tar.gz

go test $(go list ./... | grep -v "mock" | grep -v "wrappers") -coverprofile cover.out
