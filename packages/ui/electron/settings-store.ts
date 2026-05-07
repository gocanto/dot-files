import type { RuntimeSettings } from "@dot-files/bridge";
import { app } from "electron";
import {
  copyFileSync,
  existsSync,
  mkdirSync,
  readFileSync,
  statSync,
  unlinkSync,
  writeFileSync,
} from "node:fs";
import { dirname, join } from "node:path";

export function settingsPath() {
  const devSettingsPath = process.env.DOT_FILES_SETTINGS_PATH?.trim();

  if (devSettingsPath) {
    return devSettingsPath;
  }

  return join(app.getPath("userData"), "settings.json");
}

export function readSavedSettings(): Partial<RuntimeSettings> {
  try {
    const data = JSON.parse(readFileSync(settingsPath(), "utf8")) as Partial<RuntimeSettings>;

    return cleanSettings(data);
  } catch {
    return {};
  }
}

export function writeSavedSettings(settings: Partial<RuntimeSettings>) {
  mkdirSync(dirname(settingsPath()), { recursive: true });
  writeFileSync(settingsPath(), JSON.stringify(cleanSettings(settings), null, 2) + "\n", "utf8");
}

export function cleanSettings(settings: Partial<RuntimeSettings>): Partial<RuntimeSettings> {
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

export function moveWorkflowDatabase(fromPath?: string, toPath?: string) {
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
