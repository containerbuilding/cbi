FROM golang:1.10 AS compile
COPY . /go/src/github.com/containerbuilding/cbi
RUN go build -o /cbid -a -tags netgo -installsuffix netgo --ldflags '-extldflags "-static"' github.com/containerbuilding/cbi/cmd/cbid \
    && go build -o /cbi-dockercli -a -tags netgo -installsuffix netgo --ldflags '-extldflags "-static"' github.com/containerbuilding/cbi/cmd/cbi-dockercli

# TODO: use alpine
FROM debian:9 AS base

FROM base AS cbid
COPY --from=compile /cbid /cbid
ENTRYPOINT ["/cbid"]

FROM base AS cbi-dockercli
COPY --from=compile /cbi-dockercli /cbi-dockercli
ENTRYPOINT ["/cbi-dockercli"]

FROM alpine
RUN echo "nothing defined here, specify the target via `docker build --target`."; false
