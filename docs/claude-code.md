# agentctl + Claude Code

agentctl integrates with Claude Code via its **PreToolUse hook** to automatically gate every Bash, Write, and WebFetch call against your local policy — no changes to prompts or agent code required.

## Installation

```bash
go install github.com/chocks/agentctl/cmd/agentctl@latest
```

Or download a pre-built binary from the releases page and place it on your `$PATH`.

Verify:

```bash
agentctl version
```

## Hook setup

Add the following to `.claude/settings.json` in your repo (or `~/.claude/settings.json` for global use):

```json
{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Bash|Write|WebFetch",
        "hooks": [
          {
            "type": "command",
            "command": "agentctl hook claude-code"
          }
        ]
      }
    ]
  }
}
```

Drop an `agentctl.policy.yaml` in your repo root. agentctl loads it automatically. See the example at the root of this repo.

## How it works

Before each Bash, Write, or WebFetch call, Claude Code runs `agentctl hook claude-code` and passes the tool event as JSON on stdin. agentctl evaluates the action against your policy and exits:

| Exit code | Meaning |
|-----------|---------|
| 0 | Allow — Claude Code proceeds |
| 2 | Block — Claude Code shows the reason to the user |

On infrastructure failure (agentctl not found, policy unreadable), the hook exits 0 so Claude Code is never blocked by a broken gate.

## Policy example

```yaml
# agentctl.policy.yaml
actions:
  run_code:
    blocked_patterns:
      - "rm -rf"
      - "curl.*| bash"
  install_package:
    require_approval: always
  call_external_api:
    allowed_domains:
      - "api.github.com"
      - "api.openai.com"
  write_file:
    blocked_patterns:
      - ".env"
      - "*.pem"
  access_secret:
    require_approval: always
```

## Checking traces after a session

```bash
# Recent decisions
agentctl trace list --last 20

# All decisions for a session
agentctl trace search --session <session_id>

# Pending approvals
agentctl approval list --status pending

# Approve an escalation
agentctl approval approve <approval_id> --by alice
```

## Flags

| Flag | Description |
|------|-------------|
| `--agent <name>` | Agent name annotated on each trace (default: `claude-code`) |
| `--model <name>` | Model name annotated on each trace |
| `--session <id>` | Override session id (default: taken from hook event) |

Example with flags:

```json
{
  "command": "agentctl hook claude-code --agent claude-code --model claude-opus-4-6"
}
```

## Environment variables

| Variable | Description |
|----------|-------------|
| `AGENTCTL_TRACE_FILE` | Override trace file path (default: `~/.agentctl/traces.jsonl`) |
| `AGENTCTL_APPROVAL_FILE` | Override approvals file path (default: `~/.agentctl/approvals.jsonl`) |
| `AGENTCTL_HOME` | Override the agentctl home directory |

## Difference from MCP

| | Hook (`agentctl hook claude-code`) | MCP (`agentctl mcp`) |
|---|---|---|
| Intercepts | All Bash/Write/WebFetch calls automatically | Only explicit `agentctl_*` tool calls |
| Setup | `PreToolUse` in settings.json | `mcpServers` in settings.json |
| Best for | Transparent policy enforcement on any Claude Code session | Opt-in gating from within custom agent code |

See [docs/mcp.md](mcp.md) for the MCP server setup.
