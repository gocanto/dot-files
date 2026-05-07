import type { AddressInfo } from "node:net";
import type { ViteDevServer } from "vite";

export function waitForViteReady(server: ViteDevServer): void {
  const httpServer = server.httpServer;

  if (!httpServer) {
    throw new Error("Vite dev server did not create an HTTP server");
  }

  if (!httpServer.listening) {
    throw new Error("Vite dev server is not listening");
  }

  const address = httpServer.address();

  if (!address || typeof address === "string") {
    throw new Error("Vite dev server did not expose a TCP port");
  }

  const { port } = address as AddressInfo;

  if (!Number.isInteger(port) || port <= 0) {
    throw new Error("Vite dev server exposed an invalid TCP port");
  }
}
