test:
	./scripts/validate-license.sh
	rm -rf data-test
	go fmt ./cmd
	go mod tidy
	./scripts/test-pkg.sh
	golangci-lint run -v
testIntegration:
	CONFIG=config_test.yaml go test -tags=integration -race ./pkg/queue
build-dev:
	goreleaser build --rm-dist --skip-validate
	mv ./dist/file-sync_linux_amd64/file-sync ./file-sync
	docker build --pull . -t paskalmaksim/file-sync:dev
run:
	rm -rf data
	go run --race ./cmd -log.level=DEBUG -log.pretty -redis.enabled -dir.src=data-src
clean:
	rm -rf file-sync
	docker-compose down --remove-orphans 
runDocker:
	goreleaser build --rm-dist --skip-validate
	mv ./dist/file-sync_linux_amd64/file-sync ./file-sync
	docker-compose down --remove-orphans && docker-compose up
testPut:
	curl --data-binary '@examples/put.json' http://localhost:9335/api/endpoint
	cat data/test.txt
testDelete:
	curl --data-binary '@examples/delete.json' http://localhost:9335/api/endpoint
redisStart:
	docker run --name some-redis -p 6379:6379 -d redis
initSSL:
	rm -rf ssl
	mkdir ssl
	mkdir ssl/db
	mkdir ssl/db/certs
	mkdir ssl/db/newcerts
	touch ssl/db/index.txt
	echo "01" > ssl/db/serial
	openssl req -x509 -nodes -days 3650 -newkey rsa:2048 -keyout ssl/server.key -out ssl/server.crt -subj "/C=GB/ST=London/L=London/O=GLOBAL/OU=DEVOPS/CN=*.global"
	openssl req -new -newkey rsa:2048 -nodes -keyout ssl/ca.key -x509 -days 3650 \
	-subj "/C=GB/ST=London/L=London/O=GLOBAL/OU=CA/CN=*.global/emailAddress=ca@cluster.global" \
	-out ssl/ca.crt
	openssl req -new -newkey rsa:2048 -nodes -keyout ssl/client01.key \
	-subj "/C=GB/ST=London/L=London/O=GLOBAL/OU=CLIENT/CN=*.global/emailAddress=client@cluster.global" \
	-out ssl/client01.csr
	openssl x509 -req -days 3650 -in ssl/client01.csr -CA ssl/ca.crt -CAkey ssl/ca.key -set_serial 01 -out ssl/client01.crt
	openssl verify -verbose -CAfile ssl/ca.crt ssl/client01.crt
testSSL:
	curl http://localhost:9336/api/queue

	curl -k --key ssl/client01.key --cert ssl/client01.crt https://localhost:9335/api/sync
	curl -k https://localhost:9335/api/sync
upgrade:
	go get -v -u all
	go mod tidy
bulk:
	while true; do curl "http://localhost:9336/api/queue?force=true&value=put:test.txt" ; sleep 0.1; done
heap:
	go tool pprof -http=127.0.0.1:8080 http://localhost:9336/debug/pprof/heap