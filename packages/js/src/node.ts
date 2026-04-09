import { AgentctlClient } from "./client";
import {
  Action,
  type ActionParams,
} from "../../../sdk/js/src/index";

export type GuardContext = {
  sessionId?: string;
  agent?: string;
  model?: string;
  actor?: string;
  team?: string;
  turn?: number;
};

export type FetchLike = (input: string, init?: { method?: string }) => Promise<unknown>;
export type ExecLike = (command: string) => Promise<unknown>;
export type WriteFileLike = (path: string, content: string) => Promise<unknown>;

export class AgentctlNodeGuard {
  constructor(
    private readonly client: AgentctlClient,
    private readonly defaultContext: GuardContext = {},
  ) {}

  wrapExec(execFn: ExecLike, context: GuardContext = {}): ExecLike {
    return async (command: string) => {
      await this.assertAllowed(
        Action.RunCode,
        { language: "bash", command },
        `execute command: ${command}`,
        context,
      );
      return execFn(command);
    };
  }

  wrapWriteFile(writeFileFn: WriteFileLike, context: GuardContext = {}): WriteFileLike {
    return async (path: string, content: string) => {
      await this.assertAllowed(
        Action.WriteFile,
        {
          path,
          operation: "overwrite",
          sizeBytes: new TextEncoder().encode(content).length,
        },
        `write file: ${path}`,
        context,
      );
      return writeFileFn(path, content);
    };
  }

  wrapFetch(fetchFn: FetchLike, context: GuardContext = {}): FetchLike {
    return async (input: string, init?: { method?: string }) => {
      const url = new URL(input);
      await this.assertAllowed(
        Action.CallExternalApi,
        {
          url: url.toString(),
          method: init?.method ?? "GET",
          domain: url.hostname,
        },
        `call external API: ${url.hostname}`,
        context,
      );
      return fetchFn(input, init);
    };
  }

  private async assertAllowed(
    action: Action,
    params: ActionParams,
    reason: string,
    context: GuardContext,
  ): Promise<void> {
    const requestContext = buildContext(this.defaultContext, context);
    const decision = await this.client.gate({
      action,
      params,
      reason,
      context: requestContext,
    });

    if (decision.verdict !== "allow") {
      throw new Error(`agentctl ${decision.verdict}: ${decision.reason}`);
    }
  }
}

function buildContext(defaults: GuardContext, current: GuardContext) {
  const merged = { ...defaults, ...current };
  return {
    sessionId: merged.sessionId ?? `node-${Date.now()}`,
    agent: merged.agent,
    model: merged.model,
    actor: merged.actor,
    team: merged.team,
    turn: merged.turn,
    timestamp: new Date(),
  };
}
