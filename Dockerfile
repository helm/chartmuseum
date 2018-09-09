FROM alpine:3.6
RUN apk add --no-cache ca-certificates \
&& adduser -D chartmuseum
COPY bin/linux/amd64/chartmuseum /chartmuseum
USER chartmuseum
ENTRYPOINT ["/chartmuseum"]
