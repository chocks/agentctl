import {
  Configuration,
  GateApi,
  ReplayApi,
  TraceApi,
  type Action,
  type ActionParams,
  type Decision,
  type GateRequest,
  type ModelRequestContext,
  type ReplayResponse,
  type TraceListResponse,
  type Verdict,
} from "../../../sdk/js/src/index";

export type AgentctlClientOptions = {
  baseUrl?: string;
  token?: string;
  fetchApi?: typeof fetch;
  defaultContext?: Partial<ModelRequestContext>;
};

export type TraceFilter = {
  sessionId?: string;
  action?: Action;
  verdict?: Verdict;
  packageName?: string;
  since?: Date;
  until?: Date;
  limit?: number;
};

export type ReplayInput = {
  sessionId: string;
  policyPath?: string;
  limit?: number;
};

export class AgentctlClient {
  private readonly gateApi: GateApi;
  private readonly replayApi: ReplayApi;
  private readonly traceApi: TraceApi;
  private readonly defaultContext?: Partial<ModelRequestContext>;

  constructor(options: AgentctlClientOptions = {}) {
    const config = new Configuration({
      basePath: options.baseUrl ?? "http://127.0.0.1:8080",
      accessToken: options.token,
      fetchApi: options.fetchApi,
    });

    this.gateApi = new GateApi(config);
    this.replayApi = new ReplayApi(config);
    this.traceApi = new TraceApi(config);
    this.defaultContext = options.defaultContext;
  }

  async gate(input: {
    action: Action;
    params: ActionParams;
    reason: string;
    context?: Partial<ModelRequestContext>;
  }): Promise<Decision> {
    return this.gateApi.gateAction({
      gateRequest: {
        action: input.action,
        params: input.params,
        reason: input.reason,
        context: mergeContext(this.defaultContext, input.context),
      },
    });
  }

  async listTraces(filter: TraceFilter = {}): Promise<TraceListResponse> {
    return this.traceApi.listTraces({
      sessionId: filter.sessionId,
      action: filter.action,
      verdict: filter.verdict,
      _package: filter.packageName,
      since: filter.since,
      until: filter.until,
      limit: filter.limit,
    });
  }

  async replay(input: ReplayInput): Promise<ReplayResponse> {
    return this.replayApi.replaySession({
      replayRequest: {
        sessionId: input.sessionId,
        policyPath: input.policyPath,
        limit: input.limit,
      },
    });
  }
}

function mergeContext(
  defaults?: Partial<ModelRequestContext>,
  input?: Partial<ModelRequestContext>,
): ModelRequestContext | undefined {
  const merged = {
    ...defaults,
    ...input,
  };

  if (!merged.sessionId || !merged.timestamp) {
    return undefined;
  }

  return merged as ModelRequestContext;
}
