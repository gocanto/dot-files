import { flushPromises, mount } from "@vue/test-utils";
import { beforeEach, describe, expect, it, vi } from "vitest";
import App from "../src/App.vue";
import type { MacOSApi, RunEvent, SettingsResponse, Workflow } from "../src/types/api";

const workflows: Workflow[] = [
  {
    id: "check-setup",
    name: "Check Setup",
    description: "Check whether setup state looks correct.",
    changesMac: "No",
    phases: [{ id: "run-health-checks", name: "Run health checks", enabled: true }],
    confirmation: {
      title: "Check Setup",
      message: "Run health checks only.",
      options: [{ id: "run-now", label: "Run now", description: "continue", continue: true, back: false }],
    },
  },
  {
    id: "install-apps",
    name: "Install Apps",
    description: "Install configured applications.",
    changesMac: "Yes",
    phases: [{ id: "install-homebrew-apps", name: "Install Homebrew apps", enabled: true }],
    confirmation: {
      title: "Install Apps",
      message: "Install configured applications.",
      options: [{ id: "install-now", label: "Install now", description: "continue", continue: true, back: false }],
    },
  },
];

function installApi(overrides: Partial<MacOSApi> = {}) {
  const api: MacOSApi = {
    workflows: vi.fn().mockResolvedValue(workflows),
    runs: vi.fn().mockResolvedValue([
      {
        id: "run-1",
        workflowId: "check-setup",
        workflowName: "Check Setup",
        confirmationOptionId: "run-now",
        confirmationOptionLabel: "Run now",
        mode: "live",
        status: "completed",
        startedAt: "2026-05-04T00:00:00Z",
      },
    ]),
    runLog: vi.fn().mockResolvedValue({
      run: {
        id: "run-1",
        workflowId: "check-setup",
        workflowName: "Check Setup",
        confirmationOptionId: "run-now",
        confirmationOptionLabel: "Run now",
        mode: "live",
        status: "completed",
        startedAt: "2026-05-04T00:00:00Z",
      },
      events: [{ id: 1, runId: "run-1", seq: 1, type: "phase_output", message: "ok", createdAt: "2026-05-04T00:00:00Z" }],
    }),
    settings: vi.fn().mockResolvedValue({
      valid: true,
      settings: {
        repoRoot: "/repo",
        appsConfigPath: "/repo/apps.yaml",
        secretsConfigPath: "/repo/secrets.yaml",
        generatedAppsPath: "/repo/apps.generated.yaml",
        archiveRoot: "/Users/gus/.local/state/macos-settings-archives",
        workflowDbPath: "/Users/gus/Library/Application Support/mac-os/workflows.sqlite3",
        opVault: "Private",
        opItem: "Mac Migration Archive",
      },
      checks: [
        { key: "repo_root", label: "Repository root", path: "/repo", status: "ok", message: "ok" },
        { key: "workflow_db_path", label: "Workflow SQLite database", path: "/Users/gus/Library/Application Support/mac-os/workflows.sqlite3", status: "ok", message: "ok" },
      ],
    }),
    validateSettings: vi.fn().mockImplementation(async (settings) => ({
      valid: true,
      settings,
      checks: [{ key: "repo_root", label: "Repository root", path: settings.repoRoot, status: "ok", message: "ok" }],
    })),
    saveSettings: vi.fn().mockImplementation(async (settings) => ({
      valid: true,
      settings,
      checks: [{ key: "repo_root", label: "Repository root", path: settings.repoRoot, status: "ok", message: "ok" }],
    })),
    chooseDirectory: vi.fn().mockResolvedValue("/chosen"),
    chooseFile: vi.fn().mockResolvedValue("/chosen/file.yaml"),
    chooseSaveFile: vi.fn().mockResolvedValue("/chosen/workflows.sqlite3"),
    getUserPreferences: vi.fn().mockResolvedValue({ theme: "light" }),
    saveUserPreferences: vi.fn().mockImplementation(async (theme: string) => ({ theme, updatedAt: new Date().toISOString() })),
    runWorkflow: vi.fn().mockImplementation(async (_request, onEvent: (event: RunEvent) => void) => {
      onEvent({ runId: "run-2", seq: 1, type: "phase_started", phaseId: "run-health-checks", phaseName: "Run health checks", status: "running" });
      onEvent({ runId: "run-2", seq: 2, type: "phase_output", phaseId: "run-health-checks", phaseName: "Run health checks", message: "healthy" });
      onEvent({ runId: "run-2", seq: 3, type: "run_finished", status: "completed" });

      return { exitCode: 0 };
    }),
    ...overrides,
  };

  window.macOS = api;

  return api;
}

