import os from "node:os";
import {
  app,
  BrowserWindow,
  dialog,
  ipcMain,
  type OpenDialogOptions,
  type SaveDialogOptions,
} from "electron";
import {
  createWorkflowBridgeClient,
  type RuntimeSettings,
  type SettingsResponse,
  unixTarget,
  waitForReady,
  type RunWorkflowRequest,
  type WorkflowBridgeClient,
  type WorkflowEvent,
} from "@dot-files/bridge";
import { execFileSync, type ChildProcess, spawn } from "node:child_process";
import {
  copyFileSync,
  existsSync,
  mkdirSync,
  readFileSync,
  rmSync,
  statSync,
  unlinkSync,
  writeFileSync,
} from "node:fs";
import { tmpdir } from "node:os";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";

const __dirname = dirname(fileURLToPath(import.meta.url));
const repoRoot = resolve(__dirname, "..", "..", "..");
const macbookDir = join(repoRoot, "packages", "macbook");

let mainWindow: BrowserWindow | null = null;
let devToolsWindow: BrowserWindow | null = null;
let bridgeClient: WorkflowBridgeClient | null = null;
let bridgeProcess: ChildProcess | null = null;
let bridgeSocketPath = "";
const externalBridgeSocketPath = process.env.MAC_OS_BRIDGE_SOCKET?.trim() ?? "";
let bridgeStartup: Promise<void> | null = null;
let savedSettings: Partial<RuntimeSettings> = {};
const appWindowWidth = 2000;
const appWindowHeight = 1280;
const devToolsWindowWidth = 400;
const devToolsWindowHeight = 400;

function architectureLabel(architecture: string) {
  if (architecture === "arm64") return "Apple silicon";
  if (architecture === "x64") return "Intel";

  return architecture;
}

function osLabel() {
  const type = os.type();
  const release = os.release();

  return type === "Darwin" ? `macOS ${release}` : `${type} ${release}`;
}

function dsclRead(user: string, key: "JPEGPhoto" | "Picture") {
  try {
    return execFileSync("dscl", [".", "-read", `/Users/${user}`, key], {
      encoding: "utf8",
      maxBuffer: 1024 * 1024 * 4,
      stdio: ["ignore", "pipe", "ignore"],
      timeout: 1000,
    });
  } catch {
    return "";
  }
}

function accountAvatarUrl() {
  if (process.platform !== "darwin") {
    return undefined;
  }

  const username = os.userInfo().username;
  const jpegPhoto = dsclRead(username, "JPEGPhoto");
  const jpegHex = jpegPhoto.replace(/^JPEGPhoto:\s*/u, "").replace(/[^a-fA-F0-9]/gu, "");

  if (jpegHex.length >= 2) {
    try {
      const image = Buffer.from(jpegHex, "hex");

      if (image.length > 0) {
        return `data:image/jpeg;base64,${image.toString("base64")}`;
      }
    } catch {
      // Fall back to the Picture path below.
    }
  }

  const picture = dsclRead(username, "Picture")
    .replace(/^Picture:\s*/u, "")
    .trim();

  if (!picture || !existsSync(picture)) {
    return undefined;
  }

  return pathToFileURL(picture).toString();
}

const singleInstanceLock = app.requestSingleInstanceLock();

if (!singleInstanceLock) {
  app.quit();
} else {
  app.on("second-instance", () => {
    if (!mainWindow || mainWindow.isDestroyed()) {
      return;
    }

    if (mainWindow.isMinimized()) {
      mainWindow.restore();
    }

    mainWindow.center();
    mainWindow.focus();
  });
}

function createWindow() {
  if (mainWindow && !mainWindow.isDestroyed()) {
    mainWindow.focus();
    return;
  }

  mainWindow = new BrowserWindow({
    width: appWindowWidth,
    height: appWindowHeight,
    center: true,
    resizable: false,
    maximizable: false,
    fullscreenable: false,
    title: "Mac OS Manager",
    vibrancy: "sidebar",
    visualEffectState: "active",
    backgroundColor: "#00000000",
    webPreferences: {
      preload: join(__dirname, "preload.cjs"),
      contextIsolation: true,
      nodeIntegration: false,
    },
  });

  mainWindow.on("closed", () => {
    if (devToolsWindow && !devToolsWindow.isDestroyed()) {
      devToolsWindow.close();
    }

    mainWindow = null;
  });

  const devServer = process.env.VITE_DEV_SERVER_URL;

  if (devServer) {
    void mainWindow.loadURL(devServer);
  } else {
    void mainWindow.loadFile(join(repoRoot, "packages", "ui", "dist", "index.html"));
  }
}

