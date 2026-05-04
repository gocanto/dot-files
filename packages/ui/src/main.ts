import { createApp } from "vue";
import App from "./App.vue";
import { installBrowserFallback } from "./lib/browser-fallback";
import "./style.css";

installBrowserFallback();

createApp(App).mount("#app");
