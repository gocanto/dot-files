<script setup lang="ts">
import { Skeleton } from "@ui/skeleton";
import StatusBadge from "@components/StatusBadge.vue";
import MachineAvatar from "@app/MachineAvatar.vue";
import { panelHeaderClass } from "@app/styles";
import { formatDate } from "@lib/format";
import { getWorkflowDetail } from "@lib/workflowDetails";
import { cn } from "@lib/utils";
import type { AppDiagnostic, RunLog, Workflow } from "@api";

defineProps<{
  section: string;
  hasStepMeta: boolean;
  selectedWorkflow: Workflow | undefined;
  selectedRunLog: RunLog | null;
  selectedAppDiagnostic: AppDiagnostic | null;
  runLogLoading: boolean;
  runStatus: string;
}>();

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
        <div class="truncate font-semibold">{{ selectedAppDiagnostic.source }}</div>
        <div class="line-clamp-1 text-xs">{{ selectedAppDiagnostic.message }}</div>
        <div class="line-clamp-1 text-xs">
          <span class="font-medium">Recorded:</span>
          {{ formatDate(selectedAppDiagnostic.createdAt) }}
        </div>
      </div>
    </div>

    <div class="ml-auto flex items-center gap-2">
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
