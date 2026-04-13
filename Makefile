GOLANGCI_LINT ?= golangci-lint

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X main.version=$(VERSION)

.PHONY: fmt lint test build install install-tools ci

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
