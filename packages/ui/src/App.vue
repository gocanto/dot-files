<script setup lang="ts">
import {
  Activity,
  AlertTriangle,
  Apple,
  CheckCircle2,
  Circle,
  Database,
  FileText,
  FolderOpen,
  HardDrive,
  History,
  Inbox,
  KeyRound,
  Loader2,
  MoreVertical,
  Play,
  RefreshCw,
  RotateCcw,
  Save,
  Search,
  Send,
  Settings,
  ShieldCheck,
  TerminalSquare,
} from "lucide-vue-next";
import { computed, onMounted, ref } from "vue";
import {
  AlertDialog,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { Badge, type BadgeVariants } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import OutputBlock from "@/components/OutputBlock.vue";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { ResizableHandle, ResizablePanel, ResizablePanelGroup } from "@/components/ui/resizable";
import { ScrollArea } from "@/components/ui/scroll-area";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Separator } from "@/components/ui/separator";
import { Skeleton } from "@/components/ui/skeleton";
import { Switch } from "@/components/ui/switch";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Textarea } from "@/components/ui/textarea";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";
import type { ConfirmationOption, Phase, RunEvent, RunLog, RunSummary, RuntimeSettings, SettingsResponse, Workflow } from "./types/api";

type SectionId = "workflows" | "logs" | "this-mac" | "snapshots" | "health" | "settings";

const section = ref<SectionId>("workflows");
const profile = ref("current-mac");
const workflows = ref<Workflow[]>([]);
const runs = ref<RunSummary[]>([]);
const selectedWorkflowId = ref("");
const selectedRunId = ref("");
const selectedRunLog = ref<RunLog | null>(null);
const enabledPhaseIds = ref<Set<string>>(new Set());
const pendingOption = ref<ConfirmationOption | null>(null);
const runEvents = ref<RunEvent[]>([]);
const running = ref(false);
const initialLoading = ref(true);
const loadError = ref("");
const navCollapsed = ref(false);
const searchQuery = ref("");
const workflowTab = ref("all");
const logTab = ref("all");
const mutedNotes = ref(false);
const noteText = ref("");
const settingsResponse = ref<SettingsResponse | null>(null);
const settingsForm = ref<RuntimeSettings>(emptySettings());
const settingsSaving = ref(false);
const settingsMessage = ref("");
const settingsError = ref("");

const primaryNavItems = computed(() => [
  { id: "workflows" as const, label: "Workflows", icon: Inbox, count: workflows.value.length },
  { id: "logs" as const, label: "Logs", icon: History, count: runs.value.length },
]);

const secondaryNavItems = computed(() => [
  { id: "this-mac" as const, label: "This Mac", icon: Apple },
  { id: "snapshots" as const, label: "Snapshots", icon: HardDrive },
  { id: "health" as const, label: "Health Checks", icon: Activity },
  { id: "settings" as const, label: "Settings", icon: Settings },
]);

const selectedWorkflow = computed(() => workflows.value.find((workflow) => workflow.id === selectedWorkflowId.value));
const selectedRun = computed(() => runs.value.find((run) => run.id === selectedRunId.value));
const settingsDirty = computed(() => JSON.stringify(settingsForm.value) !== JSON.stringify(settingsResponse.value?.settings ?? emptySettings()));
const settingsChecks = computed(() => settingsResponse.value?.checks ?? []);

const settingsGroups = computed(() => [
  {
    id: "repository",
    label: "Repository",
    icon: FolderOpen,
    count: settingsChecks.value.filter((check) => ["repo_root", "stow"].includes(check.key)).filter((check) => check.status !== "ok").length,
  },
  {
    id: "manifests",
    label: "Manifests",
    icon: FileText,
    count: settingsChecks.value.filter((check) => ["apps_config_path", "secrets_config_path", "generated_apps_path", "private_gitconfig_path"].includes(check.key)).filter((check) => check.status !== "ok").length,
  },
  {
    id: "storage",
    label: "Storage",
    icon: Database,
    count: settingsChecks.value.filter((check) => check.key === "workflow_db_path").filter((check) => check.status !== "ok").length,
  },
  {
    id: "operations",
    label: "Operations",
    icon: KeyRound,
    count: settingsChecks.value.filter((check) => check.key === "archive_root").filter((check) => check.status !== "ok").length,
  },
]);

const displayPhases = computed(() => {
  if (!selectedWorkflow.value) {
    return [];
  }

  return selectedWorkflow.value.phases.map((phase) => ({
    ...phase,
    enabled: enabledPhaseIds.value.has(phase.id),
  }));
});

const normalizedSearch = computed(() => searchQuery.value.trim().toLowerCase());

const matchingWorkflows = computed(() => {
  const query = normalizedSearch.value;
  const filtered = query
    ? workflows.value.filter((workflow) =>
        [workflow.name, workflow.description, workflow.changesMac]
          .join(" ")
          .toLowerCase()
          .includes(query),
      )
    : workflows.value;

  if (workflowTab.value === "safe") {
    return filtered.filter((workflow) => workflow.changesMac === "No");
  }

  if (workflowTab.value === "changes") {
    return filtered.filter((workflow) => workflow.changesMac !== "No");
  }

  return filtered;
});

const matchingRuns = computed(() => {
  const query = normalizedSearch.value;
  const filtered = query
    ? runs.value.filter((run) =>
        [
          run.workflowName,
          run.status,
          run.mode,
          run.confirmationOptionLabel,
          run.errorMessage ?? "",
        ]
          .join(" ")
          .toLowerCase()
          .includes(query),
      )
    : runs.value;

  if (logTab.value === "failed") {
    return filtered.filter((run) => run.status === "failed");
  }

  if (logTab.value === "active") {
    return filtered.filter((run) => ["running", "pending"].includes(run.status));
  }

  return filtered;
});