function openDevToolsPanel(parentWindow: BrowserWindow) {
  if (devToolsWindow && !devToolsWindow.isDestroyed()) {
    devToolsWindow.close();
  }

  devToolsWindow = new BrowserWindow({
    width: devToolsWindowWidth,
    height: devToolsWindowHeight,
    minWidth: devToolsWindowWidth,
    minHeight: devToolsWindowHeight,
    maxWidth: devToolsWindowWidth,
    maxHeight: devToolsWindowHeight,
    resizable: false,
    show: false,
    title: "Mac OS Manager DevTools",
  });

  devToolsWindow.on("closed", () => {
    devToolsWindow = null;
  });

  devToolsWindow.once("ready-to-show", () => {
    const parentBounds = parentWindow.getBounds();
    devToolsWindow?.setSize(devToolsWindowWidth, devToolsWindowHeight, false);
    devToolsWindow?.setBounds({
      x: parentBounds.x + parentBounds.width,
      y: parentBounds.y,
      width: devToolsWindowWidth,
      height: devToolsWindowHeight,
    });
    devToolsWindow?.show();
  });

  parentWindow.webContents.setDevToolsWebContents(devToolsWindow.webContents);
  parentWindow.webContents.openDevTools({ mode: "detach" });
}

app.whenReady().then(() => {
  try {
    savedSettings = readSavedSettings();
    createWindow();
  } catch (error) {
    console.error(error);
    app.quit();
    return;
  }

  void startBridgeIfNeeded().catch((error: unknown) => {
    console.error("Failed to start mac-os HTTP bridge", error);
  });
});

app.on("window-all-closed", () => {
  if (process.platform !== "darwin") {
    app.quit();
  }
});

app.on("activate", () => {
  createWindow();
});

app.on("before-quit", stopWorkflowBridge);

ipcMain.handle("workflows:list", async () => {
  const response = await (await client()).listWorkflows();

  return response.workflows ?? [];
});

ipcMain.handle("runs:list", async (_event, limit: number) => {
  const response = await (await client()).listRuns({ limit });

  return response.runs ?? [];
});

ipcMain.handle("runs:log", async (_event, runId: string) => (await client()).runLog({ runId }));

ipcMain.handle("template-files:list", async () => {
  const response = await (await client()).listTemplateFiles();

  return response.files ?? [];
});

ipcMain.handle("template-files:read", async (_event, path: string) =>
  (await client()).readTemplateFile({ path }),
);

ipcMain.handle("template-files:save", async (_event, path: string, content: string) =>
  (await client()).saveTemplateFile({ path, content }),
);

ipcMain.handle("settings:get", async () => (await client()).getSettings());

ipcMain.handle("settings:validate", async (_event, settings: RuntimeSettings) =>
  (await client()).validateSettings({ settings }),
);

ipcMain.handle("settings:save", async (_event, settings: RuntimeSettings) =>
  saveSettings(settings),
);

ipcMain.handle("preferences:get", async () => (await client()).getUserPreferences());

ipcMain.handle("preferences:save", async (_event, theme: string) =>
  (await client()).saveUserPreferences({ theme }),
);

ipcMain.handle("op:list-vaults", async () => {
  try {
    const response = await (await client()).listOpVaults();

    return { ok: true as const, vaults: response.vaults ?? [] };
  } catch (error) {
    return opErrorEnvelope(error);
  }
});

ipcMain.handle("op:list-items", async (_event, vault: string) => {
  try {
    const response = await (await client()).listOpItems({ vault });

    return { ok: true as const, items: response.items ?? [] };
  } catch (error) {
    return opErrorEnvelope(error);
  }
});

ipcMain.handle("op:signin", async () => {
  return openTerminalCommand(
    'op signin && echo "\\n[Signed in. You can close this window and return to Mac OS Manager.]"',
  );
});

ipcMain.handle("op:install-dependencies", async () => {
  return openTerminalCommand(
    [
      'if ! command -v brew >/dev/null 2>&1; then echo "Homebrew is required. Run ./setup.sh first, then retry."; exit 1; fi',
      "brew install --cask 1password 1password-cli",
      'echo "\\n[1Password and 1Password CLI install finished. Open 1Password, enable CLI integration if needed, then return to Mac OS Manager.]"',
    ].join("; "),
  );
});

ipcMain.handle("settings:choose-directory", async (_event, defaultPath?: string) => {
  const options: OpenDialogOptions = {
    defaultPath,
    properties: ["openDirectory", "createDirectory"],
  };
  const result = mainWindow
    ? await dialog.showOpenDialog(mainWindow, options)
    : await dialog.showOpenDialog(options);

  return result.canceled ? null : (result.filePaths[0] ?? null);
});

ipcMain.handle("settings:choose-file", async (_event, defaultPath?: string) => {
  const options: OpenDialogOptions = {
    defaultPath,
    properties: ["openFile"],
  };
  const result = mainWindow
    ? await dialog.showOpenDialog(mainWindow, options)
    : await dialog.showOpenDialog(options);

  return result.canceled ? null : (result.filePaths[0] ?? null);
});

