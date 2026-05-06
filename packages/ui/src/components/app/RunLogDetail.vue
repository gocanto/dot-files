<script setup lang="ts">
import { TerminalSquare } from "lucide-vue-next";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Separator } from "@/components/ui/separator";
import { Skeleton } from "@/components/ui/skeleton";
import RunOutputSections from "@/components/app/RunOutputSections.vue";
import StatusBadge from "@/components/StatusBadge.vue";
import type { RunOutputSection } from "@/components/app/types";
import { formatDate, initials } from "@/lib/format";
import type { RunLog } from "@/types/api";

defineProps<{
  runLogLoading: boolean;
  selectedRunLog: RunLog | null;
  selectedRunOutputSections: RunOutputSection[];
}>();
</script>

<template>
  <div v-if="runLogLoading" data-testid="run-log-skeleton" class="flex min-h-0 flex-1 flex-col">
    <div class="flex items-start bg-section p-4">
      <Skeleton class="size-10 rounded-full" />
      <div class="ml-4 grid flex-1 gap-2">
        <Skeleton class="h-4 w-48" />
        <Skeleton class="h-3 w-56" />
        <Skeleton class="h-3 w-40" />
      </div>
      <Skeleton class="h-5 w-20 rounded-full" />
    </div>
    <Separator />
    <div class="min-h-0 flex-1 bg-terminal p-4">
      <Skeleton class="h-3 w-3/4 bg-white/10" />
      <Skeleton class="mt-2 h-3 w-1/2 bg-white/10" />
      <Skeleton class="mt-2 h-3 w-2/3 bg-white/10" />
      <Skeleton class="mt-2 h-3 w-3/5 bg-white/10" />
    </div>
    <Separator />
    <div class="border-t border-section-border bg-section p-4">
      <Skeleton class="h-20 w-full" />
    </div>
  </div>

  <div v-else-if="selectedRunLog" class="flex min-h-0 flex-1 flex-col">
    <div class="flex items-start bg-section p-4">
      <div class="flex items-start gap-4 text-sm">
        <Avatar size="sm">
          <AvatarFallback>{{ initials(selectedRunLog.run.workflowName) }}</AvatarFallback>
        </Avatar>
        <div class="grid gap-1">
          <div class="font-semibold">{{ selectedRunLog.run.workflowName }}</div>
          <div class="line-clamp-1 text-xs">
            {{ selectedRunLog.run.mode }} - {{ selectedRunLog.run.confirmationOptionLabel }}
          </div>
          <div class="line-clamp-1 text-xs">
            <span class="font-medium">Started:</span> {{ formatDate(selectedRunLog.run.startedAt) }}
          </div>
        </div>
      </div>
      <StatusBadge class="ml-auto" :status="selectedRunLog.run.status" />
    </div>
    <Separator />
    <ScrollArea class="min-h-0 flex-1 bg-terminal text-terminal-foreground">
      <RunOutputSections
        :sections="selectedRunOutputSections"
        empty-text="No log output recorded."
      />
    </ScrollArea>
  </div>

  <div v-else class="grid flex-1 place-items-center p-8 text-center text-sm text-muted-foreground">
    <div>
      <TerminalSquare class="mx-auto mb-3 size-8" />
      <p>Select a run to inspect its persisted output.</p>
    </div>
  </div>
</template>
