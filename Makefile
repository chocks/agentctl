GOLANGCI_LINT ?= golangci-lint
OPENAPI_GENERATOR ?= npx openapi-generator-cli
OPENAPI_SPEC ?= api/openapi.yaml

.PHONY: fmt lint test build codegen-js codegen-py

fmt:
	gofmt -w ./cmd ./pkg

lint:
	$(GOLANGCI_LINT) run

test:
	go test ./...

build:
	go build ./...

codegen-js:
	$(OPENAPI_GENERATOR) generate -i $(OPENAPI_SPEC) -g typescript-fetch -o sdk/js

codegen-py:
	$(OPENAPI_GENERATOR) generate -i $(OPENAPI_SPEC) -g python -o sdk/python
