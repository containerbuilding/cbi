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
  validation:
    openAPIV3Schema:
      properties:
        spec:
          properties:
            replicas:
              type: integer
              minimum: 1
              maximum: 10

---

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

apiVersion: apps/v1
kind: Deployment
metadata:
  name: cbid-deployment
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
        args: ["-v=4"]
        imagePullPolicy: Always
EOF
echo "generated ${out}"
