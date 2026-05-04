import type { Client, ClientReadableStream, ServiceError } from "@grpc/grpc-js";

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

export interface WorkflowBridgeClient extends Client {
  listWorkflows(
    request: Record<string, never>,
    callback: (error: ServiceError | null, response: { workflows: Workflow[] }) => void,
  ): void;
  runWorkflow(request: RunWorkflowRequest): ClientReadableStream<WorkflowEvent>;
  listRuns(request: { limit: number }, callback: (error: ServiceError | null, response: { runs: RunSummary[] }) => void): void;
  runLog(request: { runId: string }, callback: (error: ServiceError | null, response: RunLog) => void): void;
  getSettings(
    request: Record<string, never>,
    callback: (error: ServiceError | null, response: SettingsResponse) => void,
  ): void;
  validateSettings(
    request: { settings: RuntimeSettings },
    callback: (error: ServiceError | null, response: SettingsResponse) => void,
  ): void;
}

export const workflowProtoPath: string;
export function unixTarget(socketPath: string): string;
export function createWorkflowBridgeClient(target: string): WorkflowBridgeClient;
export function waitForReady(client: Client, timeoutMs?: number): Promise<void>;
