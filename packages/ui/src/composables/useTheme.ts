import { ref } from "vue";

export type Theme = "light" | "dark";

const STORAGE_KEY = "ui:theme";

function isTheme(value: unknown): value is Theme {
  return value === "light" || value === "dark";
}

function readCachedTheme(): Theme | null {
  try {
    const stored = globalThis.localStorage?.getItem(STORAGE_KEY);

    return isTheme(stored) ? stored : null;
  } catch {
    return null;
  }
}

const theme = ref<Theme>(readCachedTheme() ?? "light");

function applyTheme(t: Theme): void {
  const root = globalThis.document?.documentElement;

  if (!root) {
    return;
  }

  root.classList.toggle("dark", t === "dark");
  root.dataset.colorMode = t;
  root.dataset.lightTheme = "light";
  root.dataset.darkTheme = "dark";
}

function writeCache(t: Theme): void {
  try {
    globalThis.localStorage?.setItem(STORAGE_KEY, t);
  } catch {
    // ignore
  }
}

async function persistTheme(t: Theme): Promise<void> {
  if (!window.macOS?.saveUserPreferences) {
    return;
  }

  try {
    await window.macOS.saveUserPreferences(t);
  } catch (error) {
    console.error("Failed to persist theme preference", error);
  }
}

export async function setTheme(t: Theme): Promise<void> {
  theme.value = t;
  applyTheme(t);
  writeCache(t);
  await persistTheme(t);
}

export async function toggleTheme(): Promise<void> {
  await setTheme(theme.value === "dark" ? "light" : "dark");
}

export function initTheme(): void {
  applyTheme(theme.value);
}

export async function loadThemeFromBackend(): Promise<void> {
  if (!window.macOS?.getUserPreferences) {
    return;
  }

  try {
    const prefs = await window.macOS.getUserPreferences();

    if (isTheme(prefs.theme) && prefs.theme !== theme.value) {
      theme.value = prefs.theme;
      applyTheme(prefs.theme);
      writeCache(prefs.theme);
    }
  } catch (error) {
    console.error("Failed to load theme preference", error);
  }
}

export function useTheme() {
  return { theme, setTheme, toggleTheme };
}
