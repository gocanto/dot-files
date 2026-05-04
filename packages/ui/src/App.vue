<script setup lang="ts">
import {
  AlertTriangle,
  CheckCircle2,
  ChevronRight,
  Circle,
  Loader2,
  Play,
  RotateCcw,
} from "lucide-vue-next";
import { computed, onMounted, ref } from "vue";
import AppSidebar from "./components/AppSidebar.vue";
import type { ConfirmationOption, Phase, RunEvent, RunLog, RunSummary, Workflow } from "./types/api";

const section = ref("workflows");
const workflows = ref<Workflow[]>([]);
const runs = ref<RunSummary[]>([]);
const selectedWorkflowId = ref("");
const enabledPhaseIds = ref<Set<string>>(new Set());
const pendingOption = ref<ConfirmationOption | null>(null);
const runEvents = ref<RunEvent[]>([]);
const running = ref(false);
const loadError = ref("");
const selectedRunId = ref("");
const selectedRunLog = ref<RunLog | null>(null);

const selectedWorkflow = computed(() => workflows.value.find((workflow) => workflow.id === selectedWorkflowId.value));

const displayPhases = computed(() => {
  if (!selectedWorkflow.value) {
    return [];
  }

  return selectedWorkflow.value.phases.map((phase) => ({
    ...phase,
    enabled: enabledPhaseIds.value.has(phase.id),
  }));
});

const runStatus = computed(() => {
  const last = [...runEvents.value].reverse().find((event) => event.type.startsWith("run_"));

  return last?.status ?? (running.value ? "running" : "idle");
});

onMounted(async () => {
  await loadAll();
});

async function loadAll() {
  try {
    workflows.value = await window.macOS.workflows();
    runs.value = await window.macOS.runs(25);
    selectedWorkflowId.value = workflows.value[0]?.id ?? "";
    resetEnabledPhases();
  } catch (error) {
    loadError.value = error instanceof Error ? error.message : String(error);
  }
}

function selectSection(next: string) {
  section.value = next;

  if (next === "logs") {
    void refreshRuns();
  }
}

function selectWorkflow(workflow: Workflow) {
  selectedWorkflowId.value = workflow.id;
  resetEnabledPhases();
  runEvents.value = [];
}

function resetEnabledPhases() {
  enabledPhaseIds.value = new Set(selectedWorkflow.value?.phases.filter((phase) => phase.enabled).map((phase) => phase.id));
}

function togglePhase(phase: Phase) {
  const next = new Set(enabledPhaseIds.value);

  if (next.has(phase.id)) {
    next.delete(phase.id);
  } else {
    next.add(phase.id);
  }

  enabledPhaseIds.value = next;
}

function openConfirmation(option?: ConfirmationOption) {
  if (!selectedWorkflow.value?.confirmation) {
    return;
  }

  pendingOption.value = option ?? selectedWorkflow.value.confirmation.options[0] ?? null;
}

async function runSelected(option: ConfirmationOption) {
  if (!selectedWorkflow.value || option.back) {
    pendingOption.value = null;

    return;
  }

  pendingOption.value = null;
  running.value = true;
  runEvents.value = [];

  const phases = option.phases && option.phases.length > 0 ? option.phases : displayPhases.value;
  const enabledIds = phases.filter((phase) => enabledPhaseIds.value.has(phase.id)).map((phase) => phase.id);

  try {
    await window.macOS.runWorkflow(
      {
        workflowId: selectedWorkflow.value.id,
        confirmationOptionId: option.id,
        enabledPhaseIds: enabledIds,
      },
      (event) => runEvents.value.push(event),
    );
  } finally {
    running.value = false;
    await refreshRuns();
  }
}

async function refreshRuns() {
  runs.value = await window.macOS.runs(25);
}

async function openRun(run: RunSummary) {
  selectedRunId.value = run.id;
  selectedRunLog.value = await window.macOS.runLog(run.id);
}

function phaseStatus(phase: Phase) {
  const events = runEvents.value.filter((event) => event.phaseId === phase.id);
  const finish = [...events].reverse().find((event) => event.type === "phase_finished" || event.type === "phase_skipped");

  return finish?.status ?? events.at(-1)?.status ?? (enabledPhaseIds.value.has(phase.id) ? "pending" : "skipped");
}

function badgeClass(status: string) {
  if (["completed", "ok"].includes(status)) {
    return "border-emerald-200 bg-emerald-50 text-emerald-700";
  }

  if (["failed"].includes(status)) {
    return "border-red-200 bg-red-50 text-red-700";
  }

  if (["running"].includes(status)) {
    return "border-cyan-200 bg-cyan-50 text-cyan-700";
  }

  if (["stopped", "skipped"].includes(status)) {
    return "border-zinc-200 bg-zinc-100 text-zinc-500";
  }

  return "border-zinc-200 bg-white text-zinc-500";
}
</script>

