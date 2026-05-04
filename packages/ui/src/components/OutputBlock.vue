<script setup lang="ts">
import { highlightAnsiOutput } from "@/lib/shiki";
import { computed, ref, watchEffect } from "vue";

const props = withDefaults(
  defineProps<{
    code?: string;
    emptyText: string;
  }>(),
  {
    code: "",
  },
);

const displayCode = computed(() => props.code || props.emptyText);
const highlightedHtml = ref("");

watchEffect((onCleanup) => {
  const source = displayCode.value;
  let cancelled = false;

  highlightedHtml.value = "";
  onCleanup(() => {
    cancelled = true;
  });

  void highlightAnsiOutput(source)
    .then((html) => {
      if (!cancelled) {
        highlightedHtml.value = html;
      }
    })
    .catch(() => {
      if (!cancelled) {
        highlightedHtml.value = "";
      }
    });
});
</script>

<template>
  <div class="output-block min-h-full font-mono" data-testid="output-block">
    <div v-if="highlightedHtml" v-html="highlightedHtml" />
    <pre v-else class="min-h-full whitespace-pre-wrap p-4">{{ displayCode }}</pre>
  </div>
</template>

<style scoped>
.output-block {
  background: var(--card);
  color: var(--card-foreground);
}

.output-block :deep(pre.shiki) {
  min-height: 100%;
  margin: 0;
  overflow: visible;
  padding: 1rem;
  white-space: pre-wrap;
}

.output-block :deep(code) {
  font-family: inherit;
}
</style>
