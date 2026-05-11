import { DOMWrapper, flushPromises, mount } from "@vue/test-utils";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { defineComponent } from "vue";
import App from "@entry/App.vue";
import WorkflowListPanel from "@app/WorkflowListPanel.vue";
import { useAppController } from "@composables/useAppController";
import { templateFileLanguage } from "@lib/templateFileLanguage";
import type { MacOSApi, RunEvent, SettingsResponse, Workflow } from "@api";

const accountAvatarUrl =
  "data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 40 40'%3E%3Crect width='40' height='40' fill='%23262626'/%3E%3Ccircle cx='20' cy='15' r='7' fill='%23f8fafc'/%3E%3Cpath d='M8 36c2.5-8 7.2-12 12-12s9.5 4 12 12' fill='%23f8fafc'/%3E%3C/svg%3E";

vi.mock("@app/MonacoFileEditor.vue", () => ({
  default: {
    props: ["modelValue", "path", "loading"],
    emits: ["update:modelValue"],
    template:
      '<textarea data-testid="monaco-editor" :value="modelValue" @input="$emit(\'update:modelValue\', $event.target.value)" />',
  },
}));

const workflows: Workflow[] = [
  {
    id: "review-template",
    name: "Review Template",
    description: "Validate and print the tracked source of truth.",
    changesMac: "No",
    phases: [
      {
        id: "validate-template-files",
        name: "Validate template files",
        enabled: true,
      },
      {
        id: "print-tracked-homebrew-bundle",
        name: "Print tracked Homebrew bundle",
        enabled: true,
      },
      { id: "list-tracked-apps", name: "List tracked apps", enabled: true },
      {
        id: "list-tracked-macos-settings",
        name: "List tracked macOS settings",
        enabled: true,
      },
      {
        id: "list-tracked-dotfile-bundles",
        name: "List tracked dotfile bundles",
        enabled: true,
      },
    ],
    confirmation: {
      title: "Review Template",
      message: "Validate and print the template.",
      options: [
        {
          id: "run-now",
          label: "Run now",
          description: "continue",
          continue: true,
          back: false,
          requiresApproval: false,
        },
        {
          id: "back",
          label: "Back",
          description: "return to workflow menu",
          continue: false,
          back: true,
          requiresApproval: false,
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
      {
        id: "save-current-mac-snapshot",
        name: "Save current Mac snapshot",
        enabled: true,
      },
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
          requiresApproval: false,
          phases: [
            {
              id: "save-current-mac-snapshot",
              name: "Save current Mac snapshot",
              enabled: true,
            },
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
          requiresApproval: false,
          phases: [
            {
              id: "save-current-mac-snapshot",
              name: "Save current Mac snapshot",
              enabled: true,
            },
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
        {
          id: "run-now",
          label: "Run now",
          description: "continue",
          continue: true,
          back: false,
          requiresApproval: false,
        },
      ],
    },
  },
  {
    id: "converge-to-template",
    name: "Converge to Template",
    description: "Install configured applications.",
    changesMac: "Yes",
    phases: [
      {
        id: "install-homebrew-apps",
        name: "Install Homebrew apps",
        enabled: true,
      },
    ],
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
          requiresApproval: true,
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
        path: "/repo/apps.generated.yaml",
        relative: "apps.generated.yaml",
        kind: "apps",
        size: 18194,
        exists: true,
      },
      {
        path: "/repo/apps.yaml",
        relative: "apps.yaml",
        kind: "apps",
        size: 10897,
        exists: true,
      },
      {
        path: "/repo/secrets.yaml",
        relative: "secrets.yaml",
        kind: "secrets",
        size: 271,
        exists: true,
      },
      {
        path: "/repo/stow/ghostty/.config/ghostty/config",
        relative: "stow/ghostty/.config/ghostty/config",
        kind: "stow",
        size: 1520,
        exists: true,
      },
      {
        path: "/repo/stow/git/.config/git/ignore",
        relative: "stow/git/.config/git/ignore",
        kind: "stow",
        size: 31,
        exists: true,
      },
      {
        path: "/repo/stow/shell/.zshrc",
        relative: "stow/shell/.zshrc",
        kind: "stow",
        size: 5069,
        exists: true,
      },
      {
        path: "/repo/stow/vim/.vimrc",
        relative: "stow/vim/.vimrc",
        kind: "stow",
        size: 2815,
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
        workflowDbPath: "/Users/gus/Library/Application Support/gus-mac/workflows.sqlite3",
        opVault: "Private",
        opItem: "Mac Migration Archive",
      },
      checks: [
        {
          key: "repo_root",
          label: "Repository root",
          path: "/repo",
          status: "ok",
          message: "ok",
        },
        {
          key: "workflow_db_path",
          label: "Workflow SQLite database",
          path: "/Users/gus/Library/Application Support/gus-mac/workflows.sqlite3",
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
    listOpItems: vi.fn().mockResolvedValue({
      ok: true,
      items: [{ id: "i1", title: "Mac Migration Archive" }],
    }),
    signinOpCli: vi.fn().mockResolvedValue({ ok: true }),
    installOpDependencies: vi.fn().mockResolvedValue({ ok: true }),
    openDevTools: vi.fn().mockResolvedValue(undefined),
    appDiagnostics: vi.fn().mockResolvedValue([]),
    onAppDiagnostic: vi.fn().mockReturnValue(() => {}),
    reportRendererError: vi.fn().mockResolvedValue(undefined),
    macName: vi.fn().mockResolvedValue("Local Mac"),
    macHostname: vi.fn().mockResolvedValue("localhost"),
    macSystemInfo: vi.fn().mockResolvedValue({
      name: "Local Mac",
      hostname: "localhost",
      osLabel: "macOS 15",
      architectureLabel: "Apple silicon",
      avatarUrl: accountAvatarUrl,
    }),
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
        onEvent({
          runId: "run-2",
          seq: 3,
          type: "run_finished",
          status: "completed",
        });

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

async function findWrapperButton(wrapper: ReturnType<typeof mount>, text: string) {
  for (let attempt = 0; attempt < 10; attempt += 1) {
    const button = wrapper.findAll("button").find((item) => item.text().includes(text));

    if (button) {
      return button;
    }

    await flushPromises();
    await new Promise((resolve) => setTimeout(resolve, 0));
  }

  return wrapper.findAll("button").find((item) => item.text().includes(text));
}

async function clickRunOutputSectionButton(text: string) {
  const buttons = [
    ...document.body.querySelectorAll('[data-testid="run-output-sections"] button'),
  ].filter((button) => button.textContent?.includes(text));
  const button = buttons.at(-1);

  if (button) {
    await new DOMWrapper(button).trigger("click");
  }
}

function findDialogButton(text: string) {
  return [
    ...(document.body
      .querySelector('[data-slot="alert-dialog-content"]')
      ?.querySelectorAll("button") ?? []),
  ].find((button) => button.textContent?.includes(text));
}

async function flushOutputHighlighting() {
  for (let i = 0; i < 10; i += 1) {
    await flushPromises();
    await new Promise((resolve) => setTimeout(resolve, 0));
  }
}

async function flushTemplateEditorImports() {
  await Promise.all([import("@app/TemplateFilesDetail.vue"), import("@app/MonacoFileEditor.vue")]);
  await flushPromises();
}

function mountController() {
  return mount(
    defineComponent({
      setup() {
        return {
          controller: useAppController(),
        };
      },
      template: '<button data-testid="reload" @click="controller.loadAll()">Reload</button>',
    }),
  );
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

    expect(wrapper.text()).toContain("Source");
    expect(wrapper.text()).toContain("Review Template");
  });

  it("shows the loaded template file count in the template files row", () => {
    const wrapper = mount(WorkflowListPanel, {
      props: {
        stepMeta: {
          id: "template",
          title: "Source",
          summary: "Review source files.",
          emptyMessage: "No source workflows.",
        },
        searchQuery: "",
        workflows: [workflows[0]],
        selectedWorkflowId: "",
        selectedTemplateFiles: false,
        templateFilesCount: 7,
        templateFilesLoaded: true,
        templateFilesLoading: false,
        workflowsLoading: false,
      },
    });

    const templateFilesButton = wrapper
      .findAll("button")
      .find((button) => button.text().includes("Template Files"));

    expect(templateFilesButton?.text()).toContain("7 files");
    expect(templateFilesButton?.text()).not.toContain("Editable source files");
  });

  it("renders a fixed sidebar and keeps resizing inside the workspace", async () => {
    installApi();

    const wrapper = mount(App);
    await flushPromises();

    const nav = wrapper.find("#mac-nav");

    expect(nav.exists()).toBe(true);
    expect(nav.element.tagName).toBe("ASIDE");
    expect(nav.classes()).toContain("w-[300px]");
    expect(nav.classes()).toContain("min-w-[300px]");
    expect(nav.classes()).toContain("max-w-[300px]");
    expect(nav.element.nextElementSibling?.tagName).toBe("MAIN");
    expect(wrapper.find('[data-testid="workspace-resize-handle"]').exists()).toBe(true);
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
    expect(wrapper.text()).not.toContain("No source workflows registered.");
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
    expect(wrapper.text()).toContain("macOS 15");
    expect(wrapper.text()).toContain("Apple silicon");
    expect(wrapper.text()).toContain("Review and maintain the tracked source of truth");
    expect(wrapper.findAll('[data-testid="machine-avatar"]')[0].find("img").attributes("src")).toBe(
      accountAvatarUrl,
    );
    expect(api.macName).toHaveBeenCalled();
    expect(api.macHostname).toHaveBeenCalled();
    expect(api.macSystemInfo).toHaveBeenCalled();
  });

  it("keeps the sidebar machine avatar fallback when no account avatar is available", async () => {
    installApi({
      macSystemInfo: vi.fn().mockResolvedValue({
        name: "Local Mac",
        hostname: "localhost",
        osLabel: "macOS 15",
        architectureLabel: "Apple silicon",
      }),
    });

    const wrapper = mount(App);
    await flushPromises();

    const sidebarAvatar = wrapper.findAll('[data-testid="machine-avatar"]')[0];
    expect(sidebarAvatar.find("img").exists()).toBe(false);
    expect(sidebarAvatar.find('[aria-label="Default machine avatar"]').exists()).toBe(true);
  });

  it("filters workflow list with search and category nav", async () => {
    installApi();

    const wrapper = mount(App);
    await flushPromises();

    await wrapper.find('[data-testid="app-search"]').setValue("does-not-exist");
    expect(wrapper.text()).toContain("No source workflows registered.");

    await wrapper.find('[data-testid="app-search"]').setValue("");
    await wrapper
      .findAll("button")
      .find((button) => /^Apply\s*\d*$/.test(button.text().trim()))
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
      .find((button) => /^Apply\s*\d*$/.test(button.text().trim()))
      ?.trigger("click");
    await flushPromises();

    expect(
      wrapper.findAll("button").some((button) => button.text().includes("Converge to Template")),
    ).toBe(true);

    await wrapper.find('[data-testid="app-search"]').setValue("snapshot");

    expect(
      wrapper.findAll("button").some((button) => button.text().includes("Converge to Template")),
    ).toBe(false);
    expect(wrapper.text()).toContain("No apply workflows registered.");

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
    expect(document.body.textContent).not.toContain("healthy");
    await clickRunOutputSectionButton("Step 1: Validate template files");
    await flushOutputHighlighting();
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
    expect(document.body.textContent).not.toContain("healthy");
    await clickRunOutputSectionButton("Step 1: Validate template files");
    await flushOutputHighlighting();
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
          onEvent({
            runId: "run-streamed",
            seq: 5,
            type: "run_finished",
            status: "completed",
          });

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
    expect(sections?.textContent).not.toContain("Ghostty");
    expect(sections?.textContent).not.toContain("Raycast");
    await clickRunOutputSectionButton("Step 3: List tracked apps");
    await flushOutputHighlighting();
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
          onEvent({
            runId: "run-failed",
            seq: 4,
            type: "run_failed",
            status: "failed",
          });

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
    expect(document.body.textContent).not.toContain("boom");
    await clickRunOutputSectionButton("Step 2: Print tracked Homebrew bundle");
    await flushOutputHighlighting();
    expect(document.body.textContent).toContain("boom");
    expect(document.body.textContent).toContain("failed");
  });

  it("loads and saves allowlisted template files", async () => {
    const api = installApi();

    const wrapper = mount(App);
    await flushPromises();

    await (await findWrapperButton(wrapper, "Template Files"))?.trigger("click");
    await flushPromises();
    await flushPromises();
    await new Promise((resolve) => setTimeout(resolve, 0));
    await flushPromises();
    await flushTemplateEditorImports();

    expect(api.templateFiles).toHaveBeenCalled();
    expect(api.readTemplateFile).toHaveBeenCalledWith("/repo/apps.generated.yaml");
    expect(wrapper.text()).toContain("apps.yaml");

    await wrapper
      .findAll("button")
      .find((button) => button.text().trim().startsWith("apps.yaml"))
      ?.trigger("click");
    await flushPromises();

    expect(wrapper.text()).not.toContain("Saved content loaded");
    expect(wrapper.text()).toContain("Tracked app manifest");
    expect(wrapper.text()).toContain("Defines the apps this template expects");

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
    expect(wrapper.text()).toContain("Template file saved");
  });

  it("selects the first template file when the editor opens", async () => {
    const api = installApi({
      readTemplateFile: vi.fn().mockImplementation(async (path: string) => ({
        file: {
          path,
          relative: path.replace("/repo/", ""),
          kind: "apps",
          size: 32,
          exists: true,
        },
        content: `${path}\n`,
      })),
    });

    const wrapper = mount(App);
    await flushPromises();

    await wrapper
      .findAll("button")
      .find((button) => button.text().includes("Template Files"))
      ?.trigger("click");
    await flushPromises();

    expect(api.readTemplateFile).toHaveBeenCalledWith("/repo/apps.generated.yaml");
    expect(wrapper.text()).toContain("Generated app inventory");
    expect(wrapper.text()).toContain("Lists apps detected from this Mac");
    expect(wrapper.text()).not.toContain("No file selected");
    expect(
      (wrapper.find('[data-testid="monaco-editor"]').element as HTMLTextAreaElement).value,
    ).toBe("/repo/apps.generated.yaml\n");
  });

  it("opens the first existing template file when earlier allowlisted files are missing", async () => {
    const api = installApi({
      templateFiles: vi.fn().mockResolvedValue([
        {
          path: "/repo/apps.generated.yaml",
          relative: "apps.generated.yaml",
          kind: "apps",
          size: 0,
          exists: false,
        },
        {
          path: "/repo/apps.yaml",
          relative: "apps.yaml",
          kind: "apps",
          size: 32,
          exists: true,
        },
      ]),
      readTemplateFile: vi.fn().mockImplementation(async (path: string) => ({
        file: {
          path,
          relative: path.replace("/repo/", ""),
          kind: "apps",
          size: 32,
          exists: true,
        },
        content: `${path}\n`,
      })),
    });

    const wrapper = mount(App);
    await flushPromises();

    await wrapper
      .findAll("button")
      .find((button) => button.text().includes("Template Files"))
      ?.trigger("click");
    await flushPromises();

    expect(api.readTemplateFile).toHaveBeenCalledWith("/repo/apps.yaml");
    expect(api.readTemplateFile).not.toHaveBeenCalledWith("/repo/apps.generated.yaml");
    expect(wrapper.text()).toContain("Tracked app manifest");
    expect(wrapper.text()).not.toContain("Template file error");
  });

  it("keeps the selected template file when the list is refreshed", async () => {
    const api = installApi({
      readTemplateFile: vi.fn().mockImplementation(async (path: string) => ({
        file: {
          path,
          relative: path.replace("/repo/", ""),
          kind: "apps",
          size: 32,
          exists: true,
        },
        content: `${path}\n`,
      })),
    });

    const wrapper = mount(App);
    await flushPromises();

    await wrapper
      .findAll("button")
      .find((button) => button.text().includes("Template Files"))
      ?.trigger("click");
    await flushPromises();

    await wrapper
      .findAll("button")
      .find((button) => button.text().trim().startsWith("apps.yaml"))
      ?.trigger("click");
    await flushPromises();

    await wrapper.find('button[aria-label="Refresh template files"]').trigger("click");
    await flushPromises();

    expect(api.readTemplateFile).toHaveBeenLastCalledWith("/repo/apps.yaml");
    expect(wrapper.text()).toContain("Tracked app manifest");
    expect(
      (wrapper.find('[data-testid="monaco-editor"]').element as HTMLTextAreaElement).value,
    ).toBe("/repo/apps.yaml\n");
  });

  it("falls back to the first template file when the selected file disappears", async () => {
    const refreshedFiles = [
      {
        path: "/repo/secrets.yaml",
        relative: "secrets.yaml",
        kind: "secrets",
        size: 271,
        exists: true,
      },
    ];
    const api = installApi({
      templateFiles: vi
        .fn()
        .mockResolvedValueOnce([
          {
            path: "/repo/apps.generated.yaml",
            relative: "apps.generated.yaml",
            kind: "apps",
            size: 18194,
            exists: true,
          },
          {
            path: "/repo/apps.yaml",
            relative: "apps.yaml",
            kind: "apps",
            size: 10897,
            exists: true,
          },
        ])
        .mockResolvedValue(refreshedFiles),
      readTemplateFile: vi.fn().mockImplementation(async (path: string) => ({
        file: {
          path,
          relative: path.replace("/repo/", ""),
          kind: path.includes("secrets") ? "secrets" : "apps",
          size: 32,
          exists: true,
        },
        content: `${path}\n`,
      })),
    });

    const wrapper = mount(App);
    await flushPromises();

    await wrapper
      .findAll("button")
      .find((button) => button.text().includes("Template Files"))
      ?.trigger("click");
    await flushPromises();

    await wrapper
      .findAll("button")
      .find((button) => button.text().trim().startsWith("apps.yaml"))
      ?.trigger("click");
    await flushPromises();

    await wrapper.find('button[aria-label="Refresh template files"]').trigger("click");
    await flushPromises();

    expect(api.readTemplateFile).toHaveBeenLastCalledWith("/repo/secrets.yaml");
    expect(wrapper.text()).toContain("Secret reference manifest");
    expect(
      (wrapper.find('[data-testid="monaco-editor"]').element as HTMLTextAreaElement).value,
    ).toBe("/repo/secrets.yaml\n");
  });

  it("preserves the active workflow when data is reloaded", async () => {
    installApi();

    const wrapper = mountController();
    await flushPromises();
    const controller = (
      wrapper.vm as unknown as { controller: ReturnType<typeof useAppController> }
    ).controller;

    controller.selectWorkflow(workflows[1]);
    await wrapper.find('[data-testid="reload"]').trigger("click");
    await flushPromises();

    expect(controller.selectedWorkflowId.value).toBe("update-template-from-this-mac");
  });

  it("does not select the first workflow during reload while the template editor is active", async () => {
    installApi();

    const wrapper = mountController();
    await flushPromises();
    const controller = (
      wrapper.vm as unknown as { controller: ReturnType<typeof useAppController> }
    ).controller;

    controller.closeDetailPane();
    await controller.selectTemplateFiles();
    await wrapper.find('[data-testid="reload"]').trigger("click");
    await flushPromises();

    expect(controller.selectedTemplateFiles.value).toBe(true);
    expect(controller.selectedWorkflowId.value).toBe("");
  });

  it("does not show the template file editor action on the review template workflow", async () => {
    installApi();

    const wrapper = mount(App);
    await flushPromises();

    expect(wrapper.text()).toContain("Review Template");
    expect(wrapper.text()).not.toContain("Update Template Files");
    expect(wrapper.find("#template-editor").exists()).toBe(false);
  });

  it("opens template files from an update template workflow in the expanded editor", async () => {
    const api = installApi();

    const wrapper = mount(App);
    await flushPromises();

    await wrapper
      .findAll("button")
      .find((button) => button.text().includes("Update Template From This Mac"))
      ?.trigger("click");
    await flushPromises();

    expect(wrapper.text()).toContain("Update Template Files");

    await wrapper
      .findAll("button")
      .find((button) => button.text().trim().startsWith("Update Template Files"))
      ?.trigger("click");
    await flushPromises();

    expect(api.templateFiles).toHaveBeenCalled();
    expect(wrapper.find("#template-editor").exists()).toBe(true);
    expect(wrapper.find("#mac-list").exists()).toBe(false);
    expect(wrapper.find('[data-testid="expanded-template-editor"]').exists()).toBe(true);
    expect(wrapper.text()).toContain("apps.yaml");
    expect(wrapper.find('[data-testid="template-file-icon-generated-apps"]').exists()).toBe(true);
    expect(wrapper.find('[data-testid="template-file-icon-apps"]').exists()).toBe(true);
    expect(wrapper.find('[data-testid="template-file-icon-secrets"]').exists()).toBe(true);
    expect(wrapper.find('[data-testid="template-file-icon-terminal-config"]').exists()).toBe(true);
    expect(wrapper.find('[data-testid="template-file-icon-git-config"]').exists()).toBe(true);
    expect(wrapper.find('[data-testid="template-file-icon-shell-config"]').exists()).toBe(true);
    expect(wrapper.find('[data-testid="template-file-icon-vim-config"]').exists()).toBe(true);
  });

  it("returns to the previous workflow when Back is pressed from the expanded editor", async () => {
    installApi();

    const wrapper = mount(App);
    await flushPromises();

    await wrapper
      .findAll("button")
      .find((button) => button.text().includes("Update Template From This Mac"))
      ?.trigger("click");
    await flushPromises();

    await wrapper
      .findAll("button")
      .find((button) => button.text().trim().startsWith("Update Template Files"))
      ?.trigger("click");
    await flushPromises();

    await wrapper
      .findAll("button")
      .find((button) => button.text().trim() === "Back")
      ?.trigger("click");
    await flushPromises();

    expect(wrapper.find("#template-editor").exists()).toBe(false);
    expect(wrapper.find("#mac-detail").exists()).toBe(true);
    expect(wrapper.text()).toContain("Update Template From This Mac");
    expect(wrapper.text()).toContain("Save current Mac snapshot");
  });

  it("prompts before Back discards dirty template file edits", async () => {
    installApi();

    const wrapper = mount(App);
    await flushPromises();

    await wrapper
      .findAll("button")
      .find((button) => button.text().includes("Update Template From This Mac"))
      ?.trigger("click");
    await flushPromises();

    await wrapper
      .findAll("button")
      .find((button) => button.text().trim().startsWith("Update Template Files"))
      ?.trigger("click");
    await flushPromises();
    await wrapper
      .findAll("button")
      .find((button) => button.text().trim().startsWith("apps.yaml"))
      ?.trigger("click");
    await flushPromises();

    await wrapper.find('[data-testid="monaco-editor"]').setValue("apps:\n  - name: Raycast\n");
    await wrapper
      .findAll("button")
      .find((button) => button.text().trim() === "Back")
      ?.trigger("click");
    await flushPromises();

    expect(document.body.textContent).toContain("Discard unsaved changes?");
    expect(wrapper.find("#template-editor").exists()).toBe(true);

    findDocumentButton("Discard changes")?.click();
    await flushPromises();

    expect(wrapper.find("#template-editor").exists()).toBe(false);
    expect(wrapper.text()).toContain("Update Template From This Mac");
  });

  it("prompts before Cancel discards dirty template file edits and stays in the editor", async () => {
    installApi();

    const wrapper = mount(App);
    await flushPromises();

    await wrapper
      .findAll("button")
      .find((button) => button.text().includes("Update Template From This Mac"))
      ?.trigger("click");
    await flushPromises();

    await wrapper
      .findAll("button")
      .find((button) => button.text().trim().startsWith("Update Template Files"))
      ?.trigger("click");
    await flushPromises();
    await wrapper
      .findAll("button")
      .find((button) => button.text().trim().startsWith("apps.yaml"))
      ?.trigger("click");
    await flushPromises();

    const editor = wrapper.find('[data-testid="monaco-editor"]');
    await editor.setValue("apps:\n  - name: Raycast\n");
    await wrapper
      .findAll("button")
      .find((button) => button.text().trim() === "Cancel")
      ?.trigger("click");
    await flushPromises();

    expect(document.body.textContent).toContain("Discard unsaved changes?");

    findDocumentButton("Discard changes")?.click();
    await flushPromises();

    expect(wrapper.find("#template-editor").exists()).toBe(true);
    expect(
      (wrapper.find('[data-testid="monaco-editor"]').element as HTMLTextAreaElement).value,
    ).toBe("apps:\n  - name: Ghostty\n");
    expect(wrapper.text()).not.toContain("Template file changes discarded.");
  });

  it("keeps the expanded editor hidden while template files load", async () => {
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

    expect(wrapper.find('[data-testid="expanded-template-editor"]').exists()).toBe(false);
    expect(wrapper.find("#mac-detail").exists()).toBe(true);

    resolveFiles([]);
    await flushPromises();

    expect(wrapper.find('[data-testid="expanded-template-editor"]').exists()).toBe(true);
    expect(wrapper.text()).toContain("Choose a file from the list to inspect or edit it.");
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
    const toolbar = wrapper.find('[data-testid="detail-toolbar"]');
    expect(toolbar.text()).toContain("Check Setup");
    expect(toolbar.text()).toContain("completed");
    await flushOutputHighlighting();
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
    await flushPromises();
    await flushOutputHighlighting();

    expect(wrapper.text()).toContain("red output");
    expect(wrapper.text()).not.toContain("\u001B");
    expect(wrapper.html()).toContain("shiki");
  });

  it("filters workflows by sidebar step", async () => {
    installApi();

    const wrapper = mount(App);
    await flushPromises();

    // Default section is Source
    expect(wrapper.text()).toContain("Review Template");

    await wrapper
      .findAll("button")
      .find((button) => /^This Mac\s*\d*$/.test(button.text().trim()))
      ?.trigger("click");
    await flushPromises();

    expect(wrapper.text()).toContain("Inspect Current State");
    expect(wrapper.text()).not.toContain("No This Mac workflows registered.");

    await wrapper
      .findAll("button")
      .find((button) => /^Apply\s*\d*$/.test(button.text().trim()))
      ?.trigger("click");
    await flushPromises();

    expect(wrapper.text()).toContain("Converge to Template");
  });

  it("does not render settings or DevTools cards in workflow lists", async () => {
    installApi();

    const wrapper = mount(App);
    await flushPromises();

    for (const buttonText of ["Source", "This Mac", "Apply"]) {
      await wrapper
        .findAll("button")
        .find((button) => button.text().trim().startsWith(buttonText))
        ?.trigger("click");
      await flushPromises();

      const listText = wrapper.find("#mac-list").text();
      expect(listText).not.toContain("Step settings");
      expect(listText).not.toContain("Repository root");
      expect(listText).not.toContain("Archive root");
      expect(listText).not.toContain("1Password vault");
      expect(listText).not.toContain("DevTools");
    }
  });

  it("opens DevTools from the sidebar without changing sections", async () => {
    const api = installApi();

    const wrapper = mount(App);
    await flushPromises();

    await wrapper
      .findAll("button")
      .find((button) => /^This Mac\s*\d*$/.test(button.text().trim()))
      ?.trigger("click");
    await flushPromises();

    await wrapper
      .findAll("button")
      .find((button) => button.text().trim() === "DevTools")
      ?.trigger("click");
    await flushPromises();

    expect(api.openDevTools).toHaveBeenCalled();
    expect(wrapper.find("#mac-list").text()).toContain("Inspect Current State");
    expect(wrapper.find("#mac-list").text()).not.toContain("Review Template");
  });

  it("does not show the template file editor action on non-template workflows", async () => {
    installApi();

    const wrapper = mount(App);
    await flushPromises();

    await wrapper
      .findAll("button")
      .find((button) => /^This Mac\s*\d*$/.test(button.text().trim()))
      ?.trigger("click");
    await flushPromises();
    await wrapper
      .findAll("button")
      .find((button) => button.text().includes("Inspect Current State"))
      ?.trigger("click");
    await flushPromises();

    expect(wrapper.text()).not.toContain("Update Template Files");
  });

  it("maps template file names to Monaco languages", () => {
    expect(templateFileLanguage("/repo/apps.yaml")).toBe("yaml");
    expect(templateFileLanguage("/repo/secrets.yml")).toBe("yaml");
    expect(templateFileLanguage("/repo/stow/shell/.zshrc")).toBe("shell");
    expect(templateFileLanguage("/repo/stow/shell/.bash_profile")).toBe("shell");
    expect(templateFileLanguage("/repo/storage/archives/files/install.sh")).toBe("shell");
    expect(templateFileLanguage("/repo/editors/vscode/settings.json")).toBe("json");
    expect(templateFileLanguage("/repo/stow/git/.gitconfig")).toBe("plaintext");
    expect(templateFileLanguage("/repo/stow/ghostty/.config/ghostty/config")).toBe("plaintext");
  });

  it("renders settings and saves changed workflow database configuration", async () => {
    const api = installApi();

    const wrapper = mount(App);
    await flushPromises();

    await wrapper
      .findAll("button")
      .find((button) => button.text().includes("Settings"))
      ?.trigger("click");
    await flushPromises();

    expect(wrapper.text()).toContain("Workflow SQLite database");
    await wrapper.find('[data-testid="settings-workflow-db"]').setValue("/db-next.sqlite3");
    await wrapper
      .findAll("button")
      .find((button) => button.text().includes("Save settings"))
      ?.trigger("click");
    await flushPromises();

    expect(document.body.textContent).toContain("Save settings?");
    expect(api.saveSettings).not.toHaveBeenCalled();

    findDialogButton("Save settings")?.click();
    await flushPromises();

    expect(api.saveSettings).toHaveBeenCalledWith(
      expect.objectContaining({
        repoRoot: "/repo",
        workflowDbPath: "/db-next.sqlite3",
      }),
    );
    expect(wrapper.text()).toContain("Settings saved");
  });

  it("cancels settings save confirmation without persisting changes", async () => {
    const api = installApi();

    const wrapper = mount(App);
    await flushPromises();

    await wrapper
      .findAll("button")
      .find((button) => button.text().includes("Settings"))
      ?.trigger("click");
    await flushPromises();

    await wrapper.find('[data-testid="settings-workflow-db"]').setValue("/db-next.sqlite3");
    await wrapper
      .findAll("button")
      .find((button) => button.text().includes("Save settings"))
      ?.trigger("click");
    await flushPromises();

    expect(document.body.textContent).toContain("Save settings?");

    findDialogButton("Cancel")?.click();
    await flushPromises();

    expect(api.saveSettings).not.toHaveBeenCalled();
    expect(
      wrapper.find<HTMLInputElement>('[data-testid="settings-workflow-db"]').element.value,
    ).toBe("/db-next.sqlite3");
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
        {
          key: "repo_root",
          label: "Repository root",
          path: "/repo",
          status: "ok",
          message: "ok",
        },
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
            key: "workflow_db",
            label: "Workflow database",
            path: settings.workflowDbPath,
            status: "error",
            message: "missing workflow database",
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
    await wrapper.find('[data-testid="settings-workflow-db"]').setValue("/broken");
    await wrapper
      .findAll("button")
      .find((button) => button.text().includes("Save settings"))
      ?.trigger("click");
    await flushPromises();

    findDialogButton("Save settings")?.click();
    await flushPromises();

    expect(api.saveSettings).toHaveBeenCalledWith(
      expect.objectContaining({ workflowDbPath: "/broken" }),
    );
    expect(
      wrapper.find<HTMLInputElement>('[data-testid="settings-workflow-db"]').element.value,
    ).toBe("/broken");
    expect(wrapper.text()).toContain("missing workflow database");
  });

  it("shows skeleton placeholders for group counters while validation is in flight", async () => {
    let resolveValidate: ((value: SettingsResponse) => void) | undefined;
    installApi({
      validateSettings: vi.fn(
        () =>
          new Promise<SettingsResponse>((resolve) => {
            resolveValidate = resolve;
          }),
      ),
    });

    const wrapper = mount(App);
    await flushPromises();

    await wrapper
      .findAll("button")
      .find((button) => button.text().includes("Settings"))
      ?.trigger("click");
    await flushPromises();

    expect(wrapper.find('[data-testid="settings-group-repository"]').text()).toContain("1/1");

    await wrapper
      .findAll("button")
      .find((button) => button.text().trim() === "Validate")
      ?.trigger("click");
    await flushPromises();

    expect(wrapper.find('[data-testid="settings-group-repository-badge-skeleton"]').exists()).toBe(
      true,
    );
    expect(
      wrapper.find('[data-testid="settings-group-repository-subtitle-skeleton"]').exists(),
    ).toBe(true);
    expect(wrapper.find('[data-testid="settings-group-repository"]').text()).not.toContain("1/1");

    resolveValidate?.({
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
        {
          key: "repo_root",
          label: "Repository root",
          path: "/repo",
          status: "ok",
          message: "ok",
        },
      ],
    });
    await flushPromises();

    expect(wrapper.find('[data-testid="settings-group-repository-badge-skeleton"]').exists()).toBe(
      false,
    );
    expect(wrapper.find('[data-testid="settings-group-repository"]').text()).toContain("1/1");
  });

  it("scrolls the matching settings section into view when a middle-column group is clicked", async () => {
    installApi();

    const scrollIntoView = vi.fn();
    const original = Element.prototype.scrollIntoView;
    Element.prototype.scrollIntoView = scrollIntoView;

    try {
      const wrapper = mount(App, { attachTo: document.body });
      await flushPromises();

      await wrapper
        .findAll("button")
        .find((button) => button.text().includes("Settings"))
        ?.trigger("click");
      await flushPromises();

      const repoGroup = wrapper.find('[data-testid="settings-group-repository"]');
      expect(repoGroup.exists()).toBe(true);
      expect(repoGroup.text()).toContain("1/1");
      expect(repoGroup.text()).toContain("All checks passing");

      await repoGroup.trigger("click");
      await flushPromises();

      const repoCard = document.getElementById("settings-section-repository");
      expect(repoCard).not.toBeNull();
      expect(scrollIntoView).toHaveBeenCalled();
      expect(scrollIntoView.mock.instances[0]).toBe(repoCard);

      await wrapper.find('[data-testid="settings-group-storage"]').trigger("click");
      await flushPromises();

      expect(scrollIntoView).toHaveBeenCalledTimes(2);
      const storageCard = document.getElementById("settings-section-storage");
      expect(scrollIntoView.mock.instances[1]).toBe(storageCard);

      wrapper.unmount();
    } finally {
      Element.prototype.scrollIntoView = original;
    }
  });
});
