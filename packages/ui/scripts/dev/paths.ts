import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

export const uiDir = resolve(dirname(fileURLToPath(import.meta.url)), "..", "..");
export const repoRoot = resolve(uiDir, "..", "..");
export const macbookDir = join(repoRoot, "packages", "macbook");
export const storageDir = join(repoRoot, "storage", "dev");
export const settingsPath = join(storageDir, "ui-settings.json");
export const appName = "dot-files-ui";
export const portlessEnv = { PORTLESS_TLD: "localhost" };
export const portlessPort = "1355";
