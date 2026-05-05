<script setup lang="ts">
import type { HTMLAttributes } from "vue";
import { cn } from "@/lib/utils";

const props = defineProps<{ class?: HTMLAttributes["class"] }>();
</script>

<template>
  <tr :class="cn('border-b transition-colors hover:bg-accent/60 data-[state=selected]:bg-muted', props.class)">
    <slot />
  </tr>
</template>
