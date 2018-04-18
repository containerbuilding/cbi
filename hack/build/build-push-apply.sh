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

# Build images
for t in cbid cbi-docker cbi-docker-docker cbi-buildah cbi-buildah-buildah cbi-buildkit; do
  docker build -t ${REGISTRY}/${t}:${TAG} --target ${t} -f artifacts/Dockerfile .
  docker push ${REGISTRY}/${t}:${TAG}
done

# Generate and apply the manifest
./artifacts/cbi.yaml.sh ${REGISTRY} ${TAG}
kubectl apply -f./artifacts/cbi.generated.yaml 
