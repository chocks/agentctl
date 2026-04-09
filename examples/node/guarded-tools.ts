import {
  Configuration,
  GateApi,
  type Action,
  type GateRequest,
  type Decision,
} from "../../sdk/js/src";

type AgentctlClientOptions = {
  basePath?: string;
  defaultContext?: GateRequest["context"];
};

type GuardContext = {
  agent?: string;
  model?: string;
  sessionId?: string;
  turn?: number;
};

type WriteFileInput = {
  path: string;
  content: string;
  operation?: "create" | "overwrite" | "append";
};

type FetchLike = (input: string, init?: { method?: string }) => Promise<unknown>;
type ExecLike = (command: string) => Promise<unknown>;
type WriteFileLike = (path: string, content: string) => Promise<unknown>;

export class AgentctlNodeGuard {
  private readonly gateApi: GateApi;
  private readonly defaultContext?: GateRequest["context"];

  constructor(options: AgentctlClientOptions = {}) {
    this.gateApi = new GateApi(
      new Configuration({
        basePath: options.basePath ?? "http://127.0.0.1:8080",
      }),
    );
    this.defaultContext = options.defaultContext;
  }

  async gate(request: Omit<GateRequest, "context"> & { context?: GateRequest["context"] }): Promise<Decision> {
    const response = await this.gateApi.gateAction({
      gateRequest: {
        ...request,
        context: {
          ...this.defaultContext,
          ...request.context,
        },
      },
    });

    return response;
  }

  wrapExec(execFn: ExecLike, context: GuardContext = {}): ExecLike {
    return async (command: string) => {
      await this.assertAllowed("run_code", {
        language: "bash",
        command,
      }, `execute command: ${command}`, context);

      return execFn(command);
    };
  }

  wrapWriteFile(writeFileFn: WriteFileLike, context: GuardContext = {}): WriteFileLike {
    return async (path: string, content: string) => {
      await this.assertAllowed("write_file", {
        path,
        operation: "overwrite",
        sizeBytes: Buffer.byteLength(content),
      }, `write file: ${path}`, context);

      return writeFileFn(path, content);
    };
  }

  wrapFetch(fetchFn: FetchLike, context: GuardContext = {}): FetchLike {
    return async (input: string, init?: { method?: string }) => {
      const url = new URL(input);

      await this.assertAllowed("call_external_api", {
        url: url.toString(),
        method: init?.method ?? "GET",
        domain: url.hostname,
      }, `call external API: ${url.hostname}`, context);

      return fetchFn(input, init);
    };
  }

  async guardedWriteFile(
    writeFileFn: WriteFileLike,
    input: WriteFileInput,
    context: GuardContext = {},
  ): Promise<unknown> {
    const operation = input.operation ?? "overwrite";

    await this.assertAllowed("write_file", {
      path: input.path,
      operation,
      sizeBytes: Buffer.byteLength(input.content),
    }, `write file: ${input.path}`, context);

    return writeFileFn(input.path, input.content);
  }

  private async assertAllowed(
    action: Action,
    params: GateRequest["params"],
    reason: string,
    context: GuardContext,
  ): Promise<void> {
    const decision = await this.gate({
      action,
      params,
      reason,
      context: {
        sessionId: context.sessionId ?? `node-${Date.now()}`,
        agent: context.agent,
        model: context.model,
        turn: context.turn,
        timestamp: new Date(),
      },
    });

    if (decision.verdict !== "allow") {
      throw new Error(`agentctl ${decision.verdict}: ${decision.reason}`);
    }
  }
}
