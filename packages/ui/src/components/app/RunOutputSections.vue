<script setup lang="ts">
import OutputBlock from "@components/OutputBlock.vue";
import StatusBadge from "@components/StatusBadge.vue";
import type { RunOutputSection } from "@app/types";
import { ChevronDown } from "lucide-vue-next";
import { ref, watch } from "vue";

const props = withDefaults(
  defineProps<{
    sections: RunOutputSection[];
    emptyText: string;
    defaultExpanded?: boolean;
  }>(),
  {
    defaultExpanded: false,
  },
);

const expandedSectionIds = ref(new Set<string>());

watch(
  () => props.sections.map((section) => section.id),
  (sectionIds) => {
    if (props.defaultExpanded) {
      expandedSectionIds.value = new Set(sectionIds);
      return;
    }

    expandedSectionIds.value = new Set(
      [...expandedSectionIds.value].filter((sectionId) => sectionIds.includes(sectionId)),
    );
  },
  { immediate: true },
);

function isExpanded(sectionId: string) {
  return expandedSectionIds.value.has(sectionId);
}

function toggleSection(sectionId: string) {
  const nextExpandedSectionIds = new Set(expandedSectionIds.value);

  if (nextExpandedSectionIds.has(sectionId)) {
    nextExpandedSectionIds.delete(sectionId);
  } else {
    nextExpandedSectionIds.add(sectionId);
  }

  expandedSectionIds.value = nextExpandedSectionIds;
}
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
      <button
        type="button"
        class="flex w-full cursor-pointer items-center justify-between gap-3 bg-white/[0.06] px-3 py-2 text-left transition-colors hover:bg-white/[0.09] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-terminal"
        :class="isExpanded(section.id) ? 'border-b border-white/10' : ''"
        :aria-expanded="isExpanded(section.id)"
        :aria-controls="`run-output-section-${section.id}`"
        @click="toggleSection(section.id)"
      >
        <div class="flex min-w-0 items-center gap-2">
          <ChevronDown
            class="size-4 shrink-0 text-terminal-foreground/60 transition-transform"
            :class="isExpanded(section.id) ? 'rotate-0' : '-rotate-90'"
            aria-hidden="true"
          />
          <div class="min-w-0">
            <div class="text-[11px] font-medium uppercase text-terminal-foreground/60">
              {{ section.context }}
            </div>
            <div class="truncate text-xs font-semibold text-terminal-foreground">
              {{ section.label }}
            </div>
          </div>
        </div>
        <StatusBadge :status="section.status" />
      </button>
      <OutputBlock
        v-if="isExpanded(section.id)"
        :id="`run-output-section-${section.id}`"
        :code="section.code"
        :empty-text="emptyText"
        class="text-xs leading-5"
      />
    </section>
  </div>
</template>
