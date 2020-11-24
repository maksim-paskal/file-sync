set -euo pipefail

export CGO_ENABLED=0
export GO111MODULE=on
export TAGS=""
export GOFLAGS="-trimpath"
export LDFLAGS="-X main.buildTime=`date +\"%Y%m%d%H%M%S\"` -X main.buildGitTag=`git describe --exact-match --tags $(git log -n1 --pretty='%h')`"
export TARGETS="darwin/amd64 linux/amd64"
export BINNAME="file-sync"
export GOX="go run github.com/mitchellh/gox"

rm -rf _dist

$GOX -parallel=3 -output="_dist/$BINNAME-{{.OS}}-{{.Arch}}" -osarch="$TARGETS" -tags "$TAGS" -ldflags "$LDFLAGS" ./cmd/