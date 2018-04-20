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
      containers:
      - name: cbi-docker
        image: ${REGISTRY}/cbi-docker:${TAG}
        args: ["-logtostderr", "-v=4", "-docker-image=${REGISTRY}/cbi-docker-docker:${TAG}"]
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
## CBI plugin stuff (buildkit)
apiVersion: apps/v1
kind: Deployment
metadata:
  name: cbi-buildkit
  labels:
    app: cbi-buildkit
spec:
  replicas: 1
  selector:
    matchLabels:
      app: cbi-buildkit
  template:
    metadata:
      labels:
        app: cbi-buildkit
    spec:
      containers:
      - name: cbi-buildkit
        image: ${REGISTRY}/cbi-buildkit:${TAG}
        args: ["-logtostderr", "-v=4", "-buildctl-image=tonistiigi/buildkit", "-buildkitd-addr=tcp://cbi-buildkit-buildkitd:1234"]
        imagePullPolicy: Always
        ports:
        - containerPort: 12111
---
apiVersion: apps/v1
# TODO: workers should be StatefulSet
kind: Deployment
metadata:
  name: cbi-buildkit-buildkitd
  labels:
    app: cbi-buildkit-buildkitd
spec:
  replicas: 1
  selector:
    matchLabels:
      app: cbi-buildkit-buildkitd
  template:
    metadata:
      labels:
        app: cbi-buildkit-buildkitd
    spec:
      containers:
      - name: cbi-buildkit-buildkitd
        image: tonistiigi/buildkit
        args: ["--addr", "tcp://0.0.0.0:1234"]
        imagePullPolicy: Always
        ports:
        - containerPort: 1234
        securityContext:
          privileged: true
---
apiVersion: v1
kind: Service
metadata:
  name: cbi-buildkit
  labels:
    app: cbi-buildkit
spec:
  ports:
  - port: 12111
    protocol: TCP
  selector:
    app: cbi-buildkit
---
apiVersion: v1
kind: Service
metadata:
  name: cbi-buildkit-buildkitd
  labels:
    app: cbi-buildkit-buildkitd
spec:
  ports:
  - port: 1234
    protocol: TCP
  selector:
    app: cbi-buildkit-buildkitd
---
## CBI plugin stuff (kaniko)
apiVersion: apps/v1
kind: Deployment
metadata:
  name: cbi-kaniko
  labels:
    app: cbi-kaniko
spec:
  replicas: 1
  selector:
    matchLabels:
      app: cbi-kaniko
  template:
    metadata:
      labels:
        app: cbi-kaniko
    spec:
      containers:
      - name: cbi-kaniko
        image: ${REGISTRY}/cbi-kaniko:${TAG}
        args: ["-logtostderr", "-v=4", "-kaniko-image=gcr.io/kaniko-project/executor:latest"]
        imagePullPolicy: Always
        ports:
        - containerPort: 12111
---
apiVersion: v1
kind: Service
metadata:
  name: cbi-kaniko
  labels:
    app: cbi-kaniko
spec:
  ports:
  - port: 12111
    protocol: TCP
  selector:
    app: cbi-kaniko
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
        args: ["-logtostderr", "-v=4", "-cbi-plugins=cbi-docker,cbi-buildah,cbi-buildkit,cbi-kaniko"]
        imagePullPolicy: Always
EOF
echo "generated ${out}"
