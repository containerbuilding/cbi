FROM golang:1.10 AS compile
COPY . /go/src/github.com/containerbuilding/cbi
WORKDIR /go/src/github.com/containerbuilding/cbi
# We can't build them statically, as glog depends on os/user.
# See https://github.com/moby/moby/pull/29478 for the os/user issue.
RUN for f in $(find ./cmd -mindepth 1 -maxdepth 1 -type d -printf '%f\n'); \
  do echo $f; go build -o /$f github.com/containerbuilding/cbi/cmd/$f; done

FROM debian:9-slim AS base

### CBI Controller
FROM base AS cbid
COPY --from=compile /cbid /cbid
ENTRYPOINT ["/cbid"]

### CBI Common Plugin Helper
FROM base AS cbipluginhelper
RUN apt update && apt install -y -q git openssh-client
COPY --from=compile /cbipluginhelper /cbipluginhelper
ENTRYPOINT ["/cbipluginhelper", "--debug"]

### Docker
FROM base AS cbi-docker
COPY --from=compile /cbi-docker /cbi-docker
ENTRYPOINT ["/cbi-docker"]

FROM docker:17.09 AS cbi-docker-docker
ADD artifacts/dockerfile.d/docker-build-push.sh /

### Buildah
FROM base AS cbi-buildah
COPY --from=compile /cbi-buildah /cbi-buildah
ENTRYPOINT ["/cbi-buildah"]

FROM fedora:rawhide AS cbi-buildah-buildah
RUN dnf install -y buildah && dnf clean all
ADD artifacts/dockerfile.d/docker-build-push.sh /

### BuildKit
FROM base AS cbi-buildkit
COPY --from=compile /cbi-buildkit /cbi-buildkit
ENTRYPOINT ["/cbi-buildkit"]

### Kaniko
FROM base AS cbi-kaniko
COPY --from=compile /cbi-kaniko /cbi-kaniko
ENTRYPOINT ["/cbi-kaniko"]

### Dummy
FROM alpine
CMD echo 'nothing defined here, specify the target via `docker build --target`.'; false
