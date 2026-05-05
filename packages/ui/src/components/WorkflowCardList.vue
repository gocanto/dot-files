<script setup lang="ts">
import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/utils";
import { getWorkflowDetail, workflowActionPillClass } from "@/lib/workflowDetails";
import { workflowActionIcon } from "@/lib/workflowIcons";
import type { Workflow } from "@/types/api";

defineProps<{
  workflows: Workflow[];
  selectedId: string;
  emptyMessage?: string;
}>();

const emit = defineEmits<{
  (event: "select", workflow: Workflow): void;
}>();

const listItemClass = "bg-section border-section-border shadow-sm hover:border-primary/40 hover:bg-accent";
const selectedListItemClass = "border-primary/50 bg-accent text-accent-foreground shadow-sm";
</script>

<template>
  <div class="flex flex-col gap-2 p-4 pt-0">
    <button
      v-for="workflow in workflows"
      :key="workflow.id"
      :class="cn(
        'flex flex-col items-start gap-2 rounded-lg border p-3 text-left text-sm transition-all hover:bg-accent',
        listItemClass,
        selectedId === workflow.id && selectedListItemClass,
      )"
      @click="emit('select', workflow)"
    >
      <div class="flex w-full flex-col gap-1">
        <div class="flex min-w-0 items-center gap-2">
          <div class="truncate font-semibold">{{ workflow.name }}</div>
          <span v-if="workflow.id === selectedId" class="flex size-2 rounded-full bg-primary" />
          <Badge variant="outline" :class="cn('ml-auto', workflowActionPillClass(workflow.id))">
            <component :is="workflowActionIcon(workflow.id)" />
            {{ getWorkflowDetail(workflow.id).action || workflow.changesMac }}
          </Badge>
        </div>
        <div class="text-xs font-medium text-muted-foreground">
          {{ workflow.phases.length }} phases
        </div>
      </div>
      <div class="line-clamp-2 text-xs leading-5 text-muted-foreground">
        {{ workflow.description }}
      </div>
    </button>

    <div v-if="workflows.length === 0" class="rounded-lg border border-dashed border-section-border bg-section p-8 text-center text-sm text-muted-foreground">
      {{ emptyMessage ?? "No workflows match this view." }}
    </div>
  </div>
</template>
