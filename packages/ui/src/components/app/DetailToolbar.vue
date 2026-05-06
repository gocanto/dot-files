<script setup lang="ts">
import { Avatar, AvatarFallback } from "@ui/avatar";
import { Skeleton } from "@ui/skeleton";
import StatusBadge from "@components/StatusBadge.vue";
import { panelHeaderClass } from "@app/styles";
import { formatDate, initials } from "@lib/format";
import { getWorkflowDetail } from "@lib/workflowDetails";
import { cn } from "@lib/utils";
import type { RunLog, Workflow } from "@api";

defineProps<{
  section: string;
  hasStepMeta: boolean;
  selectedWorkflow: Workflow | undefined;
  selectedRunLog: RunLog | null;
  runLogLoading: boolean;
  runStatus: string;
}>();
</script>

<template>
  <div
    data-testid="detail-toolbar"
    :class="cn('flex min-h-[var(--panel-header-h)] items-center gap-3 px-4', panelHeaderClass)"
  >
    <div v-if="hasStepMeta && selectedWorkflow" class="flex items-start gap-3 text-sm">
      <Avatar size="sm">
        <AvatarFallback>{{ initials(selectedWorkflow.name) }}</AvatarFallback>
      </Avatar>
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
      <Avatar size="sm">
        <AvatarFallback>{{ initials(selectedRunLog.run.workflowName) }}</AvatarFallback>
      </Avatar>
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

    <div class="ml-auto flex items-center gap-2">
      <StatusBadge v-if="hasStepMeta && selectedWorkflow" :status="runStatus" />
      <Skeleton v-else-if="section === 'logs' && runLogLoading" class="h-5 w-20 rounded-full" />
      <StatusBadge
        v-else-if="section === 'logs' && selectedRunLog"
        :status="selectedRunLog.run.status"
      />
    </div>
  </div>
</template>
