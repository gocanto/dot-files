import { constants as fsConstants } from "node:fs";
import { access, mkdir, readFile, rm, stat } from "node:fs/promises";
import { request as httpRequest } from "node:http";
import { request as httpsRequest } from "node:https";
import os from "node:os";
import { dirname, join, resolve } from "node:path";
import { spawn } from "node:child_process";
import { fileURLToPath } from "node:url";
import { watch } from "node:fs";

const uiDir = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const repoRoot = resolve(uiDir, "..", "..");
const macbookDir = join(repoRoot, "macbook");
const storageDir = join(repoRoot, "storage", "dev");
const settingsPath = join(storageDir, "ui-settings.json");
const appName = "dot-files-ui";

const children = new Map();
let stopping = false;
let stoppingChildren = false;
let restarting = Promise.resolve();
let restartTimer;

await mkdir(storageDir, { recursive: true });
installSignalHandlers();
installWatchers();
restarting = restart("initial start");
await restarting;

function installSignalHandlers() {
  for (const signal of ["SIGINT", "SIGTERM"]) {
    process.on(signal, async () => {
      if (stopping) {
        return;
      }

      stopping = true;
      clearTimeout(restartTimer);
      await stopChildren();
      process.exit(signal === "SIGINT" ? 130 : 143);
    });
  }
}

function installWatchers() {
  const paths = [
    join(uiDir, "src"),
    join(uiDir, "electron"),
    join(uiDir, "index.html"),
    join(uiDir, "vite.config.ts"),
    join(uiDir, "vite.electron.config.ts"),
    join(uiDir, "tsconfig.electron.json"),
    join(repoRoot, "packages", "bridge"),
    macbookDir,
    storageDir,
  ];

  for (const path of paths) {
    watchIfExists(path, () => scheduleRestart(path));
  }
}

function watchIfExists(path, onChange) {
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

function shouldIgnoreChange(path) {
  return path.includes("/node_modules/") || path.includes("/dist/") || path.includes("/dist-electron/");
}

function scheduleRestart(reason) {
  if (stopping) {
    return;
  }

  clearTimeout(restartTimer);
  restartTimer = setTimeout(() => {
    restarting = restarting.then(() => restart(reason)).catch((error) => {
      console.error(error);
      process.exitCode = 1;
    });
  }, 350);
}

async function restart(reason) {
  console.log(`\nRestarting dev stack: ${reason}`);
  await stopChildren();
  await compileElectron();

  const backendSocketPath = join(os.tmpdir(), `mac-os-dev-${process.pid}-${Date.now()}.sock`);
  await rm(backendSocketPath, { force: true });

  start("vite", ["pnpm", ["exec", "portless", "--name", appName, "--force", "--", "vite"], uiDir]);
  const devServerUrl = await waitForPortless();

  start("backend", ["go", ["run", "./cmd", "serve-http", "--socket", backendSocketPath, ...settingsArgs(await readSettings())], macbookDir]);
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

async function compileElectron() {
  console.log("Compiling Electron main/preload");
  await run("pnpm", ["exec", "tsc", "-p", "tsconfig.electron.json", "--pretty", "false"], uiDir);
  await run("pnpm", ["exec", "vite", "build", "--config", "vite.electron.config.ts"], uiDir);
}

function start(name, [command, args, cwd, env = {}]) {
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
      void stopChildren().then(() => process.exit(code ?? 1));
    }
  });
}

function run(command, args, cwd) {
  return new Promise((resolveRun, rejectRun) => {
    const child = spawn(command, args, { cwd, stdio: "inherit" });

    child.on("exit", (code, signal) => {
      if (code === 0) {
        resolveRun();
        return;
      }

      rejectRun(new Error(`${command} ${args.join(" ")} exited with ${code ?? signal ?? "unknown status"}`));
    });
  });
}

async function stopChildren() {
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

  await new Promise((resolveStop) => setTimeout(resolveStop, 500));

  for (const child of running) {
    if (child.pid) {
      try {
        process.kill(-child.pid, "SIGKILL");
      } catch {}
    }
  }

  stoppingChildren = false;
}

async function waitForPortless() {
  console.log("Waiting for portless route");

  for (;;) {
    try {
      const url = await output("pnpm", ["exec", "portless", "get", appName], uiDir);
      const trimmed = url.trim();

      if (trimmed) {
        await requestOk(trimmed);
        return trimmed;
      }
    } catch {}

    await delay(500);
  }
}

async function waitForBackend(socketPath) {
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

function backendHealthz(socketPath) {
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

function output(command, args, cwd) {
  return new Promise((resolveOutput, rejectOutput) => {
    const child = spawn(command, args, { cwd, stdio: ["ignore", "pipe", "pipe"] });
    let stdout = "";
    let stderr = "";

    child.stdout.on("data", (chunk) => {
      stdout += chunk.toString("utf8");
    });

    child.stderr.on("data", (chunk) => {
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

function requestOk(url) {
  return new Promise((resolveRequest, rejectRequest) => {
    const request = url.startsWith("https:") ? httpsRequest : httpRequest;
    const options = url.startsWith("https:") ? { rejectUnauthorized: false } : {};
    const req = request(url, options, (res) => {
      res.resume();
      res.on("end", () => {
        if (res.statusCode && res.statusCode >= 200 && res.statusCode < 500) {
          resolveRequest();
          return;
        }

        rejectRequest(new Error(`unexpected status ${res.statusCode}`));
      });
    });

    req.on("error", rejectRequest);
    req.end();
  });
}

async function readSettings() {
  try {
    return JSON.parse(await readFile(settingsPath, "utf8"));
  } catch {
    return {};
  }
}

function settingsArgs(settings) {
  const pairs = [
    ["--repo-root", settings.repoRoot],
    ["--apps-config", settings.appsConfigPath],
    ["--secrets-config", settings.secretsConfigPath],
    ["--generated-apps", settings.generatedAppsPath],
    ["--archive-root", settings.archiveRoot],
    ["--workflow-db", settings.workflowDbPath],
    ["--op-vault", settings.opVault],
    ["--op-item", settings.opItem],
  ];

  return pairs.flatMap(([flag, value]) => (typeof value === "string" && value.trim() ? [flag, value] : []));
}

function delay(ms) {
  return new Promise((resolveDelay) => setTimeout(resolveDelay, ms));
}
