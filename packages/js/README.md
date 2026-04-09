# `@agentctl/client`

This package is the ergonomic JavaScript layer over the generated OpenAPI client.

Use it when you want:

- one `AgentctlClient` instead of separate API classes
- a small Node guard wrapper for risky tools
- stable access to the generated types from `sdk/js`

Example:

```ts
import { AgentctlClient, AgentctlNodeGuard } from "@agentctl/client";

const client = new AgentctlClient({
  baseUrl: "http://127.0.0.1:8080",
  token: process.env.AGENTCTL_AUTH_TOKEN,
  defaultContext: {
    actor: "alice",
    team: "platform",
  },
});

const guard = new AgentctlNodeGuard(client, {
  sessionId: "run-123",
  agent: "codex-runner",
  model: "gpt-5",
});
```