<template>
  <div class="flex min-h-screen bg-white text-zinc-950">
    <AppSidebar
      :active="section"
      :workflow-count="workflows.length"
      :run-count="runs.length"
      @select="selectSection"
    />

    <main class="flex min-w-0 flex-1 flex-col">
      <header class="flex h-14 items-center gap-2 border-b border-zinc-200 px-5">
        <span class="text-sm font-medium text-zinc-500">Mac OS Manager</span>
        <ChevronRight class="size-4 text-zinc-400" />
        <span class="text-sm font-semibold capitalize text-zinc-950">{{ section.replace('-', ' ') }}</span>
      </header>

      <div v-if="loadError" class="m-5 rounded-md border border-red-200 bg-red-50 p-4 text-sm text-red-700">
        {{ loadError }}
      </div>

      <section v-else-if="section === 'workflows'" class="grid min-h-0 flex-1 grid-cols-[360px_1fr]">
        <div class="border-r border-zinc-200 bg-zinc-50/70 p-4">
          <div class="mb-3 flex items-center justify-between">
            <h1 class="text-base font-semibold">Workflows</h1>
            <button
              class="inline-flex h-8 items-center gap-2 rounded-md border border-zinc-200 bg-white px-2.5 text-xs font-medium text-zinc-700 shadow-sm"
              @click="loadAll"
            >
              <RotateCcw class="size-3.5" />
              Refresh
            </button>
          </div>

          <div class="space-y-2">
            <button
              v-for="workflow in workflows"
              :key="workflow.id"
              class="w-full rounded-md border p-3 text-left transition"
              :class="
                workflow.id === selectedWorkflowId
                  ? 'border-zinc-300 bg-white shadow-sm'
                  : 'border-transparent bg-transparent hover:border-zinc-200 hover:bg-white'
              "
              @click="selectWorkflow(workflow)"
            >
              <div class="flex items-center justify-between gap-3">
                <p class="truncate text-sm font-medium text-zinc-950">{{ workflow.name }}</p>
                <span class="rounded border px-2 py-0.5 text-[11px]" :class="badgeClass(workflow.changesMac === 'No' ? 'ok' : 'running')">
                  {{ workflow.changesMac }}
                </span>
              </div>
              <p class="mt-1 line-clamp-2 text-xs leading-5 text-zinc-500">{{ workflow.description }}</p>
            </button>
          </div>
        </div>

        <div v-if="selectedWorkflow" class="min-w-0 overflow-y-auto p-6">
          <div class="mb-5 flex items-start justify-between gap-6">
            <div>
              <h2 class="text-xl font-semibold">{{ selectedWorkflow.name }}</h2>
              <p class="mt-1 max-w-3xl text-sm leading-6 text-zinc-500">{{ selectedWorkflow.description }}</p>
            </div>
            <span class="rounded-md border px-2.5 py-1 text-xs font-medium" :class="badgeClass(runStatus)">
              {{ runStatus }}
            </span>
          </div>

          <div class="grid grid-cols-[minmax(0,1fr)_360px] gap-6">
            <div>
              <div class="mb-3 flex items-center justify-between">
                <h3 class="text-sm font-semibold">Phases</h3>
                <button class="text-xs font-medium text-zinc-500 hover:text-zinc-950" @click="resetEnabledPhases">
                  Reset
                </button>
              </div>

              <div class="overflow-hidden rounded-md border border-zinc-200">
                <button
                  v-for="phase in displayPhases"
                  :key="phase.id"
                  class="flex w-full items-center gap-3 border-b border-zinc-200 px-3 py-3 text-left last:border-b-0 hover:bg-zinc-50"
                  @click="togglePhase(phase)"
                >
                  <CheckCircle2 v-if="enabledPhaseIds.has(phase.id)" class="size-4 text-emerald-600" />
                  <Circle v-else class="size-4 text-zinc-300" />
                  <span class="min-w-0 flex-1 truncate text-sm">{{ phase.name }}</span>
                  <span class="rounded border px-2 py-0.5 text-[11px]" :class="badgeClass(phaseStatus(phase))">
                    {{ phaseStatus(phase) }}
                  </span>
                </button>
              </div>
            </div>

            <div>
              <h3 class="mb-3 text-sm font-semibold">Confirmation</h3>
              <div class="rounded-md border border-zinc-200 bg-zinc-50 p-3">
                <p class="text-sm font-medium">{{ selectedWorkflow.confirmation?.title }}</p>
                <p class="mt-1 text-xs leading-5 text-zinc-500">{{ selectedWorkflow.confirmation?.message }}</p>
                <div class="mt-3 space-y-2">
                  <button
                    v-for="option in selectedWorkflow.confirmation?.options"
                    :key="option.id"
                    class="flex w-full items-center justify-between gap-3 rounded-md border border-zinc-200 bg-white px-3 py-2 text-left text-sm shadow-sm hover:border-zinc-300"
                    @click="openConfirmation(option)"
                  >
                    <span>
                      <span class="block font-medium">{{ option.label }}</span>
                      <span class="block text-xs text-zinc-500">{{ option.description }}</span>
                    </span>
                    <Play v-if="option.continue" class="size-4 text-zinc-500" />
                  </button>
                </div>
              </div>
            </div>
          </div>

          <div class="mt-6">
            <h3 class="mb-3 text-sm font-semibold">Output</h3>
            <pre class="h-72 overflow-auto rounded-md border border-zinc-200 bg-zinc-950 p-4 text-xs leading-5 text-zinc-100">{{ runEvents.map((event) => event.message || `${event.type} ${event.status || ''}`).filter(Boolean).join('\n') }}</pre>
          </div>
        </div>
      </section>

      <section v-else-if="section === 'logs'" class="grid min-h-0 flex-1 grid-cols-[420px_1fr]">
        <div class="border-r border-zinc-200 bg-zinc-50/70 p-4">
          <div class="mb-3 flex items-center justify-between">
            <h1 class="text-base font-semibold">Logs</h1>
            <button class="rounded-md border border-zinc-200 bg-white px-2.5 py-1.5 text-xs font-medium shadow-sm" @click="refreshRuns">
              Refresh
            </button>
          </div>
          <div class="space-y-2">
            <button
              v-for="run in runs"
              :key="run.id"
              class="w-full rounded-md border p-3 text-left transition"
              :class="run.id === selectedRunId ? 'border-zinc-300 bg-white shadow-sm' : 'border-transparent hover:border-zinc-200 hover:bg-white'"
              @click="openRun(run)"
            >
              <div class="flex items-center justify-between gap-3">
                <p class="truncate text-sm font-medium">{{ run.workflowName }}</p>
                <span class="rounded border px-2 py-0.5 text-[11px]" :class="badgeClass(run.status)">
                  {{ run.status }}
                </span>
              </div>
              <p class="mt-1 text-xs text-zinc-500">{{ run.startedAt }}</p>
            </button>
          </div>
        </div>

        <div class="overflow-y-auto p-6">
          <template v-if="selectedRunLog">
            <div class="mb-5">
              <h2 class="text-xl font-semibold">{{ selectedRunLog.run.workflowName }}</h2>
              <p class="mt-1 text-sm text-zinc-500">{{ selectedRunLog.run.mode }} · {{ selectedRunLog.run.status }}</p>
            </div>
            <pre class="h-[calc(100vh-180px)] overflow-auto rounded-md border border-zinc-200 bg-zinc-950 p-4 text-xs leading-5 text-zinc-100">{{ selectedRunLog.events.map((event) => event.message || `${event.type} ${event.status || ''}`).filter(Boolean).join('\n') }}</pre>
          </template>
          <div v-else class="grid h-full place-items-center text-sm text-zinc-500">Select a run</div>
        </div>
      </section>

      <section v-else class="flex flex-1 items-center justify-center text-sm text-zinc-500">
        {{ section.replace('-', ' ') }}
      </section>
    </main>

    <div v-if="pendingOption" class="fixed inset-0 grid place-items-center bg-black/30 p-6">
      <div class="w-full max-w-md rounded-lg bg-white p-5 shadow-xl">
        <div class="mb-4 flex items-start gap-3">
          <AlertTriangle class="mt-0.5 size-5 text-amber-600" />
          <div>
            <h2 class="text-base font-semibold">{{ pendingOption.label }}</h2>
            <p class="mt-1 text-sm leading-6 text-zinc-500">{{ pendingOption.description }}</p>
          </div>
        </div>
        <div class="flex justify-end gap-2">
          <button class="rounded-md border border-zinc-200 px-3 py-2 text-sm font-medium" @click="pendingOption = null">
            Cancel
          </button>
          <button
            class="inline-flex items-center gap-2 rounded-md bg-zinc-900 px-3 py-2 text-sm font-medium text-white disabled:opacity-60"
            :disabled="running"
            @click="runSelected(pendingOption)"
          >
            <Loader2 v-if="running" class="size-4 animate-spin" />
            Continue
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
