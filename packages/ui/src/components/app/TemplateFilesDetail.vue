<script setup lang="ts">
import { computed, ref } from "vue";
import {
  AlertTriangle,
  AppWindow,
  ArrowLeft,
  FileCode2,
  FileText,
  Github,
  LockKeyhole,
  RefreshCw,
  RotateCcw,
  Save,
  Search,
  Shell,
  Sparkles,
  SquarePen,
  TerminalSquare,
} from "lucide-vue-next";
import {
  AlertDialog,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Skeleton } from "@/components/ui/skeleton";
import MonacoFileEditor from "@/components/app/MonacoFileEditor.vue";
import { cn } from "@/lib/utils";
import type { TemplateFileSummary } from "@/types/api";

const props = defineProps<{
  theme: "light" | "dark";
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
  (event: "cancel-edit"): void;
  (event: "back"): void;
}>();

const search = ref("");
const pendingDirtyAction = ref<"back" | "cancel" | null>(null);

const filteredFiles = computed(() => {
  const query = search.value.trim().toLowerCase();

  if (!query) return props.files;

  return props.files.filter((file) =>
    [file.relative, file.kind, file.path].join(" ").toLowerCase().includes(query),
  );
});

const selectedFile = computed(() => props.files.find((file) => file.path === props.selectedPath));

function templateFileIcon(file: TemplateFileSummary) {
  const relative = file.relative.toLowerCase();

  if (relative.includes("secrets")) return LockKeyhole;
  if (relative.includes("apps.generated")) return Sparkles;
  if (relative === "apps.yaml" || file.kind === "apps") return AppWindow;
  if (relative.includes("ghostty")) return TerminalSquare;
  if (relative.includes("/git/") || relative.includes(".git")) return Github;
  if (relative.includes("/shell/") || relative.includes(".zsh") || relative.includes(".bash")) {
    return Shell;
  }
  if (relative.includes("/vim/") || relative.includes(".vimrc")) return SquarePen;
  if (file.kind === "stow") return FileCode2;

  return FileText;
}

function templateFileIconName(file: TemplateFileSummary) {
  const relative = file.relative.toLowerCase();

  if (relative.includes("secrets")) return "secrets";
  if (relative.includes("apps.generated")) return "generated-apps";
  if (relative === "apps.yaml" || file.kind === "apps") return "apps";
  if (relative.includes("ghostty")) return "terminal-config";
  if (relative.includes("/git/") || relative.includes(".git")) return "git-config";
  if (relative.includes("/shell/") || relative.includes(".zsh") || relative.includes(".bash")) {
    return "shell-config";
  }
  if (relative.includes("/vim/") || relative.includes(".vimrc")) return "vim-config";
  if (file.kind === "stow") return "stow-file";

  return "template-file";
}

function requestBack() {
  if (props.dirty) {
    pendingDirtyAction.value = "back";
    return;
  }

  emit("back");
}

function requestCancel() {
  if (props.dirty) {
    pendingDirtyAction.value = "cancel";
    return;
  }

  emit("cancel-edit");
}

function continueDirtyAction() {
  const action = pendingDirtyAction.value;
  pendingDirtyAction.value = null;

  if (action === "back") {
    emit("back");
    return;
  }

  if (action === "cancel") {
    emit("cancel-edit");
  }
}
</script>

