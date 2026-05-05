<script setup lang="ts">
import { Play, RefreshCw, RotateCcw } from "lucide-vue-next";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
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
  section: string;
  workflowsLoading: boolean;
  runsLoading: boolean;
  settingsLoading: boolean;
  running: boolean;
}>();

const emit = defineEmits<{
  (event: "refresh"): void;
  (event: "reset-phases"): void;
  (event: "open-confirmation"): void;
}>();
</script>

<template>
  <div
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
      <Tooltip>
        <TooltipTrigger as-child>
          <Button
            variant="ghost"
            size="icon"
            :disabled="section === 'logs' || workflowsLoading || runsLoading || settingsLoading"
            @click="emit('refresh')"
          >
            <RefreshCw class="size-4" />
            <span class="sr-only">Refresh</span>
          </Button>
        </TooltipTrigger>
        <TooltipContent>Refresh</TooltipContent>
      </Tooltip>

      <Tooltip>
        <TooltipTrigger as-child>
          <Button
            variant="ghost"
            size="icon"
            :disabled="!hasStepMeta || !selectedWorkflow"
            @click="emit('reset-phases')"
          >
            <RotateCcw class="size-4" />
            <span class="sr-only">Reset phases</span>
          </Button>
        </TooltipTrigger>
        <TooltipContent>Reset phases</TooltipContent>
      </Tooltip>

      <Tooltip>
        <TooltipTrigger as-child>
          <Button
            variant="ghost"
            size="icon"
            :disabled="!selectedWorkflow?.confirmation || running"
            @click="emit('open-confirmation')"
          >
            <Play class="size-4" />
            <span class="sr-only">Run workflow</span>
          </Button>
        </TooltipTrigger>
        <TooltipContent>Run workflow</TooltipContent>
      </Tooltip>
    </div>
  </div>
</template>
