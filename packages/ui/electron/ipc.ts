import {
  type RuntimeSettings,
  type SettingsResponse,
  type RunWorkflowRequest,
  type WorkflowEvent,
} from "@dot-files/bridge";
import {
  BrowserWindow,
  dialog,
  ipcMain,
  type OpenDialogOptions,
  type SaveDialogOptions,
} from "electron";
import os from "node:os";
import {
  client,
  getBridgeSettings,
  hasExternalBridge,
  setBridgeSettings,
  startBridgeIfNeeded,
  stopWorkflowBridge,
} from "#electron/bridge.js";
import { moveWorkflowDatabase, writeSavedSettings } from "#electron/settings-store.js";
import { accountAvatarUrl, architectureLabel, osLabel } from "#electron/system-info.js";
import { openTerminalCommand } from "#electron/terminal.js";

type IpcDeps = {
  getMainWindow: () => BrowserWindow | null;
  openDevToolsPanel: (window: BrowserWindow) => void;
};

export function registerIpcHandlers(deps: IpcDeps) {
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
      'op signin && echo "\\n[Signed in. You can close this window and return to gus-macbook.]"',
    );
  });

  ipcMain.handle("op:install-dependencies", async () => {
    return openTerminalCommand(
      [
        'if ! command -v brew >/dev/null 2>&1; then echo "Homebrew is required. Run ./setup.sh first, then retry."; exit 1; fi',
        "brew install --cask 1password 1password-cli",
        'echo "\\n[1Password and 1Password CLI install finished. Open 1Password, enable CLI integration if needed, then return to gus-macbook.]"',
      ].join("; "),
    );
  });

  ipcMain.handle("settings:choose-directory", async (_event, defaultPath?: string) => {
    const options: OpenDialogOptions = {
      defaultPath,
      properties: ["openDirectory", "createDirectory"],
    };
    const mainWindow = deps.getMainWindow();
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
    const mainWindow = deps.getMainWindow();
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
    const mainWindow = deps.getMainWindow();
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
    const mainWindow = deps.getMainWindow();

    if (!mainWindow) {
      return;
    }

    deps.openDevToolsPanel(mainWindow);
  });

  ipcMain.handle(
    "workflow:run",
    async (event, request: RunWorkflowRequest, eventChannel: string) => {
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
    },
  );
}

async function saveSettings(settings: RuntimeSettings): Promise<SettingsResponse> {
  const validation = await (await client()).validateSettings({ settings });

  if (!validation.valid || !validation.settings) {
    return validation;
  }

  const current = await (await client()).getSettings();
  const previousSettings = getBridgeSettings();
  const nextSettings = validation.settings;
  let rollbackDatabaseMove = () => {};
  let bridgeStopped = false;

  try {
    if (!hasExternalBridge()) {
      stopWorkflowBridge();
      bridgeStopped = true;
    }

    rollbackDatabaseMove = moveWorkflowDatabase(
      current.settings?.workflowDbPath,
      nextSettings.workflowDbPath,
    );
    setBridgeSettings(nextSettings);
    writeSavedSettings(nextSettings);

    if (hasExternalBridge()) {
      return validation;
    }

    return await (await client()).getSettings();
  } catch (error) {
    rollbackDatabaseMove();
    setBridgeSettings(previousSettings);
    writeSavedSettings(previousSettings);

    if (bridgeStopped) {
      await startBridgeIfNeeded();
    }

    throw error;
  }
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
