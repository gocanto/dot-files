import { createApp } from "vue";
import App from "./App.vue";
import { initTheme } from "./composables/useTheme";
import { installBrowserFallback } from "./lib/browser-fallback";
import "@fontsource/inter/400.css";
import "@fontsource/inter/500.css";
import "@fontsource/inter/600.css";
import "@fontsource-variable/jetbrains-mono";
import "./style.css";

installBrowserFallback();
initTheme();

createApp(App).mount("#app");
