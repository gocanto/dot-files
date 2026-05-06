<script setup lang="ts">
import type { StepperTitleProps } from "reka-ui";
import type { HTMLAttributes } from "vue";
import { reactiveOmit } from "@vueuse/core";
import { StepperTitle } from "reka-ui";
import { cn } from "@lib/utils";

const props = defineProps<StepperTitleProps & { class?: HTMLAttributes["class"] }>();
const delegatedProps = reactiveOmit(props, "class");
</script>

<template>
  <StepperTitle
    data-slot="stepper-title"
    v-bind="delegatedProps"
    :class="cn('text-sm font-medium leading-none', props.class)"
  >
    <slot />
  </StepperTitle>
</template>
