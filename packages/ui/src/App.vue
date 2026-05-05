<script setup lang="ts">
import {
  Activity,
  AlertTriangle,
  Apple,
  AppWindow,
  ArchiveRestore,
  ArrowLeft,
  Beer,
  Camera,
  CheckCircle2,
  Circle,
  Database,
  Download,
  Eye,
  FileCheck2,
  FileCode2,
  Files,
  FileText,
  FolderOpen,
  Github,
  HardDrive,
  History,
  Inbox,
  KeyRound,
  Link2,
  Loader2,
  Lock,
  ListChecks,
  Moon,
  Play,
  Printer,
  RefreshCw,
  RotateCcw,
  Save,
  Search,
  Send,
  Settings,
  ShieldCheck,
  Sliders,
  Sun,
  TerminalSquare,
  Trash2,
  Wand2,
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
import { Badge } from "@/components/ui/badge";
import StatusBadge from "@/components/StatusBadge.vue";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import OutputBlock from "@/components/OutputBlock.vue";
import WorkflowCardList from "@/components/WorkflowCardList.vue";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Progress } from "@/components/ui/progress";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { ResizableHandle, ResizablePanel, ResizablePanelGroup } from "@/components/ui/resizable";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Separator } from "@/components/ui/separator";
import { Skeleton } from "@/components/ui/skeleton";
import { Switch } from "@/components/ui/switch";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Textarea } from "@/components/ui/textarea";
import { ToastViewport, type ToastItem, type ToastTone } from "@/components/ui/toast";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { loadThemeFromBackend, useTheme } from "@/composables/useTheme";
import { cn } from "@/lib/utils";
import { confirmationStyle } from "@/lib/confirmationDisplay";
import { getWorkflowDetail, workflowsInCategory, type WorkflowCategory } from "@/lib/workflowDetails";
import type { ConfirmationOption, OpItem, OpVault, Phase, RunEvent, RunLog, RunSummary, RuntimeSettings, SettingsResponse, Workflow } from "./types/api";

type SectionId = "template" | "current" | "update" | "settings" | "logs";

type StepSettingsKey = keyof RuntimeSettings;

const { theme, toggleTheme } = useTheme();

const panelFrameClass = "bg-panel";
const panelHeaderClass = "bg-section";
const searchBarClass = "border-b border-section-border bg-section-muted/95 px-4 pt-4 pb-2 backdrop-blur supports-[backdrop-filter]:bg-section-muted/80";
const listItemClass = "bg-section border-section-border shadow-sm hover:border-primary/40 hover:bg-accent";
const selectedListItemClass = "border-primary/50 bg-accent text-accent-foreground shadow-sm";
const detailSectionClass = "rounded-lg border border-section-border bg-section p-4 shadow-sm";
const detailSectionBodyClass = "mt-3 grid gap-3 rounded-md border border-section-border bg-section-muted p-3";

const section = ref<SectionId>("template");
const selectedSettingsKey = ref<StepSettingsKey | null>(null);
const macName = ref("");
const macHostname = ref("");
const workflows = ref<Workflow[]>([]);
const runs = ref<RunSummary[]>([]);
const selectedWorkflowId = ref("");
const selectedRunId = ref("");
const selectedRunLog = ref<RunLog | null>(null);
const enabledPhaseIds = ref<Set<string>>(new Set());
const pendingOption = ref<ConfirmationOption | null>(null);
const runEvents = ref<RunEvent[]>([]);
const running = ref(false);
const workflowsLoading = ref(true);
const runsLoading = ref(true);
const initialLoading = ref(true);
const runLogLoading = ref(false);
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
const settingsLoading = ref(false);
const settingsValidating = ref(false);
const settingsPickerField = ref<keyof RuntimeSettings | null>(null);
const settingsMessage = ref("");
const settingsError = ref("");
const opVaults = ref<OpVault[]>([]);
const opItems = ref<OpItem[]>([]);
const opVaultsLoading = ref(false);
const opItemsLoading = ref(false);
const opVaultsError = ref("");
const opItemsError = ref("");
const opItemsLoadedFor = ref<string>("");
const opSigninLoading = ref(false);
const opInstallLoading = ref(false);
const toasts = ref<ToastItem[]>([]);

const stepNavItems = computed(() => [
  { id: "template" as const, label: "Template", icon: FileText, count: workflowsLoading.value ? null : workflowsInCategory(workflows.value, "template").length },
  { id: "current" as const, label: "Current state", icon: Eye, count: workflowsLoading.value ? null : workflowsInCategory(workflows.value, "current").length },
  { id: "update" as const, label: "Update", icon: Wand2, count: workflowsLoading.value ? null : workflowsInCategory(workflows.value, "update").length },
]);

const auxNavItems = computed(() => [
  { id: "settings" as const, label: "Settings", icon: Settings, count: null as number | null },
  { id: "logs" as const, label: "Logs", icon: History, count: runsLoading.value ? null : runs.value.length },
]);

const stepSectionMeta: Record<"template" | "current" | "update", { title: string; emptyMessage: string; settingsKeys: StepSettingsKey[] }> = {
  template: {
    title: "Template",
    emptyMessage: "No template workflows registered.",
    settingsKeys: ["repoRoot", "appsConfigPath", "secretsConfigPath"],
  },
  current: {
    title: "Current state",
    emptyMessage: "No current-state workflows registered.",
    settingsKeys: ["archiveRoot", "generatedAppsPath", "workflowDbPath"],
  },
  update: {
    title: "Update",
    emptyMessage: "No update workflows registered.",
    settingsKeys: ["archiveRoot", "opVault", "opItem"],
  },
};

const stepMeta = computed(() => {
  if (section.value === "template" || section.value === "current" || section.value === "update") {
    return stepSectionMeta[section.value];
  }
  return null;
});

const settingsKeyLabels: Record<StepSettingsKey, string> = {
  repoRoot: "Repository root",
  appsConfigPath: "Apps manifest",
  secretsConfigPath: "Secrets manifest",
  generatedAppsPath: "Generated apps output",
  archiveRoot: "Archive root",
  workflowDbPath: "Workflow SQLite database",
  opVault: "1Password vault",
  opItem: "1Password item",
};

const settingsWorkflows = computed(() => workflowsInCategory(workflows.value, "settings"));

const selectedWorkflow = computed(() => workflows.value.find((workflow) => workflow.id === selectedWorkflowId.value));
const selectedWorkflowDetail = computed(() => (selectedWorkflow.value ? getWorkflowDetail(selectedWorkflow.value.id) : null));
const selectedRun = computed(() => runs.value.find((run) => run.id === selectedRunId.value));
const settingsDirty = computed(() => JSON.stringify(settingsForm.value) !== JSON.stringify(settingsResponse.value?.settings ?? emptySettings()));
const settingsChecks = computed(() => settingsResponse.value?.checks ?? []);

const opVaultOptions = computed(() => {
  const options = opVaults.value.map((vault) => ({ value: vault.name, label: vault.name, missing: false }));
  const current = settingsForm.value.opVault;

  if (current && !options.some((option) => option.value === current)) {
    options.unshift({ value: current, label: `${current} (not found)`, missing: true });
  }

  return options;
});

const opItemOptions = computed(() => {
  const options = opItems.value.map((item) => ({ value: item.title, label: item.title, missing: false }));
  const current = settingsForm.value.opItem;
  const vault = settingsForm.value.opVault;

  if (current && vault && opItemsLoadedFor.value === vault && !options.some((option) => option.value === current)) {
    options.unshift({ value: current, label: `${current} (not found)`, missing: true });
  }

  return options;
});

