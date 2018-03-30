FROM golang:1.10 AS compile
COPY . /go/src/github.com/containerbuilding/cbi
WORKDIR /go/src/github.com/containerbuilding/cbi
# We can't build them statically, as glog depends on os/user.
# See https://github.com/moby/moby/pull/29478 for the os/user issue.
RUN for f in $(find ./cmd -mindepth 1 -maxdepth 1 -type d -printf '%f\n'); \
  do echo $f; go build -o /$f github.com/containerbuilding/cbi/cmd/$f; done

FROM debian:9-slim AS base

FROM base AS cbid
COPY --from=compile /cbid /cbid
ENTRYPOINT ["/cbid"]

FROM base AS cbi-docker
COPY --from=compile /cbi-docker /cbi-docker
ENTRYPOINT ["/cbi-docker"]

FROM alpine
CMD echo 'nothing defined here, specify the target via `docker build --target`.'; false
