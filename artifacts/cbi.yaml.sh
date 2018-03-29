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
  name: cbid-serviceaccount
  namespace: default
---
apiVersion: rbac.authorization.k8s.io/v1beta1
kind: ClusterRoleBinding
metadata:
  name: cbid-rbac
subjects:
  - kind: ServiceAccount
    name: cbid-serviceaccount
    namespace: default
roleRef:
  kind: ClusterRole
#FIXME
  name: cluster-admin
  apiGroup: rbac.authorization.k8s.io
---
## CBI plugin stuff (dockercli)
apiVersion: apps/v1
kind: Deployment
metadata:
  name: cbi-dockercli
  labels:
    app: cbi-dockercli
spec:
  replicas: 1
  selector:
    matchLabels:
      app: cbi-dockercli
  template:
    metadata:
      labels:
        app: cbi-dockercli
    spec:
      serviceAccountName: cbid-serviceaccount
      containers:
      - name: cbi-dockercli
        image: ${REGISTRY}/cbi-dockercli:${TAG}
        args: ["-v=4"]
        imagePullPolicy: Always
        ports:
        - containerPort: 12111
---
apiVersion: v1
kind: Service
metadata:
  name: cbi-dockercli
  labels:
    app: cbi-dockercli
spec:
  ports:
  - port: 12111
    protocol: TCP
  selector:
    app: cbi-dockercli
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
      serviceAccountName: cbid-serviceaccount
      containers:
      - name: cbid
        image: ${REGISTRY}/cbid:${TAG}
        args: ["-v=4", "-cbi-plugins=cbi-dockercli"]
        imagePullPolicy: Always
EOF
echo "generated ${out}"
