import { uiDir } from "#scripts/dev/paths.js";
import { run, type StartSpec } from "#scripts/dev/processes.js";

export async function compileElectron(): Promise<void> {
  console.log("Compiling Electron main/preload");
  await run("pnpm", ["exec", "tsc", "-p", "tsconfig.electron.json", "--pretty", "false"], uiDir);
  await run("pnpm", ["exec", "vite", "build", "--config", "vite.electron.config.ts"], uiDir);
}

export function electronStartSpec(
  backendSocketPath: string,
  settingsPath: string,
  devServerUrl: string,
): StartSpec {
  return [
    "node",
    ["node_modules/electron/cli.js", "."],
    uiDir,
    {
      API_BRIDGE_SOCKET: backendSocketPath,
      DOT_FILES_SETTINGS_PATH: settingsPath,
      VITE_DEV_SERVER_URL: devServerUrl,
    },
  ];
}
