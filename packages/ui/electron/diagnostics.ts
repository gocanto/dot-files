import { app, BrowserWindow, ipcMain } from "electron";
import { randomUUID } from "node:crypto";
import { stopWorkflowBridge } from "#electron/bridge.js";

export type AppDiagnosticLevel = "info" | "warning" | "error";

export interface AppDiagnostic {
  id: string;
  level: AppDiagnosticLevel;
  source: string;
  message: string;
  details?: string;
  createdAt: string;
}

const diagnostics: AppDiagnostic[] = [];
const maxDiagnostics = 200;

export function recordDiagnostic(input: {
  level: AppDiagnosticLevel;
  source: string;
  message: string;
  details?: string;
}) {
  const diagnostic: AppDiagnostic = {
    id: `${Date.now()}-${randomUUID()}`,
    createdAt: new Date().toISOString(),
    ...input,
  };

  diagnostics.push(diagnostic);

  if (diagnostics.length > maxDiagnostics) {
    diagnostics.splice(0, diagnostics.length - maxDiagnostics);
  }

  for (const window of BrowserWindow.getAllWindows()) {
    window.webContents.send("diagnostics:event", diagnostic);
  }

  return diagnostic;
}

export function registerDiagnosticsIpc() {
  ipcMain.handle("diagnostics:list", () => diagnostics);
  ipcMain.handle(
    "diagnostics:renderer-error",
    (_event, payload: { message: string; details?: string }) => {
      recordDiagnostic({
        level: "error",
        source: "Renderer",
        message: payload.message,
        details: payload.details,
      });
    },
  );
}

export function attachWindowDiagnostics(window: BrowserWindow) {
  window.webContents.on("console-message", (_event, level, message, line, sourceId) => {
    if (level < 2) {
      return;
    }

    recordDiagnostic({
      level: level >= 3 ? "error" : "warning",
      source: "Renderer console",
      message,
      details: [sourceId, line ? `line ${line}` : ""].filter(Boolean).join(" "),
    });
  });

  window.webContents.on(
    "did-fail-load",
    (_event, errorCode, errorDescription, validatedURL, isMainFrame) => {
      recordDiagnostic({
        level: "error",
        source: "Window load",
        message: errorDescription,
        details: `${isMainFrame ? "main frame" : "subframe"} ${errorCode} ${validatedURL}`,
      });
    },
  );

  window.webContents.on("render-process-gone", (_event, details) => {
    recordDiagnostic({
      level: "error",
      source: "Renderer process",
      message: `Renderer process ${details.reason}`,
      details: `exitCode=${details.exitCode}`,
    });
  });

  window.on("unresponsive", () => {
    recordDiagnostic({
      level: "warning",
      source: "Window",
      message: "The app window became unresponsive.",
    });
  });
}

function shutdownAfterFatal(error: Error) {
  recordDiagnostic({
    level: "error",
    source: "Main process",
    message: error.message,
    details: error.stack,
  });

  try {
    stopWorkflowBridge();
  } catch (cleanupError) {
    console.error("Cleanup failed during fatal shutdown", cleanupError);
  }

  app.exit(1);
}

process.on("uncaughtException", (error) => {
  shutdownAfterFatal(error);
});

process.on("unhandledRejection", (reason) => {
  const error = reason instanceof Error ? reason : new Error(String(reason));
  shutdownAfterFatal(error);
});
