import { spawn, type ChildProcess } from "node:child_process";
import { constants as fsConstants, watch } from "node:fs";
import { access, mkdir, readFile, rm, stat } from "node:fs/promises";
import { request as httpRequest, type IncomingMessage } from "node:http";
import { request as httpsRequest } from "node:https";
import type { AddressInfo } from "node:net";
import os from "node:os";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { createServer, type ViteDevServer } from "vite";

const uiDir = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const repoRoot = resolve(uiDir, "..", "..");
const macbookDir = join(repoRoot, "macbook");
const storageDir = join(repoRoot, "storage", "dev");
const settingsPath = join(storageDir, "ui-settings.json");
const appName = "dot-files-ui";
const portlessEnv = { PORTLESS_TLD: "localhost" };
const portlessPort = "1355";

type StartSpec = readonly [command: string, args: string[], cwd: string, env?: NodeJS.ProcessEnv];
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

const children = new Map<string, ChildProcess>();
let viteServer: ViteDevServer | undefined;
let stopping = false;
let stoppingChildren = false;
let restarting: Promise<void> = Promise.resolve();
let restartTimer: NodeJS.Timeout | undefined;

await mkdir(storageDir, { recursive: true });
installSignalHandlers();
installWatchers();
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

function installWatchers(): void {
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
    watchIfExists(path, (changed) => scheduleRestart(changed));
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

  const backendSocketPath = join(os.tmpdir(), `mac-os-dev-${process.pid}-${Date.now()}.sock`);
  await rm(backendSocketPath, { force: true });

  const devServerUrl = await startDevServer();
  console.log(`\nDev server ready: ${devServerUrl}`);

  start("backend", [
    "go",
    [
      "run",
      "./cmd",
      "serve-http",
      "--socket",
      backendSocketPath,
      ...settingsArgs(await readSettings()),
    ],
    macbookDir,
  ]);
  await waitForBackend(backendSocketPath);

  start("electron", [
    "node",
    ["node_modules/electron/cli.js", "."],
    uiDir,
    {
      MAC_OS_BRIDGE_SOCKET: backendSocketPath,
      MAC_OS_SETTINGS_PATH: settingsPath,
      VITE_DEV_SERVER_URL: devServerUrl,
    },
  ]);
}

async function compileElectron(): Promise<void> {
  console.log("Compiling Electron main/preload");
  await run("pnpm", ["exec", "tsc", "-p", "tsconfig.electron.json", "--pretty", "false"], uiDir);
  await run("pnpm", ["exec", "vite", "build", "--config", "vite.electron.config.ts"], uiDir);
}

async function startDevServer(): Promise<string> {
  console.log("Starting Vite dev server");

  viteServer = await createServer({
    configFile: join(uiDir, "vite.config.ts"),
    root: uiDir,
    server: {
      host: "127.0.0.1",
      port: devServerPortPreference(),
      strictPort: false,
    },
  });

  await viteServer.listen();

  const port = devServerPort(viteServer);
  await ensurePortlessProxy();
  await run(
    "pnpm",
    ["exec", "portless", "alias", appName, String(port), "--force"],
    uiDir,
    portlessEnv,
  );

  return waitForPortless(`http://127.0.0.1:${port}`);
}

async function stopDevServer(): Promise<void> {
  if (!viteServer) {
    return;
  }

  const server = viteServer;
  viteServer = undefined;
  await Promise.all([server.close(), removePortlessAlias()]);
}

function devServerPort(server: ViteDevServer): number {
  const address = server.httpServer?.address();

  if (!address || typeof address === "string") {
    throw new Error("Vite dev server did not expose a TCP port");
  }

  return (address as AddressInfo).port;
}

function devServerPortPreference(): number | undefined {
  const port = Number.parseInt(process.env.PORT ?? "", 10);
  return Number.isInteger(port) && port > 0 ? port : undefined;
}

async function ensurePortlessProxy(): Promise<void> {
  try {
    await output(
      "pnpm",
      [
        "exec",
        "portless",
        "proxy",
        "start",
        "--port",
        portlessPort,
        "--https",
        "--tld",
        "localhost",
      ],
      uiDir,
      portlessEnv,
    );
  } catch (error) {
    if (!(error instanceof Error) || !error.message.includes("different config")) {
      throw error;
    }

    await output(
      "pnpm",
      ["exec", "portless", "proxy", "stop", "--port", portlessPort],
      uiDir,
      portlessEnv,
    );
    await output(
      "pnpm",
      [
        "exec",
        "portless",
        "proxy",
        "start",
        "--port",
        portlessPort,
        "--https",
        "--tld",
        "localhost",
      ],
      uiDir,
      portlessEnv,
    );
  }
}

