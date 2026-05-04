import { app, BrowserWindow, dialog, ipcMain, type OpenDialogOptions, type SaveDialogOptions } from "electron";
import {
  createWorkflowBridgeClient,
  type RuntimeSettings,
  type SettingsResponse,
  type UserPreferencesResponse,
  unixTarget,
  waitForReady,
  type RunWorkflowRequest,
  type WorkflowBridgeClient,
  type WorkflowEvent,
} from "@dot-files/bridge";
import { type ChildProcess, spawn } from "node:child_process";
import { copyFileSync, existsSync, mkdirSync, readFileSync, rmSync, statSync, unlinkSync, writeFileSync } from "node:fs";
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
let savedSettings: Partial<RuntimeSettings> = {};

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
  Promise.resolve()
    .then(() => {
      savedSettings = readSavedSettings();
    })
    .then(startWorkflowBridge)
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

ipcMain.handle("settings:get", async () => unary<SettingsResponse>((callback) => client().getSettings({}, callback)));

ipcMain.handle("settings:validate", async (_event, settings: RuntimeSettings) =>
  unary<SettingsResponse>((callback) => client().validateSettings({ settings }, callback)),
);

ipcMain.handle("settings:save", async (_event, settings: RuntimeSettings) => saveSettings(settings));

ipcMain.handle("preferences:get", async () =>
  unary<UserPreferencesResponse>((callback) => client().getUserPreferences({}, callback)),
);

ipcMain.handle("preferences:save", async (_event, theme: string) =>
  unary<UserPreferencesResponse>((callback) => client().saveUserPreferences({ theme }, callback)),
);

ipcMain.handle("settings:choose-directory", async (_event, defaultPath?: string) => {
  const options: OpenDialogOptions = {
    defaultPath,
    properties: ["openDirectory", "createDirectory"],
  };
  const result = mainWindow ? await dialog.showOpenDialog(mainWindow, options) : await dialog.showOpenDialog(options);

  return result.canceled ? null : result.filePaths[0] ?? null;
});

ipcMain.handle("settings:choose-file", async (_event, defaultPath?: string) => {
  const options: OpenDialogOptions = {
    defaultPath,
    properties: ["openFile"],
  };
  const result = mainWindow ? await dialog.showOpenDialog(mainWindow, options) : await dialog.showOpenDialog(options);

  return result.canceled ? null : result.filePaths[0] ?? null;
});

ipcMain.handle("settings:choose-save-file", async (_event, defaultPath?: string) => {
  const options: SaveDialogOptions = {
    defaultPath,
    properties: ["createDirectory", "showOverwriteConfirmation"],
  };
  const result = mainWindow ? await dialog.showSaveDialog(mainWindow, options) : await dialog.showSaveDialog(options);

  return result.canceled ? null : result.filePath ?? null;
});

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

  const child = spawn(command.command, [...command.args, "serve-grpc", "--socket", bridgeSocketPath, ...settingsArgs(savedSettings)], {
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

async function saveSettings(settings: RuntimeSettings): Promise<SettingsResponse> {
  const validation = await unary<SettingsResponse>((callback) => client().validateSettings({ settings }, callback));

  if (!validation.valid || !validation.settings) {
    return validation;
  }

  const current = await unary<SettingsResponse>((callback) => client().getSettings({}, callback));
  const previousSettings = savedSettings;
  const nextSettings = validation.settings;
  let rollbackDatabaseMove = () => {};

  stopWorkflowBridge();

  try {
    rollbackDatabaseMove = moveWorkflowDatabase(current.settings?.workflowDbPath, nextSettings.workflowDbPath);
    savedSettings = nextSettings;
    writeSavedSettings(nextSettings);
    await startWorkflowBridge();

    return unary<SettingsResponse>((callback) => client().getSettings({}, callback));
  } catch (error) {
    rollbackDatabaseMove();
    savedSettings = previousSettings;
    writeSavedSettings(previousSettings);
    await startWorkflowBridge();
    throw error;
  }
}

function settingsPath() {
  return join(app.getPath("userData"), "settings.json");
}

function readSavedSettings(): Partial<RuntimeSettings> {
  try {
    const data = JSON.parse(readFileSync(settingsPath(), "utf8")) as Partial<RuntimeSettings>;

    return cleanSettings(data);
  } catch {
    return {};
  }
}

function writeSavedSettings(settings: Partial<RuntimeSettings>) {
  mkdirSync(dirname(settingsPath()), { recursive: true });
  writeFileSync(settingsPath(), JSON.stringify(cleanSettings(settings), null, 2) + "\n", "utf8");
}

function cleanSettings(settings: Partial<RuntimeSettings>): Partial<RuntimeSettings> {
  return {
    repoRoot: settings.repoRoot ?? "",
    appsConfigPath: settings.appsConfigPath ?? "",
    secretsConfigPath: settings.secretsConfigPath ?? "",
    generatedAppsPath: settings.generatedAppsPath ?? "",
    archiveRoot: settings.archiveRoot ?? "",
    workflowDbPath: settings.workflowDbPath ?? "",
    opVault: settings.opVault ?? "",
    opItem: settings.opItem ?? "",
  };
}

function settingsArgs(settings: Partial<RuntimeSettings>) {
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

function moveWorkflowDatabase(fromPath?: string, toPath?: string) {
  if (!fromPath || !toPath || fromPath === toPath) {
    return () => {};
  }

  if (existsSync(toPath) && statSync(toPath).isDirectory()) {
    throw new Error(`Workflow database path is a directory: ${toPath}`);
  }

  mkdirSync(dirname(toPath), { recursive: true });

  if (!existsSync(fromPath)) {
    return () => {};
  }

  if (existsSync(toPath)) {
    throw new Error(`Workflow database already exists: ${toPath}`);
  }

  copyFileSync(fromPath, toPath);
  unlinkSync(fromPath);

  return () => {
    if (existsSync(toPath) && !existsSync(fromPath)) {
      mkdirSync(dirname(fromPath), { recursive: true });
      copyFileSync(toPath, fromPath);
      unlinkSync(toPath);
    }
  };
}
