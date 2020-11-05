lint:
	go fmt ./cmd
	go mod tidy
	go test ./cmd
	golangci-lint run --allow-parallel-runners -v --enable-all --fix
run:
	go build -o file-sync ./cmd
	./file-sync
runDocker:
	docker-compose up
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
	curl http://localhost:9336/api/endpoint
	curl http://localhost:9336/api/queue

	curl -k --key ssl/client01.key --cert ssl/client01.crt https://localhost:9335/api/endpoint
	curl -k https://localhost:9335/api/endpoint