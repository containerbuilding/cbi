#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail
set -o errtrace

set -x

cd $(dirname $0)/../..

export PATH=~/.kubeadm-dind-cluster:$PATH
./hack/dind/up.sh  > dind.log 2>&1 || (cat dind.log; false)
DOCKER_HOST=localhost:62375 ./hack/build/build-push-apply.sh cbi-registry:5000/cbi travis
kubectl create -f artifacts/examples/ex0.yaml
# FIXME
sleep 10
kubectl get job $(kubectl get buildjob ex0 --output=jsonpath={.status.job})
sleep 60
kubectl logs -f $(kubectl get pods --selector=job-name=ex0-job --show-all --output=jsonpath={.items..metadata.name})
