import type { MacOSApi, RunEvent, Workflow } from "../types/api";

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

  const api: MacOSApi = {
    workflows: async () => fallbackWorkflows,
    runs: async () => [],
    runLog: async (runId) => ({
      run: {
        id: runId,
        workflowId: "check-setup",
        workflowName: "Check Setup",
        confirmationOptionId: "run-now",
        confirmationOptionLabel: "Run now",
        mode: "preview",
        status: "completed",
        startedAt: new Date().toISOString(),
      },
      events: [],
    }),
    runWorkflow: async (request, onEvent) => {
      const runId = crypto.randomUUID();
      let seq = 1;
      const events: RunEvent[] = [{ runId, seq: seq++, type: "run_started", status: "running", message: request.workflowId }];

      for (const phaseId of request.enabledPhaseIds) {
        events.push({ runId, seq: seq++, type: "phase_started", phaseId, status: "running" });
        events.push({ runId, seq: seq++, type: "phase_output", phaseId, message: "preview complete" });
        events.push({ runId, seq: seq++, type: "phase_finished", phaseId, status: "ok" });
      }

      events.push({ runId, seq: seq++, type: "run_finished", status: "completed" });

      for (const event of events) {
        onEvent(event);
      }

      return { exitCode: 0 };
    },
  };

  window.macOS = api;
}
