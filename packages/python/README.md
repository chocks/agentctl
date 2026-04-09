# `agentctl-client`

This package is the ergonomic Python layer over the generated client in [`sdk/python`](/Users/chockalingameswaramurthy/Documents/repos/agentctl/sdk/python).

Use it when you want:

- one `AgentctlClient` wrapper around the generated APIs
- Python guard helpers for command execution, file writes, and fetch-like calls

Example:

```python
from agentctl_client import AgentctlClient, AgentctlPythonGuard

client = AgentctlClient(
    base_url="http://127.0.0.1:8080",
    token="dev-secret",
    default_context={
        "actor": "alice",
        "team": "platform",
    },
)

guard = AgentctlPythonGuard(client, default_context={
    "session_id": "run-123",
    "agent": "claude-code",
    "model": "claude-opus",
})
```
