# agentctl MCP Server

agentctl can run as an [MCP (Model Context Protocol)](https://spec.modelcontextprotocol.io/) server. This lets any MCP-compatible client — Claude Code, Codex CLI, or your own tooling — call agentctl gate tools directly, with full policy evaluation and tracing, without needing a separate HTTP server.

## How it works

`agentctl mcp` speaks JSON-RPC 2.0 over stdio. The MCP client starts it as a subprocess and calls tools over stdin/stdout. Each tool call is evaluated against your local `agentctl.policy.yaml`, a trace is appended to `~/.agentctl/traces.jsonl`, and the verdict is returned as the tool result.

- **Allow** → `isError: false`, JSON summary with `verdict`, `trace_id`, `risk_score`
- **Deny** → `isError: true`, human-readable reason and trace ID
- **Escalate** → `isError: true`, reason plus `agentctl approval approve <id>` hint

## Available tools

| Tool | Gates |
|------|-------|
| `agentctl_install_package` | pip, npm, yarn, cargo, go get |
| `agentctl_run_code` | shell and code execution |
| `agentctl_write_file` | file create / overwrite / append |
| `agentctl_access_secret` | credential and secret reads |
| `agentctl_call_external_api` | outbound HTTP requests |

Each tool accepts a `reason` argument (required) so the purpose of every action is captured in the trace.

## Claude Code setup

Add to `.claude/settings.json` in your repo (or `~/.claude/settings.json` for global):

```json
{
  "mcpServers": {
    "agentctl": {
      "command": "agentctl",
      "args": ["mcp"],
      "env": {
        "AGENTCTL_SESSION": "claude-code"
      }
    }
  }
}
```

Drop an `agentctl.policy.yaml` in your repo root (see the example at the root of this repo). agentctl loads it automatically.

To verify the server is visible to Claude Code, ask it to list its MCP tools — `agentctl_install_package` and friends should appear.

### Difference from the hook adapter

| | Hook (`agentctl hook claude-code`) | MCP (`agentctl mcp`) |
|---|---|---|
| Intercepts | All native tool calls automatically | Only explicit `agentctl_*` tool calls |
| Setup | `PreToolUse` in settings.json | `mcpServers` in settings.json |
| Best for | Transparent policy enforcement | Opt-in gating from within agent code |

Use the hook for automatic interception of Bash/Write/WebFetch. Use MCP when you want an agent to explicitly call `agentctl_run_code` before executing a command (useful when you control the agent's system prompt).

## Codex CLI setup

Add to `~/.codex/config.yaml`:

```yaml
mcp_servers:
  agentctl:
    command: agentctl
    args: [mcp]
    env:
      AGENTCTL_SESSION: codex
```

Codex will see the five `agentctl_*` tools and can call them when the system prompt instructs it to gate risky actions.

## Session and trace correlation

All tool calls within one `agentctl mcp` process share a session ID, so you can query them together:

```bash
agentctl trace search --session mcp-1234567890
agentctl replay mcp-1234567890 --policy stricter.policy.yaml
```

Set `AGENTCTL_SESSION` to a stable value (e.g. a git commit SHA or PR number) to make traces queryable across runs.

## Approvals

When a tool call returns escalate, the trace is held as a pending approval:

```bash
agentctl approval list --status pending
agentctl approval approve <approval_id> --by alice
```

The MCP client will see `isError: true` with the approval ID and a hint to run the approve command. Once approved (or denied), the caller can retry.
