# Build Binary
FROM golang:1.8.3 as gobuild
RUN apt-get update && \
    apt-get -y install ca-certificates
WORKDIR /go/src/github.com/kubernetes-helm/chartmuseum
COPY . .
RUN make bootstrap
RUN make build_linux

# Build Image
FROM scratch
COPY --from=gobuild /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=gobuild /go/src/github.com/kubernetes-helm/chartmuseum/bin/linux/amd64/chartmuseum /chartmuseum
ENTRYPOINT ["/chartmuseum"]