const runStatus = computed(() => {
  const last = [...runEvents.value].reverse().find((event) => event.type.startsWith("run_"));

  return last?.status ?? (running.value ? "running" : "idle");
});

const outputText = computed(() =>
  runEvents.value
    .map((event) => event.message || `${event.type} ${event.status || ""}`.trim())
    .filter(Boolean)
    .join("\n"),
);

const selectedRunOutput = computed(() =>
  selectedRunLog.value?.events
    .map((event) => event.message || `${event.type} ${event.status || ""}`.trim())
    .filter(Boolean)
    .join("\n") ?? "",
);

onMounted(async () => {
  await loadAll();
});

async function loadAll() {
  try {
    loadError.value = "";
    workflows.value = await window.macOS.workflows();
    runs.value = await window.macOS.runs(25);
    await loadSettings();

    if (!selectedWorkflowId.value || !workflows.value.some((workflow) => workflow.id === selectedWorkflowId.value)) {
      selectedWorkflowId.value = workflows.value[0]?.id ?? "";
    }

    resetEnabledPhases();
  } catch (error) {
    loadError.value = error instanceof Error ? error.message : String(error);
  } finally {
    initialLoading.value = false;
  }
}

function selectSection(next: SectionId) {
  section.value = next;
  searchQuery.value = "";

  if (next === "logs") {
    void refreshRuns();
  }

  if (next === "settings") {
    void loadSettings();
  }
}

function selectWorkflow(workflow: Workflow) {
  selectedWorkflowId.value = workflow.id;
  resetEnabledPhases();
  runEvents.value = [];
}

function resetEnabledPhases() {
  enabledPhaseIds.value = new Set(selectedWorkflow.value?.phases.filter((phase) => phase.enabled).map((phase) => phase.id));
}

function togglePhase(phase: Phase) {
  const next = new Set(enabledPhaseIds.value);

  if (next.has(phase.id)) {
    next.delete(phase.id);
  } else {
    next.add(phase.id);
  }

  enabledPhaseIds.value = next;
}

function openConfirmation(option?: ConfirmationOption) {
  if (!selectedWorkflow.value?.confirmation) {
    return;
  }

  pendingOption.value = option ?? selectedWorkflow.value.confirmation.options[0] ?? null;
}

function updateConfirmationOpen(open: boolean) {
  if (!open) {
    pendingOption.value = null;
  }
}

async function runSelected(option: ConfirmationOption) {
  if (!selectedWorkflow.value || option.back) {
    pendingOption.value = null;

    return;
  }

  pendingOption.value = null;
  running.value = true;
  runEvents.value = [];

  const phases = option.phases && option.phases.length > 0 ? option.phases : displayPhases.value;
  const enabledIds = phases.filter((phase) => enabledPhaseIds.value.has(phase.id)).map((phase) => phase.id);

  try {
    await window.macOS.runWorkflow(
      {
        workflowId: selectedWorkflow.value.id,
        confirmationOptionId: option.id,
        enabledPhaseIds: enabledIds,
      },
      (event) => runEvents.value.push(event),
    );
  } finally {
    running.value = false;
    await refreshRuns();
  }
}

async function refreshRuns() {
  runs.value = await window.macOS.runs(25);
}

async function openRun(run: RunSummary) {
  selectedRunId.value = run.id;
  selectedRunLog.value = await window.macOS.runLog(run.id);
}

function emptySettings(): RuntimeSettings {
  return {
    repoRoot: "",
    appsConfigPath: "",
    secretsConfigPath: "",
    generatedAppsPath: "",
    archiveRoot: "",
    workflowDbPath: "",
    opVault: "",
    opItem: "",
  };
}

async function loadSettings() {
  const response = await window.macOS.settings();

  settingsResponse.value = response;
  settingsForm.value = { ...response.settings };
  settingsError.value = "";
}

async function validateSettings() {
  settingsMessage.value = "";
  settingsError.value = "";
  settingsResponse.value = await window.macOS.validateSettings({ ...settingsForm.value });
}

async function saveSettings() {
  settingsSaving.value = true;
  settingsMessage.value = "";
  settingsError.value = "";

  try {
    const response = await window.macOS.saveSettings({ ...settingsForm.value });
    settingsResponse.value = response;

    if (!response.valid) {
      settingsError.value = "Fix the highlighted settings before saving.";
      return;
    }

    settingsForm.value = { ...response.settings };
    settingsMessage.value = "Settings saved. The workflow bridge was restarted.";
    workflows.value = await window.macOS.workflows();
    runs.value = await window.macOS.runs(25);
  } catch (error) {
    settingsError.value = error instanceof Error ? error.message : String(error);
  } finally {
    settingsSaving.value = false;
  }
}

function resetSettingsForm() {
  settingsForm.value = { ...(settingsResponse.value?.settings ?? emptySettings()) };
  settingsError.value = "";
  settingsMessage.value = "";
}

async function chooseDirectory(field: keyof RuntimeSettings) {
  const path = await window.macOS.chooseDirectory(settingsForm.value[field]);

  if (path) {
    settingsForm.value = { ...settingsForm.value, [field]: path };
  }
}

async function chooseFile(field: keyof RuntimeSettings) {
  const path = await window.macOS.chooseFile(settingsForm.value[field]);

  if (path) {
    settingsForm.value = { ...settingsForm.value, [field]: path };
  }
}

