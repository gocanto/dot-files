import type {
  MacOSApi,
  RunEvent,
  RunLog,
  RunSummary,
  RuntimeSettings,
  SettingsResponse,
  TemplateFileContent,
  TemplateFileSummary,
  UserPreferences,
  Workflow,
} from "@api";

const fallbackWorkflows: Workflow[] = [
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
      message: "Validate and print the tracked source of truth.",
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
    description: "Save this Mac and generate review-candidate template updates.",
    changesMac: "Writes review candidates",
    phases: [
      { id: "save-current-mac-snapshot", name: "Save current Mac snapshot", enabled: true },
      {
        id: "generate-installed-app-review-candidate",
        name: "Generate installed app review candidate",
        enabled: true,
      },
      {
        id: "generate-dotfile-review-candidates",
        name: "Generate dotfile review candidates",
        enabled: true,
      },
    ],
    confirmation: {
      title: "Update Template From This Mac",
      message: "Generate review candidates without overwriting tracked template files.",
      options: [
        {
          id: "preview-only",
          label: "Preview only",
          description: "show what would happen",
          continue: true,
          back: false,
        },
        {
          id: "run-now",
          label: "Run now",
          description: "save snapshot and write candidates",
          continue: true,
          back: false,
        },
      ],
    },
  },
  {
    id: "inspect-current-state",
    name: "Inspect Current State",
    description: "Check whether prerequisites, tools, and expected setup state look correct.",
    changesMac: "No",
    phases: [{ id: "run-health-checks", name: "Run health checks", enabled: true }],
    confirmation: {
      title: "Check Setup",
      message: "Run health checks only.",
      options: [
        { id: "run-now", label: "Run now", description: "continue", continue: true, back: false },
      ],
    },
  },
];

