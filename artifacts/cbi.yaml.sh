#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail
set -o errtrace

if [ "$#" -ne 2 ]; then
    echo "Usage: $0 REGISTRY TAG"
    exit 1
fi
REGISTRY="$1"
TAG="$2"

out=$(dirname $0)/$(basename $0 | sed -e s/.yaml.sh/.generated.yaml/)
cat > ${out} << EOF
# Autogenarated by $0 at $(date)
## CRD
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  name: buildjobs.cbi.containerbuilding.github.io
spec:
  group: cbi.containerbuilding.github.io
  version: v1alpha1
  names:
    kind: BuildJob
    plural: buildjobs
  scope: Namespaced
---
## RBAC stuff
apiVersion: v1
kind: ServiceAccount
metadata:
  name: cbi
  namespace: default
---
apiVersion: rbac.authorization.k8s.io/v1beta1
kind: ClusterRoleBinding
metadata:
  name: cbi
subjects:
  - kind: ServiceAccount
    name: cbi
    namespace: default
roleRef:
  kind: ClusterRole
#FIXME
  name: cluster-admin
  apiGroup: rbac.authorization.k8s.io
---
## CBI plugin stuff (docker)
apiVersion: apps/v1
kind: Deployment
metadata:
  name: cbi-docker
  labels:
    app: cbi-docker
spec:
  replicas: 1
  selector:
    matchLabels:
      app: cbi-docker
  template:
    metadata:
      labels:
        app: cbi-docker
    spec:
      serviceAccountName: cbi
      containers:
      - name: cbi-docker
        image: ${REGISTRY}/cbi-docker:${TAG}
        args: ["-logtostderr", "-v=4"]
        imagePullPolicy: Always
        ports:
        - containerPort: 12111
---
apiVersion: v1
kind: Service
metadata:
  name: cbi-docker
  labels:
    app: cbi-docker
spec:
  ports:
  - port: 12111
    protocol: TCP
  selector:
    app: cbi-docker
---
## CBI plugin stuff (buildah)
apiVersion: apps/v1
kind: Deployment
metadata:
  name: cbi-buildah
  labels:
    app: cbi-buildah
spec:
  replicas: 1
  selector:
    matchLabels:
      app: cbi-buildah
  template:
    metadata:
      labels:
        app: cbi-buildah
    spec:
      serviceAccountName: cbi
      containers:
      - name: cbi-buildah
        image: ${REGISTRY}/cbi-buildah:${TAG}
        args: ["-logtostderr", "-v=4", "-buildah-image=${REGISTRY}/cbi-buildah-buildah:${TAG}"]
        imagePullPolicy: Always
        ports:
        - containerPort: 12111
---
apiVersion: v1
kind: Service
metadata:
  name: cbi-buildah
  labels:
    app: cbi-buildah
spec:
  ports:
  - port: 12111
    protocol: TCP
  selector:
    app: cbi-buildah
---
## CBID stuff
apiVersion: apps/v1
kind: Deployment
metadata:
  name: cbid
  labels:
    app: cbid
spec:
  replicas: 1
  selector:
    matchLabels:
      app: cbid
  template:
    metadata:
      labels:
        app: cbid
    spec:
      serviceAccountName: cbi
      containers:
      - name: cbid
        image: ${REGISTRY}/cbid:${TAG}
        args: ["-logtostderr", "-v=4", "-cbi-plugins=cbi-docker,cbi-buildah"]
        imagePullPolicy: Always
EOF
echo "generated ${out}"
