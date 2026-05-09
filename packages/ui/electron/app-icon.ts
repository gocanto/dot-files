import { app, nativeImage } from "electron";
import { existsSync } from "node:fs";
import { join } from "node:path";
import { repoRoot } from "#electron/paths.js";

export function appIconPath() {
  if (app.isPackaged) {
    return join(process.resourcesPath, "icon.icns");
  }

  return join(repoRoot, "packages", "ui", "build", "icon.icns");
}

export function appIcon() {
  const iconPath = appIconPath();

  if (!existsSync(iconPath)) {
    return undefined;
  }

  const icon = nativeImage.createFromPath(iconPath);
  return icon.isEmpty() ? undefined : icon;
}
