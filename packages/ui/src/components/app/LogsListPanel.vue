<script setup lang="ts">
import { Search } from "lucide-vue-next";
import StatusBadge from "@components/StatusBadge.vue";
import { Input } from "@ui/input";
import { ScrollArea } from "@ui/scroll-area";
import { Separator } from "@ui/separator";
import { Skeleton } from "@ui/skeleton";
import { Tabs, TabsList, TabsTrigger } from "@ui/tabs";
import {
  listItemClass,
  panelHeaderClass,
  searchBarClass,
  selectedListItemClass,
} from "@app/styles";
import { timeAgo } from "@lib/format";
import { cn } from "@lib/utils";
import type { RunSummary } from "@api";

defineProps<{
  logTab: string;
  searchQuery: string;
  runs: RunSummary[];
  selectedRunId: string;
  runsLoading: boolean;
}>();

const emit = defineEmits<{
  (event: "update:logTab", value: string): void;
  (event: "update:searchQuery", value: string): void;
  (event: "open-run", run: RunSummary): void;
}>();
</script>

<template>
  <Tabs
    :model-value="logTab"
    class="flex h-full min-h-0 flex-col"
    @update:model-value="emit('update:logTab', String($event))"
  >
    <div :class="cn('flex min-h-[var(--panel-header-h)] items-center px-4', panelHeaderClass)">
      <h1 class="text-xl font-bold">Logs</h1>
      <TabsList class="ml-auto">
        <TabsTrigger value="all">All</TabsTrigger>
        <TabsTrigger value="failed">Failed</TabsTrigger>
        <TabsTrigger value="active">Active</TabsTrigger>
      </TabsList>
    </div>
    <Separator />
    <div :class="searchBarClass">
      <form @submit.prevent>
        <div class="relative">
          <Search class="absolute left-2 top-2.5 size-4 text-muted-foreground" />
          <Input
            :model-value="searchQuery"
            data-testid="app-search"
            placeholder="Search logs"
            class="pl-8"
            @update:model-value="emit('update:searchQuery', String($event))"
          />
        </div>
      </form>
    </div>
    <ScrollArea class="min-h-0 flex-1">
      <div v-if="runsLoading" data-testid="runs-list-skeleton" class="flex flex-col gap-2 p-4 pt-0">
        <div
          v-for="index in 4"
          :key="index"
          class="rounded-lg border border-section-border bg-section p-3 shadow-sm"
        >
          <div class="flex items-center gap-2">
            <Skeleton class="h-4 w-44" />
            <Skeleton class="ml-auto h-5 w-16 rounded-full" />
          </div>
          <div class="mt-2 flex items-center justify-between gap-3">
            <Skeleton class="h-3 w-32" />
            <Skeleton class="h-3 w-16" />
          </div>
        </div>
      </div>
      <div v-else class="flex flex-col gap-2 p-4 pt-0">
        <button
          v-for="run in runs"
          :key="run.id"
          :class="
            cn(
              'flex flex-col items-start gap-2 rounded-lg border p-3 text-left text-sm transition-all hover:bg-accent',
              listItemClass,
              selectedRunId === run.id && selectedListItemClass,
            )
          "
          @click="emit('open-run', run)"
        >
          <div class="flex w-full flex-col gap-1">
            <div class="flex min-w-0 items-center gap-2">
              <div class="truncate font-semibold">{{ run.workflowName }}</div>
              <StatusBadge class="ml-auto" :status="run.status" />
            </div>
            <div class="flex items-center justify-between gap-3 text-xs text-muted-foreground">
              <span class="truncate">{{ run.mode }} - {{ run.confirmationOptionLabel }}</span>
              <span class="shrink-0">{{ timeAgo(run.startedAt) }}</span>
            </div>
          </div>
        </button>

        <div
          v-if="runs.length === 0"
          class="rounded-lg border border-dashed border-section-border bg-section p-8 text-center text-sm text-muted-foreground"
        >
          No logs match this view.
        </div>
      </div>
    </ScrollArea>
  </Tabs>
</template>
