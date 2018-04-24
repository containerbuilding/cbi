FROM golang:1.10-alpine AS compile
COPY . /go/src/github.com/containerbuilding/cbi
WORKDIR /go/src/github.com/containerbuilding/cbi
RUN for f in cmd/* ; \
  do echo $f; go build -o /$(basename $f) github.com/containerbuilding/cbi/$f; done

FROM alpine:3.7 AS base

### CBI Controller
FROM base AS cbid
COPY --from=compile /cbid /cbid
ENTRYPOINT ["/cbid"]

### CBI Common Plugin Helper
FROM base AS cbipluginhelper
RUN apk add --no-cache git openssh-client
COPY --from=compile /cbipluginhelper /cbipluginhelper
ENTRYPOINT ["/cbipluginhelper", "--debug"]

### Docker
FROM base AS cbi-docker
COPY --from=compile /cbi-docker /cbi-docker
ENTRYPOINT ["/cbi-docker"]

FROM docker:18.03 AS cbi-docker-docker
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
FROM base
CMD echo 'nothing defined here, specify the target via `docker build --target`.'; false
