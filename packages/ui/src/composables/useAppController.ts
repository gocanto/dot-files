import {
  Database,
  Eye,
  FileText,
  FolderOpen,
  History,
  KeyRound,
  Settings,
  Wand2,
} from "lucide-vue-next";
import { computed, onMounted, ref } from "vue";
import type { SectionId, StepSettingsKey } from "@/components/app/types";
import type { ToastItem, ToastTone } from "@/components/ui/toast";
import { loadThemeFromBackend, useTheme } from "@/composables/useTheme";
import { errorMessage } from "@/lib/format";
import {
  getWorkflowDetail,
  workflowsInCategory,
  type WorkflowCategory,
} from "@/lib/workflowDetails";
import type {
  ConfirmationOption,
  OpItem,
  OpVault,
  Phase,
  RunEvent,
  RunLog,
  RunSummary,
  RuntimeSettings,
  SettingsResponse,
  Workflow,
} from "@/types/api";

export function useAppController() {
  const { theme, toggleTheme } = useTheme();

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
    {
      id: "template" as const,
      label: "Template",
      icon: FileText,
      count: workflowsLoading.value
        ? null
        : workflowsInCategory(workflows.value, "template").length,
    },
    {
      id: "current" as const,
      label: "Current state",
      icon: Eye,
      count: workflowsLoading.value ? null : workflowsInCategory(workflows.value, "current").length,
    },
    {
      id: "update" as const,
      label: "Update",
      icon: Wand2,
      count: workflowsLoading.value ? null : workflowsInCategory(workflows.value, "update").length,
    },
  ]);

  const auxNavItems = computed(() => [
    { id: "settings" as const, label: "Settings", icon: Settings, count: null as number | null },
    {
      id: "logs" as const,
      label: "Logs",
      icon: History,
      count: runsLoading.value ? null : runs.value.length,
    },
  ]);

  const stepSectionMeta: Record<
    "template" | "current" | "update",
    { title: string; emptyMessage: string; settingsKeys: StepSettingsKey[] }
  > = {
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

  const selectedWorkflow = computed(() =>
    workflows.value.find((workflow) => workflow.id === selectedWorkflowId.value),
  );
  const selectedWorkflowDetail = computed(() =>
    selectedWorkflow.value ? getWorkflowDetail(selectedWorkflow.value.id) : null,
  );
  const settingsDirty = computed(
    () =>
      JSON.stringify(settingsForm.value) !==
      JSON.stringify(settingsResponse.value?.settings ?? emptySettings()),
  );
  const settingsChecks = computed(() => settingsResponse.value?.checks ?? []);

  const opVaultOptions = computed(() => {
    const options = opVaults.value.map((vault) => ({
      value: vault.name,
      label: vault.name,
      missing: false,
    }));
    const current = settingsForm.value.opVault;

    if (current && !options.some((option) => option.value === current)) {
      options.unshift({ value: current, label: `${current} (not found)`, missing: true });
    }

    return options;
  });

  const opItemOptions = computed(() => {
    const options = opItems.value.map((item) => ({
      value: item.title,
      label: item.title,
      missing: false,
    }));
    const current = settingsForm.value.opItem;
    const vault = settingsForm.value.opVault;

    if (
      current &&
      vault &&
      opItemsLoadedFor.value === vault &&
      !options.some((option) => option.value === current)
    ) {
      options.unshift({ value: current, label: `${current} (not found)`, missing: true });
    }

    return options;
  });

  const opItemSelectDisabled = computed(() => {
    return (
      !settingsForm.value.opVault ||
      opVaultsError.value !== "" ||
      opItemsLoading.value ||
      (opItemsError.value !== "" && opItemsLoadedFor.value === settingsForm.value.opVault)
    );
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
      count: settingsChecks.value
        .filter((check) => ["repo_root", "stow"].includes(check.key))
        .filter((check) => check.status !== "ok").length,
    },
    {
      id: "manifests",
      label: "Manifests",
      icon: FileText,
      count: settingsChecks.value
        .filter((check) =>
          [
            "apps_config_path",
            "secrets_config_path",
            "generated_apps_path",
            "private_gitconfig_path",
          ].includes(check.key),
        )
        .filter((check) => check.status !== "ok").length,
    },
    {
      id: "storage",
      label: "Storage",
      icon: Database,
      count: settingsChecks.value
        .filter((check) => check.key === "workflow_db_path")
        .filter((check) => check.status !== "ok").length,
    },
    {
      id: "operations",
      label: "Operations",
      icon: KeyRound,
      count: settingsChecks.value
        .filter((check) => check.key === "archive_root")
        .filter((check) => check.status !== "ok").length,
    },
  ]);

  const displayPhases = computed(() => {
    if (!selectedWorkflow.value) {
      return [];
    }

    return selectedWorkflow.value.phases.map((phase) => ({
      ...phase,
      enabled: enabledPhaseIds.value.has(phase.id),
      status: phaseStatus(phase),
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
          [workflow.name, workflow.description].join(" ").toLowerCase().includes(query),
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

    const completed = phases.filter((phase) =>
      ["completed", "ok", "skipped"].includes(phase.status),
    ).length;

    return Math.round((completed / phases.length) * 100);
  });

  const selectedRunOutput = computed(
    () =>
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
      const [
        workflowsResult,
        runsResult,
        settingsResult,
        themeResult,
        macNameResult,
        macHostnameResult,
      ] = await Promise.allSettled([
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
      macHostname.value =
        macHostnameResult.status === "fulfilled" ? macHostnameResult.value : "local";
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
        if (
          !selectedWorkflowId.value ||
          !next.some((workflow) => workflow.id === selectedWorkflowId.value)
        ) {
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

    await Promise.allSettled([
      workflowsPromise,
      runsPromise,
      settingsPromise,
      themePromise,
      macNamePromise,
      macHostnamePromise,
    ]);
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
    enabledPhaseIds.value = new Set(
      selectedWorkflow.value?.phases.filter((phase) => phase.enabled).map((phase) => phase.id),
    );
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
    const enabledIds = phases
      .filter((phase) => enabledPhaseIds.value.has(phase.id))
      .map((phase) => phase.id);

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
    const requestedRunId = run.id;

    try {
      const nextRunLog = await window.macOS.runLog(requestedRunId);

      if (selectedRunId.value === requestedRunId) {
        selectedRunLog.value = nextRunLog;
      }
    } finally {
      if (selectedRunId.value === requestedRunId) {
        runLogLoading.value = false;
      }
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
    const previous =
      tone === "loading" ? toasts.value : toasts.value.filter((toast) => toast.tone !== "loading");

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

  function updateSetting(key: keyof RuntimeSettings, value: string) {
    settingsForm.value = { ...settingsForm.value, [key]: value };
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
    const finish = [...events]
      .reverse()
      .find((event) => event.type === "phase_finished" || event.type === "phase_skipped");

    return (
      finish?.status ??
      events.at(-1)?.status ??
      (enabledPhaseIds.value.has(phase.id) ? "pending" : "skipped")
    );
  }

  return {
    theme,
    toggleTheme,
    section,
    selectedSettingsKey,
    macName,
    macHostname,
    selectedWorkflowId,
    selectedRunId,
    selectedRunLog,
    pendingOption,
    running,
    workflowsLoading,
    runsLoading,
    initialLoading,
    runLogLoading,
    loadError,
    navCollapsed,
    searchQuery,
    logTab,
    mutedNotes,
    noteText,
    settingsResponse,
    settingsForm,
    settingsSaving,
    settingsLoading,
    settingsValidating,
    settingsPickerField,
    settingsMessage,
    settingsError,
    opVaultsLoading,
    opItemsLoading,
    opVaultsError,
    opItemsError,
    opItemsLoadedFor,
    opSigninLoading,
    opInstallLoading,
    toasts,
    stepNavItems,
    auxNavItems,
    stepMeta,
    settingsKeyLabels,
    settingsWorkflows,
    selectedWorkflow,
    selectedWorkflowDetail,
    settingsDirty,
    settingsChecks,
    opVaultOptions,
    opItemOptions,
    opItemSelectDisabled,
    opSavedFields,
    settingsGroups,
    displayPhases,
    matchingWorkflows,
    matchingRuns,
    runStatus,
    outputText,
    workflowProgress,
    selectedRunOutput,
    loadAll,
    selectSection,
    selectStepSetting,
    loadOpVaults,
    onOpVaultChange,
    onOpItemChange,
    signinOpCli,
    installOpDependencies,
    openDevTools,
    selectWorkflow,
    resetEnabledPhases,
    togglePhase,
    openConfirmation,
    updateConfirmationOpen,
    runSelected,
    openRun,
    validateSettings,
    saveSettings,
    dismissToast,
    resetSettingsForm,
    updateSetting,
    chooseDirectory,
    chooseFile,
    chooseSaveFile,
  };
}