async function removePortlessAlias(): Promise<void> {
  try {
    await run("pnpm", ["exec", "portless", "alias", "--remove", appName], uiDir, portlessEnv);
  } catch {}
}

function start(name: string, [command, args, cwd, env = {}]: StartSpec): void {
  const child = spawn(command, args, {
    cwd,
    detached: true,
    env: { ...process.env, ...env },
    stdio: "inherit",
  });

  children.set(name, child);

  child.on("exit", (code, signal) => {
    children.delete(name);

    if (!stopping && !stoppingChildren && code !== 0) {
      console.error(`${name} exited with ${code ?? signal ?? "unknown status"}`);
      stopping = true;
      void stopDevServer()
        .then(() => stopChildren())
        .then(() => process.exit(code ?? 1));
    }
  });
}

function run(
  command: string,
  args: string[],
  cwd: string,
  env: NodeJS.ProcessEnv = {},
): Promise<void> {
  return new Promise((resolveRun, rejectRun) => {
    const child = spawn(command, args, { cwd, env: { ...process.env, ...env }, stdio: "inherit" });

    child.on("exit", (code, signal) => {
      if (code === 0) {
        resolveRun();
        return;
      }

      rejectRun(
        new Error(`${command} ${args.join(" ")} exited with ${code ?? signal ?? "unknown status"}`),
      );
    });
  });
}

async function stopChildren(): Promise<void> {
  stoppingChildren = true;
  const running = [...children.values()];
  children.clear();

  for (const child of running) {
    if (child.pid) {
      try {
        process.kill(-child.pid, "SIGTERM");
      } catch {}
    }
  }

  await delay(500);

  for (const child of running) {
    if (child.pid) {
      try {
        process.kill(-child.pid, "SIGKILL");
      } catch {}
    }
  }

  stoppingChildren = false;
}

async function waitForPortless(readinessUrl: string): Promise<string> {
  console.log("Waiting for portless route");

  for (;;) {
    try {
      const url = await output("pnpm", ["exec", "portless", "get", appName], uiDir, portlessEnv);
      const trimmed = url.trim();

      if (trimmed) {
        await requestOk(readinessUrl);
        return trimmed;
      }
    } catch {}

    await delay(500);
  }
}

async function waitForBackend(socketPath: string): Promise<void> {
  for (;;) {
    try {
      await access(socketPath, fsConstants.F_OK);
      await backendHealthz(socketPath);
      return;
    } catch {
      await delay(100);
    }
  }
}

function backendHealthz(socketPath: string): Promise<void> {
  return new Promise((resolveHealthz, rejectHealthz) => {
    const req = httpRequest({ socketPath, path: "/v1/healthz", method: "GET" }, (res) => {
      res.resume();
      res.on("end", () => {
        if (res.statusCode === 200) {
          resolveHealthz();
          return;
        }

        rejectHealthz(new Error(`backend healthz failed with ${res.statusCode}`));
      });
    });

    req.on("error", rejectHealthz);
    req.end();
  });
}

function output(
  command: string,
  args: string[],
  cwd: string,
  env: NodeJS.ProcessEnv = {},
): Promise<string> {
  return new Promise((resolveOutput, rejectOutput) => {
    const child = spawn(command, args, {
      cwd,
      env: { ...process.env, ...env },
      stdio: ["ignore", "pipe", "pipe"],
    });
    let stdout = "";
    let stderr = "";

    child.stdout.on("data", (chunk: Buffer) => {
      stdout += chunk.toString("utf8");
    });

    child.stderr.on("data", (chunk: Buffer) => {
      stderr += chunk.toString("utf8");
    });

    child.on("exit", (code) => {
      if (code === 0) {
        resolveOutput(stdout);
        return;
      }

      rejectOutput(new Error(stderr || `${command} ${args.join(" ")} exited with ${code}`));
    });
  });
}

function requestOk(url: string): Promise<void> {
  return new Promise((resolveRequest, rejectRequest) => {
    const onResponse = (res: IncomingMessage) => {
      res.resume();
      res.on("end", () => {
        if (res.statusCode && res.statusCode >= 200 && res.statusCode < 500) {
          resolveRequest();
          return;
        }

        rejectRequest(new Error(`unexpected status ${res.statusCode}`));
      });
    };

    const req = url.startsWith("https:")
      ? httpsRequest(url, { rejectUnauthorized: false }, onResponse)
      : httpRequest(url, onResponse);

    req.setTimeout(2_000, () => {
      req.destroy(new Error(`timed out waiting for ${url}`));
    });
    req.on("error", rejectRequest);
    req.end();
  });
}

async function readSettings(): Promise<Settings> {
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

function settingsArgs(settings: Settings): string[] {
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

function delay(ms: number): Promise<void> {
  return new Promise((resolveDelay) => setTimeout(resolveDelay, ms));
}
