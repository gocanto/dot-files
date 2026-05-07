<script setup lang="ts">
import { computed } from "vue";
import { KeyRound, RefreshCw, ShieldCheck, TriangleAlert } from "lucide-vue-next";
import { Button } from "@ui/button";
import { Badge } from "@ui/badge";
import { ScrollArea } from "@ui/scroll-area";
import { Separator } from "@ui/separator";
import { Skeleton } from "@ui/skeleton";
import StatusBadge from "@components/StatusBadge.vue";
import { panelHeaderClass } from "@app/styles";
import { cn } from "@lib/utils";
import type { SettingsCheck, SettingsResponse, Workflow } from "@api";

const props = defineProps<{
  settingsLoading: boolean;
  settingsResponse: SettingsResponse | null;
  workflows: Workflow[];
}>();

const emit = defineEmits<{
  (event: "refresh"): void;
}>();

const failingChecks = computed(() =>
  (props.settingsResponse?.checks ?? []).filter((check) => check.status !== "ok"),
);

const approvalOptions = computed(() =>
  props.workflows.flatMap((workflow) =>
    (workflow.confirmation?.options ?? [])
      .filter((option) => option.requiresApproval)
      .map((option) => ({
        workflow: workflow.name,
        option: option.label,
      })),
  ),
);

function checkStatus(check: SettingsCheck) {
  return check.status === "ok" ? "ok" : "failed";
}
</script>

<template>
  <div :class="cn('flex min-h-(--panel-header-h) items-center px-4 py-2', panelHeaderClass)">
    <div class="min-w-0">
      <h1 class="text-xl font-bold">System Status</h1>
      <p class="truncate text-xs text-muted-foreground">Permissions, settings, and apply gates</p>
    </div>
    <Button
      variant="ghost"
      size="icon-sm"
      class="ml-auto"
      :disabled="settingsLoading"
      @click="emit('refresh')"
    >
      <RefreshCw :class="cn('size-4', settingsLoading && 'animate-spin')" />
      <span class="sr-only">Refresh status</span>
    </Button>
  </div>
  <Separator />
  <ScrollArea class="min-h-0 flex-1">
    <div class="grid gap-4 p-4">
      <section class="rounded-lg border border-section-border bg-section p-4 shadow-sm">
        <div class="flex items-center gap-3">
          <ShieldCheck class="size-4 text-muted-foreground" />
          <div class="min-w-0 flex-1">
            <h2 class="text-sm font-semibold">Configuration</h2>
            <p class="text-xs text-muted-foreground">Paths and manifests used by workflows</p>
          </div>
          <Skeleton v-if="settingsLoading && !settingsResponse" class="h-5 w-20 rounded-full" />
          <StatusBadge
            v-else
            :status="settingsResponse?.valid ? 'ok' : 'failed'"
            :label="settingsResponse?.valid ? 'Ready' : 'Failed'"
          />
        </div>

        <div v-if="settingsLoading && !settingsResponse" class="mt-4 grid gap-2">
          <Skeleton v-for="i in 4" :key="i" class="h-10 rounded-md" />
        </div>
        <div v-else class="mt-4 overflow-hidden rounded-md border border-section-border">
          <div
            v-for="check in settingsResponse?.checks ?? []"
            :key="check.key"
            class="grid grid-cols-[minmax(0,1fr)_auto] gap-3 border-b border-section-border bg-section-muted px-3 py-2 text-sm last:border-b-0"
          >
            <div class="min-w-0">
              <div class="truncate font-medium">{{ check.label }}</div>
              <div class="truncate text-xs text-muted-foreground">{{ check.message }}</div>
            </div>
            <StatusBadge
              :status="checkStatus(check)"
              :label="check.status === 'ok' ? 'Ready' : 'Failed'"
            />
          </div>
        </div>
      </section>

      <section class="rounded-lg border border-section-border bg-section p-4 shadow-sm">
        <div class="flex items-center gap-3">
          <KeyRound class="size-4 text-muted-foreground" />
          <div class="min-w-0 flex-1">
            <h2 class="text-sm font-semibold">Live Apply Prompts</h2>
            <p class="text-xs text-muted-foreground">Required only before live apply actions</p>
          </div>
          <StatusBadge
            status="ok"
            :label="approvalOptions.length ? 'Configured' : 'Ready'"
          />
        </div>

        <div class="mt-4 grid gap-2">
          <div
            v-for="item in approvalOptions"
            :key="`${item.workflow}-${item.option}`"
            class="flex items-center gap-3 rounded-md border border-section-border bg-section-muted px-3 py-2 text-sm"
          >
            <div class="min-w-0 flex-1">
              <div class="truncate font-medium">{{ item.workflow }}</div>
              <div class="truncate text-xs text-muted-foreground">{{ item.option }}</div>
            </div>
            <Badge variant="outline">Prompt on apply</Badge>
          </div>
        </div>
      </section>

      <section
        v-if="failingChecks.length"
        class="rounded-lg border border-destructive/40 bg-section p-4 shadow-sm"
      >
        <div class="flex items-center gap-2 text-sm font-semibold text-destructive">
          <TriangleAlert class="size-4" />
          Failed Checks
        </div>
        <ul class="mt-3 grid gap-2 text-sm text-muted-foreground">
          <li v-for="check in failingChecks" :key="check.key">
            {{ check.label }}: {{ check.message }}
          </li>
        </ul>
      </section>
    </div>
  </ScrollArea>
</template>
