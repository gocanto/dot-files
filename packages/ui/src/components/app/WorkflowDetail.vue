<script setup lang="ts">
import { CheckCircle2, Circle, Files, KeyRound } from "lucide-vue-next";
import { Button } from "@ui/button";
import { Progress } from "@ui/progress";
import { ScrollArea } from "@ui/scroll-area";
import StatusBadge from "@components/StatusBadge.vue";
import { detailSectionBodyClass, detailSectionClass } from "@app/styles";
import type { DisplayPhase } from "@app/types";
import { confirmationStyle } from "@lib/confirmationDisplay";
import type { WorkflowDetail as WorkflowDetailInfo } from "@lib/workflowDetails";
import { confirmationIcon, phaseIcon } from "@lib/workflowIcons";
import { cn } from "@lib/utils";
import type { ConfirmationOption, Phase, Workflow } from "@api";

defineProps<{
  selectedWorkflow: Workflow;
  selectedWorkflowDetail: WorkflowDetailInfo | null;
  displayPhases: DisplayPhase[];
  workflowProgress: number;
}>();

const emit = defineEmits<{
  (event: "reset-phases"): void;
  (event: "toggle-phase", phase: Phase): void;
  (event: "open-confirmation", option: ConfirmationOption): void;
  (event: "open-template-files"): void;
  (event: "close-detail"): void;
}>();
</script>

<template>
  <ScrollArea class="min-h-0 flex-1">
    <div class="grid gap-5 p-4">
      <section
        v-if="
          selectedWorkflowDetail &&
          (selectedWorkflowDetail.purpose ||
            selectedWorkflowDetail.details ||
            selectedWorkflowDetail.whenToRun ||
            selectedWorkflowDetail.sideEffects.length ||
            selectedWorkflowDetail.prerequisites.length)
        "
        :class="detailSectionClass"
      >
        <h2 class="mb-2 text-sm font-semibold">About this workflow</h2>
        <div :class="detailSectionBodyClass">
          <div v-if="selectedWorkflowDetail.purpose">
            <div class="text-xs font-medium uppercase tracking-wide text-muted-foreground">
              Purpose
            </div>
            <p class="mt-1 text-sm leading-6">
              {{ selectedWorkflowDetail.purpose }}
            </p>
          </div>
          <div v-if="selectedWorkflowDetail.details">
            <div class="text-xs font-medium uppercase tracking-wide text-muted-foreground">
              What it does
            </div>
            <p class="mt-1 text-sm leading-6">
              {{ selectedWorkflowDetail.details }}
            </p>
          </div>
          <div v-if="selectedWorkflowDetail.whenToRun">
            <div class="text-xs font-medium uppercase tracking-wide text-muted-foreground">
              When to run
            </div>
            <p class="mt-1 text-sm leading-6">
              {{ selectedWorkflowDetail.whenToRun }}
            </p>
          </div>
          <div v-if="selectedWorkflowDetail.sideEffects.length">
            <div class="text-xs font-medium uppercase tracking-wide text-muted-foreground">
              Side effects
            </div>
            <ul class="mt-1 list-disc pl-5 text-sm leading-6">
              <li v-for="effect in selectedWorkflowDetail.sideEffects" :key="effect">
                {{ effect }}
              </li>
            </ul>
          </div>
          <div v-if="selectedWorkflowDetail.prerequisites.length">
            <div class="text-xs font-medium uppercase tracking-wide text-muted-foreground">
              Prerequisites
            </div>
            <ul class="mt-1 list-disc pl-5 text-sm leading-6">
              <li v-for="prereq in selectedWorkflowDetail.prerequisites" :key="prereq">
                {{ prereq }}
              </li>
            </ul>
          </div>
        </div>
      </section>

      <section
        v-if="
          selectedWorkflowDetail?.category === 'template' &&
          selectedWorkflow.id !== 'review-template'
        "
        :class="detailSectionClass"
      >
        <h2 class="mb-1 text-sm font-semibold">Update Template Files</h2>
        <p class="mb-3 text-sm leading-6 text-muted-foreground">
          Open the expanded editor for allowlisted manifests and stow dotfiles. Back returns here
          with your selected phases preserved.
        </p>
        <Button
          variant="ghost"
          class="h-auto w-full justify-start gap-3 whitespace-normal border border-section-border bg-section-muted px-3 py-2 text-left text-foreground hover:bg-accent hover:text-foreground"
          @click="emit('open-template-files')"
        >
          <Files class="size-4 shrink-0 text-muted-foreground" />
          <span class="min-w-0 flex-1">
            <span class="block font-medium">Update Template Files</span>
            <span class="block text-xs text-muted-foreground">
              Edit apps, secrets, generated app lists, and stow files
            </span>
          </span>
        </Button>
      </section>

      <section :class="detailSectionClass">
        <div class="mb-2 flex items-center justify-between gap-3">
          <h2 class="text-sm font-semibold">Phases</h2>
          <div class="flex items-center gap-3">
            <Progress :value="workflowProgress" class="w-28" />
            <Button variant="ghost" size="sm" @click="emit('reset-phases')">Reset</Button>
          </div>
        </div>
        <div class="overflow-hidden rounded-lg border border-section-border bg-section-muted">
          <button
            v-for="phase in displayPhases"
            :key="phase.id"
            class="flex w-full items-center gap-3 border-b border-section-border px-3 py-3 text-left text-sm transition-colors last:border-b-0 hover:bg-accent"
            @click="emit('toggle-phase', phase)"
          >
            <CheckCircle2 v-if="phase.enabled" class="size-4 shrink-0 text-primary" />
            <Circle v-else class="size-4 shrink-0 text-muted-foreground" />
            <component :is="phaseIcon(phase.id)" class="size-4 shrink-0 text-muted-foreground" />
            <span class="min-w-0 flex-1 truncate">{{ phase.name }}</span>
            <StatusBadge :status="phase.status" />
          </button>
        </div>
      </section>

      <section v-if="selectedWorkflow.confirmation" :class="detailSectionClass">
        <h2 class="mb-2 text-sm font-semibold">
          {{ selectedWorkflow.confirmation.title }}
        </h2>
        <p class="mb-3 text-sm leading-6 text-muted-foreground">
          {{ selectedWorkflow.confirmation.message }}
        </p>
        <div class="grid gap-2">
          <Button
            v-for="option in selectedWorkflow.confirmation.options"
            :key="option.id"
            variant="outline"
            :class="
              cn(
                'h-auto justify-start gap-3 whitespace-normal px-3 py-2 text-left',
                confirmationStyle(option.id).buttonClass,
              )
            "
            @click="option.back ? emit('close-detail') : emit('open-confirmation', option)"
          >
            <component
              :is="confirmationIcon(option.id)"
              :class="cn('size-4 shrink-0', confirmationStyle(option.id).iconClass)"
            />
            <span class="min-w-0 flex-1">
              <span class="block font-medium">{{ option.label }}</span>
              <span class="block text-xs text-muted-foreground">{{ option.description }}</span>
              <span
                v-if="option.requiresApproval"
                class="mt-1 flex items-center gap-1 text-xs font-medium text-amber-700 dark:text-amber-300"
              >
                <KeyRound class="size-3" />
                Host password approval required before apply
              </span>
            </span>
          </Button>
        </div>
      </section>
    </div>
  </ScrollArea>
</template>
