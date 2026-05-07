import { constants as fsConstants } from "node:fs";
import { access, rm } from "node:fs/promises";
import { request as httpRequest } from "node:http";
import os from "node:os";
import { join } from "node:path";
import { macbookDir } from "./paths.js";
import type { StartSpec } from "./processes.js";
import { readSettings, settingsArgs } from "./settings.js";
import { delay } from "./timing.js";

export async function backendSocketPath(): Promise<string> {
  const path = join(os.tmpdir(), `mac-os-dev-${process.pid}-${Date.now()}.sock`);
  await rm(path, { force: true });

  return path;
}

export async function backendStartSpec(socketPath: string): Promise<StartSpec> {
  return [
    "go",
    ["run", "./cmd", "serve-http", "--socket", socketPath, ...settingsArgs(await readSettings())],
    macbookDir,
  ];
}

export async function waitForBackend(socketPath: string, timeoutMs = 30000): Promise<void> {
  const deadline = Date.now() + timeoutMs;

  for (;;) {
    try {
      await access(socketPath, fsConstants.F_OK);
      await backendHealthz(socketPath);
      return;
    } catch {
      if (Date.now() >= deadline) {
        throw new Error(`backend was not ready after ${timeoutMs}ms at ${socketPath}`);
      }

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
