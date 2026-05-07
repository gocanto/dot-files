import type { ViteDevServer } from "vite";
import { appName, portlessEnv, portlessPort, uiDir } from "./paths.js";
import { waitForViteReady } from "./http.js";
import { output, run } from "./processes.js";
import { delay } from "./timing.js";

export async function startPortlessRoute(server: ViteDevServer, port: number): Promise<string> {
  await ensurePortlessProxy();
  await run(
    "pnpm",
    ["exec", "portless", "alias", appName, String(port), "--force"],
    uiDir,
    portlessEnv,
  );

  return waitForPortless(server);
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

export async function removePortlessAlias(): Promise<void> {
  try {
    await run("pnpm", ["exec", "portless", "alias", "--remove", appName], uiDir, portlessEnv);
  } catch {}
}

async function waitForPortless(server: ViteDevServer, timeoutMs = 30000): Promise<string> {
  console.log("Waiting for portless route");
  const deadline = Date.now() + timeoutMs;

  for (;;) {
    try {
      const url = await output("pnpm", ["exec", "portless", "get", appName], uiDir, portlessEnv);
      const trimmed = url.trim();

      if (trimmed) {
        waitForViteReady(server);
        return trimmed;
      }
    } catch {}

    if (Date.now() >= deadline) {
      throw new Error(`portless route was not ready after ${timeoutMs}ms`);
    }

    await delay(500);
  }
}
