<script setup lang="ts">
import type { HTMLAttributes } from "vue";
import { LaptopMinimal } from "lucide-vue-next";
import { Avatar, AvatarFallback, AvatarImage, type AvatarVariants } from "@ui/avatar";
import { cn } from "@lib/utils";

const props = withDefaults(
  defineProps<{
    src?: string;
    alt?: string;
    fallbackLabel?: string;
    class?: HTMLAttributes["class"];
    iconClass?: HTMLAttributes["class"];
    size?: AvatarVariants["size"];
    shape?: AvatarVariants["shape"];
  }>(),
  {
    alt: "Machine avatar",
    fallbackLabel: "Default machine avatar",
    size: "sm",
    shape: "circle",
  },
);
</script>

<template>
  <Avatar
    :size="size"
    :shape="shape"
    :class="
      cn('border border-border bg-background/70 text-muted-foreground shadow-sm', props.class)
    "
    data-testid="machine-avatar"
  >
    <AvatarImage v-if="src" :src="src" :alt="alt" />
    <AvatarFallback
      :aria-label="fallbackLabel"
      class="flex h-full w-full items-center justify-center"
    >
      <LaptopMinimal :class="cn('size-5', iconClass)" aria-hidden="true" />
      <span class="sr-only">{{ fallbackLabel }}</span>
    </AvatarFallback>
  </Avatar>
</template>
