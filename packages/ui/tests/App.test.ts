import { flushPromises, mount } from "@vue/test-utils";
import { beforeEach, describe, expect, it, vi } from "vitest";
import App from "../src/App.vue";
import type { MacOSApi, RunEvent, SettingsResponse, Workflow } from "../src/types/api";

const workflows: Workflow[] = [
  {
    id: "review-template",
    name: "Review Template",
    description: "Validate and print the tracked source of truth.",
    changesMac: "No",
    phases: [
      { id: "validate-template-files", name: "Validate template files", enabled: true },
      { id: "print-tracked-homebrew-bundle", name: "Print tracked Homebrew bundle", enabled: true },
      { id: "list-tracked-apps", name: "List tracked apps", enabled: true },
      { id: "list-tracked-macos-settings", name: "List tracked macOS settings", enabled: true },
      { id: "list-tracked-dotfile-bundles", name: "List tracked dotfile bundles", enabled: true },
    ],
    confirmation: {
      title: "Review Template",
      message: "Validate and print the template.",
      options: [
        { id: "run-now", label: "Run now", description: "continue", continue: true, back: false },
        {
          id: "back",
          label: "Back",
          description: "return to workflow menu",
          continue: false,
          back: true,
        },
      ],
    },
  },
  {
    id: "update-template-from-this-mac",
    name: "Update Template From This Mac",
    description: "Generate review candidates from this Mac.",
    changesMac: "Writes review candidates",
    phases: [
      { id: "save-current-mac-snapshot", name: "Save current Mac snapshot", enabled: true },
      {
        id: "generate-installed-app-review-candidate",
        name: "Generate installed app review candidate",
        enabled: true,
      },
    ],
    confirmation: {
      title: "Update Template From This Mac",
      message: "Generate review candidates.",
      options: [
        {
          id: "preview-only",
          label: "Preview only",
          description: "show what would happen",
          continue: true,
          back: false,
          phases: [
            { id: "save-current-mac-snapshot", name: "Save current Mac snapshot", enabled: true },
            {
              id: "generate-installed-app-review-candidate",
              name: "Generate installed app review candidate",
              enabled: true,
            },
          ],
        },
        {
          id: "run-now",
          label: "Run now",
          description: "make the described changes",
          continue: true,
          back: false,
          phases: [
            { id: "save-current-mac-snapshot", name: "Save current Mac snapshot", enabled: true },
            {
              id: "generate-installed-app-review-candidate",
              name: "Generate installed app review candidate",
              enabled: true,
            },
          ],
        },
      ],
    },
  },
  {
    id: "inspect-current",
    name: "Inspect Current State",
    description: "Check whether setup state looks correct.",
    changesMac: "No",
    phases: [{ id: "run-health-checks", name: "Run health checks", enabled: true }],
    confirmation: {
      title: "Inspect Current State",
      message: "Run health checks only.",
      options: [
        { id: "run-now", label: "Run now", description: "continue", continue: true, back: false },
      ],
    },
  },
  {
    id: "converge-to-template",
    name: "Converge to Template",
    description: "Install configured applications.",
    changesMac: "Yes",
    phases: [{ id: "install-homebrew-apps", name: "Install Homebrew apps", enabled: true }],
    confirmation: {
      title: "Converge to Template",
      message: "Install configured applications.",
      options: [
        {
          id: "install-now",
          label: "Install now",
          description: "continue",
          continue: true,
          back: false,
        },
      ],
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
      events: [
        {
          id: 1,
          runId: "run-1",
          seq: 1,
          type: "phase_output",
          phaseId: "doctor",
          phaseName: "Run health checks",
          message: "ok",
          createdAt: "2026-05-04T00:00:00Z",
        },
      ],
    }),
    templateFiles: vi.fn().mockResolvedValue([
      {
        path: "/repo/apps.yaml",
        relative: "apps.yaml",
        kind: "apps",
        size: 64,
        exists: true,
      },
      {
        path: "/repo/stow/shell/.zshrc",
        relative: "stow/shell/.zshrc",
        kind: "stow",
        size: 20,
        exists: true,
      },
    ]),
    readTemplateFile: vi.fn().mockResolvedValue({
      file: {
        path: "/repo/apps.yaml",
        relative: "apps.yaml",
        kind: "apps",
        size: 64,
        exists: true,
      },
      content: "apps:\n  - name: Ghostty\n",
    }),
    saveTemplateFile: vi.fn().mockImplementation(async (path: string, content: string) => ({
      file: {
        path,
        relative: path.replace("/repo/", ""),
        kind: "apps",
        size: content.length,
        exists: true,
      },
      content,
    })),
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
        {
          key: "workflow_db_path",
          label: "Workflow SQLite database",
          path: "/Users/gus/Library/Application Support/mac-os/workflows.sqlite3",
          status: "ok",
          message: "ok",
        },
      ],
    }),
    validateSettings: vi.fn().mockImplementation(async (settings) => ({
      valid: true,
      settings,
      checks: [
        {
          key: "repo_root",
          label: "Repository root",
          path: settings.repoRoot,
          status: "ok",
          message: "ok",
        },
      ],
    })),
    saveSettings: vi.fn().mockImplementation(async (settings) => ({
      valid: true,
      settings,
      checks: [
        {
          key: "repo_root",
          label: "Repository root",
          path: settings.repoRoot,
          status: "ok",
          message: "ok",
        },
      ],
    })),
    chooseDirectory: vi.fn().mockResolvedValue("/chosen"),
    chooseFile: vi.fn().mockResolvedValue("/chosen/file.yaml"),
    chooseSaveFile: vi.fn().mockResolvedValue("/chosen/workflows.sqlite3"),
    listOpVaults: vi.fn().mockResolvedValue({ ok: true, vaults: [{ id: "v1", name: "Private" }] }),
    listOpItems: vi
      .fn()
      .mockResolvedValue({ ok: true, items: [{ id: "i1", title: "Mac Migration Archive" }] }),
    signinOpCli: vi.fn().mockResolvedValue({ ok: true }),
    installOpDependencies: vi.fn().mockResolvedValue({ ok: true }),
    openDevTools: vi.fn().mockResolvedValue(undefined),
    macName: vi.fn().mockResolvedValue("Local Mac"),
    macHostname: vi.fn().mockResolvedValue("localhost"),
    getUserPreferences: vi.fn().mockResolvedValue({ theme: "light" }),
    saveUserPreferences: vi.fn().mockImplementation(async (theme: string) => ({
      theme,
      updatedAt: new Date().toISOString(),
    })),
    runWorkflow: vi
      .fn()
      .mockImplementation(async (_request, onEvent: (event: RunEvent) => void) => {
        onEvent({
          runId: "run-2",
          seq: 1,
          type: "phase_started",
          phaseId: "validate-template-files",
          phaseName: "Validate template files",
          status: "running",
        });
        onEvent({
          runId: "run-2",
          seq: 2,
          type: "phase_output",
          phaseId: "validate-template-files",
          phaseName: "Validate template files",
          message: "healthy",
        });
        onEvent({ runId: "run-2", seq: 3, type: "run_finished", status: "completed" });

        return { exitCode: 0 };
      }),
    ...overrides,
  };

  window.macOS = api;

  return api;
}

