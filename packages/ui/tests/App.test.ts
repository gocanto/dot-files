import { flushPromises, mount } from "@vue/test-utils";
import { beforeEach, describe, expect, it, vi } from "vitest";
import App from "../src/App.vue";
import type { MacOSApi, RunEvent, Workflow } from "../src/types/api";

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

describe("App", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  it("renders workflow navigation and details", async () => {
    installApi();

    const wrapper = mount(App);
    await flushPromises();

    expect(wrapper.text()).toContain("Workflows");
    expect(wrapper.text()).toContain("Check Setup");
    expect(wrapper.text()).toContain("Run health checks");
  });

  it("runs a workflow and appends streamed output", async () => {
    const api = installApi();

    const wrapper = mount(App);
    await flushPromises();

    await wrapper.findAll("button").find((button) => button.text().includes("Run now"))?.trigger("click");
    await wrapper.findAll("button").find((button) => button.text().includes("Continue"))?.trigger("click");
    await flushPromises();

    expect(api.runWorkflow).toHaveBeenCalledWith(
      { workflowId: "check-setup", confirmationOptionId: "run-now", enabledPhaseIds: ["run-health-checks"] },
      expect.any(Function),
    );
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
    expect(wrapper.text()).toContain("live");
  });
});
