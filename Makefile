GOLANGCI_LINT ?= golangci-lint
OPENAPI_GENERATOR ?= npx openapi-generator-cli
OPENAPI_SPEC ?= api/openapi.yaml

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X main.version=$(VERSION)

.PHONY: fmt lint test build install install-tools ci codegen-js codegen-py

fmt:
	gofmt -w ./cmd ./pkg

lint:
	$(GOLANGCI_LINT) run

test:
	go test ./...

build:
	go build -ldflags "$(LDFLAGS)" -o bin/agentctl ./cmd/agentctl

ci: fmt lint test build

install-tools:
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.11.4

install:
	go install -ldflags "$(LDFLAGS)" ./cmd/agentctl

codegen-js:
	$(OPENAPI_GENERATOR) generate -i $(OPENAPI_SPEC) -g typescript-fetch -o sdk/js

codegen-py:
	$(OPENAPI_GENERATOR) generate -i $(OPENAPI_SPEC) -g python -o sdk/python