ipcMain.handle("settings:choose-save-file", async (_event, defaultPath?: string) => {
  const options: SaveDialogOptions = {
    defaultPath,
    properties: ["createDirectory", "showOverwriteConfirmation"],
  };
  const result = mainWindow
    ? await dialog.showSaveDialog(mainWindow, options)
    : await dialog.showSaveDialog(options);

  return result.canceled ? null : (result.filePath ?? null);
});

ipcMain.handle("system:macName", () => os.userInfo().username);
ipcMain.handle("system:macHostname", () => os.hostname());
ipcMain.handle("system:macSystemInfo", () => ({
  name: os.userInfo().username,
  hostname: os.hostname(),
  osLabel: osLabel(),
  architectureLabel: architectureLabel(os.arch()),
  avatarUrl: accountAvatarUrl(),
}));
ipcMain.handle("system:openDevTools", () => {
  if (!mainWindow || mainWindow.isDestroyed()) {
    return;
  }

  openDevToolsPanel(mainWindow);
});

ipcMain.handle("workflow:run", async (event, request: RunWorkflowRequest, eventChannel: string) => {
  const c = await client();

  return new Promise<{ exitCode: number }>((resolveResult, reject) => {
    const stream = c.runWorkflow(request);
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

  if (externalBridgeSocketPath) {
    const httpClient = createWorkflowBridgeClient(unixTarget(externalBridgeSocketPath));
    await waitForReady(httpClient);
    bridgeSocketPath = externalBridgeSocketPath;
    bridgeClient = httpClient;

    return;
  }

  const command = goCommand();
  bridgeSocketPath = join(tmpdir(), `mac-os-${process.pid}-${Date.now()}.sock`);

  const child = spawn(
    command.command,
    [...command.args, "serve-http", "--socket", bridgeSocketPath, ...settingsArgs(savedSettings)],
    {
      cwd: macbookDir,
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
      console.error(`mac-os HTTP bridge exited with ${code ?? signal ?? "unknown status"}`);
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
    throw new Error(stderr || (error instanceof Error ? error.message : String(error)));
  }
}

function stopWorkflowBridge() {
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

function startBridgeIfNeeded(): Promise<void> {
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

async function client(): Promise<WorkflowBridgeClient> {
  if (!bridgeClient) {
    await startBridgeIfNeeded();
  }

  if (!bridgeClient) {
    throw new Error("mac-os HTTP bridge is not running");
  }

  return bridgeClient;
}

async function saveSettings(settings: RuntimeSettings): Promise<SettingsResponse> {
  const validation = await (await client()).validateSettings({ settings });

  if (!validation.valid || !validation.settings) {
    return validation;
  }

  const current = await (await client()).getSettings();
  const previousSettings = savedSettings;
  const nextSettings = validation.settings;
  let rollbackDatabaseMove = () => {};

  try {
    rollbackDatabaseMove = moveWorkflowDatabase(
      current.settings?.workflowDbPath,
      nextSettings.workflowDbPath,
    );
    savedSettings = nextSettings;
    writeSavedSettings(nextSettings);

    if (externalBridgeSocketPath) {
      return validation;
    }

    stopWorkflowBridge();

    return await (await client()).getSettings();
  } catch (error) {
    rollbackDatabaseMove();
    savedSettings = previousSettings;
    writeSavedSettings(previousSettings);

    if (!externalBridgeSocketPath) {
      await startBridgeIfNeeded();
    }

    throw error;
  }
}

function settingsPath() {
  const devSettingsPath = process.env.MAC_OS_SETTINGS_PATH?.trim();

  if (devSettingsPath) {
    return devSettingsPath;
  }

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

function opErrorEnvelope(error: unknown) {
  const message = error instanceof Error ? error.message : String(error);
  const code =
    error &&
    typeof error === "object" &&
    "code" in error &&
    typeof (error as { code: unknown }).code === "string"
      ? (error as { code: string }).code
      : "op_failed";

  return { ok: false as const, code, message };
}

function openTerminalCommand(
  command: string,
): Promise<{ ok: true } | { ok: false; message: string }> {
  return new Promise((resolveResult) => {
    const script = [
      'tell application "Terminal"',
      "  activate",
      `  do script "${appleScriptString(command)}"`,
      "end tell",
    ].join("\n");
    const child = spawn("osascript", ["-e", script], { stdio: ["ignore", "ignore", "pipe"] });
    let stderr = "";

    child.stderr?.on("data", (chunk: Buffer) => {
      stderr += chunk.toString("utf8");
    });

    child.on("error", (error) => {
      resolveResult({ ok: false, message: error.message });
    });

    child.on("exit", (code) => {
      if (code === 0) {
        resolveResult({ ok: true });

        return;
      }

      resolveResult({
        ok: false,
        message: stderr.trim() || `osascript exited with code ${code ?? "unknown"}`,
      });
    });
  });
}

function appleScriptString(value: string): string {
  return value.replaceAll("\\", "\\\\").replaceAll('"', '\\"');
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
