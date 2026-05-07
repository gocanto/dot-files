import type { RunWorkflowRequest, WorkflowRunStream } from "#bridge/types.js";
export declare function runWorkflowStream(
  socketPath: string,
  request: RunWorkflowRequest,
): WorkflowRunStream;
