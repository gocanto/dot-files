<script setup lang="ts">
import { CheckCircle2, Info, Loader2, X, XCircle } from "lucide-vue-next";
import type { Component, HTMLAttributes } from "vue";
import { cn } from "@/lib/utils";

export type ToastTone = "info" | "success" | "error" | "loading";

export interface ToastItem {
  id: string;
  title: string;
  description?: string;
  tone?: ToastTone;
}

const props = defineProps<{
  toasts: ToastItem[];
  class?: HTMLAttributes["class"];
}>();

const emit = defineEmits<{
  dismiss: [id: string];
}>();

const icons: Record<ToastTone, Component> = {
  info: Info,
  success: CheckCircle2,
  error: XCircle,
  loading: Loader2,
};

function toneClass(tone: ToastTone = "info") {
  if (tone === "success") return "border-[var(--status-success-border)]";
  if (tone === "error") return "border-[var(--status-danger-border)]";
  return "border-border";
}

function iconClass(tone: ToastTone = "info") {
  if (tone === "success") return "text-[var(--status-success-fg)]";
  if (tone === "error") return "text-[var(--status-danger-fg)]";
  if (tone === "loading") return "animate-spin text-primary";
  return "text-primary";
}
</script>

<template>
  <div
    v-if="toasts.length"
    data-testid="toast-viewport"
    :class="cn('pointer-events-none fixed right-4 bottom-4 z-50 grid w-80 gap-2', props.class)"
  >
    <div
      v-for="toast in toasts"
      :key="toast.id"
      :class="cn('pointer-events-auto rounded-md border bg-popover p-3 text-popover-foreground shadow-overlay', toneClass(toast.tone))"
    >
      <div class="flex items-start gap-3">
        <component :is="icons[toast.tone ?? 'info']" :class="cn('mt-0.5 size-4 shrink-0', iconClass(toast.tone))" />
        <div class="min-w-0 flex-1">
          <div class="text-sm font-medium">{{ toast.title }}</div>
          <div v-if="toast.description" class="mt-1 text-xs text-muted-foreground">{{ toast.description }}</div>
        </div>
        <button
          type="button"
          class="rounded-sm p-1 text-muted-foreground hover:bg-muted hover:text-foreground"
          @click="emit('dismiss', toast.id)"
        >
          <X class="size-3.5" />
          <span class="sr-only">Dismiss</span>
        </button>
      </div>
    </div>
  </div>
</template>
