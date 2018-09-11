FROM alpine:3.6
RUN apk add cifs-utils --no-cache ca-certificates \
&& adduser -D -u 1000 chartmuseum
COPY bin/linux/amd64/chartmuseum /chartmuseum
USER 1000
ENTRYPOINT ["/chartmuseum"]