.PHONY: help deps fmt vet test build run infra-up infra-down

help:
	@echo "deps       download and tidy Go dependencies"
	@echo "fmt        format Go source"
	@echo "vet        run go vet"
	@echo "test       run tests with race detection"
	@echo "build      build the server"
	@echo "run        run the server"
	@echo "infra-up   start PostgreSQL and Redis"
	@echo "infra-down stop PostgreSQL and Redis"

deps:
	go mod download
	go mod tidy

fmt:
	gofmt -w $$(find . -name '*.go' -not -path './vendor/*')

vet:
	go vet ./...

test:
	go test -race ./...

build:
	mkdir -p bin
	go build -trimpath -o bin/xy-wealth ./cmd/server

run:
	go run ./cmd/server

infra-up:
	docker compose up -d postgres redis

infra-down:
	docker compose down

