<script setup lang="ts">
import { computed, nextTick, ref, watch } from "vue";
import { AlertTriangle, Check, Circle, Dot, Loader2, X } from "lucide-vue-next";
import {
  AlertDialog,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@ui/alert-dialog";
import { Button } from "@ui/button";
import { ScrollArea } from "@ui/scroll-area";
import { Skeleton } from "@ui/skeleton";
import {
  Stepper,
  StepperDescription,
  StepperIndicator,
  StepperItem,
  StepperSeparator,
  StepperTitle,
  StepperTrigger,
} from "@ui/stepper";
import RunOutputSections from "@app/RunOutputSections.vue";
import StatusBadge from "@components/StatusBadge.vue";
import { cn } from "@lib/utils";
import type { ConfirmationOption } from "@api";
import type { DisplayPhase, RunOutputSection } from "@app/types";

const props = defineProps<{
  pendingOption: ConfirmationOption | null;
  title: string;
  summary: string;
  running: boolean;
  phases: DisplayPhase[];
  outputText: string;
  outputSections: RunOutputSection[];
  runStatus: string;
}>();

const emit = defineEmits<{
  (event: "update:open", open: boolean): void;
  (event: "continue", option: ConfirmationOption): void;
}>();

const bodyBottom = ref<HTMLElement | null>(null);
const headerTitle = computed(() => props.title || props.pendingOption?.label || "Run workflow");
const headerSummary = computed(() => props.summary || props.pendingOption?.description || "");
const terminalRunStatuses = new Set(["completed", "failed", "stopped"]);

const activeStep = computed(() => {
  const runningIndex = props.phases.findIndex((phase) => phase.status === "running");

  if (runningIndex >= 0) {
    return runningIndex + 1;
  }

  const failedIndex = props.phases.findIndex((phase) => phase.status === "failed");

  if (failedIndex >= 0) {
    return failedIndex + 1;
  }

  const firstPending = props.phases.findIndex((phase) => phase.status === "pending");

  return firstPending >= 0 ? firstPending + 1 : Math.max(1, props.phases.length);
});

function phaseTone(status: string) {
  if (status === "failed") return "failed";
  if (status === "running") return "running";
  if (["ok", "completed", "skipped"].includes(status)) return "completed";

  return "pending";
}

watch(
  () => [props.runStatus, props.running] as const,
  async ([status, running], [previousStatus, previousRunning]) => {
    if (
      running ||
      !terminalRunStatuses.has(status) ||
      (previousStatus === status && previousRunning === running)
    ) {
      return;
    }

    await nextTick();

    bodyBottom.value?.scrollIntoView({ block: "end", behavior: "smooth" });
  },
);
</script>

<template>
  <AlertDialog
    :open="pendingOption !== null"
    @update:open="running ? emit('update:open', true) : emit('update:open', $event)"
  >
    <AlertDialogContent
      class="h-[min(760px,calc(100vh-2rem))] grid-rows-[auto_minmax(0,1fr)_auto] sm:max-w-4xl"
    >
      <AlertDialogHeader>
        <AlertDialogTitle class="flex items-center gap-2">
          <AlertTriangle class="size-5 text-destructive" />
          {{ headerTitle }}
        </AlertDialogTitle>
        <AlertDialogDescription>{{ headerSummary }}</AlertDialogDescription>
      </AlertDialogHeader>

      <ScrollArea class="min-h-0 bg-background" data-testid="alert-dialog-body">
        <div class="grid min-h-full grid-rows-[auto_minmax(24rem,1fr)] gap-5 p-6">
          <div
            v-if="running && phases.length === 0"
            class="grid gap-3"
            data-testid="progress-skeleton"
          >
            <div class="grid grid-cols-3 gap-3">
              <Skeleton v-for="index in 3" :key="index" class="h-16 rounded-lg" />
            </div>
            <Skeleton class="h-48 rounded-md" />
          </div>

          <Stepper v-else :model-value="activeStep" class="items-start gap-2">
            <StepperItem
              v-for="(phase, index) in phases"
              :key="phase.id"
              :step="index + 1"
              class="relative flex w-full flex-col items-center justify-center"
            >
              <StepperSeparator
                v-if="index !== phases.length - 1"
                class="absolute left-[calc(50%+20px)] right-[calc(-50%+10px)] top-5 block h-0.5 rounded-full"
                :class="phaseTone(phase.status) === 'completed' && 'bg-primary'"
              />
              <StepperTrigger>
                <StepperIndicator
                  class="z-10 bg-background"
                  :class="
                    cn(
                      phaseTone(phase.status) === 'completed' &&
                        'border-success bg-success text-success-foreground',
                      phaseTone(phase.status) === 'running' && 'border-primary ring-2 ring-ring',
                      phaseTone(phase.status) === 'failed' &&
                        'border-destructive bg-destructive text-destructive-foreground',
                    )
                  "
                >
                  <Check v-if="phaseTone(phase.status) === 'completed'" class="size-4" />
                  <Loader2
                    v-else-if="phaseTone(phase.status) === 'running'"
                    class="size-4 animate-spin"
                  />
                  <X v-else-if="phaseTone(phase.status) === 'failed'" class="size-4" />
                  <Circle v-else-if="phase.enabled" class="size-4" />
                  <Dot v-else class="size-4" />
                </StepperIndicator>
              </StepperTrigger>
              <div class="mt-3 flex max-w-36 flex-col items-center text-center">
                <StepperTitle class="line-clamp-2 text-xs">{{ phase.name }}</StepperTitle>
                <StepperDescription class="mt-1">{{ phase.status }}</StepperDescription>
              </div>
            </StepperItem>
          </Stepper>

          <div
            class="grid min-h-96 grid-rows-[auto_minmax(0,1fr)] gap-2"
            data-testid="run-output-panel"
          >
            <div class="flex items-center justify-between text-xs text-muted-foreground">
              <span>Run output</span>
              <StatusBadge :status="runStatus" />
            </div>
            <ScrollArea
              class="min-h-0 rounded-md border border-section-border bg-terminal text-terminal-foreground"
            >
              <RunOutputSections
                :sections="outputSections"
                empty-text="Waiting for workflow output..."
              />
            </ScrollArea>
          </div>
          <div ref="bodyBottom" data-testid="dialog-body-bottom" aria-hidden="true" />
        </div>
      </ScrollArea>

      <AlertDialogFooter>
        <AlertDialogCancel :disabled="running">
          {{ running ? "Running" : outputText ? "Close" : "Cancel" }}
        </AlertDialogCancel>
        <Button
          :disabled="running || !pendingOption || Boolean(outputText)"
          @click="pendingOption && emit('continue', pendingOption)"
        >
          <Loader2 v-if="running" class="size-4 animate-spin" />
          Continue
        </Button>
      </AlertDialogFooter>
    </AlertDialogContent>
  </AlertDialog>
</template>
