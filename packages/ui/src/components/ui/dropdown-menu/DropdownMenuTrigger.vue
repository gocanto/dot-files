<script setup lang="ts">
import type { DropdownMenuTriggerProps } from "reka-ui";
import { DropdownMenuTrigger, useForwardProps } from "reka-ui";

const props = defineProps<DropdownMenuTriggerProps>();

const forwardedProps = useForwardProps(props);
</script>

<template>
  <DropdownMenuTrigger
    class="outline-hidden focus-visible:ring-2 focus-visible:ring-ring"
    v-bind="forwardedProps"
  >
    <slot />
  </DropdownMenuTrigger>
</template>
