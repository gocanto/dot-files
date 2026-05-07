<script setup lang="ts">
import type { StepperTriggerProps } from "reka-ui";
import { StepperTrigger } from "reka-ui";

const props = defineProps<StepperTriggerProps>();
</script>

<template>
  <StepperTrigger data-slot="stepper-trigger" v-bind="props">
    <slot />
  </StepperTrigger>
</template>
