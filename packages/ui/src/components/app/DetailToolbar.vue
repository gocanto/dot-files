<script setup lang="ts">
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import StatusBadge from "@/components/StatusBadge.vue";
import { panelHeaderClass } from "@/components/app/styles";
import { initials } from "@/lib/format";
import { getWorkflowDetail } from "@/lib/workflowDetails";
import { cn } from "@/lib/utils";
import type { Workflow } from "@/types/api";

defineProps<{
  hasStepMeta: boolean;
  selectedWorkflow: Workflow | undefined;
  runStatus: string;
}>();
</script>

<template>
  <div
    data-testid="detail-toolbar"
    :class="cn('flex min-h-[var(--panel-header-h)] items-start gap-3 px-2 py-2', panelHeaderClass)"
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

    <div class="ml-auto flex items-center gap-2">
      <StatusBadge v-if="hasStepMeta && selectedWorkflow" :status="runStatus" />
    </div>
  </div>
</template>
