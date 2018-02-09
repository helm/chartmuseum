# Build Binary
FROM golang:1.8.3 as gobuild
WORKDIR /go/src/github.com/kubernetes-helm/chartmuseum
COPY . .
RUN make bootstrap
RUN make build_linux

# Build Image
FROM alpine:3.6
RUN apk add --no-cache ca-certificates
COPY --from=gobuild /go/src/github.com/kubernetes-helm/chartmuseum/bin/linux/amd64/chartmuseum /chartmuseum
ENTRYPOINT ["/chartmuseum"]