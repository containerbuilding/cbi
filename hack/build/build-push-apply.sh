#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail
set -o errtrace

set -x

if [ "$#" -ne 2 ]; then
    echo "Usage: $0 REGISTRY TAG"
    exit 1
fi
REGISTRY="$1"
TAG="$2"

if echo ${REGISTRY} | grep '/$' > /dev/null; then
    echo "REGISTRY must not contain a trailing slash".
    exit 1
fi

cd $(dirname $0)/../..

DOCKER_BUILD_FLAGS=

# TODO: compare version as well
if [[ `docker info --format '{{json .ExperimentalBuild}}'` = true ]]; then
    export BUILD_STREAM_PROTOCOL=diffcopy
    DOCKER_BUILD_FLAGS="--stream"
fi

# Build images
for t in $((cd artifacts; ls Dockerfile.*) | sed -e s/Dockerfile\.//g); do
    docker build -t ${REGISTRY}/${t}:${TAG} -f artifacts/Dockerfile.${t} ${DOCKER_BUILD_FLAGS} .
    docker push ${REGISTRY}/${t}:${TAG}
done

# Generate and apply the manifest
./artifacts/cbi.yaml.sh ${REGISTRY} ${TAG}
kubectl apply -f./artifacts/cbi.generated.yaml 
