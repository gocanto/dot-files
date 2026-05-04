import { app, BrowserWindow, ipcMain } from "electron";
import { spawn } from "node:child_process";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

interface RunRequest {
  workflowId: string;
  confirmationOptionId: string;
  enabledPhaseIds: string[];
}

const __dirname = dirname(fileURLToPath(import.meta.url));
const repoRoot = resolve(__dirname, "..", "..", "..");
const macbookDir = join(repoRoot, "macbook");

let mainWindow: BrowserWindow | null = null;

function createWindow() {
  mainWindow = new BrowserWindow({
    width: 1280,
    height: 820,
    minWidth: 1040,
    minHeight: 700,
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
  } else {
    void mainWindow.loadFile(join(repoRoot, "packages", "ui", "dist", "index.html"));
  }
}

app.whenReady().then(createWindow);

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

ipcMain.handle("workflows:list", () => runJSON(["ui", "workflows"]));
ipcMain.handle("runs:list", (_event, limit: number) => runJSON(["ui", "runs", "--limit", String(limit)]));
ipcMain.handle("runs:log", (_event, runId: string) => runJSON(["ui", "run-log", "--run-id", runId]));

ipcMain.handle("workflow:run", async (event, request: RunRequest, eventChannel: string) => {
  const result = await runStreaming(["ui", "run"], JSON.stringify(request), (line) => {
    event.sender.send(eventChannel, JSON.parse(line));
  });

  return { exitCode: result.exitCode };
});

function goCommand() {
  const packaged = join(process.resourcesPath || "", "mac-os");

  if (app.isPackaged) {
    return { command: packaged, args: [] };
  }

  return { command: "go", args: ["run", "./cmd"] };
}

async function runJSON(args: string[]) {
  const result = await runBuffered(args);

  if (result.exitCode !== 0) {
    throw new Error(result.stderr || `mac-os exited with ${result.exitCode}`);
  }

  return JSON.parse(result.stdout || "null");
}

function runBuffered(args: string[], input?: string) {
  return new Promise<{ stdout: string; stderr: string; exitCode: number }>((resolveResult, reject) => {
    const child = spawnMacOS(args);
    let stdout = "";
    let stderr = "";

    child.stdout.on("data", (chunk: Buffer) => {
      stdout += chunk.toString("utf8");
    });

    child.stderr.on("data", (chunk: Buffer) => {
      stderr += chunk.toString("utf8");
    });

    child.on("error", reject);
    child.on("close", (exitCode) => resolveResult({ stdout, stderr, exitCode: exitCode ?? 1 }));

    if (input) {
      child.stdin.end(input);
    } else {
      child.stdin.end();
    }
  });
}

function runStreaming(args: string[], input: string, onLine: (line: string) => void) {
  return new Promise<{ stderr: string; exitCode: number }>((resolveResult, reject) => {
    const child = spawnMacOS(args);
    let stderr = "";
    let pending = "";

    child.stdout.on("data", (chunk: Buffer) => {
      pending += chunk.toString("utf8");
      const lines = pending.split("\n");
      pending = lines.pop() ?? "";

      for (const line of lines) {
        if (line.trim() !== "") {
          onLine(line);
        }
      }
    });

    child.stderr.on("data", (chunk: Buffer) => {
      stderr += chunk.toString("utf8");
    });

    child.on("error", reject);
    child.on("close", (exitCode) => {
      if (pending.trim() !== "") {
        onLine(pending);
      }

      resolveResult({ stderr, exitCode: exitCode ?? 1 });
    });

    child.stdin.end(input);
  });
}

function spawnMacOS(args: string[]) {
  const command = goCommand();

  return spawn(command.command, [...command.args, ...args], {
    cwd: macbookDir,
    env: process.env,
    stdio: ["pipe", "pipe", "pipe"],
  });
}
