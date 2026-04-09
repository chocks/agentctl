/**
 * Standalone HTTP client for the agentctl server.
 * Uses the built-in fetch API (Node 18+, VS Code 1.85+).
 * Keeps snake_case JSON shapes from the server; callers use snake_case field names.
 */

export interface ApprovalRecord {
  approval_id: string;
  trace_id: string;
  session_id: string;
  action: string;
  status: "pending" | "approved" | "denied";
  reason: string;
  requested_at: string;
  resolved_at?: string;
  resolved_by?: string;
}

export interface TraceRecord {
  trace_id: string;
  verdict: string;
  risk_score: number;
  reason: string;
  timestamp: string;
  request: {
    action: string;
    reason: string;
    context: {
      session_id: string;
      actor?: string;
      agent?: string;
    };
    params?: Record<string, string>;
  };
}

export class AgentctlClient {
  private readonly baseUrl: string;
  private readonly token: string;

  constructor(baseUrl: string, token: string) {
    this.baseUrl = baseUrl.replace(/\/$/, "");
    this.token = token;
  }

  private headers(actor?: string): Record<string, string> {
    const h: Record<string, string> = {};
    if (this.token) {
      h["Authorization"] = `Bearer ${this.token}`;
    }
    if (actor) {
      h["X-Agentctl-Actor"] = actor;
    }
    return h;
  }

  async healthz(): Promise<boolean> {
    try {
      const res = await fetch(`${this.baseUrl}/healthz`, {
        signal: AbortSignal.timeout(2000),
      });
      return res.ok;
    } catch {
      return false;
    }
  }

  async listApprovals(status?: string): Promise<ApprovalRecord[]> {
    const url = status
      ? `${this.baseUrl}/v1/approvals?status=${encodeURIComponent(status)}`
      : `${this.baseUrl}/v1/approvals`;
    const res = await fetch(url, { headers: this.headers() });
    if (!res.ok) {
      throw new Error(`listApprovals HTTP ${res.status}`);
    }
    const data = (await res.json()) as { approvals: ApprovalRecord[] };
    return data.approvals ?? [];
  }

  async approve(approvalId: string, actor?: string): Promise<ApprovalRecord> {
    const res = await fetch(
      `${this.baseUrl}/v1/approvals/${encodeURIComponent(approvalId)}/approve`,
      { method: "POST", headers: this.headers(actor) },
    );
    if (!res.ok) {
      throw new Error(`approve HTTP ${res.status}`);
    }
    return res.json() as Promise<ApprovalRecord>;
  }

  async deny(approvalId: string, actor?: string): Promise<ApprovalRecord> {
    const res = await fetch(
      `${this.baseUrl}/v1/approvals/${encodeURIComponent(approvalId)}/deny`,
      { method: "POST", headers: this.headers(actor) },
    );
    if (!res.ok) {
      throw new Error(`deny HTTP ${res.status}`);
    }
    return res.json() as Promise<ApprovalRecord>;
  }

  async listTraces(limit = 50): Promise<TraceRecord[]> {
    const res = await fetch(`${this.baseUrl}/v1/traces?limit=${limit}`, {
      headers: this.headers(),
    });
    if (!res.ok) {
      throw new Error(`listTraces HTTP ${res.status}`);
    }
    const data = (await res.json()) as { traces: TraceRecord[] };
    return data.traces ?? [];
  }
}
