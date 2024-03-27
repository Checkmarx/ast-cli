#!/bin/bash

wget https://sca-downloads.s3.amazonaws.com/cli/latest/ScaResolver-linux64.tar.gz
tar -xzvf ScaResolver-linux64.tar.gz -C /tmp
rm -rf ScaResolver-linux64.tar.gz

GOEXPERIMENT=nocoverageredesign go test ./... -coverprofile cover.out
