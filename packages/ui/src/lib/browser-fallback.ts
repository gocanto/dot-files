import type { MacOSApi, RunEvent, RunLog, RunSummary, RuntimeSettings, SettingsResponse, UserPreferences, Workflow } from "../types/api";

const fallbackWorkflows: Workflow[] = [
  {
    id: "set-up-this-mac",
    name: "Set Up This Mac",
    description: "Run the complete setup flow for this Mac.",
    changesMac: "Yes",
    phases: [
      { id: "check-install-prerequisites", name: "Check/install prerequisites", enabled: true },
      { id: "install-homebrew-packages", name: "Install Homebrew packages", enabled: true },
      { id: "run-health-checks", name: "Run health checks", enabled: true },
    ],
    confirmation: {
      title: "Set Up This Mac",
      message: "Run the complete setup flow for a clean or intentionally reconfigured Mac.",
      options: [
        { id: "preview-only", label: "Preview only", description: "show what would happen", continue: true, back: false },
        { id: "run-now", label: "Run now", description: "make the described changes", continue: true, back: false },
      ],
    },
  },
  {
    id: "check-setup",
    name: "Check Setup",
    description: "Check whether prerequisites, tools, and expected setup state look correct.",
    changesMac: "No",
    phases: [{ id: "run-health-checks", name: "Run health checks", enabled: true }],
    confirmation: {
      title: "Check Setup",
      message: "Run health checks only.",
      options: [{ id: "run-now", label: "Run now", description: "continue", continue: true, back: false }],
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

  const settingsResponse = (settings: RuntimeSettings): SettingsResponse => ({
    settings,
    valid: true,
    checks: [
      { key: "repo_root", label: "Repository root", path: settings.repoRoot, status: "ok", message: "ok" },
      { key: "apps_config_path", label: "Apps manifest", path: settings.appsConfigPath, status: "ok", message: "ok" },
      { key: "secrets_config_path", label: "Secrets manifest", path: settings.secretsConfigPath, status: "ok", message: "ok" },
      { key: "workflow_db_path", label: "Workflow SQLite database", path: settings.workflowDbPath, status: "ok", message: "ok" },
    ],
  });

  const api: MacOSApi = {
    workflows: async () => fallbackWorkflows,
    runs: async (limit = 25) => fallbackRuns.map((run) => run.run).slice(0, limit),
    settings: async () => settingsResponse(fallbackSettings),
    validateSettings: async (settings) => settingsResponse(settings),
    saveSettings: async (settings) => {
      fallbackSettings = settings;

      return settingsResponse(fallbackSettings);
    },
    chooseDirectory: async (defaultPath) => defaultPath ?? fallbackSettings.repoRoot,
    chooseFile: async (defaultPath) => defaultPath ?? fallbackSettings.appsConfigPath,
    chooseSaveFile: async (defaultPath) => defaultPath ?? fallbackSettings.workflowDbPath,
    macName: async () => "Local Mac",
    macHostname: async () => "localhost",
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
      const option = workflow?.confirmation?.options.find((entry) => entry.id === request.confirmationOptionId);
      let seq = 1;
      const events: RunEvent[] = [{ runId, seq: seq++, type: "run_started", status: "running", message: request.workflowId }];

      for (const phaseId of request.enabledPhaseIds) {
        const phaseName = workflow?.phases.find((phase) => phase.id === phaseId)?.name;

        events.push({ runId, seq: seq++, type: "phase_started", phaseId, phaseName, status: "running" });
        events.push({ runId, seq: seq++, type: "phase_output", phaseId, phaseName, message: "preview complete" });
        events.push({ runId, seq: seq++, type: "phase_finished", phaseId, phaseName, status: "ok" });
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
