<script setup lang="ts">
import { Separator } from "@ui/separator";
import { Skeleton } from "@ui/skeleton";
import { detailSectionClass } from "@app/styles";
</script>

<template>
  <div data-testid="workflow-detail-skeleton" class="flex min-h-0 flex-1 flex-col">
    <div class="flex items-start bg-section p-4">
      <Skeleton class="size-10 rounded-full" />
      <div class="ml-4 grid flex-1 gap-2">
        <Skeleton class="h-4 w-48" />
        <Skeleton class="h-3 w-72" />
        <Skeleton class="h-3 w-40" />
      </div>
      <Skeleton class="h-5 w-20 rounded-full" />
    </div>
    <Separator />
    <div class="grid gap-5 p-4">
      <section :class="detailSectionClass">
        <div class="mb-2 flex items-center justify-between">
          <Skeleton class="h-4 w-20" />
          <Skeleton class="h-8 w-16" />
        </div>
        <div class="overflow-hidden rounded-lg border border-section-border bg-section-muted">
          <div
            v-for="index in 4"
            :key="index"
            class="flex items-center gap-3 border-b px-3 py-3 last:border-b-0"
          >
            <Skeleton class="size-4 rounded-full" />
            <Skeleton class="h-4 flex-1" />
            <Skeleton class="h-5 w-16 rounded-full" />
          </div>
        </div>
      </section>
      <section :class="detailSectionClass">
        <Skeleton class="mb-2 h-4 w-28" />
        <Skeleton class="h-72 w-full rounded-lg" />
      </section>
    </div>
  </div>
</template>