const opItemSelectDisabled = computed(() => {
  return !settingsForm.value.opVault || opVaultsError.value !== "" || opItemsLoading.value || (opItemsError.value !== "" && opItemsLoadedFor.value === settingsForm.value.opVault);
});

const opSavedFields = computed(() => [
  {
    key: "opVault",
    label: "Vault",
    saved: settingsResponse.value?.settings.opVault ?? "",
    pending: settingsForm.value.opVault,
  },
  {
    key: "opItem",
    label: "Item",
    saved: settingsResponse.value?.settings.opItem ?? "",
    pending: settingsForm.value.opItem,
  },
]);

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

const sectionWorkflows = computed(() => {
  if (section.value === "logs" || section.value === "settings") return [];
  return workflowsInCategory(workflows.value, section.value as WorkflowCategory);
});

const matchingWorkflows = computed(() => {
  const query = normalizedSearch.value;
  const source = sectionWorkflows.value;
  return query
    ? source.filter((workflow) =>
        [workflow.name, workflow.description]
          .join(" ")
          .toLowerCase()
          .includes(query),
      )
    : source;
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

const workflowProgress = computed(() => {
  const phases = displayPhases.value;

  if (phases.length === 0) {
    return 0;
  }

  const completed = phases.filter((phase) => ["completed", "ok", "skipped"].includes(phaseStatus(phase))).length;

  return Math.round((completed / phases.length) * 100);
});

const selectedRunOutput = computed(() =>
  selectedRunLog.value?.events
    .map((event) => event.message || `${event.type} ${event.status || ""}`.trim())
    .filter(Boolean)
    .join("\n") ?? "",
);

onMounted(() => {
  void loadAll();
});

async function loadAll() {
  loadError.value = "";
  const isInitialLoad = initialLoading.value;

  if (!isInitialLoad) {
    workflowsLoading.value = true;
    runsLoading.value = true;
  }

  if (isInitialLoad) {
    const [workflowsResult, runsResult, settingsResult, themeResult, macNameResult, macHostnameResult] = await Promise.allSettled([
      window.macOS.workflows(),
      window.macOS.runs(25),
      window.macOS.settings(),
      loadThemeFromBackend(),
      window.macOS.macName(),
      window.macOS.macHostname(),
    ]);

    if (workflowsResult.status === "fulfilled") {
      workflows.value = workflowsResult.value;
      selectedWorkflowId.value = workflowsResult.value[0]?.id ?? "";
      resetEnabledPhases();
    } else {
      loadError.value = errorMessage(workflowsResult.reason);
    }

    if (runsResult.status === "fulfilled") {
      runs.value = runsResult.value;
    } else {
      console.error("Failed to load runs", runsResult.reason);
    }

    if (settingsResult.status === "fulfilled") {
      settingsResponse.value = settingsResult.value;
      settingsForm.value = { ...settingsResult.value.settings };
      settingsError.value = "";
    } else {
      console.error("Failed to load settings", settingsResult.reason);
    }

    if (themeResult.status === "rejected") {
      console.error("Failed to load theme preference", themeResult.reason);
    }

    macName.value = macNameResult.status === "fulfilled" ? macNameResult.value : "Mac";
    macHostname.value = macHostnameResult.status === "fulfilled" ? macHostnameResult.value : "local";
    workflowsLoading.value = false;
    runsLoading.value = false;
    settingsLoading.value = false;
    initialLoading.value = false;

    return;
  }

  const workflowsPromise = window.macOS
    .workflows()
    .then((next) => {
      workflows.value = next;
      if (!selectedWorkflowId.value || !next.some((workflow) => workflow.id === selectedWorkflowId.value)) {
        selectedWorkflowId.value = next[0]?.id ?? "";
      }
      resetEnabledPhases();
    })
    .catch((error) => {
      loadError.value = error instanceof Error ? error.message : String(error);
    })
    .finally(() => {
      workflowsLoading.value = false;
    });

  const runsPromise = window.macOS
    .runs(25)
    .then((next) => {
      runs.value = next;
    })
    .catch((error) => {
      console.error("Failed to load runs", error);
    })
    .finally(() => {
      runsLoading.value = false;
    });

  const settingsPromise = loadSettings().catch((error) => {
    console.error("Failed to load settings", error);
  });

  const themePromise = loadThemeFromBackend();

  const macNamePromise = window.macOS
    .macName()
    .then((name) => {
      macName.value = name;
    })
    .catch(() => {
      macName.value ||= "Mac";
    });

  const macHostnamePromise = window.macOS
    .macHostname()
    .then((name) => {
      macHostname.value = name;
    })
    .catch(() => {
      macHostname.value ||= "local";
    });

  await Promise.allSettled([workflowsPromise, runsPromise, settingsPromise, themePromise, macNamePromise, macHostnamePromise]);
}

function errorMessage(error: unknown) {
  return error instanceof Error ? error.message : String(error);
}

function selectSection(next: SectionId) {
  section.value = next;
  searchQuery.value = "";
  selectedWorkflowId.value = "";
  selectedSettingsKey.value = null;

  if (next === "logs") {
    void refreshRuns();
  }

  if (next === "settings") {
    void loadSettings();
    void loadOpVaults();
  }
}

function selectStepSetting(key: StepSettingsKey) {
  selectedWorkflowId.value = "";
  selectedSettingsKey.value = key;
  void loadSettings();

  if (key === "opVault" || key === "opItem") {
    void loadOpVaults();
  }
}

async function loadOpVaults() {
  opVaultsLoading.value = true;
  opVaultsError.value = "";

  try {
    const result = await window.macOS.listOpVaults();

    if (result.ok) {
      opVaults.value = result.vaults;

      const currentVault = settingsForm.value.opVault;

      if (currentVault) {
        await loadOpItems(currentVault);
      }

      return;
    }

    opVaultsError.value = result.message;
    opVaults.value = [];
    opItems.value = [];
    opItemsLoadedFor.value = "";
  } finally {
    opVaultsLoading.value = false;
  }
}

async function loadOpItems(vault: string) {
  if (!vault) {
    opItems.value = [];
    opItemsLoadedFor.value = "";
    opItemsError.value = "";

    return;
  }

  opItemsLoading.value = true;
  opItemsError.value = "";
  opItemsLoadedFor.value = vault;

  try {
    const result = await window.macOS.listOpItems(vault);

    if (opItemsLoadedFor.value !== vault) {
      return;
    }

    if (result.ok) {
      opItems.value = result.items;

      return;
    }

    opItemsError.value = result.message;
    opItems.value = [];
  } finally {
    if (opItemsLoadedFor.value === vault) {
      opItemsLoading.value = false;
    }
  }
}

function onOpVaultChange(value: unknown) {
  const next = typeof value === "string" ? value : "";

  if (settingsForm.value.opVault === next) {
    return;
  }

  settingsForm.value = { ...settingsForm.value, opVault: next, opItem: "" };
  void loadOpItems(next);
}

function onOpItemChange(value: unknown) {
  const next = typeof value === "string" ? value : "";

  if (settingsForm.value.opItem === next) {
    return;
  }

  settingsForm.value = { ...settingsForm.value, opItem: next };
}

async function signinOpCli() {
  opSigninLoading.value = true;

  try {
    const result = await window.macOS.signinOpCli();

    if (!result.ok) {
      pushToast("Could not open Terminal", result.message, "error");

      return;
    }

    pushToast("Terminal opened", "Complete `op signin` in Terminal, then click Retry.", "info");
  } finally {
    opSigninLoading.value = false;
  }
}

async function installOpDependencies() {
  opInstallLoading.value = true;

  try {
    const result = await window.macOS.installOpDependencies();

    if (!result.ok) {
      pushToast("Could not open Terminal", result.message, "error");

      return;
    }

    pushToast("Terminal opened", "Install 1Password and the CLI, then click Retry.", "info");
  } finally {
    opInstallLoading.value = false;
  }
}

async function openDevTools() {
  try {
    await window.macOS.openDevTools();
  } catch (error) {
    pushToast("Failed to open DevTools", errorMessage(error), "error");
  }
}

function selectWorkflow(workflow: Workflow) {
  selectedWorkflowId.value = workflow.id;
  selectedSettingsKey.value = null;
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
    pushToast("Workflow started", selectedWorkflow.value.name, "loading");
    await window.macOS.runWorkflow(
      {
        workflowId: selectedWorkflow.value.id,
        confirmationOptionId: option.id,
        enabledPhaseIds: enabledIds,
      },
      (event) => runEvents.value.push(event),
    );
    pushToast("Workflow completed", selectedWorkflow.value.name, "success");
  } catch (error) {
    pushToast("Workflow failed", errorMessage(error), "error");
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
  selectedRunLog.value = null;
  runLogLoading.value = true;

  try {
    selectedRunLog.value = await window.macOS.runLog(run.id);
  } finally {
    runLogLoading.value = false;
  }
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
  settingsLoading.value = true;

  try {
    const response = await window.macOS.settings();

    settingsResponse.value = response;
    settingsForm.value = { ...response.settings };
    settingsError.value = "";
  } finally {
    settingsLoading.value = false;
  }
}

async function validateSettings() {
  settingsMessage.value = "";
  settingsError.value = "";
  settingsValidating.value = true;

  try {
    settingsResponse.value = await window.macOS.validateSettings({ ...settingsForm.value });
  } finally {
    settingsValidating.value = false;
  }
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
      pushToast("Settings need review", settingsError.value, "error");
      return;
    }

    settingsForm.value = { ...response.settings };
    settingsMessage.value = "Settings saved. The workflow bridge was restarted.";
    pushToast("Settings saved", "The workflow bridge was restarted.", "success");
    workflows.value = await window.macOS.workflows();
    runs.value = await window.macOS.runs(25);
  } catch (error) {
    settingsError.value = error instanceof Error ? error.message : String(error);
    pushToast("Settings save failed", settingsError.value, "error");
  } finally {
    settingsSaving.value = false;
  }
}

