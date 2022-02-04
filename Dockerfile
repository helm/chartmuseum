FROM alpine:latest

# TARGETARCH is predefined by Docker
# See https://docs.docker.com/engine/reference/builder/#automatic-platform-args-in-the-global-scope
ARG TARGETARCH

RUN apk add --no-cache cifs-utils ca-certificates

COPY ./_dist/linux-$TARGETARCH/chartmuseum /chartmuseum

USER 1000:1000

ENTRYPOINT ["/chartmuseum"]
