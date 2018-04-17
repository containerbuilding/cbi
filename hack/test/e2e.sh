#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail
set -o errtrace

set -x

# for grepping kubectl result
export LANG=C LC_ALL=C

if [ "$#" -ne 1 ]; then
    echo "Usage: $0 REGISTRY"
    exit 1
fi
REGISTRY="$1"

if echo ${REGISTRY} | grep '/$' > /dev/null; then
    echo "REGISTRY must not contain a trailing slash".
    exit 1
fi

cd $(dirname $0)/../..

tag="test-$(date +%s)"
./hack/build/build-push-apply.sh ${REGISTRY} ${tag}

# TODO: move to golang
for ex in ex0; do
    for plugin in docker buildkit buildah; do
        echo "========== Testing ${ex} using ${plugin} plugin =========="
        # create a BuildJob
        (cat artifacts/examples/${ex}.yaml; echo "  pluginSelector: plugin.name=${plugin}") | kubectl create -f -
        # wait for the underlying job
        pod=
        while [[ -z $pod ]]; do
            pod=$(kubectl get pods --selector=job-name=${ex}-job --show-all --output=jsonpath={.items..metadata.name})
            sleep 10
        done
        until kubectl logs ${pod} > /dev/null 2>&1; do sleep 10; done
        # show the log and wait for completion
        kubectl logs -f ${pod}
        # delete the BuildJob
        kubectl delete buildjob ${ex}
    done
done
