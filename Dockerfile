FROM alpine:latest

# use goreleaser to build binnary
COPY ./file-sync /app/file-sync

WORKDIR /app

# Change this files in production
COPY ./examples/ssl/ca.crt /app/ssl/ca.crt
COPY ./examples/ssl/server.crt /app/ssl/server.crt
COPY ./examples/ssl/server.key /app/ssl/server.key
COPY ./examples/ssl/client01.crt /app/ssl/client01.crt
COPY ./examples/ssl/client01.key /app/ssl/client01.key
COPY ./examples/tests/test.txt /tmp/test.txt

RUN addgroup -g 101 -S app \
&& adduser -u 101 -D -S -G app app \
&& chown -R 101 /app

USER 101

ENTRYPOINT ["/app/file-sync"]