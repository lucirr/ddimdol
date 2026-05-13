.PHONY: build build-static test lint tidy

build:
	go build -o bin/edge-agent ./cmd/agent/...

build-static:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
		go build -ldflags="-s -w" -o bin/edge-agent-linux-amd64 ./cmd/agent/...

test:
	go test ./... -v

lint:
	golangci-lint run ./...

tidy:
	go mod tidy
