.PHONY: build run test clean docker-up docker-down

build:
	go build -o bin/server cmd/server/main.go

run: build
	./bin/server

test:
	go test ./...

load-test:
	./scripts/run_tests.sh

docker-up:
	docker-compose up -d

docker-down:
	docker-compose down

clean:
	rm -rf bin/
	docker-compose down -v

dev: docker-up
	go run cmd/server/main.go

.DEFAULT_GOAL := run
