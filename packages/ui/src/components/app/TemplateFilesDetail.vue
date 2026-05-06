<script setup lang="ts">
import { computed, ref } from "vue";
import { AlertTriangle, FileText, RefreshCw, Save, Search } from "lucide-vue-next";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Skeleton } from "@/components/ui/skeleton";
import { Textarea } from "@/components/ui/textarea";
import { detailSectionBodyClass, detailSectionClass } from "@/components/app/styles";
import { cn } from "@/lib/utils";
import type { TemplateFileSummary } from "@/types/api";

const props = defineProps<{
  files: TemplateFileSummary[];
  filesLoading: boolean;
  contentLoading: boolean;
  saving: boolean;
  dirty: boolean;
  selectedPath: string;
  draft: string;
  error: string;
  message: string;
}>();

const emit = defineEmits<{
  (event: "update:draft", value: string): void;
  (event: "refresh-files"): void;
  (event: "select-file", file: TemplateFileSummary): void;
  (event: "save-file"): void;
}>();

const search = ref("");

const filteredFiles = computed(() => {
  const query = search.value.trim().toLowerCase();

  if (!query) return props.files;

  return props.files.filter((file) =>
    [file.relative, file.kind, file.path].join(" ").toLowerCase().includes(query),
  );
});

const selectedFile = computed(() => props.files.find((file) => file.path === props.selectedPath));
</script>

<template>
  <div class="grid min-h-0 flex-1 grid-cols-[minmax(260px,32%)_1fr]">
    <div class="flex min-h-0 flex-col border-r border-section-border bg-section">
      <div class="space-y-3 border-b border-section-border p-4">
        <div class="flex items-center gap-2">
          <FileText class="size-4 text-muted-foreground" />
          <h2 class="text-sm font-semibold">Template Files</h2>
          <Button
            type="button"
            variant="ghost"
            size="icon"
            class="ml-auto size-8"
            :disabled="filesLoading"
            aria-label="Refresh template files"
            @click="emit('refresh-files')"
          >
            <RefreshCw :class="cn('size-4', filesLoading && 'animate-spin')" />
          </Button>
        </div>
        <div class="relative">
          <Search class="absolute left-2 top-2.5 size-4 text-muted-foreground" />
          <Input v-model="search" placeholder="Search files" class="pl-8" />
        </div>
      </div>

      <ScrollArea class="min-h-0 flex-1">
        <div v-if="filesLoading" class="grid gap-2 p-4" data-testid="template-files-skeleton">
          <div v-for="index in 8" :key="index" class="rounded-lg border border-section-border p-3">
            <Skeleton class="h-4 w-40" />
            <Skeleton class="mt-2 h-3 w-24" />
          </div>
        </div>
        <div v-else class="grid gap-1 p-4">
          <button
            v-for="file in filteredFiles"
            :key="file.path"
            class="rounded-lg border border-section-border bg-section-muted p-3 text-left text-sm transition-colors hover:bg-accent"
            :class="selectedPath === file.path && 'border-primary bg-accent'"
            @click="emit('select-file', file)"
          >
            <div class="truncate font-medium">{{ file.relative }}</div>
            <div class="mt-1 flex items-center gap-2 text-xs text-muted-foreground">
              <span>{{ file.kind }}</span>
              <span>{{ file.exists ? `${file.size} bytes` : "not created yet" }}</span>
            </div>
          </button>
          <p v-if="filteredFiles.length === 0" class="px-2 py-6 text-sm text-muted-foreground">
            No template files match the search.
          </p>
        </div>
      </ScrollArea>
    </div>

    <div class="flex min-h-0 flex-col">
      <div
        class="flex min-h-[var(--panel-header-h)] items-center gap-3 border-b border-section-border px-4"
      >
        <div class="min-w-0 flex-1">
          <h2 class="truncate text-sm font-semibold">
            {{ selectedFile?.relative ?? "Select a template file" }}
          </h2>
          <p class="truncate text-xs text-muted-foreground">
            {{ selectedFile?.path ?? "Allowlisted manifests and stow dotfiles only" }}
          </p>
        </div>
        <Button
          type="button"
          size="sm"
          :disabled="!selectedPath || !dirty || saving || contentLoading"
          @click="emit('save-file')"
        >
          <Save class="size-4" />
          {{ saving ? "Saving" : "Save" }}
        </Button>
      </div>

      <ScrollArea class="min-h-0 flex-1">
        <div class="grid gap-4 p-4">
          <section v-if="error" :class="detailSectionClass">
            <div class="flex items-center gap-2 text-sm font-semibold text-destructive">
              <AlertTriangle class="size-4" />
              Template file error
            </div>
            <p class="mt-2 text-sm text-muted-foreground">{{ error }}</p>
          </section>

          <section v-if="message" :class="detailSectionClass">
            <p class="text-sm text-success">{{ message }}</p>
          </section>

          <section :class="detailSectionClass">
            <div :class="detailSectionBodyClass">
              <template v-if="contentLoading">
                <Skeleton class="h-4 w-44" />
                <Skeleton class="h-[520px] w-full rounded-md" />
              </template>
              <template v-else-if="selectedPath">
                <Textarea
                  :model-value="draft"
                  class="min-h-[520px] resize-none font-mono text-xs leading-5"
                  spellcheck="false"
                  @update:model-value="emit('update:draft', String($event))"
                />
                <div class="flex items-center text-xs text-muted-foreground">
                  <span>{{ dirty ? "Unsaved changes" : "Saved content loaded" }}</span>
                  <Skeleton v-if="saving" class="ml-auto h-4 w-20" />
                </div>
              </template>
              <p v-else class="py-20 text-center text-sm text-muted-foreground">
                Choose a file from the list to inspect or edit it.
              </p>
            </div>
          </section>
        </div>
      </ScrollArea>
    </div>
  </div>
</template>
