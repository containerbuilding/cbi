# CBI: Container Builder Interface for Kubernetes

CBI provides a vendor-neutral interface for building (and pushing) container images on top of a Kubernetes cluster,
with support for several backends such as [Docker](https://www.docker.com), [img](https://github.com/genuinetools/img), [BuildKit](https://github.com/moby/buildkit), and [Buildah](https://github.com/projectatomic/buildah).

## Current status

### Specification

* CBI CRD: pre-alpha, see [`pkg/apis/cbi/v1alpha1/types.go`](pkg/apis/cbi/v1alpha1/types.go).
* CBI plugin API: pre-alpha, see [`pkg/plugin/api/plugin.proto`](pkg/plugin/api/plugin.proto).

### Implementation

* CBI controller daemon (`cbid`): pre-alpha, see [`cmd/cbid`](cmd/cbid).

* Plugins (all of them are pre-alpha or even hasn't been started to work on):

Plugin | Support Dockerfile | Support `cloudbuild.yaml` | Support LLB
--- | --- | --- | ---
[Docker](https://www.docker.com) | Yes ✅| |
[BuildKit](https://github.com/moby/buildkit) | Yes ✅| Planned? (TBD) | Planned
[Buildah](https://github.com/projectatomic/buildah) | Yes ✅ | |
[kaniko](https://github.com/GoogleCloudPlatform/kaniko) | Yes ✅ | |

* Planned: [img](https://github.com/genuinetools/img), [Google Cloud Container Builder](https://cloud.google.com/container-builder/), [OpenShift Image Builder](https://github.com/openshift/imagebuilder), [Orca](https://github.com/cyphar/orca-build), ...

<!-- TODO: figure out possibility for supporting Bazel, OpenShift S2I, Singularity... -->


* Context providers (all plugins above implements these basic context providers via [`cmd/cbipluginhelper`](cmd/cbipluginhelper).)
    * ConfigMap
    * Git, with support for SSH secret
    * Planned: BuildKitSession, S3, GCS, Swift, "Flex", ...

Please feel free to open PRs to add other plugins.

## Quick start

Requires Kubernetes 1.9 or later.

### Installation

```
$ ./hack/build/build-push-apply.sh your-registry.example.com:5000/cbi test20180501
```

This command performs:

* Build and push CBI images as `your-registry.example.com:5000/cbi/{cbid,cbi-docker,...}:test20180501`
* Generate `artifacts/cbi.generated.yaml` so that the manifest uses the images on `your-registry.example.com:5000/cbi/{cbid,cbi-docker,...}:test20180501`.
    * `CustomResourceDefinition`: `BuildJob`
    * `ServiceAccount`: `cbi`
    * `ClusterRoleBinding`: `cbi`
    * `Deployment`: `cbid`, `cbi-docker`, ...
* Execute `kubectl apply -f artifacts/cbi.generated.yaml`.

By default, the following plugins will be installed:

Plugin   | Note
---      | ---
Docker (highest priority)   | Docker needs to be installed on the hosts
Buildah  | Privileged containers needs to be enabled
BuildKit | Privileged containers needs to be enabled
kaniko   | N/A

You may execute `kubectl edit deployment cbid` to remove unneeded plugins or change the priorities.

### Run your first `buildjob`

Create a buildjob `ex-git-nopush` from [`artifacts/examples/ex-git-nopush.yaml`](artifacts/examples/ex-git-nopush.yaml):
```console
$ kubectl create -f artifacts/examples/ex-git-nopush.yaml
buildjob "ex-git-nopush" created
```

Make sure the buildjob is created:
```console
$ kubectl get buildjobs
NAME      AGE
ex-git-nopush       3s
```

Inspect the underlying job and the result:
```console
$ kubectl get job $(kubectl get buildjob ex-git-nopush --output=jsonpath={.status.job})
NAME      DESIRED   SUCCESSFUL   AGE
ex-git-nopush-job   1         1            30s
$ kubectl logs $(kubectl get pods --selector=job-name=ex-git-nopush-job --show-all --output=jsonpath={.items..metadata.name})
Sending build context to Docker daemon 79.87 kB
Step 1 : FROM alpine:latest
...
Successfully built bef4a548fb02
```

Delete the buildjob (and the underlying job)
```console
$ kubectl delete buildjobs ex-git-nopush
buildjob "ex-git-nopush" deleted
```

### Advanced usage

#### Specifying plugin:

Specify the `pluginSelector` constraint as follows:

```yaml
apiVersion: cbi.containerbuilding.github.io/v1alpha1
kind: BuildJob
metadata:
  name: ex-git-nopush
  ...
spec:
  pluginSelector: plugin.name=buildah
  ...
```

#### Secrets

e.g.

```yaml
apiVersion: cbi.containerbuilding.github.io/v1alpha1
kind: BuildJob
metadata:
  name: ex
spec:
  registry:
    target: example.com/foo/bar:baz
    push: true
# `kubectl create secret docker-registry ...`
    secretRef:
      name: docker-registry-secret-name
  language:
    kind: Dockerfile
  context:
    kind: git
    git:
      url: ssh://me@git.example.com/foo/bar.git
# `kubectl create secret generic ssh-secret-name --from-file=id_rsa=$HOME/.ssh/id_rsa --from-file=config=$HOME/.ssh/config --from-file=known_hosts=$HOME/.ssh/known_hosts`
      sshSecretRef:
        name: ssh-secret-name

```

#### Large context  (*UNIMPLEMENTED YET*):

To send a large build context using [BuildKit session gRPC](https://github.com/moby/buildkit/blob/9f6d9a9e78f18b2ffc6bc4f211092722685cc853/session/filesync/filesync.proto), with support for diffcopy

```console
$ go get github.com/containerbuilding/cbi/cmd/cbictl
$ cbictl build -t your-registry.example.com/foo/bar:baz --push .
```

## Design (subject to change)

### Components

CBI is composed of the following specifications and implementations.

Specifications:

* CBI CRD: Kubernetes custom resource definition for `buildjob` objects.
* CBI plugin API: gRPC API used for connecting `cbid` to plugins.

Implementations:

* CBI controller daemon (`cbid`): a controller that watches creation of CBI CRD objects and creates [Kubernetes Job](https://kubernetes.io/docs/concepts/workloads/controllers/jobs-run-to-completion/#what-is-a-job) objects correspondingly.
* CBI CLI (`cbictl`): a reference CLI implementation for `cbid`
* CBI plugins: the actual pods that build and push images.
* CBI session manager (`cbism`): pods that speak [BuildKit session gRPC](https://github.com/moby/buildkit/blob/9f6d9a9e78f18b2ffc6bc4f211092722685cc853/session/filesync/filesync.proto) (or other similar protocols) for supporting large build context and diffcopy.

The concept of CBI session manager (`cbism`) is decoupled from `cbid`, so as to make `cbid` free from I/O overhead.

![cbi.png](./docs/cbi.png)

### Build context

CBI supports the following values for `context.kind`:

* `ConfigMap`: Kubernetes config map. Only suitable for small contexts.
* `Git`: git repository, with support for Kubernetes secrets 
* `BuildkitSession`: BuildKit session gRPC

However, some backend driver may not implement all of them.

#### Session

If `BuildkitSession` is specified as `context.kind`, the pod ID of a CBI session manager, TCP port number, and the session ID would be set to the status fields of the `BuildJob` object.

The client is expected to send the context to the specified session manager pod using BuildKit session gRPC (via [the HTTP/1.1 gate](https://github.com/moby/buildkit/blob/b7424f41fdf60b178c5227abdd54cb615161123d/session/manager.go#L46)).
To connect to the pod, the client may use `kubectl port-forward` or `kubectl exec ... socat`.

Future version would also provide [Ingress](https://kubernetes.io/docs/concepts/services-networking/ingress/) for exposing the CBI session manager in more efficient ways.

## Contribute to CBI

* Vendoring is managed via [dep](https://github.com/golang/dep).
* To update CRD definition, please edit [`pkg/apis/cbi/v1alpha1/types.go`](pkg/apis/cbi/v1alpha1/types.go) and run `hack/codegen/update-codegen.sh`. Please do not edit autogenerated files manually.

### Local testing with DinD

You may use `hack/dind/up.sh` for setting up a local Kubernetes cluster and a local registry using Docker-in-Docker.

```console
$ ./hack/dind/up.sh
$ DOCKER_HOST=localhost:62375 ./hack/build/build-push-apply.sh cbi-registry:5000/cbi test20180501
$ ./hack/dind/down.sh
```
The Kubernetes cluster and the "bootstrap" Docker listening at `localhost:62375` can connect to `cbi-registry:5000` without auth.


## FAQs

### Q: Does CBI standardize the Dockerfile specification?

A: No, the Dockerfile specification has been maintained by Docker, Inc.

CBI itself is neutral to any image building instruction language (e.g. Dockerfile).

However, most backend implementations would accept Dockerfile.

### Q: Does CBI replace BuildKit?

A: No, CBI just provides an abstract interface for several backends such as BuildKit.

### Q: Is CBI a part of Kubernetes, a Kubernetes incubator, or a CNCF project?

A: Currently no, unlike CRI/CNI/CSI.

But it'd be good to donate CBI to such a vendor-neutral organization if CBI becomes popular.
