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