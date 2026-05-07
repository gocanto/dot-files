<script setup lang="ts">
import type { SelectLabelProps } from "reka-ui";
import type { HTMLAttributes } from "vue";
import { reactiveOmit } from "@vueuse/core";
import { SelectLabel, useForwardProps } from "reka-ui";
import { cn } from "@lib/utils";

const props = defineProps<SelectLabelProps & { class?: HTMLAttributes["class"] }>();
const delegatedProps = reactiveOmit(props, "class");
const forwardedProps = useForwardProps(delegatedProps);
</script>

<template>
  <SelectLabel
    v-bind="forwardedProps"
    :class="cn('px-2 py-1.5 text-sm font-semibold', props.class)"
  >
    <slot />
  </SelectLabel>
</template>
