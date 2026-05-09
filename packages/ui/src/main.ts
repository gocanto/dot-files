import { createApp } from "vue";
import App from "@entry/App.vue";
import { initTheme } from "@composables/useTheme";
import { installBrowserFallback } from "@lib/browser-fallback";
import "./style.css";

installBrowserFallback();
initTheme();

window.addEventListener("error", (event) => {
  void window.macOS?.reportRendererError?.(
    event.message || "Unhandled renderer error",
    [event.filename, event.lineno ? `line ${event.lineno}` : "", event.error?.stack]
      .filter(Boolean)
      .join("\n"),
  );
});

window.addEventListener("unhandledrejection", (event) => {
  const reason = event.reason;
  const message = reason instanceof Error ? reason.message : String(reason);
  const details = reason instanceof Error ? reason.stack : undefined;

  void window.macOS?.reportRendererError?.(message || "Unhandled renderer rejection", details);
});

createApp(App).mount("#app");
