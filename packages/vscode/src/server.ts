import * as vscode from "vscode";
import { spawn, type ChildProcess } from "child_process";

type ServerState = "stopped" | "starting" | "running";

export class ServerManager implements vscode.Disposable {
  private proc?: ChildProcess;
  private state: ServerState = "stopped";
  private readonly statusBar: vscode.StatusBarItem;
  private readonly output: vscode.OutputChannel;

  constructor() {
    this.statusBar = vscode.window.createStatusBarItem(
      vscode.StatusBarAlignment.Right,
      100,
    );
    this.statusBar.command = "agentctl.toggleServer";
    this.output = vscode.window.createOutputChannel("agentctl");
    this.setState("stopped");
    this.statusBar.show();
  }

  get running(): boolean {
    return this.proc !== undefined;
  }

  start(addr: string, token: string, onReady: () => void): void {
    if (this.running) {
      return;
    }

    const args = ["serve", "--addr", addr];
    if (token) {
      args.push("--auth-token", token);
    }

    this.proc = spawn("agentctl", args, { stdio: ["ignore", "pipe", "pipe"] });
    this.setState("starting");

    let readyCalled = false;
    const callReady = () => {
      if (!readyCalled) {
        readyCalled = true;
        this.setState("running");
        onReady();
      }
    };

    this.proc.stdout?.on("data", (chunk: Buffer) => {
      this.output.append(chunk.toString());
    });

    this.proc.stderr?.on("data", (chunk: Buffer) => {
      const text = chunk.toString();
      this.output.append(text);
      if (text.includes("listening")) {
        callReady();
      }
    });

    this.proc.on("error", (err) => {
      this.output.appendLine(`agentctl failed to start: ${err.message}`);
      this.proc = undefined;
      this.setState("stopped");
      readyCalled = true; // prevent double-call
      vscode.window.showErrorMessage(
        `agentctl server failed to start: ${err.message}`,
      );
    });

    this.proc.on("exit", (code) => {
      this.output.appendLine(`\nagentctl exited (code ${code ?? "?"})`);
      this.proc = undefined;
      this.setState("stopped");
    });

    // Fallback: assume ready after 1.5s if no "listening" line appears
    setTimeout(callReady, 1500);
  }

  stop(): void {
    if (this.proc) {
      this.proc.kill("SIGTERM");
      this.proc = undefined;
      this.setState("stopped");
    }
  }

  showOutput(): void {
    this.output.show();
  }

  private setState(s: ServerState): void {
    this.state = s;
    switch (s) {
      case "stopped":
        this.statusBar.text = "$(circle-slash) agentctl";
        this.statusBar.tooltip = "agentctl server stopped — click to start";
        break;
      case "starting":
        this.statusBar.text = "$(sync~spin) agentctl";
        this.statusBar.tooltip = "agentctl server starting…";
        break;
      case "running":
        this.statusBar.text = "$(circle-filled) agentctl";
        this.statusBar.tooltip = "agentctl server running — click to stop";
        break;
    }
  }

  dispose(): void {
    this.stop();
    this.statusBar.dispose();
    this.output.dispose();
  }
}
