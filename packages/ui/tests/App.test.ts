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
});
