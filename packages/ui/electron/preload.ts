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

contextBridge.exposeInMainWorld("macOS", {
  workflows: () => ipcRenderer.invoke("workflows:list"),
  runs: (limit?: number) => ipcRenderer.invoke("runs:list", limit ?? 50),
  runLog: (runId: string) => ipcRenderer.invoke("runs:log", runId),
  runWorkflow: (request: RunRequest, onEvent: (event: RunEvent) => void) => {
    const channel = `workflow:event:${crypto.randomUUID()}`;
    const listener = (_: Electron.IpcRendererEvent, event: RunEvent) => onEvent(event);

    ipcRenderer.on(channel, listener);

    return ipcRenderer
      .invoke("workflow:run", request, channel)
      .finally(() => ipcRenderer.removeListener(channel, listener));
  },
});
