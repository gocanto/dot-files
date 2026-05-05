import tailwindcss from "@tailwindcss/vite";
import vue from "@vitejs/plugin-vue";
import { resolve } from "node:path";
import { defineConfig } from "vite";

export default defineConfig({
  plugins: [vue(), tailwindcss()],
  cacheDir: resolve(__dirname, "../../storage/.cache/vite/ui"),
  resolve: {
    alias: {
      "@": resolve(__dirname, "./src"),
    },
  },
  server: {
    host: "127.0.0.1",
    allowedHosts: ["dot-files-ui.localhost"],
    hmr: {
      protocol: "wss",
      host: "dot-files-ui.localhost",
      clientPort: 1355,
    },
  },
  test: {
    environment: "happy-dom",
    globals: true,
  },
});
