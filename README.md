# agentctl

`agentctl` is a local control plane for coding agents. It gates a small set of risky actions, records a trace for every decision, and can replay prior sessions against a different policy.

## Install

```bash
go install github.com/chocks/agentctl/cmd/agentctl@latest
agentctl version
```

`agentctl` stores all of its state under `~/.agentctl/`:

- `policy.yaml`
- `traces.jsonl`
- `approvals.jsonl`

There is no repo-local policy file and no HTTP server.

## Quick Start

Attach to a supported agent. `attach` bootstraps `~/.agentctl/` and writes a default policy if one does not exist yet.

```bash
agentctl attach claude-code
# or
agentctl attach codex
```

Verify the install:

```bash
agentctl doctor
```

Launch the terminal UI:

```bash
agentctl ui
```

## Governed Actions

| Action | What it covers |
|---|---|
| `install_package` | pip, npm, cargo, go installs |
| `run_code` | shell execution, script runs |
| `access_secret` | reading secrets, tokens, credentials |
| `write_file` | file creation, overwrites, appends |
| `call_external_api` | outbound HTTP to external services |

Everything else stays out of the control path.

## CLI

```text
agentctl attach <agent>        Configure agent integration and bootstrap ~/.agentctl/
agentctl detach <agent>        Remove agent integration
agentctl doctor                Check policy, trace store, approvals, and agent status
agentctl gate                  Evaluate one action from stdin
agentctl trace list            Show recent traces
agentctl trace search          Search traces
agentctl replay <session_id>   Re-evaluate a recorded session
agentctl approval list         List approvals
agentctl approval approve <id> Approve a pending escalation
agentctl approval deny <id>    Deny a pending escalation
agentctl ui                    Terminal UI for traces and approvals
agentctl hook claude-code      Claude Code PreToolUse hook adapter
agentctl mcp                   MCP server for Codex and other MCP clients
```

## Policy

`agentctl` loads exactly one policy file: `~/.agentctl/policy.yaml`.

- Missing file: built-in safe defaults are used.
- Malformed file: `gate`, `doctor`, and `mcp` fail loudly.
- Hook mode (`agentctl hook claude-code`) fails open on malformed policy and writes the error to stderr.

Default policy written by `attach`:

```yaml
actions:
  install_package:
    require_hashes: true

  run_code:
    block_patterns:
      - "| bash"
      - "| sh"
      - "| python"
    network: deny

  access_secret:
    require_approval: always
    max_ttl: 300

  write_file:
    block_patterns:
      - ".env"
      - "*.pem"
      - "*.key"

  call_external_api:
    allowed_domains: []
```

`allowed_domains: []` means deny all outbound calls. Omitting `allowed_domains` means no domain restriction.

## Replay

Record a session under a stable session ID:

```bash
echo '{"action":"call_external_api","params":{"url":"https://api.openai.com/v1/responses","method":"POST"},"reason":"call provider"}' \
  | agentctl gate --session demo-1
```

Replay that session against the current global policy:

```bash
agentctl replay demo-1
```

Or replay against an alternate policy file:

```bash
agentctl replay demo-1 --policy ./stricter-policy.yaml
```

## Docs

- [Claude Code setup](docs/claude-code.md)
- [MCP / Codex setup](docs/mcp.md)

## Development

```bash
make fmt
make build
make test
make lint
```

## License

MIT. See `LICENSE`.
