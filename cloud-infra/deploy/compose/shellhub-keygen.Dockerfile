FROM alpine:3.24.2

RUN apk add --no-cache openssl
COPY deploy/compose/shellhub-keygen.sh /usr/local/bin/shellhub-keygen
RUN chmod 0755 /usr/local/bin/shellhub-keygen && \
    mkdir -p /keys && \
    chown 65532:65532 /keys

USER 65532:65532
ENTRYPOINT ["/usr/local/bin/shellhub-keygen"]