async function chooseSaveFile(field: keyof RuntimeSettings) {
  const path = await window.macOS.chooseSaveFile(settingsForm.value[field]);

  if (path) {
    settingsForm.value = { ...settingsForm.value, [field]: path };
  }
}

function phaseStatus(phase: Phase) {
  const events = runEvents.value.filter((event) => event.phaseId === phase.id);
  const finish = [...events].reverse().find((event) => event.type === "phase_finished" || event.type === "phase_skipped");

  return finish?.status ?? events.at(-1)?.status ?? (enabledPhaseIds.value.has(phase.id) ? "pending" : "skipped");
}

function badgeVariant(status: string): BadgeVariants["variant"] {
  if (status === "failed" || status === "Yes") {
    return "destructive";
  }

  if (["running", "completed", "ok", "No"].includes(status)) {
    return "default";
  }

  if (["stopped", "skipped", "pending"].includes(status)) {
    return "secondary";
  }

  return "outline";
}

function formatDate(value?: string) {
  if (!value) {
    return "Not recorded";
  }

  const date = new Date(value);

  if (Number.isNaN(date.getTime())) {
    return value;
  }

  return new Intl.DateTimeFormat(undefined, {
    dateStyle: "medium",
    timeStyle: "short",
  }).format(date);
}

function timeAgo(value?: string) {
  if (!value) {
    return "";
  }

  const date = new Date(value);

  if (Number.isNaN(date.getTime())) {
    return value;
  }

  const seconds = Math.max(1, Math.round((Date.now() - date.getTime()) / 1000));
  const units: Array<[Intl.RelativeTimeFormatUnit, number]> = [
    ["year", 60 * 60 * 24 * 365],
    ["month", 60 * 60 * 24 * 30],
    ["week", 60 * 60 * 24 * 7],
    ["day", 60 * 60 * 24],
    ["hour", 60 * 60],
    ["minute", 60],
  ];
  const formatter = new Intl.RelativeTimeFormat(undefined, { numeric: "auto" });
  const match = units.find(([, unitSeconds]) => seconds >= unitSeconds);

  if (!match) {
    return "just now";
  }

  const [unit, unitSeconds] = match;

  return formatter.format(-Math.floor(seconds / unitSeconds), unit);
}

function initials(value: string) {
  return value
    .split(/\s+/)
    .filter(Boolean)
    .slice(0, 3)
    .map((chunk) => chunk[0]?.toUpperCase())
    .join("");
}
</script>

