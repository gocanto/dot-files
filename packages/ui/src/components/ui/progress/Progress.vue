<script setup lang="ts">
import type { HTMLAttributes } from "vue";
import { computed } from "vue";
import { cn } from "@lib/utils";

const props = withDefaults(
  defineProps<{
    value?: number;
    class?: HTMLAttributes["class"];
  }>(),
  {
    value: 0,
  },
);

const normalizedValue = computed(() => {
  if (!Number.isFinite(props.value)) {
    return 0;
  }

  return Math.min(100, Math.max(0, props.value));
});
</script>

<template>
  <div
    data-slot="progress"
    :class="cn('h-2 overflow-hidden rounded-full border bg-muted', props.class)"
    role="progressbar"
    :aria-valuenow="normalizedValue"
    aria-valuemin="0"
    aria-valuemax="100"
  >
    <div
      class="h-full rounded-full bg-success transition-[width] duration-300"
      :style="{ width: `${normalizedValue}%` }"
    />
  </div>
</template>
