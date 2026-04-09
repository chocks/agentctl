# agentctl

Trace and control dangerous agent actions.

`agentctl` is a focused v1 control layer for coding agents. It gates a small set of high-risk actions, records structured traces for every decision, and replays prior sessions against a different policy.

The current implementation is a small Go CLI with:

- `gate`: evaluate one risky action against policy
- `trace`: inspect prior decisions from the local trace store
- `replay`: re-run a recorded session with another policy file

The repo is now also set up to be schema-first for future SDKs:

- [`api/openapi.yaml`](/Users/chockalingameswaramurthy/Documents/repos/agentctl/api/openapi.yaml) is the cross-language API contract
- `pkg/schema` remains the Go runtime model
- JS and Python clients should be generated from the OpenAPI spec

## Scope

This repo intentionally covers only the dangerous operations:

- `install_package`
- `run_code`
- `access_secret`
- `write_file`
- `call_external_api`

Everything else stays out of the control path for now.

## Quick Start

```bash
make fmt
go run ./cmd/agentctl gate <<'EOF'
{"action":"install_package","params":{"manager":"pip","package":"requests","version":"2.31.0","hash":"sha256:abc","pinned":true},"reason":"HTTP client"}
EOF
```

Example with a stable session id for replay:

```bash
go run ./cmd/agentctl gate --session demo-1 <<'EOF'
{"action":"call_external_api","params":{"url":"https://api.openai.com/v1/responses","method":"POST"},"reason":"call model provider"}
EOF
```

```bash
go run ./cmd/agentctl replay demo-1 --policy agentctl.policy.yaml
```

Traces are stored in `~/.agentctl/traces.jsonl` by default. Set `AGENTCTL_HOME` or `AGENTCTL_TRACE_FILE` to override that location.

## Policy File

`agentctl` loads `agentctl.policy.yaml` from the repo root by default.

```yaml
actions:
  install_package:
    require_hashes: true
    require_lockfile: true

  run_code:
    block_patterns:
      - "| bash"
      - "| sh"
    network: deny

  access_secret:
    require_approval: always
    max_ttl: 300
```

## Development

This repo follows basic Go hygiene by default:

- format with `gofmt`
- keep packages small and standard-library first
- test core policy and trace behavior
- lint with `golangci-lint`

Common commands:

```bash
make fmt
make lint
make test
make build
```

## SDK Codegen

To avoid hand-maintaining JS and Python models, use the OpenAPI contract in [`api/openapi.yaml`](/Users/chockalingameswaramurthy/Documents/repos/agentctl/api/openapi.yaml).

Planned generation flow:

```bash
npm install
make codegen-js
make codegen-py
```

The generator is managed as a local dev dependency in [`package.json`](/Users/chockalingameswaramurthy/Documents/repos/agentctl/package.json). The design note is in [`docs/codegen.md`](/Users/chockalingameswaramurthy/Documents/repos/agentctl/docs/codegen.md).

Generated clients live in:

- [`sdk/js`](/Users/chockalingameswaramurthy/Documents/repos/agentctl/sdk/js)
- [`sdk/python`](/Users/chockalingameswaramurthy/Documents/repos/agentctl/sdk/python)

For the easiest Node adoption path, see [`examples/node/README.md`](/Users/chockalingameswaramurthy/Documents/repos/agentctl/examples/node/README.md).

## Project Layout

```text
cmd/agentctl     CLI entrypoint
pkg/schema       canonical action and decision types
pkg/policy       YAML policy engine
pkg/gate         gate() primitive
pkg/trace        append-only JSONL trace store
```

## Status

This is a tightened bootstrap, not a full platform:

- local JSONL traces instead of SQLite/Postgres
- CLI replay only, not full simulation tooling
- YAML policy rules, not a full policy DSL
- no JS/Python SDKs yet

## License

MIT. See `LICENSE`.
