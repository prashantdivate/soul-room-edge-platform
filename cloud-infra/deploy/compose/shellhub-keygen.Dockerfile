FROM alpine:3.21

RUN apk add --no-cache openssl
COPY deploy/compose/shellhub-keygen.sh /usr/local/bin/shellhub-keygen
RUN chmod 0755 /usr/local/bin/shellhub-keygen

ENTRYPOINT ["/usr/local/bin/shellhub-keygen"]
