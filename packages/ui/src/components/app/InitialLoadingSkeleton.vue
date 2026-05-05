<script setup lang="ts">
import { Separator } from "@/components/ui/separator";
import { Skeleton } from "@/components/ui/skeleton";
</script>

<template>
  <div
    data-testid="initial-shell-skeleton"
    class="grid h-screen grid-cols-[230px_410px_1fr] overflow-hidden"
  >
    <aside class="border-r border-sidebar-border bg-sidebar">
      <div class="flex h-12 items-center gap-2 px-3">
        <Skeleton class="size-4 rounded" />
        <div class="grid flex-1 gap-1">
          <Skeleton class="h-4 w-24" />
          <Skeleton class="h-3 w-32" />
        </div>
      </div>
      <Separator />
      <div class="grid gap-1 p-2">
        <div
          v-for="index in 5"
          :key="`nav-shell-${index}`"
          class="flex h-8 items-center gap-2 rounded-md px-2"
        >
          <Skeleton class="size-4 rounded" />
          <Skeleton class="h-4 w-28" />
          <Skeleton v-if="index < 4" class="ml-auto h-4 w-5 rounded" />
        </div>
      </div>
    </aside>

    <section class="border-r border-section-border bg-panel">
      <div class="flex h-12 items-center bg-section px-4">
        <Skeleton class="h-6 w-28" />
      </div>
      <Separator />
      <div class="border-b border-section-border bg-section-muted p-4">
        <Skeleton class="h-9 w-full" />
      </div>
      <div class="grid gap-2 px-4">
        <div
          v-for="index in 6"
          :key="`list-shell-${index}`"
          class="rounded-lg border border-section-border bg-section p-3"
        >
          <div class="flex items-center gap-3">
            <Skeleton class="h-4 w-40" />
            <Skeleton class="ml-auto h-5 w-12 rounded-full" />
          </div>
          <Skeleton class="mt-3 h-3 w-24" />
          <Skeleton class="mt-3 h-3 w-full" />
          <Skeleton class="mt-2 h-3 w-4/5" />
        </div>
      </div>
    </section>

    <section class="flex min-h-0 flex-col bg-background">
      <div class="flex h-12 items-center gap-2 bg-section px-2">
        <Skeleton v-for="index in 3" :key="`toolbar-left-${index}`" class="size-8 rounded-md" />
        <div class="ml-auto flex gap-2">
          <Skeleton v-for="index in 3" :key="`toolbar-right-${index}`" class="size-8 rounded-md" />
        </div>
      </div>
      <Separator />
      <div class="flex items-start p-4">
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
        <section class="rounded-lg border border-section-border bg-section p-4 shadow-sm">
          <div class="mb-2 flex items-center justify-between">
            <Skeleton class="h-4 w-20" />
            <Skeleton class="h-8 w-16" />
          </div>
          <div class="overflow-hidden rounded-lg border border-section-border bg-section-muted">
            <div
              v-for="index in 4"
              :key="`phase-shell-${index}`"
              class="flex items-center gap-3 border-b px-3 py-3 last:border-b-0"
            >
              <Skeleton class="size-4 rounded-full" />
              <Skeleton class="h-4 flex-1" />
              <Skeleton class="h-5 w-16 rounded-full" />
            </div>
          </div>
        </section>
        <section class="rounded-lg border border-section-border bg-section p-4 shadow-sm">
          <Skeleton class="mb-2 h-4 w-28" />
          <Skeleton class="h-72 w-full rounded-lg" />
        </section>
      </div>
    </section>
  </div>
</template>
