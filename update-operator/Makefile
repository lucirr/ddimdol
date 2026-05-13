.PHONY: build test generate manifests tidy

build:
	go build -o bin/operator ./cmd/operator/...

test:
	go test ./... -v

tidy:
	go mod tidy

generate:
	controller-gen object:headerFile="hack/boilerplate.go.txt" paths="./..."

manifests:
	controller-gen crd paths="./..." output:crd:artifacts:config=config/crd
