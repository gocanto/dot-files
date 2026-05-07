import { resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { defineConfig } from "vite";

const projectDir = fileURLToPath(new URL(".", import.meta.url));

export default defineConfig({
  cacheDir: resolve(projectDir, "../../storage/.cache/vite/ui-electron"),
  resolve: {
    alias: [
      {
        find: /^#electron\/(.+)\.js$/u,
        replacement: resolve(projectDir, "electron/$1.ts"),
      },
    ],
  },
  build: {
    emptyOutDir: false,
    lib: {
      entry: resolve(projectDir, "electron/preload.ts"),
      formats: ["cjs"],
      fileName: () => "preload.cjs",
    },
    outDir: "dist-electron",
    rollupOptions: {
      external: ["electron"],
    },
  },
});
