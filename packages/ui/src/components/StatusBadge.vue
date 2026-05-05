<script setup lang="ts">
import { computed } from "vue";
import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/utils";
import { statusIconFor, statusPillClass } from "@/lib/phaseDisplay";

const props = defineProps<{ status: string; label?: string; class?: string }>();

const icon = computed(() => statusIconFor(props.status));
const isRunning = computed(() => props.status === "running");
</script>

<template>
  <Badge variant="outline" :class="cn(statusPillClass(status), props.class)">
    <component :is="icon" :class="isRunning ? 'animate-spin' : ''" />
    {{ label ?? status }}
  </Badge>
</template>
