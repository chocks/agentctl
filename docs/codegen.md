# Codegen

`agentctl` should be schema-first for client libraries.

The Go CLI is the current runtime, but JS and Python SDKs should be generated
from [`api/openapi.yaml`](/Users/chockalingameswaramurthy/Documents/repos/agentctl/api/openapi.yaml), not re-modeled by hand.

## Why OpenAPI

- good support for TypeScript and Python generators
- stable wire contract for a future local daemon or hosted service
- keeps `gate`, `trace`, and `replay` aligned across languages
- works even if the Go server implementation changes later

## Contract Rules

- Treat `api/openapi.yaml` as the cross-language source of truth.
- Keep `pkg/schema` aligned with the OpenAPI component schemas.
- Additive changes are preferred. Breaking changes should bump the API version.
- New action types must update OpenAPI, Go schema, policy evaluation, and tests together.

## Generation Targets

Recommended first generated clients:

- TypeScript fetch client
- Python client

## Example Commands

Using `openapi-generator-cli` directly:

```bash
openapi-generator-cli generate \
  -i api/openapi.yaml \
  -g typescript-fetch \
  -o sdk/js

openapi-generator-cli generate \
  -i api/openapi.yaml \
  -g python \
  -o sdk/python
```

Using the repo make targets:

```bash
make codegen-js
make codegen-py
```

## Practical Next Step

The current repo still exposes a CLI, not an HTTP server. That is acceptable for now.

The API spec should be treated as the target contract for:

- a future local daemon mode
- generated clients
- integration tests that exercise the same request/response shapes across languages
