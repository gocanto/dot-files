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

contextBridge.exposeInMainWorld("macOS", {
  workflows: () => ipcRenderer.invoke("workflows:list"),
  runs: (limit?: number) => ipcRenderer.invoke("runs:list", limit ?? 50),
  runLog: (runId: string) => ipcRenderer.invoke("runs:log", runId),
  settings: () => ipcRenderer.invoke("settings:get"),
  validateSettings: (settings: RuntimeSettings) => ipcRenderer.invoke("settings:validate", settings),
  saveSettings: (settings: RuntimeSettings) => ipcRenderer.invoke("settings:save", settings),
  getUserPreferences: () => ipcRenderer.invoke("preferences:get"),
  saveUserPreferences: (theme: string) => ipcRenderer.invoke("preferences:save", theme),
  chooseDirectory: (defaultPath?: string) => ipcRenderer.invoke("settings:choose-directory", defaultPath),
  chooseFile: (defaultPath?: string) => ipcRenderer.invoke("settings:choose-file", defaultPath),
  chooseSaveFile: (defaultPath?: string) => ipcRenderer.invoke("settings:choose-save-file", defaultPath),
  runWorkflow: (request: RunRequest, onEvent: (event: RunEvent) => void) => {
    const channel = `workflow:event:${crypto.randomUUID()}`;
    const listener = (_: Electron.IpcRendererEvent, event: RunEvent) => onEvent(event);

    ipcRenderer.on(channel, listener);

    return ipcRenderer
      .invoke("workflow:run", request, channel)
      .finally(() => ipcRenderer.removeListener(channel, listener));
  },
});
