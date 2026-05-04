import grpc from "@grpc/grpc-js";
import protoLoader from "@grpc/proto-loader";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

export const workflowProtoPath = join(dirname(fileURLToPath(import.meta.url)), "proto", "workflow.proto");

const definition = protoLoader.loadSync(workflowProtoPath, {
  defaults: true,
  enums: String,
  keepCase: false,
  longs: Number,
  oneofs: true,
});

const descriptor = grpc.loadPackageDefinition(definition);
const WorkflowBridge = descriptor.macos.bridge.v1.WorkflowBridge;

export function unixTarget(socketPath) {
  return `unix://${socketPath}`;
}

export function createWorkflowBridgeClient(target) {
  return new WorkflowBridge(target, grpc.credentials.createInsecure());
}

export function waitForReady(client, timeoutMs = 10000) {
  return new Promise((resolve, reject) => {
    client.waitForReady(Date.now() + timeoutMs, (error) => {
      if (error) {
        reject(error);
        return;
      }

      resolve();
    });
  });
}