<template>
  <div
    data-testid="expanded-template-editor"
    class="grid h-full min-h-0 flex-1 grid-rows-[auto_minmax(0,1fr)]"
  >
    <div
      class="flex min-h-[88px] items-center gap-3 border-b border-section-border bg-section px-4"
    >
      <Button type="button" variant="ghost" size="sm" @click="requestBack">
        <ArrowLeft class="size-4" />
        Back
      </Button>
      <div class="min-w-0 flex-1">
        <h1 class="truncate text-lg font-semibold">Update Template Files</h1>
        <p class="mt-1 max-w-4xl text-sm leading-5 text-muted-foreground">
          Edit allowlisted template manifests and stow dotfiles. Save writes the selected file; Back
          returns to the workflow you were reviewing.
        </p>
      </div>
      <Button
        type="button"
        variant="outline"
        size="sm"
        :disabled="!selectedPath || contentLoading || saving"
        @click="requestCancel"
      >
        <RotateCcw class="size-4" />
        Cancel
      </Button>
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

    <div class="grid h-full min-h-0 grid-cols-[minmax(280px,32%)_1fr]">
      <div class="flex h-full min-h-0 flex-col border-r border-section-border bg-section">
        <div class="space-y-3 border-b border-section-border p-4">
          <div class="flex items-center gap-2">
            <FileText class="size-4 text-muted-foreground" />
            <h2 class="text-sm font-semibold">Files</h2>
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
            <div
              v-for="index in 8"
              :key="index"
              class="rounded-lg border border-section-border p-3"
            >
              <Skeleton class="h-4 w-40" />
              <Skeleton class="mt-2 h-3 w-24" />
            </div>
          </div>
          <div v-else class="grid gap-1 p-4">
            <button
              v-for="file in filteredFiles"
              :key="file.path"
              class="flex items-start gap-3 rounded-lg border border-section-border bg-section-muted p-3 text-left text-sm transition-colors hover:bg-accent"
              :class="
                selectedPath === file.path &&
                'border-primary bg-primary/10 text-primary shadow-sm ring-2 ring-primary/30'
              "
              @click="emit('select-file', file)"
            >
              <span
                class="mt-0.5 grid size-8 shrink-0 place-items-center rounded-md border border-section-border bg-background text-muted-foreground"
                :class="selectedPath === file.path && 'border-primary/40 text-primary'"
              >
                <component
                  :is="templateFileIcon(file)"
                  class="size-4"
                  aria-hidden="true"
                  :data-testid="`template-file-icon-${templateFileIconName(file)}`"
                />
              </span>
              <div class="min-w-0 flex-1">
                <div class="truncate font-medium">{{ file.relative }}</div>
                <div class="mt-1 flex items-center gap-2 text-xs text-muted-foreground">
                  <span>{{ file.kind }}</span>
                  <span>{{ file.exists ? `${file.size} bytes` : "not created yet" }}</span>
                </div>
              </div>
            </button>
            <p v-if="filteredFiles.length === 0" class="px-2 py-6 text-sm text-muted-foreground">
              No template files match the search.
            </p>
          </div>
        </ScrollArea>
      </div>

      <div class="grid h-full min-h-0 grid-rows-[auto_auto_minmax(0,1fr)]">
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
        </div>

        <section v-if="error" class="border-b border-section-border px-4 py-3">
          <div class="flex items-center gap-2 text-sm font-semibold text-destructive">
            <AlertTriangle class="size-4" />
            Template file error
          </div>
          <p class="mt-1 text-sm text-muted-foreground">{{ error }}</p>
        </section>

        <div class="h-[calc(100vh-11rem)] min-h-0 p-4">
          <template v-if="contentLoading">
            <div class="grid h-full min-h-0 grid-rows-[auto_minmax(0,1fr)] gap-3">
              <Skeleton class="h-4 w-44" />
              <Skeleton class="h-full min-h-0 w-full rounded-md" />
            </div>
          </template>
          <template v-else-if="selectedPath">
            <div class="h-full min-h-0">
              <MonacoFileEditor
                :model-value="draft"
                :path="selectedPath"
                :loading="contentLoading"
                :theme="theme"
                @update:model-value="emit('update:draft', $event)"
              />
            </div>
          </template>
          <div
            v-else
            class="grid h-full place-items-center text-center text-sm text-muted-foreground"
          >
            <p>Choose a file from the list to inspect or edit it.</p>
          </div>
        </div>
      </div>
    </div>
  </div>

  <AlertDialog
    :open="pendingDirtyAction !== null"
    @update:open="!$event && (pendingDirtyAction = null)"
  >
    <AlertDialogContent>
      <AlertDialogHeader>
        <AlertDialogTitle>Discard unsaved changes?</AlertDialogTitle>
        <AlertDialogDescription>
          This file has unsaved edits. Continuing will discard the current draft.
        </AlertDialogDescription>
      </AlertDialogHeader>
      <AlertDialogFooter>
        <AlertDialogCancel>Stay</AlertDialogCancel>
        <Button variant="destructive" @click="continueDirtyAction">Discard changes</Button>
      </AlertDialogFooter>
    </AlertDialogContent>
  </AlertDialog>
</template>
