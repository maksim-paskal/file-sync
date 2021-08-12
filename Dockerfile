FROM golang:1.16 as build

COPY ./cmd /usr/src/app/cmd
COPY ./pkg /usr/src/app/pkg
COPY go.* /usr/src/app/
COPY .git /usr/src/app/

ENV GOOS=linux
ENV GOARCH=amd64
ENV CGO_ENABLED=0
ENV GOFLAGS="-trimpath"

RUN cd /usr/src/app \
  && go mod download \
  && go mod verify \
  && go build -v -o file-sync -ldflags \
  "-X github.com/maksim-paskal/file-sync/pkg/config.gitVersion=$(git describe --tags `git rev-list --tags --max-count=1`)-$(date +%Y%m%d%H%M%S)-$(git log -n1 --pretty='%h')" \
  ./cmd \
  && /usr/src/app/file-sync -version

FROM alpine:latest

COPY --from=build /usr/src/app/file-sync /app/file-sync

WORKDIR /app

# Change this files in production
COPY examples/ssl/ca.crt /app/ssl/ca.crt
COPY examples/ssl/server.crt /app/ssl/server.crt
COPY examples/ssl/server.key /app/ssl/server.key
COPY examples/ssl/client01.crt /app/ssl/client01.crt
COPY examples/ssl/client01.key /app/ssl/client01.key
COPY examples/tests/test.txt /tmp/test.txt

RUN addgroup -g 101 -S app \
&& adduser -u 101 -D -S -G app app \
&& chown -R 101 /app

USER 101

ENTRYPOINT ["/app/file-sync"]