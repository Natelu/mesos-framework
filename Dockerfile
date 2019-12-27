FROM alpine:3.7

COPY luzx-executor /opt/cke/bin/cke-executor

CMD ["/opt/cke/bin/cke-executor"]