import { createApp } from "vue";
import App from "./App.vue";
import { initTheme } from "./composables/useTheme";
import { installBrowserFallback } from "./lib/browser-fallback";
import "./style.css";

installBrowserFallback();
initTheme();

createApp(App).mount("#app");
