#!/usr/bin/env bash

# Output location
OUTPUT_LOCATION=build

# gvm use go1.17
# export VERBOSE=true
export VERBOSE=false
rm -rf $OUTPUT_LOCATION/offline-installer
rm -rf /tmp/artifacts-*
rm -rf /tmp/CustomInstallerTest
gofmt -s -w *.go
go build -o $OUTPUT_LOCATION/offline-installer github.com/ActiveState/cli/cmd/state-offline-installer
gozip -c $OUTPUT_LOCATION/offline-installer LICENSE.txt artifacts.tar.gz
# $OUTPUT_LOCATION/offline-installer install /tmp/CustomInstallerTestII
$OUTPUT_LOCATION/offline-installer install /tmp/CustomInstallerTest
$OUTPUT_LOCATION/offline-installer install /tmp/CustomInstallerTest
$OUTPUT_LOCATION/offline-installer uninstall /tmp/CustomInstallerTest
