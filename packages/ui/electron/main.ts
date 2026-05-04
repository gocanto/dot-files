import { app, BrowserWindow, ipcMain } from "electron";
import {
  createWorkflowBridgeClient,
  unixTarget,
  waitForReady,
  type RunWorkflowRequest,
  type WorkflowBridgeClient,
  type WorkflowEvent,
} from "@dot-files/bridge";
import { type ChildProcess, spawn } from "node:child_process";
import { rmSync } from "node:fs";
import { tmpdir } from "node:os";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const __dirname = dirname(fileURLToPath(import.meta.url));
const repoRoot = resolve(__dirname, "..", "..", "..");
const macbookDir = join(repoRoot, "macbook");

let mainWindow: BrowserWindow | null = null;
let bridgeClient: WorkflowBridgeClient | null = null;
let bridgeProcess: ChildProcess | null = null;
let bridgeSocketPath = "";

function createWindow() {
  mainWindow = new BrowserWindow({
    width: 2000,
    height: 1500,
    minWidth: 1024,
    minHeight: 800,
    title: "Mac OS Manager",
    webPreferences: {
      preload: join(__dirname, "preload.cjs"),
      contextIsolation: true,
      nodeIntegration: false,
    },
  });

  const devServer = process.env.VITE_DEV_SERVER_URL;

  if (devServer) {
    void mainWindow.loadURL(devServer);
    mainWindow.webContents.openDevTools();
  } else {
    void mainWindow.loadFile(join(repoRoot, "packages", "ui", "dist", "index.html"));
  }
}

app.whenReady().then(() =>
  startWorkflowBridge()
    .then(createWindow)
    .catch((error: unknown) => {
      console.error(error);
      app.quit();
    }),
);

app.on("window-all-closed", () => {
  if (process.platform !== "darwin") {
    app.quit();
  }
});

app.on("activate", () => {
  if (BrowserWindow.getAllWindows().length === 0) {
    createWindow();
  }
});

app.on("before-quit", stopWorkflowBridge);

ipcMain.handle("workflows:list", async () => {
  const response = await unary<{ workflows?: unknown[] }>((callback) => client().listWorkflows({}, callback));

  return response.workflows ?? [];
});

ipcMain.handle("runs:list", async (_event, limit: number) => {
  const response = await unary<{ runs?: unknown[] }>((callback) => client().listRuns({ limit }, callback));

  return response.runs ?? [];
});

ipcMain.handle("runs:log", (_event, runId: string) => unary((callback) => client().runLog({ runId }, callback)));

ipcMain.handle("workflow:run", (event, request: RunWorkflowRequest, eventChannel: string) => {
  return new Promise<{ exitCode: number }>((resolveResult, reject) => {
    const stream = client().runWorkflow(request);
    let exitCode = 0;

    stream.on("data", (workflowEvent: WorkflowEvent) => {
      if (workflowEvent.type === "run_failed") {
        exitCode = 1;
      }

      event.sender.send(eventChannel, workflowEvent);
    });

    stream.on("error", reject);
    stream.on("end", () => resolveResult({ exitCode }));
  });
});

function goCommand() {
  const packaged = join(process.resourcesPath || "", "mac-os");

  if (app.isPackaged) {
    return { command: packaged, args: [] };
  }

  return { command: "go", args: ["run", "./cmd"] };
}

async function startWorkflowBridge() {
  if (bridgeClient) {
    return;
  }

  const command = goCommand();
  bridgeSocketPath = join(tmpdir(), `mac-os-${process.pid}-${Date.now()}.sock`);

  const child = spawn(command.command, [...command.args, "serve-grpc", "--socket", bridgeSocketPath], {
    cwd: macbookDir,
    env: process.env,
    stdio: ["ignore", "pipe", "pipe"],
  });
  bridgeProcess = child;

  let stderr = "";

  child.stderr?.on("data", (chunk: Buffer) => {
    stderr += chunk.toString("utf8");
  });

  child.on("exit", (code, signal) => {
    if (bridgeClient) {
      console.error(`mac-os gRPC bridge exited with ${code ?? signal ?? "unknown status"}`);
    }

    bridgeClient?.close();
    bridgeClient = null;
    bridgeProcess = null;
  });

  const grpcClient = createWorkflowBridgeClient(unixTarget(bridgeSocketPath));

  try {
    await waitForReady(grpcClient);
    bridgeClient = grpcClient;
  } catch (error) {
    grpcClient.close();
    child.kill();
    throw new Error(stderr || (error instanceof Error ? error.message : String(error)));
  }
}

function stopWorkflowBridge() {
  bridgeClient?.close();
  bridgeClient = null;

  bridgeProcess?.kill();
  bridgeProcess = null;

  if (bridgeSocketPath) {
    rmSync(bridgeSocketPath, { force: true });
    bridgeSocketPath = "";
  }
}

function client() {
  if (!bridgeClient) {
    throw new Error("mac-os gRPC bridge is not running");
  }

  return bridgeClient;
}

function unary<T>(call: (callback: (error: Error | null, response: T) => void) => void) {
  return new Promise<T>((resolveResult, reject) => {
    call((error, response) => {
      if (error) {
        reject(error);
        return;
      }

      resolveResult(response);
    });
  });
}
