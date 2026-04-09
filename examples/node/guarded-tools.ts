// examples/node/guarded-tools.ts
//
// Demonstrates wrapping Node.js built-ins with agentctl guards.
// Uses the @agentctl/client package — see ../../packages/js/src/
//
// Usage:
//   AGENTCTL_URL=http://127.0.0.1:8080 npx ts-node guarded-tools.ts

import { exec } from "child_process";
import { writeFile } from "fs/promises";
import { promisify } from "util";
import { AgentctlClient, AgentctlNodeGuard } from "../../packages/js/src";

const execAsync = promisify(exec);

const client = new AgentctlClient({
  baseUrl: process.env.AGENTCTL_URL ?? "http://127.0.0.1:8080",
  token: process.env.AGENTCTL_AUTH_TOKEN,
  defaultContext: {
    actor: process.env.USER,
    team: "dev",
  },
});

const guard = new AgentctlNodeGuard(client, {
  sessionId: process.env.AGENTCTL_SESSION ?? `node-${Date.now()}`,
  agent: "node-example",
});

// Wrap exec: shell commands are gated as run_code
export const guardedExec = guard.wrapExec(
  (cmd: string) => execAsync(cmd).then(({ stdout }) => stdout),
);

// Wrap writeFile: file writes are gated as write_file
export const guardedWriteFile = guard.wrapWriteFile(
  (path: string, content: string) => writeFile(path, content, "utf8"),
);

// Wrap fetch: outbound HTTP is gated as call_external_api
export const guardedFetch = guard.wrapFetch(fetch);
