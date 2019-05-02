FROM alpine:3.9
RUN apk add --no-cache cifs-utils ca-certificates \
    && adduser -D -u 1000 chartmuseum
COPY bin/linux/amd64/chartmuseum /chartmuseum
USER 1000
ENTRYPOINT ["/chartmuseum"]
