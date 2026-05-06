<script setup lang="ts">
import type { StepperItemProps } from "reka-ui";
import type { HTMLAttributes } from "vue";
import { reactiveOmit } from "@vueuse/core";
import { StepperItem } from "reka-ui";
import { cn } from "@/lib/utils";

const props = defineProps<StepperItemProps & { class?: HTMLAttributes["class"] }>();
const delegatedProps = reactiveOmit(props, "class");
</script>

<template>
  <StepperItem data-slot="stepper-item" v-bind="delegatedProps" :class="cn('group', props.class)">
    <slot />
  </StepperItem>
</template>
