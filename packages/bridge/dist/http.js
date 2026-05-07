import { request as httpRequest } from "node:http";
const DEFAULT_REQUEST_TIMEOUT_MS = 30000;
export function requestJson(
  socketPath,
  method,
  path,
  body,
  timeoutMs = DEFAULT_REQUEST_TIMEOUT_MS,
) {
  return new Promise((resolve, reject) => {
    let settled = false;
    const headers = { Accept: "application/json" };
    const payload = body === undefined ? null : JSON.stringify(body);
    if (payload !== null) {
      headers["Content-Type"] = "application/json";
      headers["Content-Length"] = Buffer.byteLength(payload);
    }
    const fail = (error) => {
      if (settled) {
        return;
      }
      settled = true;
      reject(error);
    };
    const succeed = (value) => {
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
              succeed(JSON.parse(raw));
            } catch (error) {
              fail(error instanceof Error ? error : new Error(String(error)));
            }
            return;
          }
          if (res.statusCode === 200) {
            succeed(undefined);
            return;
          }
          const error = new Error(`${method} ${path} failed (${res.statusCode}): ${raw}`);
          const bridgeError = error;
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
export function consumeBody(res) {
  return new Promise((resolve, reject) => {
    res.setEncoding("utf8");
    let body = "";
    res.on("data", (chunk) => {
      body += chunk;
    });
    res.on("end", () => resolve(body));
    res.on("error", reject);
  });
}
function isJsonResponse(res) {
  return (res.headers["content-type"] ?? "").includes("application/json");
}
function applyJsonErrorPayload(error, raw) {
  try {
    const parsed = JSON.parse(raw);
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
