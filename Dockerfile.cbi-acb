FROM golang:1.10-alpine AS compile
COPY . /go/src/github.com/containerbuilding/cbi
RUN go build -ldflags="-s -w" -o /cbi-acb github.com/containerbuilding/cbi/cmd/cbi-acb

FROM alpine:3.7
COPY --from=compile /cbi-acb /cbi-acb
ENTRYPOINT ["/cbi-acb"]
