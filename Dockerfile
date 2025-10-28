FROM alpine:latest

RUN apk add --no-cache cifs-utils ca-certificates

COPY bin/linux/amd64/chartmuseum

USER 1001:1001

ENTRYPOINT ["/chartmuseum"]
