.PHONY: run build test migrate lint

run:
	go run ./cmd/server/...

build:
	go build -o bin/portal-api ./cmd/server/...

test:
	go test ./... -v -cover

migrate:
	@echo "Run migrations manually or via goose"

lint:
	golangci-lint run ./...
