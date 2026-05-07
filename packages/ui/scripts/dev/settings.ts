import { readFile } from "node:fs/promises";
import { settingsPath } from "#scripts/dev/paths.js";

type SettingsKey =
  | "repoRoot"
  | "appsConfigPath"
  | "secretsConfigPath"
  | "generatedAppsPath"
  | "archiveRoot"
  | "workflowDbPath"
  | "opVault"
  | "opItem";
type Settings = Partial<Record<SettingsKey, string>>;

export async function readSettings(): Promise<Settings> {
  try {
    const parsed: unknown = JSON.parse(await readFile(settingsPath, "utf8"));
    return isSettings(parsed) ? parsed : {};
  } catch {
    return {};
  }
}

function isSettings(value: unknown): value is Settings {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

export function settingsArgs(settings: Settings): string[] {
  const pairs: [flag: string, value: string | undefined][] = [
    ["--repo-root", settings.repoRoot],
    ["--apps-config", settings.appsConfigPath],
    ["--secrets-config", settings.secretsConfigPath],
    ["--generated-apps", settings.generatedAppsPath],
    ["--archive-root", settings.archiveRoot],
    ["--workflow-db", settings.workflowDbPath],
    ["--op-vault", settings.opVault],
    ["--op-item", settings.opItem],
  ];

  return pairs.flatMap(([flag, value]) =>
    typeof value === "string" && value.trim() ? [flag, value] : [],
  );
}
