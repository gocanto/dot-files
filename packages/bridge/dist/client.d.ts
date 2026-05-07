import type { UnixTarget, WorkflowBridgeClient } from "#bridge/types.js";
export declare function unixTarget(socketPath: string): UnixTarget;
export declare function createWorkflowBridgeClient(target: UnixTarget): WorkflowBridgeClient;
export declare function waitForReady(
  client: WorkflowBridgeClient,
  timeoutMs?: number,
): Promise<void>;
