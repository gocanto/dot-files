<script setup lang="ts">
import { Badge } from "@/components/ui/badge";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Separator } from "@/components/ui/separator";
import { Skeleton } from "@/components/ui/skeleton";
import StatusBadge from "@/components/StatusBadge.vue";
import WorkflowCardList from "@/components/WorkflowCardList.vue";
import { panelHeaderClass } from "@/components/app/styles";
import type { SettingsGroup } from "@/components/app/types";
import { cn } from "@/lib/utils";
import type { SettingsResponse, Workflow } from "@/types/api";

defineProps<{
  settingsLoading: boolean;
  settingsResponse: SettingsResponse | null;
  settingsGroups: SettingsGroup[];
  workflowsLoading: boolean;
  settingsWorkflows: Workflow[];
  selectedWorkflowId: string;
}>();

const emit = defineEmits<{
  (event: "select-workflow", workflow: Workflow): void;
}>();
</script>

<template>
  <div :class="cn('flex min-h-[var(--panel-header-h)] items-center px-4 py-2', panelHeaderClass)">
    <h1 class="text-xl font-bold">Settings</h1>
    <Skeleton v-if="settingsLoading && !settingsResponse" class="ml-auto h-5 w-24 rounded-full" />
    <StatusBadge
      v-else
      class="ml-auto"
      :status="settingsResponse?.valid ? 'ok' : 'failed'"
      :label="settingsResponse?.valid ? 'valid' : 'needs review'"
    />
  </div>
  <Separator />
  <ScrollArea class="min-h-0 flex-1">
    <div class="flex flex-col gap-2 p-4">
      <template v-if="settingsLoading && !settingsResponse">
        <div
          v-for="i in 4"
          :key="`settings-group-skeleton-${i}`"
          data-testid="settings-groups-skeleton"
          class="flex items-center gap-3 rounded-lg border border-section-border bg-section p-3 text-sm shadow-sm"
        >
          <Skeleton class="size-4 rounded" />
          <div class="min-w-0 flex-1 space-y-1">
            <Skeleton class="h-4 w-28" />
            <Skeleton class="h-3 w-40" />
          </div>
          <Skeleton class="h-5 w-6 rounded-full" />
        </div>
      </template>
      <template v-else>
        <div
          v-for="group in settingsGroups"
          :key="group.id"
          class="flex items-center gap-3 rounded-lg border border-section-border bg-section p-3 text-sm shadow-sm"
        >
          <component :is="group.icon" class="size-4 text-muted-foreground" />
          <div class="min-w-0 flex-1">
            <div class="font-medium">{{ group.label }}</div>
            <div class="truncate text-xs text-muted-foreground">
              {{
                group.count === 0
                  ? "No validation errors"
                  : `${group.count} issue${group.count === 1 ? "" : "s"}`
              }}
            </div>
          </div>
          <Badge :variant="group.count === 0 ? 'secondary' : 'destructive'">{{
            group.count
          }}</Badge>
        </div>
      </template>
    </div>
    <Separator />
    <div class="px-4 pt-4 pb-2 text-xs font-semibold uppercase tracking-wide text-muted-foreground">
      Workflows
    </div>
    <div
      v-if="workflowsLoading"
      data-testid="settings-workflows-skeleton"
      class="flex flex-col gap-2 p-4 pt-0"
    >
      <div
        v-for="i in 3"
        :key="`settings-wf-skeleton-${i}`"
        class="rounded-lg border border-section-border bg-section p-3 shadow-sm"
      >
        <div class="flex items-center gap-3">
          <Skeleton class="h-4 w-40" />
          <Skeleton class="ml-auto h-5 w-12 rounded-full" />
        </div>
        <Skeleton class="mt-3 h-3 w-24" />
        <Skeleton class="mt-3 h-3 w-full" />
      </div>
    </div>
    <WorkflowCardList
      v-else
      :workflows="settingsWorkflows"
      :selected-id="selectedWorkflowId"
      empty-message="No settings workflows available."
      @select="emit('select-workflow', $event)"
    />
  </ScrollArea>
</template>
