<script setup lang="ts">
import OutputBlock from "@/components/OutputBlock.vue";
import StatusBadge from "@/components/StatusBadge.vue";
import type { RunOutputSection } from "@/components/app/types";

defineProps<{
  sections: RunOutputSection[];
  emptyText: string;
}>();
</script>

<template>
  <div class="grid gap-3 p-3" data-testid="run-output-sections">
    <div v-if="sections.length === 0" class="rounded-md border border-white/10 bg-terminal p-4">
      <OutputBlock :code="''" :empty-text="emptyText" />
    </div>

    <section
      v-for="section in sections"
      :key="section.id"
      class="overflow-hidden rounded-md border border-white/10 bg-terminal"
    >
      <div
        class="flex items-center justify-between gap-3 border-b border-white/10 bg-white/[0.06] px-3 py-2"
      >
        <div class="min-w-0">
          <div class="text-[11px] font-medium uppercase text-terminal-foreground/60">
            {{ section.context }}
          </div>
          <div class="truncate text-xs font-semibold text-terminal-foreground">
            {{ section.label }}
          </div>
        </div>
        <StatusBadge :status="section.status" />
      </div>
      <OutputBlock :code="section.code" :empty-text="emptyText" class="text-xs leading-5" />
    </section>
  </div>
</template>