<template>
  <TooltipProvider :delay-duration="0">
    <div class="h-screen overflow-hidden bg-background text-foreground">
      <ResizablePanelGroup direction="horizontal" class="h-screen max-h-screen items-stretch">
        <ResizablePanel
          id="mac-nav"
          :default-size="18"
          :collapsed-size="4"
          collapsible
          :min-size="14"
          :max-size="22"
          :class="cn(navCollapsed && 'min-w-[52px] transition-all duration-300 ease-in-out')"
          @collapse="navCollapsed = true"
          @expand="navCollapsed = false"
        >
          <div :class="cn('flex h-[52px] items-center justify-center', navCollapsed ? 'px-2' : 'px-3')">
            <Select v-model="profile">
              <SelectTrigger
                aria-label="Select Mac profile"
                :class="cn(
                  'flex items-center gap-2 [&>span]:line-clamp-1 [&>span]:flex [&>span]:w-full [&>span]:items-center [&>span]:gap-2 [&>span]:truncate [&_svg]:size-4 [&_svg]:shrink-0',
                  navCollapsed && 'flex size-9 shrink-0 items-center justify-center p-0 [&>span]:w-auto [&>svg]:hidden',
                )"
              >
                <SelectValue placeholder="Select Mac profile">
                  <div class="flex items-center gap-2">
                    <Apple class="size-4" />
                    <span v-if="!navCollapsed">Current Mac</span>
                  </div>
                </SelectValue>
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="current-mac">Current Mac</SelectItem>
                <SelectItem value="local-profile">Local Profile</SelectItem>
              </SelectContent>
            </Select>
          </div>

          <Separator />

          <div :data-collapsed="navCollapsed" class="group flex flex-col gap-4 py-2 data-[collapsed=true]:py-2">
            <nav class="grid gap-1 px-2 group-[[data-collapsed=true]]:justify-center group-[[data-collapsed=true]]:px-2">
              <template v-for="item in primaryNavItems" :key="item.id">
                <Tooltip v-if="navCollapsed">
                  <TooltipTrigger as-child>
                    <Button
                      variant="ghost"
                      size="icon"
                      :class="cn(section === item.id && 'bg-primary text-primary-foreground hover:bg-primary/90 hover:text-primary-foreground')"
                      @click="selectSection(item.id)"
                    >
                      <component :is="item.icon" class="size-4" />
                      <span class="sr-only">{{ item.label }}</span>
                    </Button>
                  </TooltipTrigger>
                  <TooltipContent side="right" class="flex items-center gap-4">
                    {{ item.label }}
                    <span class="ml-auto text-muted-foreground">{{ item.count }}</span>
                  </TooltipContent>
                </Tooltip>

                <Button
                  v-else
                  variant="ghost"
                  size="sm"
                  :class="cn('justify-start', section === item.id && 'bg-primary text-primary-foreground hover:bg-primary/90 hover:text-primary-foreground')"
                  @click="selectSection(item.id)"
                >
                  <component :is="item.icon" class="mr-2 size-4" />
                  {{ item.label }}
                  <span class="ml-auto">{{ item.count }}</span>
                </Button>
              </template>
            </nav>
          </div>

          <Separator />

          <div :data-collapsed="navCollapsed" class="group flex flex-col gap-4 py-2 data-[collapsed=true]:py-2">
            <nav class="grid gap-1 px-2 group-[[data-collapsed=true]]:justify-center group-[[data-collapsed=true]]:px-2">
              <template v-for="item in secondaryNavItems" :key="item.id">
                <Tooltip v-if="navCollapsed">
                  <TooltipTrigger as-child>
                    <Button
                      variant="ghost"
                      size="icon"
                      :class="cn(section === item.id && 'bg-primary text-primary-foreground hover:bg-primary/90 hover:text-primary-foreground')"
                      @click="selectSection(item.id)"
                    >
                      <component :is="item.icon" class="size-4" />
                      <span class="sr-only">{{ item.label }}</span>
                    </Button>
                  </TooltipTrigger>
                  <TooltipContent side="right">{{ item.label }}</TooltipContent>
                </Tooltip>

                <Button
                  v-else
                  variant="ghost"
                  size="sm"
                  :class="cn('justify-start', section === item.id && 'bg-primary text-primary-foreground hover:bg-primary/90 hover:text-primary-foreground')"
                  @click="selectSection(item.id)"
                >
                  <component :is="item.icon" class="mr-2 size-4" />
                  {{ item.label }}
                </Button>
              </template>
            </nav>
          </div>
        </ResizablePanel>

        <ResizableHandle with-handle />

        <ResizablePanel id="mac-list" :default-size="32" :min-size="28">
          <div class="flex h-full min-h-0 flex-col">
            <template v-if="initialLoading">
              <div class="flex items-center px-4 py-2">
                <Skeleton class="h-7 w-32" />
                <div class="ml-auto flex gap-2">
                  <Skeleton class="h-8 w-16" />
                  <Skeleton class="h-8 w-16" />
                </div>
              </div>
              <Separator />
              <div class="p-4">
                <Skeleton class="h-9 w-full" />
              </div>
              <div class="flex flex-col gap-2 p-4 pt-0">
                <div v-for="index in 6" :key="index" class="rounded-lg border p-3">
                  <div class="flex items-center gap-3">
                    <Skeleton class="h-4 w-40" />
                    <Skeleton class="ml-auto h-5 w-12 rounded-full" />
                  </div>
                  <Skeleton class="mt-3 h-3 w-24" />
                  <Skeleton class="mt-3 h-3 w-full" />
                  <Skeleton class="mt-2 h-3 w-4/5" />
                </div>
              </div>
            </template>

            <template v-else-if="section === 'workflows'">
              <Tabs v-model="workflowTab" class="flex h-full min-h-0 flex-col">
                <div class="flex items-center px-4 py-2">
                  <h1 class="text-xl font-bold">Workflows</h1>
                  <TabsList class="ml-auto">
                    <TabsTrigger value="all">All</TabsTrigger>
                    <TabsTrigger value="safe">Safe</TabsTrigger>
                    <TabsTrigger value="changes">Changes</TabsTrigger>
                  </TabsList>
                </div>
                <Separator />
                <div class="bg-background/95 p-4 backdrop-blur supports-[backdrop-filter]:bg-background/60">
                  <form @submit.prevent>
                    <div class="relative">
                      <Search class="absolute left-2 top-2.5 size-4 text-muted-foreground" />
                      <Input v-model="searchQuery" data-testid="app-search" placeholder="Search workflows" class="pl-8" />
                    </div>
                  </form>
                </div>
                <ScrollArea class="min-h-0 flex-1">
                  <div class="flex flex-col gap-2 p-4 pt-0">
                    <button
                      v-for="workflow in matchingWorkflows"
                      :key="workflow.id"
                      :class="cn(
                        'flex flex-col items-start gap-2 rounded-lg border p-3 text-left text-sm transition-all hover:bg-accent',
                        selectedWorkflowId === workflow.id && 'bg-muted',
                      )"
                      @click="selectWorkflow(workflow)"
                    >
                      <div class="flex w-full flex-col gap-1">
                        <div class="flex min-w-0 items-center gap-2">
                          <div class="truncate font-semibold">{{ workflow.name }}</div>
                          <span v-if="workflow.id === selectedWorkflowId" class="flex size-2 rounded-full bg-primary" />
                          <Badge class="ml-auto" :variant="badgeVariant(workflow.changesMac)">
                            {{ workflow.changesMac }}
                          </Badge>
                        </div>
                        <div class="text-xs font-medium text-muted-foreground">
                          {{ workflow.phases.length }} phases
                        </div>
                      </div>
                      <div class="line-clamp-2 text-xs leading-5 text-muted-foreground">
                        {{ workflow.description }}
                      </div>
                    </button>

                    <div v-if="matchingWorkflows.length === 0" class="rounded-lg border border-dashed p-8 text-center text-sm text-muted-foreground">
                      No workflows match this view.
                    </div>
                  </div>
                </ScrollArea>
              </Tabs>
            </template>

            <template v-else-if="section === 'logs'">
              <Tabs v-model="logTab" class="flex h-full min-h-0 flex-col">
                <div class="flex items-center px-4 py-2">
                  <h1 class="text-xl font-bold">Logs</h1>
                  <TabsList class="ml-auto">
                    <TabsTrigger value="all">All</TabsTrigger>
                    <TabsTrigger value="failed">Failed</TabsTrigger>
                    <TabsTrigger value="active">Active</TabsTrigger>
                  </TabsList>
                </div>
                <Separator />
                <div class="bg-background/95 p-4 backdrop-blur supports-[backdrop-filter]:bg-background/60">
                  <form @submit.prevent>
                    <div class="relative">
                      <Search class="absolute left-2 top-2.5 size-4 text-muted-foreground" />
                      <Input v-model="searchQuery" data-testid="app-search" placeholder="Search logs" class="pl-8" />
                    </div>
                  </form>
                </div>
                <ScrollArea class="min-h-0 flex-1">
                  <div class="flex flex-col gap-2 p-4 pt-0">
                    <button
                      v-for="run in matchingRuns"
                      :key="run.id"
                      :class="cn(
                        'flex flex-col items-start gap-2 rounded-lg border p-3 text-left text-sm transition-all hover:bg-accent',
                        selectedRunId === run.id && 'bg-muted',
                      )"
                      @click="openRun(run)"
                    >
                      <div class="flex w-full flex-col gap-1">
                        <div class="flex min-w-0 items-center gap-2">
                          <div class="truncate font-semibold">{{ run.workflowName }}</div>
                          <Badge class="ml-auto" :variant="badgeVariant(run.status)">
                            {{ run.status }}
                          </Badge>
                        </div>
                        <div class="flex items-center justify-between gap-3 text-xs text-muted-foreground">
                          <span class="truncate">{{ run.mode }} - {{ run.confirmationOptionLabel }}</span>
                          <span class="shrink-0">{{ timeAgo(run.startedAt) }}</span>
                        </div>
                      </div>
                    </button>

                    <div v-if="matchingRuns.length === 0" class="rounded-lg border border-dashed p-8 text-center text-sm text-muted-foreground">
                      No logs match this view.
                    </div>
                  </div>
                </ScrollArea>
              </Tabs>
            </template>

            <template v-else-if="section === 'settings'">
              <div class="flex items-center px-4 py-2">
                <h1 class="text-xl font-bold">Settings</h1>
                <Badge class="ml-auto" :variant="settingsResponse?.valid ? 'default' : 'destructive'">
                  {{ settingsResponse?.valid ? "valid" : "needs review" }}
                </Badge>
              </div>
              <Separator />
              <ScrollArea class="min-h-0 flex-1">
                <div class="flex flex-col gap-2 p-4">
                  <div
                    v-for="group in settingsGroups"
                    :key="group.id"
                    class="flex items-center gap-3 rounded-lg border p-3 text-sm"
                  >
                    <component :is="group.icon" class="size-4 text-muted-foreground" />
                    <div class="min-w-0 flex-1">
                      <div class="font-medium">{{ group.label }}</div>
                      <div class="truncate text-xs text-muted-foreground">
                        {{ group.count === 0 ? "No validation errors" : `${group.count} issue${group.count === 1 ? "" : "s"}` }}
                      </div>
                    </div>
                    <Badge :variant="group.count === 0 ? 'secondary' : 'destructive'">{{ group.count }}</Badge>
                  </div>
                </div>
              </ScrollArea>
            </template>

            <template v-else>
              <div class="flex items-center px-4 py-2">
                <h1 class="text-xl font-bold capitalize">{{ section.replace("-", " ") }}</h1>
              </div>
              <Separator />
              <div class="bg-background/95 p-4 backdrop-blur supports-[backdrop-filter]:bg-background/60">
                <div class="relative">
                  <Search class="absolute left-2 top-2.5 size-4 text-muted-foreground" />
                  <Input v-model="searchQuery" placeholder="Search" class="pl-8" />
                </div>
              </div>
              <div class="grid min-h-0 flex-1 place-items-center p-8 text-center text-sm text-muted-foreground">
                Select Workflows or Logs to manage live app data.
              </div>
            </template>
          </div>
        </ResizablePanel>

        <ResizableHandle with-handle />

        <ResizablePanel id="mac-detail" :default-size="50" :min-size="35">
          <div class="flex h-full min-h-0 flex-col">
            <div class="flex items-center p-2">
              <div class="flex items-center gap-2">
                <Tooltip>
                  <TooltipTrigger as-child>
                    <Button variant="ghost" size="icon" :disabled="section === 'logs'" @click="loadAll">
                      <RefreshCw class="size-4" />
                      <span class="sr-only">Refresh</span>
                    </Button>
                  </TooltipTrigger>
                  <TooltipContent>Refresh</TooltipContent>
                </Tooltip>

                <Tooltip>
                  <TooltipTrigger as-child>
                    <Button variant="ghost" size="icon" :disabled="section !== 'workflows' || !selectedWorkflow" @click="resetEnabledPhases">
                      <RotateCcw class="size-4" />
                      <span class="sr-only">Reset phases</span>
                    </Button>
                  </TooltipTrigger>
                  <TooltipContent>Reset phases</TooltipContent>
                </Tooltip>

                <Tooltip>
                  <TooltipTrigger as-child>
                    <Button variant="ghost" size="icon" :disabled="section !== 'workflows' || !selectedWorkflow?.confirmation || running" @click="openConfirmation()">
                      <Play class="size-4" />
                      <span class="sr-only">Run workflow</span>
                    </Button>
                  </TooltipTrigger>
                  <TooltipContent>Run workflow</TooltipContent>
                </Tooltip>
              </div>

              <div class="ml-auto flex items-center gap-2">
                <Tooltip>
                  <TooltipTrigger as-child>
                    <Button variant="ghost" size="icon" :disabled="section !== 'logs'" @click="refreshRuns">
                      <History class="size-4" />
                      <span class="sr-only">Refresh logs</span>
                    </Button>
                  </TooltipTrigger>
                  <TooltipContent>Refresh logs</TooltipContent>
                </Tooltip>
              </div>

              <Separator orientation="vertical" class="mx-2 h-6" />

              <DropdownMenu>
                <DropdownMenuTrigger as-child>
                  <Button variant="ghost" size="icon">
                    <MoreVertical class="size-4" />
                    <span class="sr-only">More</span>
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="end">
                  <DropdownMenuItem @click="loadAll">Refresh all data</DropdownMenuItem>
                  <DropdownMenuItem :disabled="section !== 'workflows' || !selectedWorkflow" @click="resetEnabledPhases">
                    Reset selected workflow
                  </DropdownMenuItem>
                  <DropdownMenuItem :disabled="section !== 'logs'" @click="refreshRuns">
                    Refresh logs
                  </DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>
            </div>

            <Separator />

            <div v-if="initialLoading" class="flex min-h-0 flex-1 flex-col">
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
                <section>
                  <div class="mb-2 flex items-center justify-between">
                    <Skeleton class="h-4 w-20" />
                    <Skeleton class="h-8 w-16" />
                  </div>
                  <div class="overflow-hidden rounded-lg border">
                    <div v-for="index in 4" :key="index" class="flex items-center gap-3 border-b px-3 py-3 last:border-b-0">
                      <Skeleton class="size-4 rounded-full" />
                      <Skeleton class="h-4 flex-1" />
                      <Skeleton class="h-5 w-16 rounded-full" />
                    </div>
                  </div>
                </section>
                <section>
                  <Skeleton class="mb-2 h-4 w-28" />
                  <Skeleton class="h-72 w-full rounded-lg" />
                </section>
              </div>
            </div>

            <div v-else-if="loadError" class="grid flex-1 place-items-center p-8">
              <div class="max-w-xl rounded-lg border border-destructive/40 p-5">
                <div class="flex items-center gap-2 font-semibold text-destructive">
                  <AlertTriangle class="size-5" />
                  Load failed
                </div>
                <p class="mt-2 text-sm text-muted-foreground">{{ loadError }}</p>
              </div>
            </div>

            <template v-else-if="section === 'workflows' && selectedWorkflow">
              <div class="flex items-start p-4">
                <div class="flex items-start gap-4 text-sm">
                  <Avatar size="sm">
                    <AvatarFallback>{{ initials(selectedWorkflow.name) }}</AvatarFallback>
                  </Avatar>
                  <div class="grid gap-1">
                    <div class="font-semibold">{{ selectedWorkflow.name }}</div>
                    <div class="line-clamp-1 text-xs">{{ selectedWorkflow.description }}</div>
                    <div class="line-clamp-1 text-xs">
                      <span class="font-medium">Changes Mac:</span> {{ selectedWorkflow.changesMac }}
                    </div>
                  </div>
                </div>
                <Badge class="ml-auto" :variant="badgeVariant(runStatus)">{{ runStatus }}</Badge>
              </div>

              <Separator />

              <ScrollArea class="min-h-0 flex-1">
                <div class="grid gap-5 p-4">
                  <section>
                    <div class="mb-2 flex items-center justify-between gap-3">
                      <h2 class="text-sm font-semibold">Phases</h2>
                      <Button variant="ghost" size="sm" @click="resetEnabledPhases">Reset</Button>
                    </div>
                    <div class="overflow-hidden rounded-lg border">
                      <button
                        v-for="phase in displayPhases"
                        :key="phase.id"
                        class="flex w-full items-center gap-3 border-b px-3 py-3 text-left text-sm transition-colors last:border-b-0 hover:bg-accent"
                        @click="togglePhase(phase)"
                      >
                        <CheckCircle2 v-if="enabledPhaseIds.has(phase.id)" class="size-4 text-primary" />
                        <Circle v-else class="size-4 text-muted-foreground" />
                        <span class="min-w-0 flex-1 truncate">{{ phase.name }}</span>
                        <Badge :variant="badgeVariant(phaseStatus(phase))">{{ phaseStatus(phase) }}</Badge>
                      </button>
                    </div>
                  </section>

                  <section v-if="selectedWorkflow.confirmation">
                    <h2 class="mb-2 text-sm font-semibold">{{ selectedWorkflow.confirmation.title }}</h2>
                    <p class="mb-3 text-sm leading-6 text-muted-foreground">{{ selectedWorkflow.confirmation.message }}</p>
                    <div class="grid gap-2">
                      <Button
                        v-for="option in selectedWorkflow.confirmation.options"
                        :key="option.id"
                        variant="outline"
                        class="h-auto justify-between gap-3 whitespace-normal px-3 py-2 text-left"
                        @click="openConfirmation(option)"
                      >
                        <span class="min-w-0">
                          <span class="block font-medium">{{ option.label }}</span>
                          <span class="block text-xs text-muted-foreground">{{ option.description }}</span>
                        </span>
                        <Play v-if="option.continue" class="size-4 text-muted-foreground" />
                      </Button>
                    </div>
                  </section>

                  <section>
                    <h2 class="mb-2 text-sm font-semibold">Output</h2>
                    <ScrollArea class="h-72 rounded-lg border bg-primary text-primary-foreground">
                      <OutputBlock :code="outputText" empty-text="No workflow output yet." class="text-xs leading-5" />
                    </ScrollArea>
                  </section>
                </div>
              </ScrollArea>

              <Separator />

              <div class="p-4">
                <div class="grid gap-4">
                  <Textarea v-model="noteText" class="p-4" :placeholder="`Add a note for ${selectedWorkflow.name}...`" />
                  <div class="flex items-center">
                    <Label html-for="mute-notes" class="flex items-center gap-2 text-xs font-normal">
                      <Switch id="mute-notes" v-model="mutedNotes" aria-label="Mute workflow notes" />
                      Mute workflow notes
                    </Label>
                    <Button type="button" size="sm" class="ml-auto" :disabled="!noteText.trim()">
                      <Send class="size-4" />
                      Send
                    </Button>
                  </div>
                </div>
              </div>
            </template>

            <template v-else-if="section === 'logs'">
              <div v-if="selectedRunLog" class="flex min-h-0 flex-1 flex-col">
                <div class="flex items-start p-4">
                  <div class="flex items-start gap-4 text-sm">
                    <Avatar size="sm">
                      <AvatarFallback>{{ initials(selectedRunLog.run.workflowName) }}</AvatarFallback>
                    </Avatar>
                    <div class="grid gap-1">
                      <div class="font-semibold">{{ selectedRunLog.run.workflowName }}</div>
                      <div class="line-clamp-1 text-xs">{{ selectedRunLog.run.mode }} - {{ selectedRunLog.run.confirmationOptionLabel }}</div>
                      <div class="line-clamp-1 text-xs">
                        <span class="font-medium">Started:</span> {{ formatDate(selectedRunLog.run.startedAt) }}
                      </div>
                    </div>
                  </div>
                  <Badge class="ml-auto" :variant="badgeVariant(selectedRunLog.run.status)">
                    {{ selectedRunLog.run.status }}
                  </Badge>
                </div>
                <Separator />
                <ScrollArea class="min-h-0 flex-1">
                  <OutputBlock :code="selectedRunOutput" empty-text="No log output recorded." class="text-sm leading-6" />
                </ScrollArea>
                <Separator />
                <div class="p-4">
                  <div class="grid gap-4">
                    <Textarea class="p-4" :placeholder="`Add a note for ${selectedRunLog.run.workflowName}...`" />
                    <div class="flex items-center">
                      <Label html-for="mute-run-notes" class="flex items-center gap-2 text-xs font-normal">
                        <Switch id="mute-run-notes" aria-label="Mute run notes" />
                        Mute run notes
                      </Label>
                      <Button type="button" size="sm" class="ml-auto" disabled>
                        <Send class="size-4" />
                        Send
                      </Button>
                    </div>
                  </div>
                </div>
              </div>

              <div v-else class="grid flex-1 place-items-center p-8 text-center text-sm text-muted-foreground">
                <div>
                  <TerminalSquare class="mx-auto mb-3 size-8" />
                  <p>Select a run to inspect its persisted output.</p>
                </div>
              </div>
            </template>

            <template v-else-if="section === 'settings'">
              <div class="flex items-start p-4">
                <div class="flex items-start gap-4 text-sm">
                  <Avatar size="sm">
                    <AvatarFallback>SE</AvatarFallback>
                  </Avatar>
                  <div class="grid gap-1">
                    <div class="font-semibold">Settings</div>
                    <div class="line-clamp-1 text-xs">Repository, workflow storage, and operational defaults.</div>
                    <div class="line-clamp-1 text-xs">
                      <span class="font-medium">Status:</span> {{ settingsResponse?.valid ? "Valid" : "Needs review" }}
                    </div>
                  </div>
                </div>
                <Badge class="ml-auto" :variant="settingsResponse?.valid ? 'default' : 'destructive'">
                  {{ settingsResponse?.valid ? "valid" : "invalid" }}
                </Badge>
              </div>
              <Separator />
              <ScrollArea class="min-h-0 flex-1">
                <div class="grid gap-6 p-4">
                  <section class="grid gap-3">
                    <h2 class="text-sm font-semibold">Repository</h2>
                    <div class="grid gap-2">
                      <Label for="repo-root">Repo root</Label>
                      <div class="flex gap-2">
                        <Input id="repo-root" v-model="settingsForm.repoRoot" data-testid="settings-repo-root" />
                        <Tooltip>
                          <TooltipTrigger as-child>
                            <Button type="button" variant="outline" size="icon" @click="chooseDirectory('repoRoot')">
                              <FolderOpen class="size-4" />
                              <span class="sr-only">Choose repo root</span>
                            </Button>
                          </TooltipTrigger>
                          <TooltipContent>Choose repo root</TooltipContent>
                        </Tooltip>
                      </div>
                    </div>
                  </section>

                  <section class="grid gap-3">
                    <h2 class="text-sm font-semibold">Manifests</h2>
                    <div class="grid gap-2">
                      <Label for="apps-config">Apps manifest</Label>
                      <div class="flex gap-2">
                        <Input id="apps-config" v-model="settingsForm.appsConfigPath" data-testid="settings-apps-config" />
                        <Button type="button" variant="outline" size="icon" @click="chooseFile('appsConfigPath')">
                          <FileText class="size-4" />
                          <span class="sr-only">Choose apps manifest</span>
                        </Button>
                      </div>
                    </div>
                    <div class="grid gap-2">
                      <Label for="secrets-config">Secrets manifest</Label>
                      <div class="flex gap-2">
                        <Input id="secrets-config" v-model="settingsForm.secretsConfigPath" />
                        <Button type="button" variant="outline" size="icon" @click="chooseFile('secretsConfigPath')">
                          <FileText class="size-4" />
                          <span class="sr-only">Choose secrets manifest</span>
                        </Button>
                      </div>
                    </div>
                    <div class="grid gap-2">
                      <Label for="generated-apps">Generated apps output</Label>
                      <div class="flex gap-2">
                        <Input id="generated-apps" v-model="settingsForm.generatedAppsPath" />
                        <Button type="button" variant="outline" size="icon" @click="chooseSaveFile('generatedAppsPath')">
                          <Save class="size-4" />
                          <span class="sr-only">Choose generated apps output</span>
                        </Button>
                      </div>
                    </div>
                  </section>

                  <section class="grid gap-3">
                    <h2 class="text-sm font-semibold">Storage</h2>
                    <div class="grid gap-2">
                      <Label for="workflow-db">Workflow SQLite database</Label>
                      <div class="flex gap-2">
                        <Input id="workflow-db" v-model="settingsForm.workflowDbPath" data-testid="settings-workflow-db" />
                        <Button type="button" variant="outline" size="icon" @click="chooseSaveFile('workflowDbPath')">
                          <Database class="size-4" />
                          <span class="sr-only">Choose workflow database</span>
                        </Button>
                      </div>
                    </div>
                  </section>

                  <section class="grid gap-3">
                    <h2 class="text-sm font-semibold">Operations</h2>
                    <div class="grid gap-2">
                      <Label for="archive-root">Archive root</Label>
                      <div class="flex gap-2">
                        <Input id="archive-root" v-model="settingsForm.archiveRoot" />
                        <Button type="button" variant="outline" size="icon" @click="chooseDirectory('archiveRoot')">
                          <FolderOpen class="size-4" />
                          <span class="sr-only">Choose archive root</span>
                        </Button>
                      </div>
                    </div>
                    <div class="grid grid-cols-2 gap-3">
                      <div class="grid gap-2">
                        <Label for="op-vault">1Password vault</Label>
                        <Input id="op-vault" v-model="settingsForm.opVault" />
                      </div>
                      <div class="grid gap-2">
                        <Label for="op-item">1Password item</Label>
                        <Input id="op-item" v-model="settingsForm.opItem" />
                      </div>
                    </div>
                  </section>

                  <section class="grid gap-3">
                    <div class="flex items-center justify-between gap-3">
                      <h2 class="text-sm font-semibold">Validation</h2>
                      <Button type="button" variant="ghost" size="sm" @click="validateSettings">Validate</Button>
                    </div>
                    <div class="overflow-hidden rounded-lg border">
                      <div
                        v-for="check in settingsChecks"
                        :key="check.key"
                        class="grid gap-1 border-b px-3 py-3 text-sm last:border-b-0"
                      >
                        <div class="flex items-center gap-2">
                          <span class="font-medium">{{ check.label }}</span>
                          <Badge class="ml-auto" :variant="check.status === 'ok' ? 'secondary' : 'destructive'">{{ check.status }}</Badge>
                        </div>
                        <div class="truncate text-xs text-muted-foreground">{{ check.path }}</div>
                        <div v-if="check.message && check.message !== 'ok'" class="text-xs text-destructive">{{ check.message }}</div>
                      </div>
                    </div>
                    <div v-if="settingsError" class="rounded-lg border border-destructive/40 p-3 text-sm text-destructive">
                      {{ settingsError }}
                    </div>
                    <div v-if="settingsMessage" class="rounded-lg border p-3 text-sm text-muted-foreground">
                      {{ settingsMessage }}
                    </div>
                  </section>
                </div>
              </ScrollArea>
              <Separator />
              <div class="flex items-center gap-2 p-4">
                <Button type="button" variant="outline" :disabled="!settingsDirty || settingsSaving" @click="resetSettingsForm">
                  Reset
                </Button>
                <Button type="button" class="ml-auto" :disabled="!settingsDirty || settingsSaving" @click="saveSettings">
                  <Loader2 v-if="settingsSaving" class="size-4 animate-spin" />
                  <Save v-else class="size-4" />
                  Save settings
                </Button>
              </div>
            </template>

            <template v-else>
              <div class="flex items-start p-4">
                <div class="flex items-start gap-4 text-sm">
                  <Avatar size="sm">
                    <AvatarFallback>{{ initials(section.replace("-", " ")) }}</AvatarFallback>
                  </Avatar>
                  <div class="grid gap-1">
                    <div class="font-semibold capitalize">{{ section.replace("-", " ") }}</div>
                    <div class="line-clamp-1 text-xs">This area is available from the mail-style navigation.</div>
                    <div class="line-clamp-1 text-xs">
                      <span class="font-medium">Profile:</span> Current Mac
                    </div>
                  </div>
                </div>
              </div>
              <Separator />
              <div class="grid flex-1 place-items-center p-8 text-center text-sm text-muted-foreground">
                <div>
                  <ShieldCheck class="mx-auto mb-3 size-8" />
                  <p>Workflow execution and log inspection are available from Workflows and Logs.</p>
                </div>
              </div>
            </template>
          </div>
        </ResizablePanel>
      </ResizablePanelGroup>
    </div>

    <AlertDialog :open="pendingOption !== null" @update:open="updateConfirmationOpen">
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle class="flex items-center gap-2">
            <AlertTriangle class="size-5 text-destructive" />
            {{ pendingOption?.label }}
          </AlertDialogTitle>
          <AlertDialogDescription>{{ pendingOption?.description }}</AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>Cancel</AlertDialogCancel>
          <Button :disabled="running || !pendingOption" @click="pendingOption && runSelected(pendingOption)">
            <Loader2 v-if="running" class="size-4 animate-spin" />
            Continue
          </Button>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  </TooltipProvider>
</template>
