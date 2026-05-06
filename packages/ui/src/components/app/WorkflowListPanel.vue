<script setup lang="ts">
import { computed } from "vue";
import { FileCode2, Search } from "lucide-vue-next";
import { Badge } from "@ui/badge";
import { Input } from "@ui/input";
import { ScrollArea } from "@ui/scroll-area";
import { Separator } from "@ui/separator";
import { Skeleton } from "@ui/skeleton";
import WorkflowCardList from "@components/WorkflowCardList.vue";
import {
  listItemClass,
  panelHeaderClass,
  searchBarClass,
  selectedListItemClass,
} from "@app/styles";
import type { StepMeta } from "@app/types";
import { cn } from "@lib/utils";
import type { Workflow } from "@api";

const props = defineProps<{
  stepMeta: StepMeta;
  searchQuery: string;
  workflows: Workflow[];
  selectedWorkflowId: string;
  selectedTemplateFiles: boolean;
  templateFilesCount: number;
  templateFilesLoaded: boolean;
  templateFilesLoading: boolean;
  workflowsLoading: boolean;
}>();

const emit = defineEmits<{
  (event: "update:searchQuery", value: string): void;
  (event: "select-workflow", workflow: Workflow): void;
  (event: "select-template-files"): void;
}>();

const templateFilesCountLabel = computed(() => {
  if (props.templateFilesLoading || !props.templateFilesLoaded) {
    return "Template files";
  }

  return `${props.templateFilesCount} ${props.templateFilesCount === 1 ? "file" : "files"}`;
});
</script>

<template>
  <div class="flex h-full min-h-0 flex-col">
    <div
      :class="
        cn(
          'flex min-h-[var(--panel-header-h)] flex-col justify-center gap-1 px-4',
          panelHeaderClass,
        )
      "
    >
      <h1 class="text-xl font-bold">{{ stepMeta.title }}</h1>
      <p class="line-clamp-2 text-sm leading-5 text-muted-foreground">
        {{ stepMeta.summary }}
      </p>
    </div>
    <Separator />
    <div :class="searchBarClass">
      <form @submit.prevent>
        <div class="relative">
          <Search class="absolute left-2 top-2.5 size-4 text-muted-foreground" />
          <Input
            :model-value="searchQuery"
            data-testid="app-search"
            placeholder="Search workflows"
            class="pl-8"
            @update:model-value="emit('update:searchQuery', String($event))"
          />
        </div>
      </form>
    </div>
    <ScrollArea class="min-h-0 flex-1">
      <div
        v-if="workflowsLoading"
        data-testid="workflows-list-skeleton"
        class="flex flex-col gap-2 p-4"
      >
        <div
          v-for="index in 6"
          :key="index"
          class="rounded-lg border border-section-border bg-section p-3 shadow-sm"
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
      <WorkflowCardList
        v-else
        :workflows="workflows"
        :selected-id="selectedWorkflowId"
        :empty-message="stepMeta.emptyMessage"
        @select="emit('select-workflow', $event)"
      >
        <template v-if="stepMeta.id === 'template'">
          <button
            :class="
              cn(
                'flex flex-col items-start gap-2 rounded-lg border p-3 text-left text-sm transition-all hover:bg-accent',
                listItemClass,
                selectedTemplateFiles && selectedListItemClass,
              )
            "
            @click="emit('select-template-files')"
          >
            <div class="flex w-full flex-col gap-1">
              <div class="flex min-w-0 items-center gap-2">
                <div class="truncate font-semibold">Template Files</div>
                <span v-if="selectedTemplateFiles" class="flex size-2 rounded-full bg-primary" />
                <Badge
                  variant="outline"
                  class="ml-auto border-[var(--status-neutral-border)] bg-[var(--status-neutral-bg)] text-[var(--status-neutral-fg)]"
                >
                  <FileCode2 />
                  Source
                </Badge>
              </div>
              <div class="text-xs font-medium text-muted-foreground">
                {{ templateFilesCountLabel }}
              </div>
            </div>
            <div class="line-clamp-2 text-xs leading-5 text-muted-foreground">
              Open the template source files for apps, secrets, macOS defaults, and dotfiles.
            </div>
          </button>
        </template>
      </WorkflowCardList>
    </ScrollArea>
  </div>
</template>
