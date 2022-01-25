# This will be our builder image

FROM golang:alpine

ARG version=0.14.0

ARG revision=main

COPY . /go/src/github.com/helm/chartmuseum

WORKDIR /go/src/github.com/helm/chartmuseum

RUN CGO_ENABLED=0 GO111MODULE=on go build \
   -v --ldflags="-w -X main.Version=${version} -X main.Revision=${revision}" \
   -o /chartmuseum \
   cmd/chartmuseum/main.go


# This will be the final image

FROM alpine:latest

RUN apk add --no-cache cifs-utils ca-certificates

COPY --from=0 /chartmuseum /chartmuseum

USER 1000:1000

ENTRYPOINT ["/chartmuseum"]
