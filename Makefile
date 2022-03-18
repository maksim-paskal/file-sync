tag=dev
image=paskalmaksim/file-sync:$(tag)

test:
	./scripts/validate-license.sh
	rm -rf data-test
	go fmt ./cmd/main
	go mod tidy
	./scripts/test-pkg.sh
	go run github.com/golangci/golangci-lint/cmd/golangci-lint@latest run -v
	rm -rf data-test
coverage:
	go tool cover -html=coverage.out
testIntegration:
	CONFIG=config_test.yaml go test -v -count=1 -tags=integration -race ./pkg/queue
build:
	git tag -d `git tag -l "helm-chart-*"`
	go run github.com/goreleaser/goreleaser@latest build --rm-dist --snapshot --skip-validate
	mv ./dist/file-sync_linux_amd64/file-sync ./file-sync
	docker build --pull . -t $(image)
push:
	docker push $(image)
run:
	rm -rf data
	go run --race ./cmd/main \
	-log.level=DEBUG \
	-log.pretty \
	-redis.enabled \
	-sync.address=10.10.10.10,11.11.11.11,12.12.12.12 \
	-sync.timeout=1s \
	-sync.retry.count=0 \
	-dir.src=examples \
	-ssl.crt=ssl/CA.crt \
	-ssl.key=ssl/CA.key
clean:
	rm -rf file-sync
	docker-compose down --remove-orphans 
runDocker:
	make build
	docker-compose down --remove-orphans && docker-compose up
testPut:
	curl --data-binary '@examples/put.json' http://localhost:9335/api/endpoint
	cat data/test.txt
testDelete:
	curl --data-binary '@examples/delete.json' http://localhost:9335/api/endpoint
redisStart:
	docker run --name some-redis -p 6379:6379 -d redis
initSSL:
	rm -rf ./ssl/
	mkdir -p ./ssl/

	go run ./cmd/gencerts -cert.path=ssl
	kubectl create configmap ssl --dry-run=client -o yaml --from-file ssl/CA.crt --from-file ssl/CA.key
sslSSLCertificates:
	openssl rsa -in ./ssl/CA.key -check -noout
	openssl rsa -in ./ssl/test.key -check -noout
	openssl verify -CAfile ./ssl/CA.crt ./ssl/test.crt

	openssl x509 -in ./ssl/test.crt -text

	openssl x509 -pubkey -in ./ssl/CA.crt -noout | openssl md5
	openssl pkey -pubout -in ./ssl/CA.key | openssl md5

	openssl x509 -pubkey -in ./ssl/test.crt -noout | openssl md5
	openssl pkey -pubout -in ./ssl/test.key | openssl md5
testSSL:
	curl -k --key ssl/test.key --cert ssl/test.crt https://localhost:9335/api/healthz
upgrade:
	go get -v -u all
	go mod tidy
bulk:
	while true; do curl "http://localhost:9336/api/queue?force=true&value=put:test.txt" ; sleep 0.1; done
heap:
	go tool pprof -http=127.0.0.1:8080 http://localhost:9336/debug/pprof/heap
testChart:
	helm lint --strict ./charts/file-sync
	helm template ./charts/file-sync | kubectl apply --dry-run=client -f -
chart-index:
	rm -rf .cr-index
	mkdir .cr-index
	cr index \
	--owner maksim-paskal \
	--git-repo file-sync \
	--release-name-template "helm-chart-{{ .Version }}" \
	--charts-repo https://maksim-paskal.github.io/file-sync \
	--push \
	--token $(CR_TOKEN)
chart-upload:
	rm -rf .cr-release-packages
	cr package ./charts/file-sync
	cr upload \
	--owner maksim-paskal \
	--git-repo file-sync \
	--commit "`git rev-parse HEAD`" \
	--release-name-template "helm-chart-{{ .Version }}" \
	--token $(CR_TOKEN)
scan:
	@trivy image \
	-ignore-unfixed --no-progress --severity HIGH,CRITICAL \
	$(image)