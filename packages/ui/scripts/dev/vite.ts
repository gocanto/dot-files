import type { AddressInfo } from "node:net";
import { join } from "node:path";
import { createServer, type ViteDevServer } from "vite";
import { uiDir } from "./paths.js";
import { removePortlessAlias, startPortlessRoute } from "./portless.js";

let viteServer: ViteDevServer | undefined;

export async function startDevServer(): Promise<string> {
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

  return startPortlessRoute(port);
}

export async function stopDevServer(): Promise<void> {
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
