import { mkdirSync, mkdtempSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

let tempRoot = "";
let userData = "";
let home = "";
let settingsFile = "";

vi.mock("electron", () => ({
  app: {
    getPath: vi.fn((key: string) => {
      if (key === "userData") return userData;
      if (key === "home") return home;
      throw new Error(`unexpected getPath key: ${key}`);
    }),
  },
}));

async function loadModule() {
  vi.resetModules();
  return await import("#electron-src/settings-store");
}

beforeEach(() => {
  tempRoot = mkdtempSync(join(tmpdir(), "settings-store-test-"));
  home = tempRoot;
  userData = join(home, "Library", "Application Support", "gus-mac");
  settingsFile = join(userData, "settings.json");
  mkdirSync(userData, { recursive: true });
  process.env.DOT_FILES_SETTINGS_PATH = settingsFile;
});

afterEach(() => {
  delete process.env.DOT_FILES_SETTINGS_PATH;
  rmSync(tempRoot, { recursive: true, force: true });
});

describe("defaultWorkflowDbPath", () => {
  it("returns workflows.sqlite3 inside the userData directory", async () => {
    const { defaultWorkflowDbPath } = await loadModule();
    expect(defaultWorkflowDbPath()).toBe(join(userData, "workflows.sqlite3"));
  });
});

describe("cleanSettings", () => {
  it("fills missing workflowDbPath with the userData default", async () => {
    const { cleanSettings } = await loadModule();
    expect(cleanSettings({})).toMatchObject({
      workflowDbPath: join(userData, "workflows.sqlite3"),
    });
  });

  it("preserves an explicitly set workflowDbPath", async () => {
    const { cleanSettings } = await loadModule();
    expect(cleanSettings({ workflowDbPath: "/custom/path.sqlite3" })).toMatchObject({
      workflowDbPath: "/custom/path.sqlite3",
    });
  });
});

describe("readSavedSettings", () => {
  it("returns the saved workflowDbPath verbatim", async () => {
    writeFileSync(
      settingsFile,
      JSON.stringify({ repoRoot: "/repo", workflowDbPath: "/some/saved/path.sqlite3" }),
    );

    const { readSavedSettings } = await loadModule();
    const result = readSavedSettings();

    expect(result.workflowDbPath).toBe("/some/saved/path.sqlite3");
    expect(result.repoRoot).toBe("/repo");
  });

  it("falls back to the default workflowDbPath when settings.json is missing", async () => {
    rmSync(settingsFile, { force: true });

    const { readSavedSettings, defaultWorkflowDbPath } = await loadModule();
    const result = readSavedSettings();

    expect(result.workflowDbPath).toBe(defaultWorkflowDbPath());
  });
});