function pushToast(title: string, description?: string, tone: ToastTone = "info") {
  const id = crypto.randomUUID();
  const previous = tone === "loading" ? toasts.value : toasts.value.filter((toast) => toast.tone !== "loading");

  toasts.value = [...previous.slice(-2), { id, title, description, tone }];

  if (tone !== "loading") {
    window.setTimeout(() => dismissToast(id), 5000);
  }
}

function dismissToast(id: string) {
  toasts.value = toasts.value.filter((toast) => toast.id !== id);
}

function resetSettingsForm() {
  settingsForm.value = { ...(settingsResponse.value?.settings ?? emptySettings()) };
  settingsError.value = "";
  settingsMessage.value = "";
}

async function chooseDirectory(field: keyof RuntimeSettings) {
  settingsPickerField.value = field;

  try {
    const path = await window.macOS.chooseDirectory(settingsForm.value[field]);

    if (path) {
      settingsForm.value = { ...settingsForm.value, [field]: path };
    }
  } finally {
    settingsPickerField.value = null;
  }
}

async function chooseFile(field: keyof RuntimeSettings) {
  settingsPickerField.value = field;

  try {
    const path = await window.macOS.chooseFile(settingsForm.value[field]);

    if (path) {
      settingsForm.value = { ...settingsForm.value, [field]: path };
    }
  } finally {
    settingsPickerField.value = null;
  }
}

async function chooseSaveFile(field: keyof RuntimeSettings) {
  settingsPickerField.value = field;

  try {
    const path = await window.macOS.chooseSaveFile(settingsForm.value[field]);

    if (path) {
      settingsForm.value = { ...settingsForm.value, [field]: path };
    }
  } finally {
    settingsPickerField.value = null;
  }
}

function phaseStatus(phase: Phase) {
  const events = runEvents.value.filter((event) => event.phaseId === phase.id);
  const finish = [...events].reverse().find((event) => event.type === "phase_finished" || event.type === "phase_skipped");

  return finish?.status ?? events.at(-1)?.status ?? (enabledPhaseIds.value.has(phase.id) ? "pending" : "skipped");
}

const phaseIcons: Record<string, typeof Download> = {
  "check-install-prerequisites": ListChecks,
  "install-homebrew-packages": Beer,
  "set-up-github-access-and-signing": Github,
  "install-app-store-apps": Apple,
  "show-manual-app-install-notes": FileText,
  "restore-private-secrets-from-1password": Lock,
  "prepare-existing-dotfiles": FolderOpen,
  "install-oh-my-zsh": TerminalSquare,
  "link-dotfiles": Link2,
  "apply-macos-settings": Wand2,
  "apply-tracked-macos-settings": Wand2,
  "run-health-checks": Activity,
  "restore-supported-app-configs-from-latest-snapshot": ArchiveRestore,
  "restore-supported-app-settings": ArchiveRestore,
  "save-supported-app-settings-snapshot": Camera,
  "generate-installed-app-list-candidate": FileCode2,
  "print-generated-homebrew-package-list": Printer,
  "print-tracked-homebrew-bundle": Printer,
  "list-tracked-apps": AppWindow,
  "list-tracked-macos-settings": Sliders,
  "list-tracked-dotfile-bundles": Files,
  "validate-template-files": FileCheck2,
  "list-installed-homebrew-formulae-and-casks": Beer,
  "show-current-macos-defaults-values": Sliders,
  "scan-untracked-items": Search,
  "snapshot-before-remove": Camera,
  "uninstall-untracked-homebrew-formulae-and-casks": Trash2,
  "uninstall-untracked-app-store-apps-best-effort": Trash2,
};

function phaseIcon(id: string) {
  return phaseIcons[id] ?? Circle;
}

const confirmationIcons: Record<string, typeof Play> = {
  "preview-only": Eye,
  "run-now": Play,
  "already-erased-run-now": Play,
  "run-without-erasing": Play,
  "erase-first": Trash2,
  back: ArrowLeft,
};

