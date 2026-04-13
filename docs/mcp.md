# agentctl MCP Server

`agentctl mcp` exposes the five gate actions as MCP tools over stdio. This is the integration path for Codex and any other MCP client that can start a local subprocess.

There is no HTTP server. The MCP process talks over stdin/stdout only.

## Recommended Codex Setup

```bash
go install github.com/chocks/agentctl/cmd/agentctl@latest
agentctl attach codex
agentctl doctor
```

`attach codex` bootstraps `~/.agentctl/` and adds this entry to `~/.codex/config.toml`:

```toml
[mcp_servers.agentctl]
command = "agentctl"
args = ["mcp"]
```

Codex will then see these tools:

- `agentctl_install_package`
- `agentctl_run_code`
- `agentctl_write_file`
- `agentctl_access_secret`
- `agentctl_call_external_api`

Each tool requires a `reason` field so the trace records why the action was requested.

## How MCP Gating Works

For each MCP tool call:

1. `agentctl` evaluates the action against `~/.agentctl/policy.yaml`
2. Appends a decision to `~/.agentctl/traces.jsonl`
3. Returns a result to the client

Result shape:

- Allow: `isError: false`
- Deny: `isError: true` with human-readable reason
- Escalate: `isError: true` with reason and approval hint

## Manual Codex Config

If you do not want to use `attach`, add this to `~/.codex/config.toml`:

```toml
[mcp_servers.agentctl]
command = "agentctl"
args = ["mcp"]
```

## Claude Code MCP Usage

Claude Code can also use `agentctl mcp`, but the recommended Claude integration is the hook adapter because it intercepts native tool calls automatically.

Manual Claude MCP config:

```json
{
  "mcpServers": {
    "agentctl": {
      "command": "agentctl",
      "args": ["mcp"]
    }
  }
}
```

## Replay and Approvals

All tool calls handled by one MCP process share a generated session ID, so they can be queried and replayed together:

```bash
agentctl trace search --session <session_id>
agentctl replay <session_id>
```

Escalated actions are recorded in `~/.agentctl/approvals.jsonl`:

```bash
agentctl approval list --status pending
agentctl approval approve <approval_id> --by alice
agentctl approval deny <approval_id> --by alice
agentctl ui
```

The MCP caller can retry after the operator resolves the approval.
