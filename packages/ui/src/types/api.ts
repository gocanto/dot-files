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

export interface MacOSApi {
  workflows(): Promise<Workflow[]>;
  runWorkflow(request: RunRequest, onEvent: (event: RunEvent) => void): Promise<{ exitCode: number }>;
  runs(limit?: number): Promise<RunSummary[]>;
  runLog(runId: string): Promise<RunLog>;
}

declare global {
  interface Window {
    macOS: MacOSApi;
  }
}
