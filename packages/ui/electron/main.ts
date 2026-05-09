import { app } from "electron";
import { appIcon } from "#electron/app-icon.js";
import { setBridgeSettings, startBridgeIfNeeded, stopWorkflowBridge } from "#electron/bridge.js";
import { registerIpcHandlers } from "#electron/ipc.js";
import { readSavedSettings } from "#electron/settings-store.js";
import {
  createWindow,
  focusMainWindow,
  getMainWindow,
  openDevToolsPanel,
} from "#electron/windows.js";

const singleInstanceLock = app.requestSingleInstanceLock();

if (!singleInstanceLock) {
  app.quit();
} else {
  app.on("second-instance", focusMainWindow);
}

app.whenReady().then(() => {
  try {
    const icon = appIcon();
    if (icon && process.platform === "darwin" && app.dock) {
      app.dock.setIcon(icon);
    }

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
