GOLANGCI_LINT ?= golangci-lint

.PHONY: fmt lint test build

fmt:
	gofmt -w ./cmd ./pkg

lint:
	$(GOLANGCI_LINT) run

test:
	go test ./...

build:
	go build ./...
