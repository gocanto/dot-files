<script setup lang="ts">
import { TerminalSquare } from "lucide-vue-next";
import { ScrollArea } from "@ui/scroll-area";
import { Skeleton } from "@ui/skeleton";
import RunOutputSections from "@app/RunOutputSections.vue";
import type { RunOutputSection } from "@app/types";
import type { AppDiagnostic, RunLog } from "@api";

defineProps<{
  runLogLoading: boolean;
  selectedRunLog: RunLog | null;
  selectedAppDiagnostic: AppDiagnostic | null;
  selectedRunOutputSections: RunOutputSection[];
}>();
</script>

<template>
  <div v-if="runLogLoading" data-testid="run-log-skeleton" class="flex min-h-0 flex-1 flex-col">
    <div class="min-h-0 flex-1 bg-terminal p-4">
      <Skeleton class="h-3 w-3/4 bg-white/10" />
      <Skeleton class="mt-2 h-3 w-1/2 bg-white/10" />
      <Skeleton class="mt-2 h-3 w-2/3 bg-white/10" />
      <Skeleton class="mt-2 h-3 w-3/5 bg-white/10" />
    </div>
  </div>

  <div v-else-if="selectedRunLog" class="flex min-h-0 flex-1 flex-col">
    <ScrollArea class="min-h-0 flex-1 bg-terminal text-terminal-foreground">
      <RunOutputSections
        :sections="selectedRunOutputSections"
        empty-text="No log output recorded."
        default-expanded
      />
    </ScrollArea>
  </div>

  <div v-else-if="selectedAppDiagnostic" class="flex min-h-0 flex-1 flex-col">
    <ScrollArea class="min-h-0 flex-1 bg-terminal p-4 text-terminal-foreground">
      <pre class="whitespace-pre-wrap text-sm leading-6">{{
        [
          `[${selectedAppDiagnostic.createdAt}] ${selectedAppDiagnostic.level.toUpperCase()} ${selectedAppDiagnostic.source}`,
          selectedAppDiagnostic.message,
          selectedAppDiagnostic.details,
        ]
          .filter(Boolean)
          .join("\n\n")
      }}</pre>
    </ScrollArea>
  </div>

  <div v-else class="grid flex-1 place-items-center p-8 text-center text-sm text-muted-foreground">
    <div>
      <TerminalSquare class="mx-auto mb-3 size-8" />
      <p>Select a run or app diagnostic to inspect its output.</p>
    </div>
  </div>
</template>
