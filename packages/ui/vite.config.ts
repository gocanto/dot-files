import tailwindcss from "@tailwindcss/vite";
import vue from "@vitejs/plugin-vue";
import { resolve } from "node:path";
import { defineConfig } from "vite";

export default defineConfig({
  base: "./",
  plugins: [vue(), tailwindcss()],
  cacheDir: resolve(__dirname, "../../storage/.cache/vite/ui"),
  build: {
    rolldownOptions: {
      output: {
        codeSplitting: {
          minSize: 20_000,
          maxSize: 480_000,
          groups: [
            {
              name: "monaco",
              test: /node_modules[\\/](monaco-editor|monaco-yaml)[\\/]/,
              priority: 40,
            },
            {
              name: "shiki",
              test: /node_modules[\\/](@shikijs|shiki)[\\/]/,
              priority: 30,
            },
            {
              name: "terminal",
              test: /node_modules[\\/]ghostty-web[\\/]/,
              priority: 25,
            },
            {
              name: "vue-ui",
              test: /node_modules[\\/](vue|@vueuse|reka-ui|lucide-vue-next|@radix-icons|class-variance-authority|clsx|tailwind-merge)[\\/]/,
              priority: 20,
            },
            {
              name: "vendor",
              test: /node_modules[\\/]/,
              priority: 10,
            },
          ],
        },
      },
    },
  },
  resolve: {
    alias: {
      "@entry": resolve(__dirname, "./src"),
      "@app": resolve(__dirname, "./src/components/app"),
      "@components": resolve(__dirname, "./src/components"),
      "@ui": resolve(__dirname, "./src/components/ui"),
      "@composables": resolve(__dirname, "./src/composables"),
      "@lib": resolve(__dirname, "./src/lib"),
      "@api": resolve(__dirname, "./src/types/api.ts"),
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
