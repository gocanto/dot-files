import { type IncomingMessage } from "node:http";
export type HttpMethod = "GET" | "POST" | "PUT";
export type JsonBody = Record<string, unknown>;
export interface BridgeError extends Error {
  statusCode?: number;
  code?: string;
}
export declare function requestJson<Response>(
  socketPath: string,
  method: HttpMethod,
  path: string,
  body?: JsonBody,
  timeoutMs?: number,
): Promise<Response>;
export declare function consumeBody(res: IncomingMessage): Promise<string>;
