#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail
set -o errtrace

if [ "$#" -ne 3 ]; then
    echo "Usage: $0 PUSH_TARGET GOOGLE_SECRET GOOGLE_PROJECT"
    exit 1
fi
PUSH_TARGET="$1"
GOOGLE_SECRET="$2"
GOOGLE_PROJECT="$3"

out=$(dirname $0)/$(basename $0 | sed -e s/.yaml.sh/.generated.yaml/)
cat > ${out} << EOF
# Autogenarated by $0 at $(date)
apiVersion: v1
kind: ConfigMap
metadata:
  name: ex-google-cloudbuild-push-configmap
data:
  cloudbuild.yaml: |-
    steps:
    - name: 'gcr.io/cloud-builders/docker'
      args: ['build', '--tag=${PUSH_TARGET}', '.']
    images: ['${PUSH_TARGET}']
  Dockerfile: |-
    FROM busybox
    ADD hello /
    RUN cat /hello
  hello: "hello cloudbuild"

---

apiVersion: cbi.containerbuilding.github.io/v1alpha1
kind: BuildJob
metadata:
  name: ex-google-cloudbuild-push
  annotations:
    cbi-gcb/secret: ${GOOGLE_SECRET}
    cbi-gcb/project: ${GOOGLE_PROJECT}
spec:
  registry:
    push: true
  language:
    kind: Cloudbuild
  context:
    kind: ConfigMap
    configMapRef:
      name: ex-google-cloudbuild-push-configmap
EOF
echo "generated ${out}"
