export interface Phase {
  id: string;
  name: string;
  enabled: boolean;
}

export interface ConfirmationOption {
  id: string;
  label: string;
  description: string;
  continue: boolean;
  back: boolean;
  requiresApproval: boolean;
  phases?: Phase[];
}

export interface Workflow {
  id: string;
  name: string;
  description: string;
  changesMac: string;
  phases: Phase[];
  confirmation?: {
    title: string;
    message: string;
    options: ConfirmationOption[];
  };
}

export interface RunRequest {
  workflowId: string;
  confirmationOptionId: string;
  enabledPhaseIds: string[];
}

export interface RunEvent {
  runId: string;
  seq: number;
  type: string;
  phaseId?: string;
  phaseName?: string;
  status?: string;
  message?: string;
}

export interface RunSummary {
  id: string;
  workflowId: string;
  workflowName: string;
  confirmationOptionId: string;
  confirmationOptionLabel: string;
  mode: string;
  status: string;
  startedAt: string;
  completedAt?: string;
  errorMessage?: string;
}

export interface RunLog {
  run: RunSummary;
  events: Array<RunEvent & { id: number; createdAt: string }>;
}

export interface TemplateFileSummary {
  path: string;
  relative: string;
  kind: string;
  size: number;
  modifiedAt?: string;
  exists: boolean;
}

export interface TemplateFileContent {
  file: TemplateFileSummary;
  content: string;
}

export interface RuntimeSettings {
  repoRoot: string;
  appsConfigPath: string;
  secretsConfigPath: string;
  generatedAppsPath: string;
  archiveRoot: string;
  workflowDbPath: string;
  opVault: string;
  opItem: string;
}

export interface SettingsCheck {
  key: string;
  label: string;
  path: string;
  status: string;
  message: string;
}

export interface SettingsResponse {
  settings: RuntimeSettings;
  checks: SettingsCheck[];
  valid: boolean;
}

export interface UserPreferences {
  theme: string;
  updatedAt?: string;
}

export interface AppDiagnostic {
  id: string;
  level: "info" | "warning" | "error";
  source: string;
  message: string;
  details?: string;
  createdAt: string;
}

export interface MacSystemInfo {
  name: string;
  hostname: string;
  osLabel: string;
  architectureLabel: string;
  avatarUrl?: string;
}

export interface OpVault {
  id: string;
  name: string;
}

export interface OpItem {
  id: string;
  title: string;
}

export type OpVaultsResult =
  | { ok: true; vaults: OpVault[] }
  | { ok: false; code: string; message: string };
export type OpItemsResult =
  | { ok: true; items: OpItem[] }
  | { ok: false; code: string; message: string };
export type OpSigninResult = { ok: true } | { ok: false; message: string };
export type OpInstallResult = { ok: true } | { ok: false; message: string };

export interface MacOSApi {
  workflows(): Promise<Workflow[]>;
  runWorkflow(
    request: RunRequest,
    onEvent: (event: RunEvent) => void,
  ): Promise<{ exitCode: number }>;
  runs(limit?: number): Promise<RunSummary[]>;
  runLog(runId: string): Promise<RunLog>;
  templateFiles(): Promise<TemplateFileSummary[]>;
  readTemplateFile(path: string): Promise<TemplateFileContent>;
  saveTemplateFile(path: string, content: string): Promise<TemplateFileContent>;
  settings(): Promise<SettingsResponse>;
  validateSettings(settings: RuntimeSettings): Promise<SettingsResponse>;
  saveSettings(settings: RuntimeSettings): Promise<SettingsResponse>;
  chooseDirectory(defaultPath?: string): Promise<string | null>;
  chooseFile(defaultPath?: string): Promise<string | null>;
  chooseSaveFile(defaultPath?: string): Promise<string | null>;
  listOpVaults(): Promise<OpVaultsResult>;
  listOpItems(vault: string): Promise<OpItemsResult>;
  signinOpCli(): Promise<OpSigninResult>;
  installOpDependencies(): Promise<OpInstallResult>;
  openDevTools(): Promise<void>;
  appDiagnostics(): Promise<AppDiagnostic[]>;
  onAppDiagnostic(onEvent: (event: AppDiagnostic) => void): () => void;
  reportRendererError(message: string, details?: string): Promise<void>;
  macName(): Promise<string>;
  macHostname(): Promise<string>;
  macSystemInfo?(): Promise<MacSystemInfo>;
  getUserPreferences(): Promise<UserPreferences>;
  saveUserPreferences(theme: string): Promise<UserPreferences>;
}

declare global {
  interface Window {
    macOS: MacOSApi;
  }
}
