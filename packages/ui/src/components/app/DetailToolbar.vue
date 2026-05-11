<script setup lang="ts">
import { computed } from "vue";
import { X } from "lucide-vue-next";
import { Skeleton } from "@ui/skeleton";
import StatusBadge from "@components/StatusBadge.vue";
import MachineAvatar from "@app/MachineAvatar.vue";
import { panelHeaderClass } from "@app/styles";
import { formatDate } from "@lib/format";
import { getWorkflowDetail } from "@lib/workflowDetails";
import { cn } from "@lib/utils";
import type { AppDiagnostic, RunLog, Workflow } from "@api";

const props = defineProps<{
  section: string;
  hasStepMeta: boolean;
  selectedWorkflow: Workflow | undefined;
  selectedRunLog: RunLog | null;
  selectedAppDiagnostic: AppDiagnostic | null;
  runLogLoading: boolean;
  runStatus: string;
}>();

const emit = defineEmits<{
  (event: "close-detail"): void;
}>();

const showClose = computed(
  () =>
    (props.hasStepMeta && Boolean(props.selectedWorkflow)) ||
    (props.section === "logs" &&
      (props.runLogLoading ||
        Boolean(props.selectedRunLog) ||
        Boolean(props.selectedAppDiagnostic))),
);

const sectionLabels: Record<string, string> = {
  template: "Source",
  current: "This Mac",
  update: "Apply",
  logs: "Logs",
  settings: "Settings",
  status: "Status",
};

const sectionLabel = computed(() => sectionLabels[props.section] ?? "");

