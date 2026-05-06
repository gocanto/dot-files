<script setup lang="ts">
import type { StepperIndicatorProps } from "reka-ui";
import type { HTMLAttributes } from "vue";
import { reactiveOmit } from "@vueuse/core";
import { StepperIndicator } from "reka-ui";
import { cn } from "@/lib/utils";

const props = defineProps<StepperIndicatorProps & { class?: HTMLAttributes["class"] }>();
const delegatedProps = reactiveOmit(props, "class");
</script>

<template>
  <StepperIndicator
    data-slot="stepper-indicator"
    v-bind="delegatedProps"
    :class="
      cn(
        'flex size-9 shrink-0 items-center justify-center rounded-full border text-xs font-semibold transition-colors',
        props.class,
      )
    "
  >
    <slot />
  </StepperIndicator>
</template>
