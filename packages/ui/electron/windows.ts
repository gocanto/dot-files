import { app, BrowserWindow } from "electron";
import { join } from "node:path";
import { appIcon } from "#electron/app-icon.js";
import { electronDir, repoRoot } from "#electron/paths.js";

const appWindowWidth = 2000;
const appWindowHeight = 1280;
const devToolsWindowWidth = 400;
const devToolsWindowHeight = 400;

let mainWindow: BrowserWindow | null = null;
let devToolsWindow: BrowserWindow | null = null;

export function getMainWindow() {
  return mainWindow && !mainWindow.isDestroyed() ? mainWindow : null;
}

export function createWindow() {
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
    icon: appIcon(),
    webPreferences: {
      preload: join(electronDir, "preload.cjs"),
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
  } else if (app.isPackaged) {
    void mainWindow.loadFile(join(electronDir, "..", "dist", "index.html"));
  } else {
    void mainWindow.loadFile(join(repoRoot, "packages", "ui", "dist", "index.html"));
  }
}

export function focusMainWindow() {
  if (!mainWindow || mainWindow.isDestroyed()) {
    return;
  }

  if (mainWindow.isMinimized()) {
    mainWindow.restore();
  }

  mainWindow.center();
  mainWindow.focus();
}

export function openDevToolsPanel(parentWindow: BrowserWindow) {
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
    icon: appIcon(),
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
