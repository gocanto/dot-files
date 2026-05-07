import { spawn, type ChildProcess } from "node:child_process";
import { delay } from "./timing.js";

export type StartSpec = readonly [
  command: string,
  args: string[],
  cwd: string,
  env?: NodeJS.ProcessEnv,
];

const children = new Map<string, ChildProcess>();
let stoppingChildren = false;

export function start(
  name: string,
  [command, args, cwd, env = {}]: StartSpec,
  onUnexpectedExit: (name: string, code: number | null, signal: NodeJS.Signals | null) => void,
): void {
  const child = spawn(command, args, {
    cwd,
    detached: true,
    env: { ...process.env, ...env },
    stdio: "inherit",
  });

  children.set(name, child);

  child.on("exit", (code, signal) => {
    children.delete(name);

    if (!stoppingChildren && code !== 0) {
      onUnexpectedExit(name, code, signal);
    }
  });
}

export function run(
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

export function output(
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

export async function stopChildren(): Promise<void> {
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
