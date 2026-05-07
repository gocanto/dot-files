import { app } from "electron";
import { setBridgeSettings, startBridgeIfNeeded, stopWorkflowBridge } from "./bridge.js";
import { registerIpcHandlers } from "./ipc.js";
import { readSavedSettings } from "./settings-store.js";
import { createWindow, focusMainWindow, getMainWindow, openDevToolsPanel } from "./windows.js";

const singleInstanceLock = app.requestSingleInstanceLock();

if (!singleInstanceLock) {
  app.quit();
} else {
  app.on("second-instance", focusMainWindow);
}

app.whenReady().then(() => {
  try {
    setBridgeSettings(readSavedSettings());
    registerIpcHandlers({ getMainWindow, openDevToolsPanel });
    createWindow();
  } catch (error) {
    console.error(error);
    app.quit();
    return;
  }

  void startBridgeIfNeeded().catch((error: unknown) => {
    console.error("Failed to start api HTTP bridge", error);
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