function confirmationIcon(id: string) {
  return confirmationIcons[id] ?? Play;
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
      <div v-if="initialLoading" data-testid="initial-shell-skeleton" class="grid h-screen grid-cols-[230px_410px_1fr] overflow-hidden">
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
            <div v-for="index in 5" :key="`nav-shell-${index}`" class="flex h-8 items-center gap-2 rounded-md px-2">
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
            <div v-for="index in 6" :key="`list-shell-${index}`" class="rounded-lg border border-section-border bg-section p-3">
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
                <div v-for="index in 4" :key="`phase-shell-${index}`" class="flex items-center gap-3 border-b px-3 py-3 last:border-b-0">
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

      <ResizablePanelGroup v-else direction="horizontal" class="h-screen max-h-screen items-stretch">
        <ResizablePanel
          id="mac-nav"
          :default-size="18"
          :collapsed-size="4"
          collapsible
          :min-size="14"
          :max-size="22"
          :class="cn(navCollapsed && 'min-w-12 transition-all duration-300 ease-in-out')"
          @collapse="navCollapsed = true"
          @expand="navCollapsed = false"
        >
          <div :class="cn('flex h-12 items-center bg-sidebar', navCollapsed ? 'justify-center px-2' : 'gap-2 px-3')">
            <Apple class="size-4 shrink-0" />
            <div v-if="!navCollapsed" class="flex min-w-0 flex-col">
              <span class="truncate text-sm font-medium">Mac: {{ macName }}</span>
              <span class="truncate text-[10px] text-muted-foreground">{{ macHostname }}</span>
            </div>
            <Tooltip v-if="!navCollapsed">
              <TooltipTrigger as-child>
                <Button variant="ghost" size="icon-sm" class="ml-auto" @click="toggleTheme">
                  <Sun v-if="theme === 'dark'" class="size-4" />
                  <Moon v-else class="size-4" />
                  <span class="sr-only">Toggle theme</span>
                </Button>
              </TooltipTrigger>
              <TooltipContent>Toggle theme</TooltipContent>
            </Tooltip>
          </div>

          <Separator />

          <div :data-collapsed="navCollapsed" class="group flex flex-col gap-4 py-2 data-[collapsed=true]:py-2">
            <nav class="grid gap-1 px-2 group-[[data-collapsed=true]]:justify-center group-[[data-collapsed=true]]:px-2">
              <template v-for="item in stepNavItems" :key="item.id">
                <Tooltip v-if="navCollapsed">
                  <TooltipTrigger as-child>
                    <Button
                      variant="ghost"
                      size="icon"
                      :class="cn(section === item.id && 'bg-accent text-accent-foreground hover:bg-accent dark:hover:bg-accent')"
                      @click="selectSection(item.id)"
                    >
                      <component :is="item.icon" class="size-4" />
                      <span class="sr-only">{{ item.label }}</span>
                    </Button>
                  </TooltipTrigger>
                  <TooltipContent side="right" class="flex items-center gap-4">
                    {{ item.label }}
                    <Skeleton v-if="item.count === null" class="ml-auto h-4 w-6 rounded" />
                    <span v-else class="ml-auto text-muted-foreground">{{ item.count }}</span>
                  </TooltipContent>
                </Tooltip>

                <Button
                  v-else
                  variant="ghost"
                  size="sm"
                  :class="cn('justify-start', section === item.id && 'bg-accent text-accent-foreground hover:bg-accent dark:hover:bg-accent')"
                  @click="selectSection(item.id)"
                >
                  <component :is="item.icon" class="mr-2 size-4" />
                  {{ item.label }}
                  <Skeleton v-if="item.count === null" class="ml-auto h-4 w-6 rounded" />
                  <span v-else class="ml-auto">{{ item.count }}</span>
                </Button>
              </template>
            </nav>
          </div>

          <Separator />

          <div :data-collapsed="navCollapsed" class="group flex flex-col gap-4 py-2 data-[collapsed=true]:py-2">
            <nav class="grid gap-1 px-2 group-[[data-collapsed=true]]:justify-center group-[[data-collapsed=true]]:px-2">
              <template v-for="item in auxNavItems" :key="item.id">
                <Tooltip v-if="navCollapsed">
                  <TooltipTrigger as-child>
                    <Button
                      variant="ghost"
                      size="icon"
                      :class="cn(section === item.id && 'bg-accent text-accent-foreground hover:bg-accent dark:hover:bg-accent')"
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
                  :class="cn('justify-start', section === item.id && 'bg-accent text-accent-foreground hover:bg-accent dark:hover:bg-accent')"
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
          <div :class="cn('flex h-full min-h-0 flex-col', panelFrameClass)">
            <template v-if="stepMeta">
              <div class="flex h-full min-h-0 flex-col">
                <div :class="cn('flex min-h-[var(--panel-header-h)] items-center px-4', panelHeaderClass)">
                  <h1 class="text-xl font-bold">{{ stepMeta.title }}</h1>
                </div>
                <Separator />
                <div :class="searchBarClass">
                  <form @submit.prevent>
                    <div class="relative">
                      <Search class="absolute left-2 top-2.5 size-4 text-muted-foreground" />
                      <Input v-model="searchQuery" data-testid="app-search" placeholder="Search workflows" class="pl-8" />
                    </div>
                  </form>
                </div>
                <ScrollArea class="min-h-0 flex-1">
                  <div v-if="workflowsLoading" data-testid="workflows-list-skeleton" class="flex flex-col gap-2 p-4 pt-0">
                    <div v-for="index in 6" :key="index" class="rounded-lg border border-section-border bg-section p-3 shadow-sm">
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
                    :workflows="matchingWorkflows"
                    :selected-id="selectedWorkflowId"
                    :empty-message="stepMeta.emptyMessage"
                    @select="selectWorkflow"
                  />
                  <div class="px-4 pt-4 pb-2 text-xs font-semibold uppercase tracking-wide text-muted-foreground">
                    Step settings
                  </div>
                  <div class="flex flex-col gap-1 px-4 pb-4">
                    <template v-if="settingsLoading && !settingsResponse">
                      <div
                        v-for="key in stepMeta.settingsKeys"
                        :key="key"
                        data-testid="step-settings-skeleton"
                        class="flex items-center gap-3 rounded-lg border border-section-border bg-section px-3 py-2 shadow-sm"
                      >
                        <Skeleton class="size-4 rounded" />
                        <div class="min-w-0 flex-1 space-y-1">
                          <Skeleton class="h-4 w-32" />
                          <Skeleton class="h-3 w-48" />
                        </div>
                      </div>
                    </template>
                    <template v-else>
                      <button
                        v-for="key in stepMeta.settingsKeys"
                        :key="key"
                        :class="cn(
                          'flex items-center gap-3 rounded-lg border px-3 py-2 text-left text-sm transition-all hover:bg-accent',
                          listItemClass,
                          selectedSettingsKey === key && selectedListItemClass,
                        )"
                        @click="selectStepSetting(key)"
                      >
                        <Settings class="size-4 text-muted-foreground" />
                        <div class="min-w-0 flex-1">
                          <div class="font-medium">{{ settingsKeyLabels[key] }}</div>
                          <div class="truncate text-xs text-muted-foreground">{{ settingsForm[key] || "not set" }}</div>
                        </div>
                      </button>
                      <button
                        :class="cn('flex items-center gap-3 rounded-lg border px-3 py-2 text-left text-sm transition-all', listItemClass)"
                        @click="openDevTools"
                      >
                        <TerminalSquare class="size-4 text-muted-foreground" />
                        <div class="min-w-0 flex-1">
                          <div class="font-medium">DevTools</div>
                          <div class="truncate text-xs text-muted-foreground">Open developer tools</div>
                        </div>
                      </button>
                    </template>
                  </div>
                </ScrollArea>
              </div>
            </template>

            <template v-else-if="section === 'logs'">
              <Tabs v-model="logTab" class="flex h-full min-h-0 flex-col">
                <div :class="cn('flex min-h-[var(--panel-header-h)] items-center px-4', panelHeaderClass)">
                  <h1 class="text-xl font-bold">Logs</h1>
                  <TabsList class="ml-auto">
                    <TabsTrigger value="all">All</TabsTrigger>
                    <TabsTrigger value="failed">Failed</TabsTrigger>
                    <TabsTrigger value="active">Active</TabsTrigger>
                  </TabsList>
                </div>
                <Separator />
                <div :class="searchBarClass">
                  <form @submit.prevent>
                    <div class="relative">
                      <Search class="absolute left-2 top-2.5 size-4 text-muted-foreground" />
                      <Input v-model="searchQuery" data-testid="app-search" placeholder="Search logs" class="pl-8" />
                    </div>
                  </form>
                </div>
                <ScrollArea class="min-h-0 flex-1">
                  <div v-if="runsLoading" data-testid="runs-list-skeleton" class="flex flex-col gap-2 p-4 pt-0">
                    <div v-for="index in 4" :key="index" class="rounded-lg border border-section-border bg-section p-3 shadow-sm">
                      <div class="flex items-center gap-2">
                        <Skeleton class="h-4 w-44" />
                        <Skeleton class="ml-auto h-5 w-16 rounded-full" />
                      </div>
                      <div class="mt-2 flex items-center justify-between gap-3">
                        <Skeleton class="h-3 w-32" />
                        <Skeleton class="h-3 w-16" />
                      </div>
                    </div>
                  </div>
                  <div v-else class="flex flex-col gap-2 p-4 pt-0">
                    <button
                      v-for="run in matchingRuns"
                      :key="run.id"
                      :class="cn(
                        'flex flex-col items-start gap-2 rounded-lg border p-3 text-left text-sm transition-all hover:bg-accent',
                        listItemClass,
                        selectedRunId === run.id && selectedListItemClass,
                      )"
                      @click="openRun(run)"
                    >
                      <div class="flex w-full flex-col gap-1">
                        <div class="flex min-w-0 items-center gap-2">
                          <div class="truncate font-semibold">{{ run.workflowName }}</div>
                          <StatusBadge class="ml-auto" :status="run.status" />
                        </div>
                        <div class="flex items-center justify-between gap-3 text-xs text-muted-foreground">
                          <span class="truncate">{{ run.mode }} - {{ run.confirmationOptionLabel }}</span>
                          <span class="shrink-0">{{ timeAgo(run.startedAt) }}</span>
                        </div>
                      </div>
                    </button>

                    <div v-if="matchingRuns.length === 0" class="rounded-lg border border-dashed border-section-border bg-section p-8 text-center text-sm text-muted-foreground">
                      No logs match this view.
                    </div>
                  </div>
                </ScrollArea>
              </Tabs>
            </template>

            <template v-else-if="section === 'settings'">
              <div :class="cn('flex min-h-[var(--panel-header-h)] items-center px-4 py-2', panelHeaderClass)">
                <h1 class="text-xl font-bold">Settings</h1>
                <Skeleton v-if="settingsLoading && !settingsResponse" class="ml-auto h-5 w-24 rounded-full" />
                <StatusBadge
                  v-else
                  class="ml-auto"
                  :status="settingsResponse?.valid ? 'ok' : 'failed'"
                  :label="settingsResponse?.valid ? 'valid' : 'needs review'"
                />
              </div>
              <Separator />
              <ScrollArea class="min-h-0 flex-1">
                <div class="flex flex-col gap-2 p-4">
                  <template v-if="settingsLoading && !settingsResponse">
                    <div
                      v-for="i in 4"
                      :key="`settings-group-skeleton-${i}`"
                      data-testid="settings-groups-skeleton"
                      class="flex items-center gap-3 rounded-lg border border-section-border bg-section p-3 text-sm shadow-sm"
                    >
                      <Skeleton class="size-4 rounded" />
                      <div class="min-w-0 flex-1 space-y-1">
                        <Skeleton class="h-4 w-28" />
                        <Skeleton class="h-3 w-40" />
                      </div>
                      <Skeleton class="h-5 w-6 rounded-full" />
                    </div>
                  </template>
                  <template v-else>
                    <div
                      v-for="group in settingsGroups"
                      :key="group.id"
                      class="flex items-center gap-3 rounded-lg border border-section-border bg-section p-3 text-sm shadow-sm"
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
                  </template>
                </div>
                <Separator />
                <div class="px-4 pt-4 pb-2 text-xs font-semibold uppercase tracking-wide text-muted-foreground">
                  Workflows
                </div>
                <div v-if="workflowsLoading" data-testid="settings-workflows-skeleton" class="flex flex-col gap-2 p-4 pt-0">
                  <div v-for="i in 3" :key="`settings-wf-skeleton-${i}`" class="rounded-lg border border-section-border bg-section p-3 shadow-sm">
                    <div class="flex items-center gap-3">
                      <Skeleton class="h-4 w-40" />
                      <Skeleton class="ml-auto h-5 w-12 rounded-full" />
                    </div>
                    <Skeleton class="mt-3 h-3 w-24" />
                    <Skeleton class="mt-3 h-3 w-full" />
                  </div>
                </div>
                <WorkflowCardList
                  v-else
                  :workflows="settingsWorkflows"
                  :selected-id="selectedWorkflowId"
                  empty-message="No settings workflows available."
                  @select="selectWorkflow"
                />
              </ScrollArea>
            </template>
          </div>
        </ResizablePanel>

        <ResizableHandle with-handle />

        <ResizablePanel id="mac-detail" :default-size="50" :min-size="35">
          <div class="flex h-full min-h-0 flex-col bg-background">
            <div :class="cn('flex min-h-[var(--panel-header-h)] items-start gap-3 px-2 py-2', panelHeaderClass)">
              <div v-if="stepMeta && selectedWorkflow" class="flex items-start gap-3 text-sm">
                <Avatar size="sm">
                  <AvatarFallback>{{ initials(selectedWorkflow.name) }}</AvatarFallback>
                </Avatar>
                <div class="grid gap-1">
                  <div class="font-semibold">{{ selectedWorkflow.name }}</div>
                  <div class="line-clamp-1 text-xs">{{ selectedWorkflow.description }}</div>
                  <div class="line-clamp-1 text-xs">
                    <span class="font-medium">Action:</span> {{ getWorkflowDetail(selectedWorkflow.id).action || selectedWorkflow.changesMac }}
                    <span class="text-muted-foreground">· Changes Mac: {{ selectedWorkflow.changesMac }}</span>
                  </div>
                </div>
              </div>

              <div class="ml-auto flex items-center gap-2">
                <StatusBadge v-if="stepMeta && selectedWorkflow" :status="runStatus" />
                <Tooltip>
                  <TooltipTrigger as-child>
                    <Button
                      variant="ghost"
                      size="icon"
                      :disabled="section === 'logs' || workflowsLoading || runsLoading || settingsLoading"
                      @click="loadAll"
                    >
                      <RefreshCw class="size-4" />
                      <span class="sr-only">Refresh</span>
                    </Button>
                  </TooltipTrigger>
                  <TooltipContent>Refresh</TooltipContent>
                </Tooltip>

                <Tooltip>
                  <TooltipTrigger as-child>
                    <Button variant="ghost" size="icon" :disabled="!stepMeta || !selectedWorkflow" @click="resetEnabledPhases">
                      <RotateCcw class="size-4" />
                      <span class="sr-only">Reset phases</span>
                    </Button>
                  </TooltipTrigger>
                  <TooltipContent>Reset phases</TooltipContent>
                </Tooltip>

                <Tooltip>
                  <TooltipTrigger as-child>
                    <Button variant="ghost" size="icon" :disabled="!selectedWorkflow?.confirmation || running" @click="openConfirmation()">
                      <Play class="size-4" />
                      <span class="sr-only">Run workflow</span>
                    </Button>
                  </TooltipTrigger>
                  <TooltipContent>Run workflow</TooltipContent>
                </Tooltip>
              </div>

            </div>

            <Separator />

            <div v-if="loadError" class="grid flex-1 place-items-center p-8">
              <div class="max-w-xl rounded-lg border border-destructive/40 bg-section p-5 shadow-sm">
                <div class="flex items-center gap-2 font-semibold text-destructive">
                  <AlertTriangle class="size-5" />
                  Load failed
                </div>
                <p class="mt-2 text-sm text-muted-foreground">{{ loadError }}</p>
              </div>
            </div>

            <template v-else-if="stepMeta && selectedSettingsKey">
              <div class="flex items-start bg-section p-4">
                <div class="flex items-start gap-4 text-sm">
                  <Avatar size="sm">
                    <AvatarFallback>{{ initials(settingsKeyLabels[selectedSettingsKey]) }}</AvatarFallback>
                  </Avatar>
                  <div class="grid gap-1">
                    <div class="font-semibold">{{ settingsKeyLabels[selectedSettingsKey] }}</div>
                    <div class="line-clamp-1 text-xs text-muted-foreground">Step setting · {{ stepMeta.title }}</div>
                  </div>
                </div>
                <StatusBadge
                  class="ml-auto"
                  :status="settingsResponse?.valid ? 'ok' : 'failed'"
                  :label="settingsResponse?.valid ? 'valid' : 'needs review'"
                />
              </div>
              <Separator />
              <ScrollArea class="min-h-0 flex-1">
                <div class="grid gap-3 p-4">
                  <Label :for="`step-setting-${selectedSettingsKey}`">{{ settingsKeyLabels[selectedSettingsKey] }}</Label>
                  <div class="flex gap-2">
                    <Input
                      :id="`step-setting-${selectedSettingsKey}`"
                      v-model="settingsForm[selectedSettingsKey]"
                    />
                    <Button type="button" variant="outline" size="icon" @click="chooseDirectory(selectedSettingsKey)">
                      <FolderOpen class="size-4" />
                    </Button>
                  </div>
                  <p class="text-xs text-muted-foreground">Edits here apply to the same setting visible in the All settings panel. Save from there to persist.</p>
                </div>
              </ScrollArea>
            </template>

            <template v-else-if="stepMeta && workflowsLoading && !selectedWorkflow">
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
                      <div v-for="index in 4" :key="index" class="flex items-center gap-3 border-b px-3 py-3 last:border-b-0">
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

            <template v-else-if="stepMeta && !selectedWorkflow">
              <div class="grid flex-1 place-items-center p-8 text-center text-sm text-muted-foreground">
                <div>
                  <Inbox class="mx-auto mb-3 size-8" />
                  <p>Select a workflow or a step setting to begin.</p>
                </div>
              </div>
            </template>

            <template v-else-if="stepMeta && selectedWorkflow">
              <ScrollArea class="min-h-0 flex-1">
                <div class="grid gap-5 p-4">
                  <section
                    v-if="selectedWorkflowDetail && (selectedWorkflowDetail.purpose || selectedWorkflowDetail.details || selectedWorkflowDetail.whenToRun || selectedWorkflowDetail.sideEffects.length || selectedWorkflowDetail.prerequisites.length)"
                    :class="detailSectionClass"
                  >
                    <h2 class="mb-2 text-sm font-semibold">About this workflow</h2>
                    <div :class="detailSectionBodyClass">
                      <div v-if="selectedWorkflowDetail.purpose">
                        <div class="text-xs font-medium uppercase tracking-wide text-muted-foreground">Purpose</div>
                        <p class="mt-1 text-sm leading-6">{{ selectedWorkflowDetail.purpose }}</p>
                      </div>
                      <div v-if="selectedWorkflowDetail.details">
                        <div class="text-xs font-medium uppercase tracking-wide text-muted-foreground">What it does</div>
                        <p class="mt-1 text-sm leading-6">{{ selectedWorkflowDetail.details }}</p>
                      </div>
                      <div v-if="selectedWorkflowDetail.whenToRun">
                        <div class="text-xs font-medium uppercase tracking-wide text-muted-foreground">When to run</div>
                        <p class="mt-1 text-sm leading-6">{{ selectedWorkflowDetail.whenToRun }}</p>
                      </div>
                      <div v-if="selectedWorkflowDetail.sideEffects.length">
                        <div class="text-xs font-medium uppercase tracking-wide text-muted-foreground">Side effects</div>
                        <ul class="mt-1 list-disc pl-5 text-sm leading-6">
                          <li v-for="effect in selectedWorkflowDetail.sideEffects" :key="effect">{{ effect }}</li>
                        </ul>
                      </div>
                      <div v-if="selectedWorkflowDetail.prerequisites.length">
                        <div class="text-xs font-medium uppercase tracking-wide text-muted-foreground">Prerequisites</div>
                        <ul class="mt-1 list-disc pl-5 text-sm leading-6">
                          <li v-for="prereq in selectedWorkflowDetail.prerequisites" :key="prereq">{{ prereq }}</li>
                        </ul>
                      </div>
                    </div>
                  </section>

                  <section :class="detailSectionClass">
                    <div class="mb-2 flex items-center justify-between gap-3">
                      <h2 class="text-sm font-semibold">Phases</h2>
                      <div class="flex items-center gap-3">
                        <Progress :value="workflowProgress" class="w-28" />
                        <Button variant="ghost" size="sm" @click="resetEnabledPhases">Reset</Button>
                      </div>
                    </div>
                    <div class="overflow-hidden rounded-lg border border-section-border bg-section-muted">
                      <button
                        v-for="phase in displayPhases"
                        :key="phase.id"
                        class="flex w-full items-center gap-3 border-b border-section-border px-3 py-3 text-left text-sm transition-colors last:border-b-0 hover:bg-accent"
                        @click="togglePhase(phase)"
                      >
                        <CheckCircle2 v-if="enabledPhaseIds.has(phase.id)" class="size-4 shrink-0 text-primary" />
                        <Circle v-else class="size-4 shrink-0 text-muted-foreground" />
                        <component :is="phaseIcon(phase.id)" class="size-4 shrink-0 text-muted-foreground" />
                        <span class="min-w-0 flex-1 truncate">{{ phase.name }}</span>
                        <StatusBadge :status="phaseStatus(phase)" />
                      </button>
                    </div>
                  </section>

                  <section v-if="selectedWorkflow.confirmation" :class="detailSectionClass">
                    <h2 class="mb-2 text-sm font-semibold">{{ selectedWorkflow.confirmation.title }}</h2>
                    <p class="mb-3 text-sm leading-6 text-muted-foreground">{{ selectedWorkflow.confirmation.message }}</p>
                    <div class="grid gap-2">
                      <Button
                        v-for="option in selectedWorkflow.confirmation.options"
                        :key="option.id"
                        variant="outline"
                        :class="cn('h-auto justify-start gap-3 whitespace-normal px-3 py-2 text-left', confirmationStyle(option.id).buttonClass)"
                        @click="openConfirmation(option)"
                      >
                        <component :is="confirmationIcon(option.id)" :class="cn('size-4 shrink-0', confirmationStyle(option.id).iconClass)" />
                        <span class="min-w-0 flex-1">
                          <span class="block font-medium">{{ option.label }}</span>
                          <span class="block text-xs text-muted-foreground">{{ option.description }}</span>
                        </span>
                      </Button>
                    </div>
                  </section>

                  <section :class="detailSectionClass">
                    <h2 class="mb-2 text-sm font-semibold">Output</h2>
                    <ScrollArea class="h-72 rounded-md border border-section-border bg-terminal text-terminal-foreground">
                      <OutputBlock :code="outputText" empty-text="No workflow output yet." class="text-xs leading-5" />
                    </ScrollArea>
                  </section>
                </div>
              </ScrollArea>

              <Separator />

              <div class="border-t border-section-border bg-section p-4">
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
              <div v-if="runLogLoading" data-testid="run-log-skeleton" class="flex min-h-0 flex-1 flex-col">
                <div class="flex items-start bg-section p-4">
                  <Skeleton class="size-10 rounded-full" />
                  <div class="ml-4 grid flex-1 gap-2">
                    <Skeleton class="h-4 w-48" />
                    <Skeleton class="h-3 w-56" />
                    <Skeleton class="h-3 w-40" />
                  </div>
                  <Skeleton class="h-5 w-20 rounded-full" />
                </div>
                <Separator />
                <div class="min-h-0 flex-1 bg-terminal p-4">
                  <Skeleton class="h-3 w-3/4 bg-white/10" />
                  <Skeleton class="mt-2 h-3 w-1/2 bg-white/10" />
                  <Skeleton class="mt-2 h-3 w-2/3 bg-white/10" />
                  <Skeleton class="mt-2 h-3 w-3/5 bg-white/10" />
                </div>
                <Separator />
                <div class="border-t border-section-border bg-section p-4">
                  <Skeleton class="h-20 w-full" />
                </div>
              </div>

              <div v-else-if="selectedRunLog" class="flex min-h-0 flex-1 flex-col">
                <div class="flex items-start bg-section p-4">
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
                  <StatusBadge class="ml-auto" :status="selectedRunLog.run.status" />
                </div>
                <Separator />
                <ScrollArea class="min-h-0 flex-1 bg-terminal text-terminal-foreground">
                  <OutputBlock :code="selectedRunOutput" empty-text="No log output recorded." class="text-sm leading-6" />
                </ScrollArea>
                <Separator />
                <div class="border-t border-section-border bg-section p-4">
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
              <div class="flex items-start bg-section p-4">
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
                <Skeleton v-if="settingsLoading || settingsSaving || settingsValidating" class="ml-auto h-5 w-16" />
                <StatusBadge
                  v-else
                  class="ml-auto"
                  :status="settingsResponse?.valid ? 'ok' : 'failed'"
                  :label="settingsResponse?.valid ? 'valid' : 'invalid'"
                />
              </div>
              <Separator />
              <ScrollArea class="min-h-0 flex-1">
                <div class="grid gap-6 p-4">
                  <Card>
                    <CardHeader>
                      <CardTitle class="text-sm">Repository</CardTitle>
                    </CardHeader>
                    <CardContent class="grid gap-3">
                      <div class="grid gap-2">
                        <Label for="repo-root">Repo root</Label>
                        <div class="flex gap-2">
                          <Skeleton v-if="settingsLoading || settingsPickerField === 'repoRoot'" class="h-9 w-full" />
                          <Input v-else id="repo-root" v-model="settingsForm.repoRoot" data-testid="settings-repo-root" />
                          <Tooltip>
                            <TooltipTrigger as-child>
                              <Button type="button" variant="outline" size="icon" :disabled="settingsLoading || settingsPickerField !== null" @click="chooseDirectory('repoRoot')">
                                <FolderOpen class="size-4" />
                                <span class="sr-only">Choose repo root</span>
                              </Button>
                            </TooltipTrigger>
                            <TooltipContent>Choose repo root</TooltipContent>
                          </Tooltip>
                        </div>
                      </div>
                    </CardContent>
                  </Card>

                  <Card>
                    <CardHeader>
                      <CardTitle class="text-sm">Manifests</CardTitle>
                    </CardHeader>
                    <CardContent class="grid gap-3">
                      <div class="grid gap-2">
                        <Label for="apps-config">Apps manifest</Label>
                        <div class="flex gap-2">
                          <Skeleton v-if="settingsLoading || settingsPickerField === 'appsConfigPath'" class="h-9 w-full" />
                          <Input v-else id="apps-config" v-model="settingsForm.appsConfigPath" data-testid="settings-apps-config" />
                          <Button type="button" variant="outline" size="icon" :disabled="settingsLoading || settingsPickerField !== null" @click="chooseFile('appsConfigPath')">
                            <FileText class="size-4" />
                            <span class="sr-only">Choose apps manifest</span>
                          </Button>
                        </div>
                      </div>
                      <div class="grid gap-2">
                        <Label for="secrets-config">Secrets manifest</Label>
                        <div class="flex gap-2">
                          <Skeleton v-if="settingsLoading || settingsPickerField === 'secretsConfigPath'" class="h-9 w-full" />
                          <Input v-else id="secrets-config" v-model="settingsForm.secretsConfigPath" />
                          <Button type="button" variant="outline" size="icon" :disabled="settingsLoading || settingsPickerField !== null" @click="chooseFile('secretsConfigPath')">
                            <FileText class="size-4" />
                            <span class="sr-only">Choose secrets manifest</span>
                          </Button>
                        </div>
                      </div>
                      <div class="grid gap-2">
                        <Label for="generated-apps">Generated apps output</Label>
                        <div class="flex gap-2">
                          <Skeleton v-if="settingsLoading || settingsPickerField === 'generatedAppsPath'" class="h-9 w-full" />
                          <Input v-else id="generated-apps" v-model="settingsForm.generatedAppsPath" />
                          <Button type="button" variant="outline" size="icon" :disabled="settingsLoading || settingsPickerField !== null" @click="chooseSaveFile('generatedAppsPath')">
                            <Save class="size-4" />
                            <span class="sr-only">Choose generated apps output</span>
                          </Button>
                        </div>
                      </div>
                    </CardContent>
                  </Card>

                  <Card>
                    <CardHeader>
                      <CardTitle class="text-sm">Storage</CardTitle>
                    </CardHeader>
                    <CardContent class="grid gap-3">
                      <div class="grid gap-2">
                        <Label for="workflow-db">Workflow SQLite database</Label>
                        <div class="flex gap-2">
                          <Skeleton v-if="settingsLoading || settingsPickerField === 'workflowDbPath'" class="h-9 w-full" />
                          <Input v-else id="workflow-db" v-model="settingsForm.workflowDbPath" data-testid="settings-workflow-db" />
                          <Button type="button" variant="outline" size="icon" :disabled="settingsLoading || settingsPickerField !== null" @click="chooseSaveFile('workflowDbPath')">
                            <Database class="size-4" />
                            <span class="sr-only">Choose workflow database</span>
                          </Button>
                        </div>
                      </div>
                    </CardContent>
                  </Card>

	                  <Card>
	                    <CardHeader>
	                      <CardTitle class="text-sm">Operations</CardTitle>
	                    </CardHeader>
	                    <CardContent class="grid gap-3">
                      <div class="grid gap-2">
                        <Label for="archive-root">Archive root</Label>
                        <div class="flex gap-2">
                          <Skeleton v-if="settingsLoading || settingsPickerField === 'archiveRoot'" class="h-9 w-full" />
                          <Input v-else id="archive-root" v-model="settingsForm.archiveRoot" />
                          <Button type="button" variant="outline" size="icon" :disabled="settingsLoading || settingsPickerField !== null" @click="chooseDirectory('archiveRoot')">
                            <FolderOpen class="size-4" />
                            <span class="sr-only">Choose archive root</span>
	                          </Button>
	                        </div>
	                      </div>
	                    </CardContent>
	                  </Card>

	                  <Card>
	                    <CardHeader>
	                      <CardTitle class="text-sm">1Password</CardTitle>
	                    </CardHeader>
	                    <CardContent class="grid gap-3">
	                      <div v-if="opVaultsError" class="flex items-start justify-between gap-2 rounded-md border border-destructive/40 bg-destructive/10 px-3 py-2 text-xs text-destructive">
	                        <div class="flex items-start gap-2">
	                          <AlertTriangle class="mt-0.5 size-4 shrink-0" />
	                          <span>{{ opVaultsError }}</span>
                        </div>
                        <div class="flex shrink-0 items-center gap-1">
                          <Button type="button" variant="ghost" size="sm" :disabled="opInstallLoading" @click="installOpDependencies">
                            <Loader2 v-if="opInstallLoading" class="size-3.5 animate-spin" />
                            <Download v-else class="size-3.5" />
                            Install
                          </Button>
                          <Button type="button" variant="ghost" size="sm" :disabled="opSigninLoading" @click="signinOpCli">
                            <Loader2 v-if="opSigninLoading" class="size-3.5 animate-spin" />
                            <KeyRound v-else class="size-3.5" />
                            Sign in
                          </Button>
                          <Button type="button" variant="ghost" size="sm" :disabled="opVaultsLoading" @click="loadOpVaults">
                            <Loader2 v-if="opVaultsLoading" class="size-3.5 animate-spin" />
                            <RefreshCw v-else class="size-3.5" />
                            Retry
                          </Button>
                        </div>
                      </div>
                      <div class="grid grid-cols-2 gap-3">
                        <div class="grid gap-2">
                          <Label for="op-vault">1Password vault</Label>
                          <Skeleton v-if="settingsLoading || opVaultsLoading" class="h-9 w-full" />
                          <div v-else class="flex gap-2">
                            <Select :model-value="settingsForm.opVault" :disabled="opVaultsError !== ''" @update:model-value="onOpVaultChange">
                              <SelectTrigger id="op-vault" class="flex-1" data-testid="settings-op-vault">
                                <SelectValue placeholder="Select a vault" />
                              </SelectTrigger>
                              <SelectContent>
                                <SelectItem v-for="option in opVaultOptions" :key="option.value" :value="option.value">
                                  {{ option.label }}
                                </SelectItem>
                              </SelectContent>
                            </Select>
                            <Button type="button" variant="outline" size="icon" :disabled="opVaultsLoading" @click="loadOpVaults">
                              <RefreshCw class="size-4" />
                              <span class="sr-only">Refresh vaults</span>
                            </Button>
                          </div>
                        </div>
                        <div class="grid gap-2">
                          <Label for="op-item">1Password item</Label>
                          <Skeleton v-if="settingsLoading || opItemsLoading" class="h-9 w-full" />
                          <Select v-else :model-value="settingsForm.opItem" :disabled="opItemSelectDisabled" @update:model-value="onOpItemChange">
                            <SelectTrigger id="op-item" data-testid="settings-op-item">
                              <SelectValue :placeholder="settingsForm.opVault ? 'Select an item' : 'Pick a vault first'" />
                            </SelectTrigger>
                            <SelectContent>
                              <SelectItem v-for="option in opItemOptions" :key="option.value" :value="option.value">
                                {{ option.label }}
                              </SelectItem>
                            </SelectContent>
                          </Select>
	                          <p v-if="opItemsError && opItemsLoadedFor === settingsForm.opVault" class="text-xs text-destructive">{{ opItemsError }}</p>
	                        </div>
	                      </div>
	                      <div class="overflow-hidden rounded-lg border border-section-border bg-section-muted">
	                        <div class="border-b border-section-border px-3 py-2 text-xs font-medium text-muted-foreground">Saved fields</div>
	                        <div
	                          v-for="field in opSavedFields"
	                          :key="field.key"
	                          class="grid grid-cols-[8rem_minmax(0,1fr)_minmax(0,1fr)] gap-3 border-b border-section-border px-3 py-2 text-xs last:border-b-0"
	                        >
	                          <div class="font-medium">{{ field.label }}</div>
	                          <div class="min-w-0">
	                            <div class="text-muted-foreground">Saved</div>
	                            <div class="truncate">{{ field.saved || "not set" }}</div>
	                          </div>
	                          <div class="min-w-0">
	                            <div class="text-muted-foreground">Pending</div>
	                            <div class="truncate">{{ field.pending || "not set" }}</div>
	                          </div>
	                        </div>
	                      </div>
	                    </CardContent>
	                  </Card>

	                  <Card>
                    <CardHeader class="flex flex-row items-center justify-between gap-3 space-y-0">
                      <CardTitle class="text-sm">Validation</CardTitle>
                      <Button type="button" variant="ghost" size="sm" :disabled="settingsLoading || settingsValidating || settingsSaving" @click="validateSettings">
                        <Loader2 v-if="settingsValidating" class="size-4 animate-spin" />
                        Validate
                      </Button>
                    </CardHeader>
                    <CardContent class="grid gap-3">
                      <div class="overflow-hidden rounded-lg border border-section-border bg-section-muted">
                        <template v-if="settingsLoading || settingsValidating || settingsSaving">
                          <div
                            v-for="i in 4"
                            :key="`skeleton-check-${i}`"
                            class="grid gap-2 border-b px-3 py-3 last:border-b-0"
                            data-testid="settings-checks-skeleton"
                          >
                            <div class="flex items-center gap-2">
                              <Skeleton class="h-4 w-32" />
                              <Skeleton class="ml-auto h-5 w-12" />
                            </div>
                            <Skeleton class="h-3 w-3/4" />
                          </div>
                        </template>
                        <Table v-else>
                          <TableHeader>
                            <TableRow>
                              <TableHead>Check</TableHead>
                              <TableHead>Path</TableHead>
                              <TableHead class="w-24 text-right">Status</TableHead>
                            </TableRow>
                          </TableHeader>
                          <TableBody>
                            <TableRow v-for="check in settingsChecks" :key="check.key">
                              <TableCell>
                                <div class="font-medium">{{ check.label }}</div>
                                <div v-if="check.message && check.message !== 'ok'" class="mt-1 text-xs text-destructive">{{ check.message }}</div>
                              </TableCell>
                              <TableCell class="max-w-0 truncate text-xs text-muted-foreground">{{ check.path }}</TableCell>
                              <TableCell class="text-right">
                                <StatusBadge :status="check.status" />
                              </TableCell>
                            </TableRow>
                          </TableBody>
                        </Table>
                      </div>
                      <div v-if="settingsError" class="rounded-lg border border-destructive/40 bg-section-muted p-3 text-sm text-destructive">
                        {{ settingsError }}
                      </div>
                      <div v-if="settingsMessage" class="rounded-lg border border-section-border bg-section-muted p-3 text-sm text-muted-foreground">
                        {{ settingsMessage }}
                      </div>
                    </CardContent>
                  </Card>
                </div>
              </ScrollArea>
              <Separator />
              <div class="flex items-center gap-2 border-t border-section-border bg-section p-4">
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
              <div class="flex items-start bg-section p-4">
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

    <ToastViewport :toasts="toasts" @dismiss="dismissToast" />

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