function findDocumentButton(text: string) {
  return [...document.body.querySelectorAll("button")].find((button) =>
    button.textContent?.includes(text),
  );
}

async function flushOutputHighlighting() {
  for (let i = 0; i < 10; i += 1) {
    await flushPromises();
    await new Promise((resolve) => setTimeout(resolve, 0));
  }
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

    expect(wrapper.text()).toContain("Template");
    expect(wrapper.text()).toContain("Review Template");
  });

  it("keeps only status in the workflow detail toolbar actions area", async () => {
    installApi();

    const wrapper = mount(App);
    await flushPromises();

    const toolbar = wrapper.find('[data-testid="detail-toolbar"]');

    expect(toolbar.text()).toContain("idle");
    expect(toolbar.text()).not.toContain("Refresh");
    expect(toolbar.text()).not.toContain("Reset phases");
    expect(toolbar.text()).not.toContain("Run workflow");
  });

  it("closes the workflow detail pane when the confirmation Back row is clicked", async () => {
    const api = installApi();

    const wrapper = mount(App);
    await flushPromises();

    expect(wrapper.find("#mac-detail").exists()).toBe(true);

    await wrapper
      .findAll("button")
      .find((button) => button.text().includes("Back"))
      ?.trigger("click");
    await flushPromises();

    expect(wrapper.find("#mac-detail").exists()).toBe(false);
    expect(document.body.querySelector('[data-slot="alert-dialog-content"]')).toBeNull();
    expect(api.runWorkflow).not.toHaveBeenCalled();
  });

  it("shows skeletons while initial data is loading", () => {
    installApi({
      workflows: vi.fn(() => new Promise<Workflow[]>(() => {})),
    });

    const wrapper = mount(App);

    expect(wrapper.find('[data-testid="initial-shell-skeleton"]').exists()).toBe(true);
    expect(wrapper.findAll('[data-slot="skeleton"]').length).toBeGreaterThan(0);
    expect(wrapper.text()).not.toContain("No template workflows registered.");
  });

  it("replaces the initial shell with loaded data", async () => {
    const api = installApi();

    const wrapper = mount(App);

    expect(wrapper.find('[data-testid="initial-shell-skeleton"]').exists()).toBe(true);
    await flushPromises();

    expect(wrapper.find('[data-testid="initial-shell-skeleton"]').exists()).toBe(false);
    expect(wrapper.text()).toContain("Review Template");
    expect(wrapper.text()).toContain("Mac: Local Mac");
    expect(wrapper.text()).toContain("localhost");
    expect(api.macName).toHaveBeenCalled();
    expect(api.macHostname).toHaveBeenCalled();
  });

  it("filters workflow list with search and category nav", async () => {
    installApi();

    const wrapper = mount(App);
    await flushPromises();

    await wrapper.find('[data-testid="app-search"]').setValue("does-not-exist");
    expect(wrapper.text()).toContain("No template workflows registered.");

    await wrapper.find('[data-testid="app-search"]').setValue("");
    await wrapper
      .findAll("button")
      .find((button) => /^Update\s*\d*$/.test(button.text().trim()))
      ?.trigger("click");
    await flushPromises();

    expect(
      wrapper.findAll("button").some((button) => button.text().includes("Converge to Template")),
    ).toBe(true);
  });

  it("only matches the visible card content, not the hidden detail copy", async () => {
    installApi();

    const wrapper = mount(App);
    await flushPromises();

    await wrapper
      .findAll("button")
      .find((button) => /^Update\s*\d*$/.test(button.text().trim()))
      ?.trigger("click");
    await flushPromises();

    expect(
      wrapper.findAll("button").some((button) => button.text().includes("Converge to Template")),
    ).toBe(true);

    await wrapper.find('[data-testid="app-search"]').setValue("snapshot");

    expect(
      wrapper.findAll("button").some((button) => button.text().includes("Converge to Template")),
    ).toBe(false);
    expect(wrapper.text()).toContain("No update workflows registered.");

    await wrapper.find('[data-testid="app-search"]').setValue("converge");

    expect(
      wrapper.findAll("button").some((button) => button.text().includes("Converge to Template")),
    ).toBe(true);
  });

  it("runs a workflow and appends streamed output", async () => {
    const api = installApi();

    const wrapper = mount(App);
    await flushPromises();

    await wrapper
      .findAll("button")
      .find((button) => button.text().includes("Run now"))
      ?.trigger("click");
    findDocumentButton("Continue")?.click();
    await flushOutputHighlighting();

    expect(api.runWorkflow).toHaveBeenCalledWith(
      {
        workflowId: "review-template",
        confirmationOptionId: "run-now",
        enabledPhaseIds: [
          "validate-template-files",
          "print-tracked-homebrew-bundle",
          "list-tracked-apps",
          "list-tracked-macos-settings",
          "list-tracked-dotfile-bundles",
        ],
      },
      expect.any(Function),
    );
    expect(document.body.textContent).toContain("healthy");
    expect(document.body.textContent).toContain("completed");
  });

  it("shows workflow progress in a stepper dialog", async () => {
    installApi();

    const wrapper = mount(App);
    await flushPromises();

    await wrapper
      .findAll("button")
      .find((button) => button.text().includes("Run now"))
      ?.trigger("click");
    findDocumentButton("Continue")?.click();
    await flushOutputHighlighting();

    expect(document.body.querySelector('[data-slot="stepper"]')).not.toBeNull();
    expect(document.body.textContent).toContain("Print tracked Homebrew bundle");
    expect(document.body.textContent).toContain("Run output");
    expect(document.body.textContent).toContain("Step 1: Validate template files");
    expect(document.body.textContent).toContain("healthy");
    expect(document.body.textContent).not.toContain("phase_started running");
    expect(document.body.textContent).not.toContain("phase_finished ok");
  });

  it("groups streamed output chunks under the matching step", async () => {
    installApi({
      runWorkflow: vi
        .fn()
        .mockImplementation(async (_request, onEvent: (event: RunEvent) => void) => {
          onEvent({
            runId: "run-streamed",
            seq: 1,
            type: "phase_started",
            phaseId: "list-tracked-apps",
            phaseName: "List tracked apps",
            status: "running",
          });
          onEvent({
            runId: "run-streamed",
            seq: 2,
            type: "phase_output",
            phaseId: "list-tracked-apps",
            phaseName: "List tracked apps",
            message: "Ghostty\n",
          });
          onEvent({
            runId: "run-streamed",
            seq: 3,
            type: "phase_output",
            phaseId: "list-tracked-apps",
            phaseName: "List tracked apps",
            message: "Raycast\n",
          });
          onEvent({
            runId: "run-streamed",
            seq: 4,
            type: "phase_finished",
            phaseId: "list-tracked-apps",
            phaseName: "List tracked apps",
            status: "ok",
          });
          onEvent({ runId: "run-streamed", seq: 5, type: "run_finished", status: "completed" });

          return { exitCode: 0 };
        }),
    });

    const wrapper = mount(App);
    await flushPromises();

    await wrapper
      .findAll("button")
      .find((button) => button.text().includes("Run now"))
      ?.trigger("click");
    findDocumentButton("Continue")?.click();
    await flushOutputHighlighting();

    const sections = document.body.querySelector('[data-testid="run-output-sections"]');

    expect(sections?.textContent).toContain("Step 3: List tracked apps");
    expect(sections?.textContent).toContain("Ghostty");
    expect(sections?.textContent).toContain("Raycast");
  });

  it("separates confirmation dialog header body and footer surfaces", async () => {
    installApi();

    const wrapper = mount(App);
    await flushPromises();

    await wrapper
      .findAll("button")
      .find((button) => button.text().includes("Run now"))
      ?.trigger("click");

    expect(document.body.querySelector('[data-slot="alert-dialog-content"]')?.className).toContain(
      "grid-rows-[auto_minmax(0,1fr)_auto]",
    );
    expect(document.body.querySelector('[data-slot="alert-dialog-content"]')?.className).toContain(
      "h-[min(760px,calc(100vh-2rem))]",
    );
    expect(document.body.querySelector('[data-slot="alert-dialog-header"]')?.className).toContain(
      "border-b",
    );
    expect(document.body.querySelector('[data-testid="alert-dialog-body"]')?.className).toContain(
      "bg-background",
    );
    expect(document.body.querySelector('[data-testid="alert-dialog-body"]')?.className).toContain(
      "min-h-0",
    );
    expect(document.body.querySelector('[data-slot="alert-dialog-footer"]')?.className).toContain(
      "border-t",
    );
    expect(document.body.querySelector('[data-testid="run-output-panel"]')?.className).toContain(
      "min-h-96",
    );
    expect(document.body.querySelector('[data-testid="run-output-panel"]')?.innerHTML).toContain(
      "status-idle",
    );
  });

  it("uses workflow confirmation copy in the progress dialog header", async () => {
    installApi();

    const wrapper = mount(App);
    await flushPromises();

    await wrapper
      .findAll("button")
      .find((button) => button.text().includes("Run now"))
      ?.trigger("click");

    const header = document.body.querySelector('[data-slot="alert-dialog-header"]');

    expect(header?.textContent).toContain("Review Template");
    expect(header?.textContent).toContain("Validate and print the template.");
    expect(header?.textContent).not.toContain("continue");
  });

  it("scrolls the dialog body to the footer when a run completes", async () => {
    const scrollIntoView = vi.fn();
    const originalScrollIntoView = Element.prototype.scrollIntoView;

    Element.prototype.scrollIntoView = scrollIntoView;
    try {
      installApi();

      const wrapper = mount(App);
      await flushPromises();

      await wrapper
        .findAll("button")
        .find((button) => button.text().includes("Run now"))
        ?.trigger("click");
      findDocumentButton("Continue")?.click();
      await flushPromises();

      expect(scrollIntoView).toHaveBeenCalledWith(
        expect.objectContaining({ block: "end", behavior: "smooth" }),
      );
    } finally {
      Element.prototype.scrollIntoView = originalScrollIntoView;
    }
  });

  it("keeps failed workflow output visible in the progress dialog", async () => {
    installApi({
      runWorkflow: vi
        .fn()
        .mockImplementation(async (_request, onEvent: (event: RunEvent) => void) => {
          onEvent({
            runId: "run-failed",
            seq: 1,
            type: "phase_started",
            phaseId: "print-tracked-homebrew-bundle",
            phaseName: "Print tracked Homebrew bundle",
            status: "running",
          });
          onEvent({
            runId: "run-failed",
            seq: 2,
            type: "phase_output",
            phaseId: "print-tracked-homebrew-bundle",
            phaseName: "Print tracked Homebrew bundle",
            message: "boom",
          });
          onEvent({
            runId: "run-failed",
            seq: 3,
            type: "phase_finished",
            phaseId: "print-tracked-homebrew-bundle",
            phaseName: "Print tracked Homebrew bundle",
            status: "failed",
            message: "failed",
          });
          onEvent({ runId: "run-failed", seq: 4, type: "run_failed", status: "failed" });

          return { exitCode: 1 };
        }),
    });

    const wrapper = mount(App);
    await flushPromises();

    await wrapper
      .findAll("button")
      .find((button) => button.text().includes("Run now"))
      ?.trigger("click");
    findDocumentButton("Continue")?.click();
    await flushOutputHighlighting();

    expect(document.body.querySelector('[data-slot="stepper"]')).not.toBeNull();
    expect(document.body.textContent).toContain("boom");
    expect(document.body.textContent).toContain("failed");
  });

  it("loads and saves allowlisted template files", async () => {
    const api = installApi();

    const wrapper = mount(App);
    await flushPromises();

    await wrapper
      .findAll("button")
      .find((button) => button.text().includes("Template Files"))
      ?.trigger("click");
    await flushPromises();

    expect(api.templateFiles).toHaveBeenCalled();
    expect(wrapper.text()).toContain("apps.yaml");

    await wrapper
      .findAll("button")
      .find((button) => button.text().trim().startsWith("apps.yaml"))
      ?.trigger("click");
    await flushPromises();

    await wrapper.find("textarea").setValue("apps:\n  - name: Ghostty\n  - name: Raycast\n");
    await wrapper
      .findAll("button")
      .find((button) => /^Save$/.test(button.text().trim()))
      ?.trigger("click");
    await flushPromises();

    expect(api.saveTemplateFile).toHaveBeenCalledWith(
      "/repo/apps.yaml",
      "apps:\n  - name: Ghostty\n  - name: Raycast\n",
    );
    expect(wrapper.text()).toContain("Template file saved.");
  });

  it("shows skeletons while template files load", async () => {
    let resolveFiles: (value: Awaited<ReturnType<MacOSApi["templateFiles"]>>) => void = () => {};
    installApi({
      templateFiles: vi.fn(
        (): Promise<Awaited<ReturnType<MacOSApi["templateFiles"]>>> =>
          new Promise((resolve) => {
            resolveFiles = resolve;
          }),
      ),
    });

    const wrapper = mount(App);
    await flushPromises();

    await wrapper
      .findAll("button")
      .find((button) => button.text().includes("Template Files"))
      ?.trigger("click");
    await flushPromises();

    expect(wrapper.find('[data-testid="template-files-skeleton"]').exists()).toBe(true);

    resolveFiles([]);
    await flushPromises();

    expect(wrapper.find('[data-testid="template-files-skeleton"]').exists()).toBe(false);
  });

  it("opens persisted logs", async () => {
    const api = installApi();

    const wrapper = mount(App);
    await flushPromises();

    await wrapper
      .findAll("button")
      .find((button) => button.text().includes("Logs"))
      ?.trigger("click");
    await flushPromises();
    await wrapper
      .findAll("button")
      .find((button) => button.text().includes("Check Setup"))
      ?.trigger("click");
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
        events: [
          {
            id: 1,
            runId: "run-1",
            seq: 1,
            type: "phase_output",
            phaseId: "doctor",
            phaseName: "Run health checks",
            message: "\u001B[31mred output\u001B[0m",
            createdAt: "2026-05-04T00:00:00Z",
          },
        ],
      }),
    });

    const wrapper = mount(App);
    await flushPromises();

    await wrapper
      .findAll("button")
      .find((button) => button.text().includes("Logs"))
      ?.trigger("click");
    await flushPromises();
    await wrapper
      .findAll("button")
      .find((button) => button.text().includes("Check Setup"))
      ?.trigger("click");
    await flushOutputHighlighting();

    expect(wrapper.text()).toContain("red output");
    expect(wrapper.text()).not.toContain("\u001B");
    expect(wrapper.html()).toContain("shiki");
  });

  it("filters workflows by sidebar step", async () => {
    installApi();

    const wrapper = mount(App);
    await flushPromises();

    // Default section is Template
    expect(wrapper.text()).toContain("Review Template");

    await wrapper
      .findAll("button")
      .find((button) => /^Current state\s*\d*$/.test(button.text().trim()))
      ?.trigger("click");
    await flushPromises();

    expect(wrapper.text()).toContain("Inspect Current State");
    expect(wrapper.text()).not.toContain("No current-state workflows registered.");

    await wrapper
      .findAll("button")
      .find((button) => /^Update\s*\d*$/.test(button.text().trim()))
      ?.trigger("click");
    await flushPromises();

    expect(wrapper.text()).toContain("Converge to Template");
  });

  it("renders settings and saves changed repo configuration", async () => {
    const api = installApi();

    const wrapper = mount(App);
    await flushPromises();

    await wrapper
      .findAll("button")
      .find((button) => button.text().includes("Settings"))
      ?.trigger("click");
    await flushPromises();

    expect(wrapper.text()).toContain("Workflow SQLite database");
    await wrapper.find('[data-testid="settings-repo-root"]').setValue("/repo-next");
    await wrapper
      .findAll("button")
      .find((button) => button.text().includes("Save settings"))
      ?.trigger("click");
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
      checks: [
        { key: "repo_root", label: "Repository root", path: "/repo", status: "ok", message: "ok" },
      ],
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

    await wrapper
      .findAll("button")
      .find((button) => button.text().includes("Settings"))
      ?.trigger("click");
    await flushPromises();

    expect(wrapper.findAll('[data-testid="settings-checks-skeleton"]').length).toBeGreaterThan(0);

    resolveSecond?.(settledResponse);
    await flushPromises();

    expect(wrapper.findAll('[data-testid="settings-checks-skeleton"]').length).toBe(0);
    expect(wrapper.find<HTMLInputElement>('[data-testid="settings-repo-root"]').element.value).toBe(
      "/repo",
    );
  });

  it("shows settings validation failures without clearing input", async () => {
    const api = installApi({
      saveSettings: vi.fn().mockImplementation(async (settings) => ({
        valid: false,
        settings,
        checks: [
          {
            key: "repo_root",
            label: "Repository root",
            path: settings.repoRoot,
            status: "error",
            message: "missing repo markers",
          },
        ],
      })),
    });

    const wrapper = mount(App);
    await flushPromises();

    await wrapper
      .findAll("button")
      .find((button) => button.text().includes("Settings"))
      ?.trigger("click");
    await flushPromises();
    await wrapper.find('[data-testid="settings-repo-root"]').setValue("/broken");
    await wrapper
      .findAll("button")
      .find((button) => button.text().includes("Save settings"))
      ?.trigger("click");
    await flushPromises();

    expect(api.saveSettings).toHaveBeenCalledWith(expect.objectContaining({ repoRoot: "/broken" }));
    expect(wrapper.find<HTMLInputElement>('[data-testid="settings-repo-root"]').element.value).toBe(
      "/broken",
    );
    expect(wrapper.text()).toContain("missing repo markers");
  });
});
