import type { EventEmitter } from "node:events";

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
  phases: Phase[];
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

export interface RunWorkflowRequest {
  workflowId: string;
  confirmationOptionId: string;
  enabledPhaseIds: string[];
}

export interface WorkflowEvent {
  id?: number;
  runId: string;
  seq: number;
  type: string;
  phaseId?: string;
  phaseName?: string;
  status?: string;
  message?: string;
  createdAt?: string;
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
  run?: RunSummary;
  events: WorkflowEvent[];
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
  settings?: RuntimeSettings;
  checks: SettingsCheck[];
  valid: boolean;
}

export interface UserPreferencesResponse {
  theme: string;
  updatedAt: string;
}

export interface OpVault {
  id: string;
  name: string;
}

export interface OpItem {
  id: string;
  title: string;
}

export interface OpUnavailableError extends Error {
  code: "op_unavailable";
}

export interface UnixTarget {
  socketPath: string;
}

export interface WorkflowRunStream extends EventEmitter {
  on(event: "data", listener: (event: WorkflowEvent) => void): this;
  on(event: "end", listener: () => void): this;
  on(
    event: "end-info",
    listener: (info: { exitCode: number; status: string; message?: string }) => void,
  ): this;
  on(event: "error", listener: (error: Error) => void): this;
}

export interface WorkflowBridgeClient {
  close(): void;
  healthz(): Promise<void>;
  listWorkflows(): Promise<{ workflows: Workflow[] }>;
  listRuns(request: { limit: number }): Promise<{ runs: RunSummary[] }>;
  runLog(request: { runId: string }): Promise<RunLog>;
  listTemplateFiles(): Promise<{ files: TemplateFileSummary[] }>;
  readTemplateFile(request: { path: string }): Promise<TemplateFileContent>;
  saveTemplateFile(request: { path: string; content: string }): Promise<TemplateFileContent>;
  getSettings(): Promise<SettingsResponse>;
  validateSettings(request: { settings: RuntimeSettings }): Promise<SettingsResponse>;
  getUserPreferences(): Promise<UserPreferencesResponse>;
  saveUserPreferences(request: { theme: string }): Promise<UserPreferencesResponse>;
  listOpVaults(): Promise<{ vaults: OpVault[] }>;
  listOpItems(request: { vault: string }): Promise<{ items: OpItem[] }>;
  runWorkflow(request: RunWorkflowRequest): WorkflowRunStream;
}

export function unixTarget(socketPath: string): UnixTarget;
export function createWorkflowBridgeClient(target: UnixTarget): WorkflowBridgeClient;
export function waitForReady(client: WorkflowBridgeClient, timeoutMs?: number): Promise<void>;
