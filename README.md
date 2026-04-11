# agentctl

Trace and control dangerous agent actions.

`agentctl` gates a small set of high-risk actions, records structured traces for every decision, and replays prior sessions against different policies. Three primitives: **gate**, **trace**, **replay**.

## Agent Install

Paste this prompt into your coding agent (Claude Code, Cursor, Copilot, etc.) and it will install and configure `agentctl` for your project:

<details>
<summary><b>Copy the agent prompt</b></summary>

```text
Install and configure agentctl in this project. Follow these steps exactly:

1. Check prerequisites:
   - Verify Go >= 1.23 is installed (`go version`). If not, tell me to install Go first and stop.
   - Verify git is available.

2. Install the agentctl binary:
   - Run: go install github.com/chocks/agentctl/cmd/agentctl@latest
   - Verify it works: agentctl version

3. Create a starter policy file `agentctl.policy.yaml` in the project root with these defaults:

   actions:
     install_package:
       require_hashes: true
       require_lockfile: true
     run_code:
       block_patterns:
         - "| bash"
         - "| sh"
         - "curl | python"
         - "wget | python"
       network: deny
     access_secret:
       require_approval: always
       max_ttl: 300
     write_file:
       require_approval: never
     call_external_api:
       allowed_domains:
         - "*.github.com"
         - "*.pypi.org"
         - "*.npmjs.org"

4. Verify the setup by running a test gate call:
   echo '{"action":"install_package","params":{"manager":"pip","package":"requests","version":"2.31.0","hash":"sha256:abc","pinned":true},"reason":"HTTP client"}' | agentctl gate

   This should return a JSON decision with verdict "allow".

5. Print a summary of what was installed and where traces will be stored (~/.agentctl/traces.jsonl).
```

</details>

## Governed Actions

| Action | What it covers |
|---|---|
| `install_package` | pip, npm, cargo, go installs |
| `run_code` | shell execution, script runs |
| `access_secret` | reading secrets, tokens, credentials |
| `write_file` | file creation, overwrites, appends |
| `call_external_api` | outbound HTTP to external services |

Everything else stays out of the control path.

## Quick Start

```bash
go run ./cmd/agentctl gate <<'EOF'
{"action":"install_package","params":{"manager":"pip","package":"requests","version":"2.31.0","hash":"sha256:abc","pinned":true},"reason":"HTTP client"}
EOF
```

With a session id for replay:

```bash
go run ./cmd/agentctl gate --session demo-1 <<'EOF'
{"action":"call_external_api","params":{"url":"https://api.openai.com/v1/responses","method":"POST"},"reason":"call model provider"}
EOF

go run ./cmd/agentctl replay demo-1 --policy agentctl.policy.yaml
```

Traces are stored in `~/.agentctl/traces.jsonl`. Override with `AGENTCTL_HOME` or `AGENTCTL_TRACE_FILE`.

## Local API

```bash
go run ./cmd/agentctl serve
```

Open `http://127.0.0.1:8080/ui` for local traces, approvals, and replay UI.

For non-loopback, bearer auth is required:

```bash
AGENTCTL_AUTH_TOKEN=dev-secret go run ./cmd/agentctl serve --addr 0.0.0.0:8080
```

Multi-user headers: `Authorization: Bearer <token>`, `X-Agentctl-Actor`, `X-Agentctl-Team`.

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

## SDK Clients

The OpenAPI contract in `api/openapi.yaml` is the source of truth for generated clients.

```bash
npm install
make codegen-js    # → sdk/js
make codegen-py    # → sdk/python
```

Higher-level wrappers: `packages/js`, `packages/python`. Node quickstart: `examples/node/README.md`.

## Development

```bash
make fmt      # gofmt
make lint     # golangci-lint
make test     # go test ./...
make build    # go build ./...
```

## Project Layout

```text
cmd/agentctl     CLI entrypoint
pkg/schema       canonical action and decision types
pkg/policy       YAML policy engine
pkg/gate         gate() primitive
pkg/trace        append-only JSONL trace store
api/openapi.yaml cross-language API contract
```

## License

MIT. See `LICENSE`.
