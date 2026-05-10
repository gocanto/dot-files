import {
  createWorkflowBridgeClient,
  type RuntimeSettings,
  unixTarget,
  waitForReady,
  type WorkflowBridgeClient,
} from "@dot-files/bridge";
import { app } from "electron";
import { spawn, type ChildProcess } from "node:child_process";
import { rmSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { recordDiagnostic } from "#electron/diagnostics.js";
import { macbookDir } from "#electron/paths.js";

let bridgeClient: WorkflowBridgeClient | null = null;
let bridgeProcess: ChildProcess | null = null;
let bridgeSocketPath = "";
const externalBridgeSocketPath = process.env.API_BRIDGE_SOCKET?.trim() ?? "";
let bridgeStartup: Promise<void> | null = null;
let savedSettings: Partial<RuntimeSettings> = {};

export function hasExternalBridge() {
  return externalBridgeSocketPath !== "";
}

export function getBridgeSettings() {
  return savedSettings;
}

export function setBridgeSettings(settings: Partial<RuntimeSettings>) {
  savedSettings = settings;
}

function goCommand() {
  const packaged = join(process.resourcesPath || "", "api");

  if (app.isPackaged) {
    return { command: packaged, args: [] };
  }

  return { command: "go", args: ["run", "./cmd"] };
}

async function startWorkflowBridge() {
  if (bridgeClient) {
    return;
  }

  if (externalBridgeSocketPath) {
    const httpClient = createWorkflowBridgeClient(unixTarget(externalBridgeSocketPath));
    await waitForReady(httpClient);
    bridgeSocketPath = externalBridgeSocketPath;
    bridgeClient = httpClient;

    return;
  }

  const command = goCommand();
  bridgeSocketPath = join(tmpdir(), `api-${process.pid}-${Date.now()}.sock`);

  const child = spawn(
    command.command,
    [...command.args, "serve-http", "--socket", bridgeSocketPath, ...settingsArgs(savedSettings)],
    {
      cwd: app.isPackaged ? app.getPath("userData") : macbookDir,
      env: process.env,
      stdio: ["ignore", "pipe", "pipe"],
    },
  );
  bridgeProcess = child;

  let stderr = "";

  child.stderr?.on("data", (chunk: Buffer) => {
    stderr += chunk.toString("utf8");
  });

  child.on("exit", (code, signal) => {
    if (bridgeClient) {
      const status = code ?? signal ?? "unknown status";

      recordDiagnostic({
        level: "error",
        source: "Backend bridge",
        message: `api HTTP bridge exited with ${status}`,
        details: stderr.trim() || undefined,
      });
      console.error(`api HTTP bridge exited with ${status}`);
    }

    bridgeClient?.close();
    bridgeClient = null;
    bridgeProcess = null;
    bridgeStartup = null;
  });

  const httpClient = createWorkflowBridgeClient(unixTarget(bridgeSocketPath));

  try {
    await waitForReady(httpClient);
    bridgeClient = httpClient;
  } catch (error) {
    httpClient.close();
    child.kill();
    recordDiagnostic({
      level: "error",
      source: "Backend bridge",
      message: "Failed to start api HTTP bridge",
      details: stderr || (error instanceof Error ? error.stack : String(error)),
    });
    throw new Error(stderr || (error instanceof Error ? error.message : String(error)));
  }
}

export function stopWorkflowBridge() {
  bridgeClient?.close();
  bridgeClient = null;
  bridgeStartup = null;

  bridgeProcess?.kill();
  bridgeProcess = null;

  if (bridgeSocketPath && bridgeSocketPath !== externalBridgeSocketPath) {
    rmSync(bridgeSocketPath, { force: true });
  }

  bridgeSocketPath = "";
}

export function startBridgeIfNeeded(): Promise<void> {
  if (bridgeClient) {
    return Promise.resolve();
  }

  if (!bridgeStartup) {
    bridgeStartup = startWorkflowBridge().catch((error: unknown) => {
      bridgeStartup = null;
      throw error;
    });
  }

  return bridgeStartup;
}

export async function client(): Promise<WorkflowBridgeClient> {
  if (!bridgeClient) {
    await startBridgeIfNeeded();
  }

  if (!bridgeClient) {
    throw new Error("api HTTP bridge is not running");
  }

  return bridgeClient;
}

export function settingsArgs(settings: Partial<RuntimeSettings>) {
  const args: string[] = [];
  const pairs: Array<[string, string | undefined]> = [
    ["--repo-root", settings.repoRoot],
    ["--apps-config", settings.appsConfigPath],
    ["--secrets-config", settings.secretsConfigPath],
    ["--generated-apps", settings.generatedAppsPath],
    ["--archive-root", settings.archiveRoot],
    ["--workflow-db", settings.workflowDbPath],
    ["--op-vault", settings.opVault],
    ["--op-item", settings.opItem],
  ];

  for (const [flag, value] of pairs) {
    if (value?.trim()) {
      args.push(flag, value);
    }
  }

  return args;
}
