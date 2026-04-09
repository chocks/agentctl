# agentctl

Trace and control dangerous agent actions.

`agentctl` is a focused v1 control layer for coding agents. It gates a small set of high-risk actions, records structured traces for every decision, and replays prior sessions against a different policy.

The current implementation is a small Go CLI with:

- `gate`: evaluate one risky action against policy
- `trace`: inspect prior decisions from the local trace store
- `replay`: re-run a recorded session with another policy file

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
