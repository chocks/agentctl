# Node Integration

This is the thinnest realistic integration path for coding-agent users on Node:

1. run the local `agentctl` API
2. wrap risky tools with the generated JS client
3. let policy decide whether the tool actually executes

Start the local API:

```bash
go run ./cmd/agentctl serve
```

Use the wrapper in [`guarded-tools.ts`](/Users/chockalingameswaramurthy/Documents/repos/agentctl/examples/node/guarded-tools.ts):

```ts
import { exec as rawExec } from "node:child_process";
import { promisify } from "node:util";
import { writeFile as rawWriteFile } from "node:fs/promises";
import { AgentctlNodeGuard } from "./guarded-tools";

const guard = new AgentctlNodeGuard();

const exec = guard.wrapExec(promisify(rawExec), {
  agent: "codex-runner",
  model: "gpt-5",
  sessionId: "run-123",
});

const writeFile = guard.wrapWriteFile(rawWriteFile, {
  agent: "codex-runner",
  model: "gpt-5",
  sessionId: "run-123",
});

await exec("npm install axios");
await writeFile("notes.txt", "hello");
```

If a decision comes back as `deny` or `escalate`, the wrapper throws before the underlying tool runs.

This is the intended product shape for Codex or Claude Code style users:

- the model calls normal tools
- your runtime wraps those tools
- `agentctl` gates and traces the dangerous operations automatically
