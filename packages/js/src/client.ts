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

export type ApprovalRecord = {
  approvalId: string;
  traceId: string;
  sessionId: string;
  action: Action;
  status: "pending" | "approved" | "denied";
  reason: string;
  requestedAt: string;
  resolvedAt?: string;
  resolvedBy?: string;
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
  private readonly baseUrl: string;
  private readonly token?: string;
  private readonly fetchApi: typeof fetch;

  constructor(options: AgentctlClientOptions = {}) {
    this.baseUrl = options.baseUrl ?? "http://127.0.0.1:8080";
    this.token = options.token;
    this.fetchApi = options.fetchApi ?? globalThis.fetch;

    const config = new Configuration({
      basePath: this.baseUrl,
      accessToken: this.token,
      fetchApi: this.fetchApi,
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

  async waitForApproval(
    approvalId: string,
    options: { timeoutSecs?: number } = {},
  ): Promise<ApprovalRecord> {
    const timeoutSecs = options.timeoutSecs ?? 30;
    const url = `${this.baseUrl}/v1/approvals/${encodeURIComponent(approvalId)}/wait?timeout=${timeoutSecs}s`;
    const headers: Record<string, string> = {};
    if (this.token) {
      headers["Authorization"] = `Bearer ${this.token}`;
    }
    const resp = await this.fetchApi(url, { method: "GET", headers } as RequestInit);
    if (!resp.ok && (resp as Response).status !== 408) {
      throw new Error(`waitForApproval: HTTP ${(resp as Response).status}`);
    }
    const data = await (resp as Response).json();
    return {
      approvalId: data.approval_id,
      traceId: data.trace_id,
      sessionId: data.session_id,
      action: data.action,
      status: data.status,
      reason: data.reason,
      requestedAt: data.requested_at,
      resolvedAt: data.resolved_at,
      resolvedBy: data.resolved_by,
    };
  }
}

function mergeContext(
  defaults?: Partial<ModelRequestContext>,
  input?: Partial<ModelRequestContext>,
): ModelRequestContext {
  const merged: Partial<ModelRequestContext> = {
    ...defaults,
    ...input,
  };

  if (!merged.sessionId) {
    // Synthesize an ephemeral session id rather than silently dropping context.
    // Callers should set defaultContext.sessionId for reliable trace correlation.
    console.warn(
      "[agentctl] warning: no session_id provided — using ephemeral id. " +
        "Set defaultContext.sessionId for reliable trace correlation.",
    );
    merged.sessionId = `ephemeral-${Date.now()}`;
  }

  if (!merged.timestamp) {
    merged.timestamp = new Date();
  }

  return merged as ModelRequestContext;
}
