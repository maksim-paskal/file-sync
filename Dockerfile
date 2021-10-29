FROM alpine:latest

# use goreleaser to build binnary
COPY ./file-sync /app/file-sync

WORKDIR /app

RUN addgroup -g 101 -S app \
&& adduser -u 101 -D -S -G app app \
&& chown -R 101 /app

USER 101

ENTRYPOINT ["/app/file-sync"]