export function installBrowserFallback() {
  if (window.macOS) {
    return;
  }

  const fallbackRuns: RunLog[] = [];
  let fallbackPreferences: UserPreferences = { theme: "light" };
  let fallbackSettings: RuntimeSettings = {
    repoRoot: "/repo",
    appsConfigPath: "/repo/apps.yaml",
    secretsConfigPath: "/repo/secrets.yaml",
    generatedAppsPath: "/repo/apps.generated.yaml",
    archiveRoot: "/Users/local/.local/state/macos-settings-archives",
    workflowDbPath: "/Users/local/Library/Application Support/mac-os/workflows.sqlite3",
    opVault: "Private",
    opItem: "Mac Migration Archive",
  };
  let fallbackTemplateFiles: TemplateFileContent[] = [
    {
      file: {
        path: "/repo/apps.yaml",
        relative: "apps.yaml",
        kind: "apps",
        size: 48,
        exists: true,
      },
      content: "apps:\n  - name: Ghostty\n    install_method: brew\n",
    },
    {
      file: {
        path: "/repo/stow/shell/.zshrc",
        relative: "stow/shell/.zshrc",
        kind: "stow",
        size: 19,
        exists: true,
      },
      content: "export EDITOR=vim\n",
    },
  ];

  const settingsResponse = (settings: RuntimeSettings): SettingsResponse => ({
    settings,
    valid: true,
    checks: [
      {
        key: "repo_root",
        label: "Repository root",
        path: settings.repoRoot,
        status: "ok",
        message: "ok",
      },
      {
        key: "apps_config_path",
        label: "Apps manifest",
        path: settings.appsConfigPath,
        status: "ok",
        message: "ok",
      },
      {
        key: "secrets_config_path",
        label: "Secrets manifest",
        path: settings.secretsConfigPath,
        status: "ok",
        message: "ok",
      },
      {
        key: "workflow_db_path",
        label: "Workflow SQLite database",
        path: settings.workflowDbPath,
        status: "ok",
        message: "ok",
      },
    ],
  });

  const api: MacOSApi = {
    workflows: async () => fallbackWorkflows,
    runs: async (limit = 25) => fallbackRuns.map((run) => run.run).slice(0, limit),
    templateFiles: async (): Promise<TemplateFileSummary[]> =>
      fallbackTemplateFiles.map((entry) => entry.file),
    readTemplateFile: async (path): Promise<TemplateFileContent> => {
      const file = fallbackTemplateFiles.find((entry) => entry.file.path === path);

      if (!file) {
        throw new Error(`Template file not found: ${path}`);
      }

      return file;
    },
    saveTemplateFile: async (path, content): Promise<TemplateFileContent> => {
      const index = fallbackTemplateFiles.findIndex((entry) => entry.file.path === path);

      if (index < 0) {
        throw new Error(`Template file not found: ${path}`);
      }

      const current = fallbackTemplateFiles[index];
      const next = {
        file: {
          ...current.file,
          size: new TextEncoder().encode(content).length,
          modifiedAt: new Date().toISOString(),
        },
        content,
      };

      fallbackTemplateFiles = [
        ...fallbackTemplateFiles.slice(0, index),
        next,
        ...fallbackTemplateFiles.slice(index + 1),
      ];

      return next;
    },
    settings: async () => settingsResponse(fallbackSettings),
    validateSettings: async (settings) => settingsResponse(settings),
    saveSettings: async (settings) => {
      fallbackSettings = settings;

      return settingsResponse(fallbackSettings);
    },
    chooseDirectory: async (defaultPath) => defaultPath ?? fallbackSettings.repoRoot,
    chooseFile: async (defaultPath) => defaultPath ?? fallbackSettings.appsConfigPath,
    chooseSaveFile: async (defaultPath) => defaultPath ?? fallbackSettings.workflowDbPath,
    listOpVaults: async () => ({
      ok: true,
      vaults: [
        { id: "v-private", name: "Private" },
        { id: "v-shared", name: "Shared" },
      ],
    }),
    listOpItems: async (vault) => ({
      ok: true,
      items:
        vault === "Private"
          ? [
              { id: "i-mac", title: "Mac Migration Archive" },
              { id: "i-github", title: "GitHub" },
            ]
          : [{ id: "i-shared", title: "Team Secrets" }],
    }),
    signinOpCli: async () => ({ ok: true }),
    installOpDependencies: async () => ({ ok: true }),
    openDevTools: async () => {
      window.open("", "mac-os-manager-devtools");
    },
    macName: async () => "Local Mac",
    macHostname: async () => "localhost",
    macSystemInfo: async () => ({
      name: "Local Mac",
      hostname: "localhost",
      osLabel: "macOS 15",
      architectureLabel: "Apple silicon",
    }),
    getUserPreferences: async () => fallbackPreferences,
    saveUserPreferences: async (theme) => {
      fallbackPreferences = { theme, updatedAt: new Date().toISOString() };

      return fallbackPreferences;
    },
    runLog: async (runId) => {
      const run = fallbackRuns.find((entry) => entry.run.id === runId);

      if (!run) {
        throw new Error(`Run not found: ${runId}`);
      }

      return run;
    },
    runWorkflow: async (request, onEvent) => {
      const runId = crypto.randomUUID();
      const startedAt = new Date().toISOString();
      const workflow = fallbackWorkflows.find((entry) => entry.id === request.workflowId);
      const option = workflow?.confirmation?.options.find(
        (entry) => entry.id === request.confirmationOptionId,
      );
      let seq = 1;
      const events: RunEvent[] = [
        { runId, seq: seq++, type: "run_started", status: "running", message: request.workflowId },
      ];

      for (const phaseId of request.enabledPhaseIds) {
        const phaseName = workflow?.phases.find((phase) => phase.id === phaseId)?.name;

        events.push({
          runId,
          seq: seq++,
          type: "phase_started",
          phaseId,
          phaseName,
          status: "running",
        });
        events.push({
          runId,
          seq: seq++,
          type: "phase_output",
          phaseId,
          phaseName,
          message:
            phaseId === "validate-template-files" ? "validation complete" : "preview complete",
        });
        events.push({
          runId,
          seq: seq++,
          type: "phase_finished",
          phaseId,
          phaseName,
          status: "ok",
        });
      }

      events.push({ runId, seq: seq++, type: "run_finished", status: "completed" });

      for (const event of events) {
        onEvent(event);
      }

      const completedAt = new Date().toISOString();
      const run: RunSummary = {
        id: runId,
        workflowId: request.workflowId,
        workflowName: workflow?.name ?? request.workflowId,
        confirmationOptionId: request.confirmationOptionId,
        confirmationOptionLabel: option?.label ?? request.confirmationOptionId,
        mode: option?.id === "run-now" ? "live" : "preview",
        status: "completed",
        startedAt,
        completedAt,
      };

      fallbackRuns.unshift({
        run,
        events: events.map((event, index) => ({
          ...event,
          id: index + 1,
          createdAt: completedAt,
        })),
      });

      return { exitCode: 0 };
    },
  };

  window.macOS = api;
}