const ansiPattern = /\[[0-9;?]*[ -/]*[@-~]/g;

function cleanSummaryText(value: string): string {
  return value.replace(ansiPattern, "").replace(/\s+/g, " ").trim();
}

const runLogSummary = computed(() => {
  const log = props.selectedRunLog;
  if (!log) return "";
  const errorMessage = cleanSummaryText(log.run.errorMessage ?? "");
  if (errorMessage) return errorMessage;

  const events = log.events;
  const phaseIds = new Set<string>();
  for (const event of events) {
    if (event.phaseId) phaseIds.add(event.phaseId);
  }

  for (let i = events.length - 1; i >= 0; i--) {
    const message = cleanSummaryText(events[i].message ?? "");
    if (message) return message;
  }

  const phaseCount = phaseIds.size;
  const eventCount = events.length;
  const phasePart = phaseCount ? `${phaseCount} phase${phaseCount === 1 ? "" : "s"}` : "";
  const eventPart = `${eventCount} event${eventCount === 1 ? "" : "s"}`;
  return [phasePart, eventPart].filter(Boolean).join(" · ");
});

const appDiagnosticSummary = computed(() => {
  const diagnostic = props.selectedAppDiagnostic;
  if (!diagnostic) return "";
  const detail = cleanSummaryText(diagnostic.details ?? "");
  if (detail) return detail;
  return cleanSummaryText(diagnostic.message ?? "");
});

function diagnosticStatus(level: AppDiagnostic["level"]) {
  return level === "error" ? "failed" : level === "warning" ? "stopped" : "completed";
}
</script>

<template>
  <div
    data-testid="detail-toolbar"
    :class="cn('flex min-h-[var(--panel-header-h)] items-center gap-3 px-4', panelHeaderClass)"
  >
    <div v-if="hasStepMeta && selectedWorkflow" class="flex items-start gap-3 text-sm">
      <MachineAvatar :alt="`Machine avatar for ${selectedWorkflow.name}`" />
      <div class="grid gap-1">
        <div
          v-if="sectionLabel"
          class="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground"
        >
          {{ sectionLabel }}
        </div>
        <div class="font-semibold">{{ selectedWorkflow.name }}</div>
        <div class="line-clamp-1 text-xs">{{ selectedWorkflow.description }}</div>
        <div class="line-clamp-1 text-xs">
          <span class="font-medium">Action:</span>
          {{ getWorkflowDetail(selectedWorkflow.id).action || selectedWorkflow.changesMac }}
          <span class="text-muted-foreground"
            >· Changes Mac: {{ selectedWorkflow.changesMac }}</span
          >
        </div>
      </div>
    </div>

    <div v-else-if="section === 'logs' && runLogLoading" class="flex items-start gap-3 text-sm">
      <Skeleton class="size-8 rounded-full" />
      <div class="grid gap-2">
        <Skeleton class="h-3 w-12" />
        <Skeleton class="h-3 w-40" />
        <Skeleton class="h-4 w-48" />
        <Skeleton class="h-3 w-56" />
        <Skeleton class="h-3 w-40" />
      </div>
    </div>

    <div
      v-else-if="section === 'logs' && selectedRunLog"
      class="flex min-w-0 items-start gap-3 text-sm"
    >
      <MachineAvatar :alt="`Machine avatar for ${selectedRunLog.run.workflowName}`" />
      <div class="grid min-w-0 gap-1">
        <div
          v-if="sectionLabel"
          class="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground"
        >
          {{ sectionLabel }}
        </div>
        <div
          v-if="runLogSummary"
          class="line-clamp-1 text-xs text-muted-foreground"
          :title="runLogSummary"
        >
          {{ runLogSummary }}
        </div>
        <div class="truncate font-semibold">{{ selectedRunLog.run.workflowName }}</div>
        <div class="line-clamp-1 text-xs">
          {{ selectedRunLog.run.mode }} - {{ selectedRunLog.run.confirmationOptionLabel }}
        </div>
        <div class="line-clamp-1 text-xs">
          <span class="font-medium">Started:</span>
          {{ formatDate(selectedRunLog.run.startedAt) }}
        </div>
      </div>
    </div>

    <div
      v-else-if="section === 'logs' && selectedAppDiagnostic"
      class="flex min-w-0 items-start gap-3 text-sm"
    >
      <MachineAvatar :alt="`App diagnostic for ${selectedAppDiagnostic.source}`" />
      <div class="grid min-w-0 gap-1">
        <div
          v-if="sectionLabel"
          class="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground"
        >
          {{ sectionLabel }}
        </div>
        <div
          v-if="appDiagnosticSummary"
          class="line-clamp-1 text-xs text-muted-foreground"
          :title="appDiagnosticSummary"
        >
          {{ appDiagnosticSummary }}
        </div>
        <div class="truncate font-semibold">{{ selectedAppDiagnostic.source }}</div>
        <div class="line-clamp-1 text-xs">{{ selectedAppDiagnostic.message }}</div>
        <div class="line-clamp-1 text-xs">
          <span class="font-medium">Recorded:</span>
          {{ formatDate(selectedAppDiagnostic.createdAt) }}
        </div>
      </div>
    </div>

    <div class="ml-auto flex flex-col items-end gap-1.5">
      <button
        v-if="showClose"
        type="button"
        data-testid="detail-close"
        aria-label="Close detail"
        class="inline-flex size-6 items-center justify-center rounded-md text-muted-foreground transition-colors hover:bg-accent hover:text-accent-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
        @click="emit('close-detail')"
      >
        <X class="size-4" />
      </button>
      <StatusBadge v-if="hasStepMeta && selectedWorkflow" :status="runStatus" />
      <Skeleton v-else-if="section === 'logs' && runLogLoading" class="h-5 w-20 rounded-full" />
      <StatusBadge
        v-else-if="section === 'logs' && selectedRunLog"
        :status="selectedRunLog.run.status"
      />
      <StatusBadge
        v-else-if="section === 'logs' && selectedAppDiagnostic"
        :status="diagnosticStatus(selectedAppDiagnostic.level)"
        :label="selectedAppDiagnostic.level"
      />
    </div>
  </div>
</template>
