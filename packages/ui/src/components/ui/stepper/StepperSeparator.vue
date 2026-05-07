<script setup lang="ts">
import type { StepperSeparatorProps } from "reka-ui";
import type { HTMLAttributes } from "vue";
import { reactiveOmit } from "@vueuse/core";
import { StepperSeparator } from "reka-ui";
import { cn } from "@lib/utils";

const props = defineProps<StepperSeparatorProps & { class?: HTMLAttributes["class"] }>();
const delegatedProps = reactiveOmit(props, "class");
</script>

<template>
  <StepperSeparator
    data-slot="stepper-separator"
    v-bind="delegatedProps"
    :class="cn('shrink-0 bg-muted group-data-[state=completed]:bg-primary', props.class)"
  />
</template>
