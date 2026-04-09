import * as vscode from "vscode";
import { AgentctlClient, type TraceRecord } from "./client";

const VERDICT_ICON: Record<string, string> = {
  allow: "check",
  deny: "x",
  escalate: "warning",
};

export class TraceItem extends vscode.TreeItem {
  constructor(readonly record: TraceRecord) {
    const ts = new Date(record.timestamp).toLocaleTimeString();
    super(
      `${record.request.action} · ${record.verdict}`,
      vscode.TreeItemCollapsibleState.None,
    );
    this.description = `${ts} — ${record.reason}`;
    this.tooltip = new vscode.MarkdownString(
      `**${record.verdict.toUpperCase()}** \`${record.request.action}\`\n\n` +
        `Risk score: ${record.risk_score}  \n` +
        `Session: \`${record.request.context.session_id}\`  \n` +
        `Reason: ${record.reason}  \n` +
        `Time: ${record.timestamp}`,
    );
    this.iconPath = new vscode.ThemeIcon(
      VERDICT_ICON[record.verdict] ?? "circle-outline",
    );
  }
}

export class TraceProvider implements vscode.TreeDataProvider<TraceItem> {
  private readonly _onDidChangeTreeData = new vscode.EventEmitter<
    TraceItem | undefined | void
  >();
  readonly onDidChangeTreeData = this._onDidChangeTreeData.event;

  private traces: TraceRecord[] = [];

  constructor(
    private client: AgentctlClient,
    private getLimit: () => number,
  ) {}

  async refresh(): Promise<void> {
    try {
      this.traces = await this.client.listTraces(this.getLimit());
    } catch {
      this.traces = [];
    }
    this._onDidChangeTreeData.fire();
  }

  getTreeItem(element: TraceItem): vscode.TreeItem {
    return element;
  }

  getChildren(): TraceItem[] {
    return this.traces.map((r) => new TraceItem(r));
  }

  dispose(): void {
    this._onDidChangeTreeData.dispose();
  }
}
