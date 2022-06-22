#!/usr/bin/env bash

cd cli
git pull
go mod vendor
sudo /state-tool/bin/state run preprocess
cd ..
sudo chown -R as-build:as-build cli
cd cli/cmd/state-offline-installer/docker
go build -o /offline-installer-data/offline-installer github.com/ActiveState/cli/cmd/state-offline-installer
