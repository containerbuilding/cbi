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
function e2e(){
    local ex=$1
    local plugin=$2
    echo "travis_fold:start:${ex}-${plugin}"
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
    echo "travis_fold:end:${ex}-${plugin}"
}

# ex0: git context
e2e ex0 docker
e2e ex0 buildkit
e2e ex0 buildah

# ex1: configmap context
for f in docker buildkit buildah; do
    e2e ex1 $f
    kubectl delete configmap ex1-configmap
done
