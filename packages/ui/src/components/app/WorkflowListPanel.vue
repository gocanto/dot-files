<script setup lang="ts">
import { Search, Settings, TerminalSquare } from "lucide-vue-next";
import { Input } from "@/components/ui/input";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Separator } from "@/components/ui/separator";
import { Skeleton } from "@/components/ui/skeleton";
import WorkflowCardList from "@/components/WorkflowCardList.vue";
import {
  listItemClass,
  panelHeaderClass,
  searchBarClass,
  selectedListItemClass,
} from "@/components/app/styles";
import type { StepMeta, StepSettingsKey } from "@/components/app/types";
import { cn } from "@/lib/utils";
import type { RuntimeSettings, SettingsResponse, Workflow } from "@/types/api";

defineProps<{
  stepMeta: StepMeta;
  searchQuery: string;
  workflows: Workflow[];
  selectedWorkflowId: string;
  selectedSettingsKey: StepSettingsKey | null;
  workflowsLoading: boolean;
  settingsLoading: boolean;
  settingsResponse: SettingsResponse | null;
  settingsForm: RuntimeSettings;
  settingsKeyLabels: Record<StepSettingsKey, string>;
}>();

const emit = defineEmits<{
  (event: "update:searchQuery", value: string): void;
  (event: "select-workflow", workflow: Workflow): void;
  (event: "select-step-setting", key: StepSettingsKey): void;
  (event: "open-devtools"): void;
}>();
</script>

<template>
  <div class="flex h-full min-h-0 flex-col">
    <div :class="cn('flex min-h-[var(--panel-header-h)] items-center px-4', panelHeaderClass)">
      <h1 class="text-xl font-bold">{{ stepMeta.title }}</h1>
    </div>
    <Separator />
    <div :class="searchBarClass">
      <form @submit.prevent>
        <div class="relative">
          <Search class="absolute left-2 top-2.5 size-4 text-muted-foreground" />
          <Input
            :model-value="searchQuery"
            data-testid="app-search"
            placeholder="Search workflows"
            class="pl-8"
            @update:model-value="emit('update:searchQuery', String($event))"
          />
        </div>
      </form>
    </div>
    <ScrollArea class="min-h-0 flex-1">
      <div
        v-if="workflowsLoading"
        data-testid="workflows-list-skeleton"
        class="flex flex-col gap-2 p-4 pt-0"
      >
        <div
          v-for="index in 6"
          :key="index"
          class="rounded-lg border border-section-border bg-section p-3 shadow-sm"
        >
          <div class="flex items-center gap-3">
            <Skeleton class="h-4 w-40" />
            <Skeleton class="ml-auto h-5 w-12 rounded-full" />
          </div>
          <Skeleton class="mt-3 h-3 w-24" />
          <Skeleton class="mt-3 h-3 w-full" />
          <Skeleton class="mt-2 h-3 w-4/5" />
        </div>
      </div>
      <WorkflowCardList
        v-else
        :workflows="workflows"
        :selected-id="selectedWorkflowId"
        :empty-message="stepMeta.emptyMessage"
        @select="emit('select-workflow', $event)"
      />
      <div
        class="px-4 pt-4 pb-2 text-xs font-semibold uppercase tracking-wide text-muted-foreground"
      >
        Step settings
      </div>
      <div class="flex flex-col gap-1 px-4 pb-4">
        <template v-if="settingsLoading && !settingsResponse">
          <div
            v-for="key in stepMeta.settingsKeys"
            :key="key"
            data-testid="step-settings-skeleton"
            class="flex items-center gap-3 rounded-lg border border-section-border bg-section px-3 py-2 shadow-sm"
          >
            <Skeleton class="size-4 rounded" />
            <div class="min-w-0 flex-1 space-y-1">
              <Skeleton class="h-4 w-32" />
              <Skeleton class="h-3 w-48" />
            </div>
          </div>
        </template>
        <template v-else>
          <button
            v-for="key in stepMeta.settingsKeys"
            :key="key"
            :class="
              cn(
                'flex items-center gap-3 rounded-lg border px-3 py-2 text-left text-sm transition-all hover:bg-accent',
                listItemClass,
                selectedSettingsKey === key && selectedListItemClass,
              )
            "
            @click="emit('select-step-setting', key)"
          >
            <Settings class="size-4 text-muted-foreground" />
            <div class="min-w-0 flex-1">
              <div class="font-medium">{{ settingsKeyLabels[key] }}</div>
              <div class="truncate text-xs text-muted-foreground">
                {{ settingsForm[key] || "not set" }}
              </div>
            </div>
          </button>
          <button
            :class="
              cn(
                'flex items-center gap-3 rounded-lg border px-3 py-2 text-left text-sm transition-all',
                listItemClass,
              )
            "
            @click="emit('open-devtools')"
          >
            <TerminalSquare class="size-4 text-muted-foreground" />
            <div class="min-w-0 flex-1">
              <div class="font-medium">DevTools</div>
              <div class="truncate text-xs text-muted-foreground">Open developer tools</div>
            </div>
          </button>
        </template>
      </div>
    </ScrollArea>
  </div>
</template>
