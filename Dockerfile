FROM golang:1.15 as build

COPY ./cmd /usr/src/app/cmd
COPY go.* /usr/src/app/
COPY .git /usr/src/app/

ENV GOOS=linux
ENV GOARCH=amd64
ENV CGO_ENABLED=0
ENV GOFLAGS="-trimpath"

RUN cd /usr/src/app \
  && go mod download \
  && go mod verify \
  && go build -v -o file-sync -ldflags "-X main.buildTime=$(date +"%Y%m%d%H%M%S") -X main.gitVersion=`git describe --exact-match --tags $(git log -n1 --pretty='%h')`" ./cmd

FROM alpine:latest

COPY --from=build /usr/src/app/file-sync /app/file-sync

WORKDIR /app

COPY ssl/ca.crt /app/ssl/ca.crt
COPY ssl/server.crt /app/ssl/server.crt
COPY ssl/server.key /app/ssl/server.key

RUN addgroup -g 101 -S app \
&& adduser -u 101 -D -S -G app app \
&& chown -R 101 /app

USER 101

CMD /app/file-sync