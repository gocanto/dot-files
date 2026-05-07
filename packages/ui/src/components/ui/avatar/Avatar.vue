<script setup lang="ts">
import type { HTMLAttributes } from "vue";
import type { AvatarVariants } from "@ui/avatar/variants";
import { AvatarRoot } from "reka-ui";
import { cn } from "@lib/utils";
import { avatarVariant } from "@ui/avatar/variants";

const props = withDefaults(
  defineProps<{
    class?: HTMLAttributes["class"];
    size?: AvatarVariants["size"];
    shape?: AvatarVariants["shape"];
  }>(),
  {
    size: "sm",
    shape: "circle",
  },
);
</script>

<template>
  <AvatarRoot :class="cn(avatarVariant({ size, shape }), props.class)">
    <slot />
  </AvatarRoot>
</template>