function findDocumentButton(text: string) {
  return [...document.body.querySelectorAll("button")].find((button) => button.textContent?.includes(text));
}

async function flushOutputHighlighting() {
  await flushPromises();
  await new Promise((resolve) => setTimeout(resolve, 0));
  await flushPromises();
}

describe("App", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
    document.body.innerHTML = "";
  });

  it("renders workflow navigation and details", async () => {
    installApi();

    const wrapper = mount(App);
    await flushPromises();

    expect(wrapper.text()).toContain("Workflows");
    expect(wrapper.text()).toContain("Check Setup");
    expect(wrapper.text()).toContain("Run health checks");
  });

  it("shows skeletons while initial data is loading", () => {
    installApi({
      workflows: vi.fn(() => new Promise<Workflow[]>(() => {})),
    });

    const wrapper = mount(App);

    expect(wrapper.findAll('[data-slot="skeleton"]').length).toBeGreaterThan(0);
  });

  it("filters workflow list with search and workflow tabs", async () => {
    installApi();

    const wrapper = mount(App);
    await flushPromises();

    await wrapper.find('[data-testid="app-search"]').setValue("does-not-exist");
    expect(wrapper.text()).toContain("No workflows match this view.");

    await wrapper.find('[data-testid="app-search"]').setValue("");
    await wrapper.findAll("button").find((button) => button.text().includes("Changes"))?.trigger("click");
    await flushPromises();

    expect(wrapper.findAll("button").some((button) => button.text().includes("Install Apps"))).toBe(true);
  });

  it("runs a workflow and appends streamed output", async () => {
    const api = installApi();

    const wrapper = mount(App);
    await flushPromises();

    await wrapper.findAll("button").find((button) => button.text().includes("Run now"))?.trigger("click");
    findDocumentButton("Continue")?.click();
    await flushOutputHighlighting();

    expect(api.runWorkflow).toHaveBeenCalledWith(
      { workflowId: "check-setup", confirmationOptionId: "run-now", enabledPhaseIds: ["run-health-checks"] },
      expect.any(Function),
    );
    expect(wrapper.text()).toContain("healthy");
    expect(wrapper.text()).toContain("completed");
  });

  it("opens persisted logs", async () => {
    const api = installApi();

    const wrapper = mount(App);
    await flushPromises();

    await wrapper.findAll("button").find((button) => button.text().includes("Logs"))?.trigger("click");
    await flushPromises();
    await wrapper.findAll("button").find((button) => button.text().includes("Check Setup"))?.trigger("click");
    await flushPromises();

    expect(api.runLog).toHaveBeenCalledWith("run-1");
    expect(wrapper.text()).toContain("ok");
    expect(wrapper.text()).toContain("live");
  });

  it("renders ANSI persisted log output without raw escape codes", async () => {
    installApi({
      runLog: vi.fn().mockResolvedValue({
        run: {
          id: "run-1",
          workflowId: "check-setup",
          workflowName: "Check Setup",
          confirmationOptionId: "run-now",
          confirmationOptionLabel: "Run now",
          mode: "live",
          status: "completed",
          startedAt: "2026-05-04T00:00:00Z",
        },
        events: [{ id: 1, runId: "run-1", seq: 1, type: "phase_output", message: "\u001B[31mred output\u001B[0m", createdAt: "2026-05-04T00:00:00Z" }],
      }),
    });

    const wrapper = mount(App);
    await flushPromises();

    await wrapper.findAll("button").find((button) => button.text().includes("Logs"))?.trigger("click");
    await flushPromises();
    await wrapper.findAll("button").find((button) => button.text().includes("Check Setup"))?.trigger("click");
    await flushOutputHighlighting();

    expect(wrapper.text()).toContain("red output");
    expect(wrapper.text()).not.toContain("\u001B");
    expect(wrapper.html()).toContain("shiki");
  });

  it("filters workflows by sidebar category", async () => {
    installApi();

    const wrapper = mount(App);
    await flushPromises();

    expect(wrapper.text()).toContain("Check Setup");
    expect(wrapper.text()).toContain("Install Apps");

    await wrapper.findAll("button").find((button) => button.text().includes("Snapshots"))?.trigger("click");
    await flushPromises();

    expect(wrapper.text()).toContain("No snapshot workflows match this view.");

    await wrapper.findAll("button").find((button) => button.text().includes("Health Checks"))?.trigger("click");
    await flushPromises();

    expect(wrapper.text()).toContain("Check Setup");
    expect(wrapper.text()).not.toContain("No health-check workflows match this view.");
  });

  it("renders settings and saves changed repo configuration", async () => {
    const api = installApi();

    const wrapper = mount(App);
    await flushPromises();

    await wrapper.findAll("button").find((button) => button.text().includes("Settings"))?.trigger("click");
    await flushPromises();

    expect(wrapper.text()).toContain("Workflow SQLite database");
    await wrapper.find('[data-testid="settings-repo-root"]').setValue("/repo-next");
    await wrapper.findAll("button").find((button) => button.text().includes("Save settings"))?.trigger("click");
    await flushPromises();

    expect(api.saveSettings).toHaveBeenCalledWith(
      expect.objectContaining({
        repoRoot: "/repo-next",
        workflowDbPath: "/Users/gus/Library/Application Support/mac-os/workflows.sqlite3",
      }),
    );
    expect(wrapper.text()).toContain("Settings saved");
  });

  it("shows skeletons while settings load on entering Settings", async () => {
    const settledResponse: SettingsResponse = {
      valid: true,
      settings: {
        repoRoot: "/repo",
        appsConfigPath: "/repo/apps.yaml",
        secretsConfigPath: "/repo/secrets.yaml",
        generatedAppsPath: "/repo/apps.generated.yaml",
        archiveRoot: "/archive",
        workflowDbPath: "/db.sqlite3",
        opVault: "Private",
        opItem: "Mac Migration Archive",
      },
      checks: [{ key: "repo_root", label: "Repository root", path: "/repo", status: "ok", message: "ok" }],
    };
    let call = 0;
    let resolveSecond: ((value: SettingsResponse) => void) | undefined;
    installApi({
      settings: vi.fn((): Promise<SettingsResponse> => {
        call += 1;
        if (call === 1) return Promise.resolve(settledResponse);
        return new Promise<SettingsResponse>((resolve) => {
          resolveSecond = resolve;
        });
      }),
    });

    const wrapper = mount(App);
    await flushPromises();

    await wrapper.findAll("button").find((button) => button.text().includes("Settings"))?.trigger("click");
    await flushPromises();

    expect(wrapper.findAll('[data-testid="settings-checks-skeleton"]').length).toBeGreaterThan(0);

    resolveSecond?.(settledResponse);
    await flushPromises();

    expect(wrapper.findAll('[data-testid="settings-checks-skeleton"]').length).toBe(0);
    expect(wrapper.find<HTMLInputElement>('[data-testid="settings-repo-root"]').element.value).toBe("/repo");
  });

  it("shows settings validation failures without clearing input", async () => {
    const api = installApi({
      saveSettings: vi.fn().mockImplementation(async (settings) => ({
        valid: false,
        settings,
        checks: [{ key: "repo_root", label: "Repository root", path: settings.repoRoot, status: "error", message: "missing repo markers" }],
      })),
    });

    const wrapper = mount(App);
    await flushPromises();

    await wrapper.findAll("button").find((button) => button.text().includes("Settings"))?.trigger("click");
    await flushPromises();
    await wrapper.find('[data-testid="settings-repo-root"]').setValue("/broken");
    await wrapper.findAll("button").find((button) => button.text().includes("Save settings"))?.trigger("click");
    await flushPromises();

    expect(api.saveSettings).toHaveBeenCalledWith(expect.objectContaining({ repoRoot: "/broken" }));
    expect(wrapper.find<HTMLInputElement>('[data-testid="settings-repo-root"]').element.value).toBe("/broken");
    expect(wrapper.text()).toContain("missing repo markers");
  });
});
