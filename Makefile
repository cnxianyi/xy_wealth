.PHONY: help deps fmt vet test go-test build server-build run web-deps web-dev web-lint web-test web-build infra-up infra-down

help:
	@echo "deps       download Go and Web dependencies"
	@echo "fmt        format Go source"
	@echo "vet        run go vet"
	@echo "test       run Go and Web tests"
	@echo "build      build the Web app and server"
	@echo "run        run the server"
	@echo "web-dev    run the Vite development server"
	@echo "web-lint   lint the Web app"
	@echo "web-test   test the Web app"
	@echo "web-build  build the Web app"
	@echo "infra-up   start PostgreSQL and Redis"
	@echo "infra-down stop PostgreSQL and Redis"

deps:
	go mod download
	go mod tidy
	$(MAKE) web-deps

fmt:
	gofmt -w $$(find . -name '*.go' -not -path './vendor/*' -not -path './web/node_modules/*')

vet:
	go vet ./...

go-test:
	go test -race ./...

test: go-test web-test

server-build:
	mkdir -p bin
	go build -trimpath -o bin/xy-wealth ./cmd/server

build: web-build server-build

run:
	go run ./cmd/server

web-deps:
	pnpm --dir web install --frozen-lockfile

web-dev:
	pnpm --dir web dev

web-lint:
	pnpm --dir web lint

web-test:
	pnpm --dir web test

web-build:
	pnpm --dir web build

infra-up:
	docker compose up -d postgres redis

infra-down:
	docker compose down
