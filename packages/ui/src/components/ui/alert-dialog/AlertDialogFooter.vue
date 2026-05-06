<script setup lang="ts">
import type { HTMLAttributes } from "vue";
import { cn } from "@lib/utils";

const props = defineProps<{
  class?: HTMLAttributes["class"];
}>();
</script>

<template>
  <div
    data-slot="alert-dialog-footer"
    :class="
      cn(
        'flex flex-col-reverse gap-2 border-t border-section-border bg-section-muted px-6 py-4 sm:flex-row sm:justify-end',
        props.class,
      )
    "
  >
    <slot />
  </div>
</template>
