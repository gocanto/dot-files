import { requestJson } from "#bridge/http.js";
import { runWorkflowStream } from "#bridge/sse.js";
import type {
  RunLog,
  RunSummary,
  RunWorkflowRequest,
  RuntimeSettings,
  SettingsResponse,
  TemplateFileContent,
  TemplateFileSummary,
  UnixTarget,
  UserPreferencesResponse,
  Workflow,
  WorkflowBridgeClient,
  WorkflowRunStream,
  OpItem,
  OpVault,
} from "#bridge/types.js";

type ListRunsRequest = { limit?: number };

export function unixTarget(socketPath: string): UnixTarget {
  return { socketPath };
}

export function createWorkflowBridgeClient(target: UnixTarget): WorkflowBridgeClient {
  return new HttpWorkflowBridgeClient(target);
}

export function waitForReady(client: WorkflowBridgeClient, timeoutMs = 10000): Promise<void> {
  const deadline = Date.now() + timeoutMs;

  return new Promise<void>((resolve, reject) => {
    const attempt = (): void => {
      client
        .healthz()
        .then(resolve)
        .catch((error: unknown) => {
          if (Date.now() >= deadline) {
            reject(error);

            return;
          }

          setTimeout(attempt, 100);
        });
    };

    attempt();
  });
}

class HttpWorkflowBridgeClient implements WorkflowBridgeClient {
  private readonly socketPath: string;

  constructor(target: UnixTarget) {
    this.socketPath = target.socketPath;
  }

  close(): void {}

  healthz(): Promise<void> {
    return this.request<void>("GET", "/v1/healthz");
  }

  listWorkflows(): Promise<{ workflows: Workflow[] }> {
    return this.request<{ workflows: Workflow[] }>("GET", "/v1/workflows");
  }

  listRuns(request: ListRunsRequest = {}): Promise<{ runs: RunSummary[] }> {
    const query = typeof request.limit === "number" ? `?limit=${request.limit}` : "";

    return this.request<{ runs: RunSummary[] }>("GET", `/v1/runs${query}`);
  }

  runLog(request: { runId: string }): Promise<RunLog> {
    return this.request<RunLog>("GET", `/v1/runs/${encodeURIComponent(request.runId)}/log`);
  }

  listTemplateFiles(): Promise<{ files: TemplateFileSummary[] }> {
    return this.request<{ files: TemplateFileSummary[] }>("GET", "/v1/template-files");
  }

  readTemplateFile(request: { path: string }): Promise<TemplateFileContent> {
    return this.request<TemplateFileContent>(
      "GET",
      `/v1/template-files/content?path=${encodeURIComponent(request.path)}`,
    );
  }

  saveTemplateFile(request: { path: string; content: string }): Promise<TemplateFileContent> {
    return this.request<TemplateFileContent>("PUT", "/v1/template-files/content", {
      path: request.path,
      content: request.content,
    });
  }

  getSettings(): Promise<SettingsResponse> {
    return this.request<SettingsResponse>("GET", "/v1/settings");
  }

  validateSettings(request: { settings: RuntimeSettings }): Promise<SettingsResponse> {
    return this.request<SettingsResponse>("POST", "/v1/settings/validate", {
      settings: request.settings,
    });
  }

  getUserPreferences(): Promise<UserPreferencesResponse> {
    return this.request<UserPreferencesResponse>("GET", "/v1/preferences");
  }

  saveUserPreferences(request: { theme: string }): Promise<UserPreferencesResponse> {
    return this.request<UserPreferencesResponse>("POST", "/v1/preferences", {
      theme: request.theme,
    });
  }

  listOpVaults(): Promise<{ vaults: OpVault[] }> {
    return this.request<{ vaults: OpVault[] }>("GET", "/v1/onepassword/vaults");
  }

  listOpItems(request: { vault: string }): Promise<{ items: OpItem[] }> {
    return this.request<{ items: OpItem[] }>(
      "GET",
      `/v1/onepassword/items?vault=${encodeURIComponent(request.vault)}`,
    );
  }

  runWorkflow(request: RunWorkflowRequest): WorkflowRunStream {
    return runWorkflowStream(this.socketPath, request);
  }

  private request<Response>(
    method: "GET" | "POST" | "PUT",
    path: string,
    body?: Record<string, unknown>,
  ): Promise<Response> {
    return requestJson<Response>(this.socketPath, method, path, body);
  }
}
