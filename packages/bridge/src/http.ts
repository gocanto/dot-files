import { type IncomingMessage, request as httpRequest } from "node:http";

const DEFAULT_REQUEST_TIMEOUT_MS = 30000;

export type HttpMethod = "GET" | "POST" | "PUT";
export type JsonBody = Record<string, unknown>;

export interface BridgeError extends Error {
  statusCode?: number;
  code?: string;
}

interface JsonErrorPayload {
  error?: unknown;
  code?: unknown;
}

export function requestJson<Response>(
  socketPath: string,
  method: HttpMethod,
  path: string,
  body?: JsonBody,
  timeoutMs = DEFAULT_REQUEST_TIMEOUT_MS,
): Promise<Response> {
  return new Promise<Response>((resolve, reject) => {
    let settled = false;
    const headers: Record<string, string | number> = { Accept: "application/json" };
    const payload = body === undefined ? null : JSON.stringify(body);

    if (payload !== null) {
      headers["Content-Type"] = "application/json";
      headers["Content-Length"] = Buffer.byteLength(payload);
    }

    const fail = (error: Error): void => {
      if (settled) {
        return;
      }

      settled = true;
      reject(error);
    };

    const succeed = (value: Response): void => {
      if (settled) {
        return;
      }

      settled = true;
      resolve(value);
    };

    const req = httpRequest({ socketPath, method, path, headers }, (res) => {
      consumeBody(res)
        .then((raw) => {
          if (res.statusCode === 200 && isJsonResponse(res)) {
            try {
              succeed(JSON.parse(raw) as Response);
            } catch (error) {
              fail(error instanceof Error ? error : new Error(String(error)));
            }

            return;
          }

          if (res.statusCode === 200) {
            succeed(undefined as Response);

            return;
          }

          const error = new Error(`${method} ${path} failed (${res.statusCode}): ${raw}`);
          const bridgeError = error as BridgeError;
          bridgeError.statusCode = res.statusCode;

          if (isJsonResponse(res)) {
            applyJsonErrorPayload(bridgeError, raw);
          }

          fail(bridgeError);
        })
        .catch(fail);
    });

    req.setTimeout(timeoutMs, () => {
      req.destroy(new Error(`${method} ${path} timed out after ${timeoutMs}ms`));
    });

    req.on("error", fail);

    if (payload !== null) {
      req.write(payload);
    }

    req.end();
  });
}

export function consumeBody(res: IncomingMessage): Promise<string> {
  return new Promise<string>((resolve, reject) => {
    res.setEncoding("utf8");
    let body = "";

    res.on("data", (chunk: string) => {
      body += chunk;
    });

    res.on("end", () => resolve(body));
    res.on("error", reject);
  });
}

function isJsonResponse(res: IncomingMessage): boolean {
  return (res.headers["content-type"] ?? "").includes("application/json");
}

function applyJsonErrorPayload(error: BridgeError, raw: string): void {
  try {
    const parsed = JSON.parse(raw) as JsonErrorPayload | null;

    if (parsed && typeof parsed === "object") {
      if (typeof parsed.error === "string") {
        error.message = parsed.error;
      }

      if (typeof parsed.code === "string") {
        error.code = parsed.code;
      }
    }
  } catch {
    // Fall through with the generic error message.
  }
}
