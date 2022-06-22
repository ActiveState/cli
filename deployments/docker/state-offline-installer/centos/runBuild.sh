#!/usr/bin/env bash

BUILD_DIRECTORY=$(realpath ../../../../build)

# docker build --no-cache -t offline-installer-builder-c7 \
docker build -t offline-installer-builder-c7 \
	--build-arg ACTIVESTATE_API_KEY=${ACTIVESTATE_API_KEY} \
	--build-arg ACTIVESTATE_API_HOST=${ACTIVESTATE_API_HOST} \
    .


# echo "Directory"
# echo $BUILD_DIRECTORY

docker run \
    --mount src="$BUILD_DIRECTORY",target=/offline-installer-data,type=bind \
    offline-installer-builder-c7:latest \
     /usr/local/bin/runCompile
