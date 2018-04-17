#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail
set -o errtrace

set -x

cd $(dirname $0)/../..

export PATH=~/.kubeadm-dind-cluster:$PATH
./hack/dind/up.sh  > dind.log 2>&1 || (cat dind.log; false)
DOCKER_HOST=localhost:62375 ./hack/test/e2e.sh cbi-registry:5000/cbi
# no need to call ./hack/dind/down.sh on travis
