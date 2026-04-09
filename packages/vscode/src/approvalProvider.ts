import * as vscode from "vscode";
import { AgentctlClient, type ApprovalRecord } from "./client";

export class ApprovalItem extends vscode.TreeItem {
  constructor(readonly record: ApprovalRecord) {
    super(
      `[${record.action}] ${record.reason}`,
      vscode.TreeItemCollapsibleState.None,
    );
    this.contextValue = record.status;
    this.description = record.session_id;
    this.tooltip = new vscode.MarkdownString(
      `**${record.action}** · \`${record.status}\`\n\n` +
        `Session: \`${record.session_id}\`  \n` +
        `Requested: ${record.requested_at}`,
    );
    this.iconPath = new vscode.ThemeIcon(
      record.status === "pending"
        ? "clock"
        : record.status === "approved"
          ? "check"
          : "x",
    );
  }
}

export class ApprovalProvider
  implements vscode.TreeDataProvider<ApprovalItem>
{
  private readonly _onDidChangeTreeData = new vscode.EventEmitter<
    ApprovalItem | undefined | void
  >();
  readonly onDidChangeTreeData = this._onDidChangeTreeData.event;

  private approvals: ApprovalRecord[] = [];
  private pollTimer?: ReturnType<typeof setInterval>;

  constructor(
    private client: AgentctlClient,
    private getPollIntervalMs: () => number,
  ) {}

  startPolling(): void {
    this.stopPolling();
    this.pollTimer = setInterval(
      () => void this.refresh(),
      this.getPollIntervalMs(),
    );
    void this.refresh();
  }

  stopPolling(): void {
    if (this.pollTimer !== undefined) {
      clearInterval(this.pollTimer);
      this.pollTimer = undefined;
    }
  }

  async refresh(): Promise<void> {
    try {
      this.approvals = await this.client.listApprovals("pending");
    } catch {
      this.approvals = [];
    }
    this._onDidChangeTreeData.fire();
  }

  getTreeItem(element: ApprovalItem): vscode.TreeItem {
    return element;
  }

  getChildren(): ApprovalItem[] {
    return this.approvals.map((r) => new ApprovalItem(r));
  }

  dispose(): void {
    this.stopPolling();
    this._onDidChangeTreeData.dispose();
  }
}
