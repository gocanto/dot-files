import { contextBridge, ipcRenderer } from "electron";

interface RunRequest {
  workflowId: string;
  confirmationOptionId: string;
  enabledPhaseIds: string[];
}

interface RunEvent {
  runId: string;
  seq: number;
  type: string;
  phaseId?: string;
  phaseName?: string;
  status?: string;
  message?: string;
}

interface RuntimeSettings {
  repoRoot: string;
  appsConfigPath: string;
  secretsConfigPath: string;
  generatedAppsPath: string;
  archiveRoot: string;
  workflowDbPath: string;
  opVault: string;
  opItem: string;
}

interface TemplateFileContent {
  file: {
    path: string;
    relative: string;
    kind: string;
    size: number;
    modifiedAt?: string;
    exists: boolean;
  };
  content: string;
}

interface OpVault {
  id: string;
  name: string;
}

interface OpItem {
  id: string;
  title: string;
}

interface AppDiagnostic {
  id: string;
  level: "info" | "warning" | "error";
  source: string;
  message: string;
  details?: string;
  createdAt: string;
}

type OpVaultsResult =
  | { ok: true; vaults: OpVault[] }
  | { ok: false; code: string; message: string };
type OpItemsResult = { ok: true; items: OpItem[] } | { ok: false; code: string; message: string };
type OpSigninResult = { ok: true } | { ok: false; message: string };
type OpInstallResult = { ok: true } | { ok: false; message: string };

contextBridge.exposeInMainWorld("macOS", {
  macName: () => ipcRenderer.invoke("system:macName"),
  macHostname: () => ipcRenderer.invoke("system:macHostname"),
  macSystemInfo: () => ipcRenderer.invoke("system:macSystemInfo"),
  workflows: () => ipcRenderer.invoke("workflows:list"),
  runs: (limit?: number) => ipcRenderer.invoke("runs:list", limit ?? 50),
  runLog: (runId: string) => ipcRenderer.invoke("runs:log", runId),
  templateFiles: () => ipcRenderer.invoke("template-files:list"),
  readTemplateFile: (path: string): Promise<TemplateFileContent> =>
    ipcRenderer.invoke("template-files:read", path),
  saveTemplateFile: (path: string, content: string): Promise<TemplateFileContent> =>
    ipcRenderer.invoke("template-files:save", path, content),
  settings: () => ipcRenderer.invoke("settings:get"),
  validateSettings: (settings: RuntimeSettings) =>
    ipcRenderer.invoke("settings:validate", settings),
  saveSettings: (settings: RuntimeSettings) => ipcRenderer.invoke("settings:save", settings),
  getUserPreferences: () => ipcRenderer.invoke("preferences:get"),
  saveUserPreferences: (theme: string) => ipcRenderer.invoke("preferences:save", theme),
  chooseDirectory: (defaultPath?: string) =>
    ipcRenderer.invoke("settings:choose-directory", defaultPath),
  chooseFile: (defaultPath?: string) => ipcRenderer.invoke("settings:choose-file", defaultPath),
  chooseSaveFile: (defaultPath?: string) =>
    ipcRenderer.invoke("settings:choose-save-file", defaultPath),
  listOpVaults: (): Promise<OpVaultsResult> => ipcRenderer.invoke("op:list-vaults"),
  listOpItems: (vault: string): Promise<OpItemsResult> =>
    ipcRenderer.invoke("op:list-items", vault),
  signinOpCli: (): Promise<OpSigninResult> => ipcRenderer.invoke("op:signin"),
  installOpDependencies: (): Promise<OpInstallResult> =>
    ipcRenderer.invoke("op:install-dependencies"),
  openDevTools: () => ipcRenderer.invoke("system:openDevTools"),
  appDiagnostics: (): Promise<AppDiagnostic[]> => ipcRenderer.invoke("diagnostics:list"),
  reportRendererError: (message: string, details?: string) =>
    ipcRenderer.invoke("diagnostics:renderer-error", { message, details }),
  onAppDiagnostic: (onEvent: (event: AppDiagnostic) => void) => {
    const listener = (_: Electron.IpcRendererEvent, event: AppDiagnostic) => onEvent(event);

    ipcRenderer.on("diagnostics:event", listener);

    return () => ipcRenderer.removeListener("diagnostics:event", listener);
  },
  runWorkflow: (request: RunRequest, onEvent: (event: RunEvent) => void) => {
    const channel = `workflow:event:${crypto.randomUUID()}`;
    const listener = (_: Electron.IpcRendererEvent, event: RunEvent) => onEvent(event);

    ipcRenderer.on(channel, listener);

    return ipcRenderer
      .invoke("workflow:run", request, channel)
      .finally(() => ipcRenderer.removeListener(channel, listener));
  },
});
