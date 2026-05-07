import { watch } from "node:fs";
import { stat } from "node:fs/promises";
import { join } from "node:path";
import { macbookDir, repoRoot, settingsPath, storageDir, uiDir } from "#scripts/dev/paths.js";

export function installWatchers(onChange: (changed: string) => void): void {
  const paths = [
    join(uiDir, "electron"),
    join(uiDir, "vite.config.ts"),
    join(uiDir, "vite.electron.config.ts"),
    join(uiDir, "tsconfig.electron.json"),
    join(repoRoot, "packages", "bridge"),
    macbookDir,
    storageDir,
  ];

  for (const path of paths) {
    watchIfExists(path, onChange);
  }
}

function watchIfExists(path: string, onChange: (changed: string) => void): void {
  stat(path)
    .then((info) => {
      const watcher = watch(path, { recursive: info.isDirectory() }, (_event, filename) => {
        const changed = filename ? join(path, filename.toString()) : path;

        if (path === storageDir && changed !== settingsPath) {
          return;
        }

        if (shouldIgnoreChange(changed)) {
          return;
        }

        onChange(changed);
      });

      watcher.on("error", (error) => {
        console.error(`watch failed for ${path}:`, error);
      });
    })
    .catch(() => {});
}

function shouldIgnoreChange(path: string): boolean {
  return (
    path.includes("/node_modules/") || path.includes("/dist/") || path.includes("/dist-electron/")
  );
}
