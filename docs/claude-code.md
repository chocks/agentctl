# agentctl + Claude Code

`agentctl` integrates with Claude Code through a `PreToolUse` hook. The recommended setup is `agentctl attach claude-code`, which bootstraps `~/.agentctl/` and updates `~/.claude/settings.json` for you.

## Install and Attach

```bash
go install github.com/chocks/agentctl/cmd/agentctl@latest
agentctl attach claude-code
agentctl doctor
```

`attach` does two things:

1. Creates `~/.agentctl/` and writes a default `policy.yaml` if needed.
2. Adds `agentctl hook claude-code` to Claude Code's `PreToolUse` hooks.

## What Gets Intercepted

The Claude Code hook adapter maps these tools into `agentctl` actions:

- `Bash`
- `Write`
- `Edit`
- `MultiEdit`
- `WebFetch`

Before the tool call runs, Claude Code invokes `agentctl hook claude-code` with the hook event on stdin. `agentctl` evaluates policy, records a trace, and exits:

| Exit code | Meaning |
|---|---|
| `0` | Allow |
| `2` | Deny or escalate; Claude Code shows the reason |

If the global policy file exists but is malformed, the hook fails open and writes the error to stderr. This keeps a bad YAML edit from blocking every tool call.

## Global Policy

Claude Code uses the same global policy file as every other integration:

```text
~/.agentctl/policy.yaml
```

There is no repo-local override.

Recent traces and approvals live at:

```text
~/.agentctl/traces.jsonl
~/.agentctl/approvals.jsonl
```

## Manual Hook Setup

If you do not want to use `attach`, add this to `~/.claude/settings.json` yourself:

```json
{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Bash|Write|Edit|MultiEdit|WebFetch",
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

## Useful Commands

```bash
agentctl trace list --last 20
agentctl trace search --session <session_id>
agentctl approval list --status pending
agentctl approval approve <approval_id> --by alice
agentctl ui
```

## Difference From MCP

| | Hook (`agentctl hook claude-code`) | MCP (`agentctl mcp`) |
|---|---|---|
| Intercepts | Native Claude Code tool calls automatically | Only explicit `agentctl_*` tool calls |
| Setup | `attach claude-code` or manual hook entry | Manual `mcpServers` entry |
| Best for | Transparent enforcement | Opt-in gating from prompts or agent code |

See [docs/mcp.md](mcp.md) for MCP usage and Codex setup.
