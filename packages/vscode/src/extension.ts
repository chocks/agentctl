import * as vscode from "vscode";
import { AgentctlClient } from "./client";
import { ApprovalItem, ApprovalProvider } from "./approvalProvider";
import { TraceProvider } from "./traceProvider";
import { ServerManager } from "./server";

function readConfig() {
  const cfg = vscode.workspace.getConfiguration("agentctl");
  return {
    serverUrl: cfg.get<string>("serverUrl") ?? "http://127.0.0.1:8080",
    authToken: cfg.get<string>("authToken") ?? "",
    pollIntervalMs: cfg.get<number>("pollIntervalMs") ?? 5000,
    autoStartServer: cfg.get<boolean>("autoStartServer") ?? false,
    tracesLimit: cfg.get<number>("tracesLimit") ?? 50,
  };
}

function serverAddr(serverUrl: string): string {
  try {
    return new URL(serverUrl).host;
  } catch {
    return "127.0.0.1:8080";
  }
}

export async function activate(ctx: vscode.ExtensionContext): Promise<void> {
  const cfg = readConfig();
  const client = new AgentctlClient(cfg.serverUrl, cfg.authToken);

  const approvalProvider = new ApprovalProvider(
    client,
    () => readConfig().pollIntervalMs,
  );
  const traceProvider = new TraceProvider(
    client,
    () => readConfig().tracesLimit,
  );
  const serverManager = new ServerManager();

  ctx.subscriptions.push(
    approvalProvider,
    traceProvider,
    serverManager,
    vscode.window.createTreeView("agentctl.approvals", {
      treeDataProvider: approvalProvider,
    }),
    vscode.window.createTreeView("agentctl.traces", {
      treeDataProvider: traceProvider,
    }),
  );

  ctx.subscriptions.push(
    vscode.commands.registerCommand("agentctl.refreshApprovals", () => {
      void approvalProvider.refresh();
    }),

    vscode.commands.registerCommand("agentctl.refreshTraces", () => {
      void traceProvider.refresh();
    }),

    vscode.commands.registerCommand(
      "agentctl.approve",
      async (item: ApprovalItem) => {
        try {
          await client.approve(item.record.approval_id);
          void approvalProvider.refresh();
          void traceProvider.refresh();
          vscode.window.showInformationMessage(
            `Approved: ${item.record.action}`,
          );
        } catch (e) {
          vscode.window.showErrorMessage(`Approve failed: ${String(e)}`);
        }
      },
    ),

    vscode.commands.registerCommand(
      "agentctl.deny",
      async (item: ApprovalItem) => {
        try {
          await client.deny(item.record.approval_id);
          void approvalProvider.refresh();
          void traceProvider.refresh();
          vscode.window.showInformationMessage(`Denied: ${item.record.action}`);
        } catch (e) {
          vscode.window.showErrorMessage(`Deny failed: ${String(e)}`);
        }
      },
    ),

    vscode.commands.registerCommand("agentctl.startServer", () => {
      const { serverUrl, authToken } = readConfig();
      serverManager.start(serverAddr(serverUrl), authToken, () => {
        approvalProvider.startPolling();
        void traceProvider.refresh();
      });
      serverManager.showOutput();
    }),

    vscode.commands.registerCommand("agentctl.stopServer", () => {
      serverManager.stop();
      approvalProvider.stopPolling();
    }),

    vscode.commands.registerCommand("agentctl.toggleServer", () => {
      if (serverManager.running) {
        vscode.commands.executeCommand("agentctl.stopServer");
      } else {
        vscode.commands.executeCommand("agentctl.startServer");
      }
    }),

    // Restart polling whenever the poll interval setting changes
    vscode.workspace.onDidChangeConfiguration((e) => {
      if (e.affectsConfiguration("agentctl.pollIntervalMs")) {
        if (serverManager.running) {
          approvalProvider.startPolling();
        }
      }
    }),
  );

  // If server is already up, start polling without starting a new process
  if (cfg.autoStartServer) {
    vscode.commands.executeCommand("agentctl.startServer");
  } else {
    const alive = await client.healthz();
    if (alive) {
      approvalProvider.startPolling();
      void traceProvider.refresh();
    }
  }
}

export function deactivate(): void {
  // Disposables registered via ctx.subscriptions handle cleanup
}
