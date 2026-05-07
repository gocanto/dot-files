import { mkdir } from "node:fs/promises";
import { backendSocketPath, backendStartSpec, waitForBackend } from "./backend.js";
import { compileElectron, electronStartSpec } from "./electron.js";
import { settingsPath, storageDir } from "./paths.js";
import { start, stopChildren } from "./processes.js";
import { stopDevServer, startDevServer } from "./vite.js";
import { installWatchers } from "./watchers.js";

let stopping = false;
let restarting: Promise<void> = Promise.resolve();
let restartTimer: NodeJS.Timeout | undefined;

await mkdir(storageDir, { recursive: true });
installSignalHandlers();
installWatchers((changed) => scheduleRestart(changed));
restarting = restart("initial start");
await restarting;

function installSignalHandlers(): void {
  const signals: NodeJS.Signals[] = ["SIGINT", "SIGTERM"];

  for (const signal of signals) {
    process.on(signal, async () => {
      if (stopping) {
        return;
      }

      stopping = true;
      clearTimeout(restartTimer);
      await stopDevServer();
      await stopChildren();
      process.exit(signal === "SIGINT" ? 130 : 143);
    });
  }
}

function scheduleRestart(reason: string): void {
  if (stopping) {
    return;
  }

  clearTimeout(restartTimer);
  restartTimer = setTimeout(() => {
    restarting = restarting
      .then(() => restart(reason))
      .catch((error: unknown) => {
        console.error(error);
        process.exitCode = 1;
      });
  }, 350);
}

async function restart(reason: string): Promise<void> {
  console.log(`\nRestarting dev stack: ${reason}`);
  await stopDevServer();
  await stopChildren();
  await compileElectron();

  const socketPath = await backendSocketPath();
  const devServerUrl = await startDevServer();
  console.log(`\nDev server ready: ${devServerUrl}`);

  start("backend", await backendStartSpec(socketPath), handleUnexpectedExit);
  await waitForBackend(socketPath);

  start(
    "electron",
    electronStartSpec(socketPath, settingsPath, devServerUrl),
    handleUnexpectedExit,
  );
}

function handleUnexpectedExit(
  name: string,
  code: number | null,
  signal: NodeJS.Signals | null,
): void {
  if (stopping) {
    return;
  }

  console.error(`${name} exited with ${code ?? signal ?? "unknown status"}`);
  stopping = true;
  void stopDevServer()
    .then(() => stopChildren())
    .then(() => process.exit(code ?? 1));
}
