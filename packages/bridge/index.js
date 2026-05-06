import { EventEmitter } from "node:events";
import { request as httpRequest } from "node:http";

export function unixTarget(socketPath) {
  return { socketPath };
}

export function createWorkflowBridgeClient(target) {
  return new WorkflowBridgeClient(target);
}

export function waitForReady(client, timeoutMs = 10000) {
  const deadline = Date.now() + timeoutMs;

  return new Promise((resolve, reject) => {
    const attempt = () => {
      client
        .healthz()
        .then(resolve)
        .catch((error) => {
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

class WorkflowBridgeClient {
  constructor(target) {
    this.socketPath = target.socketPath;
  }

  close() {}

  healthz() {
    return this.request("GET", "/v1/healthz");
  }

  listWorkflows() {
    return this.request("GET", "/v1/workflows");
  }

  listRuns({ limit } = {}) {
    const query = typeof limit === "number" ? `?limit=${limit}` : "";

    return this.request("GET", `/v1/runs${query}`);
  }

  runLog({ runId }) {
    return this.request("GET", `/v1/runs/${encodeURIComponent(runId)}/log`);
  }

  listTemplateFiles() {
    return this.request("GET", "/v1/template-files");
  }

  readTemplateFile({ path }) {
    return this.request("GET", `/v1/template-files/content?path=${encodeURIComponent(path)}`);
  }

  saveTemplateFile({ path, content }) {
    return this.request("PUT", "/v1/template-files/content", { path, content });
  }

  getSettings() {
    return this.request("GET", "/v1/settings");
  }

  validateSettings({ settings }) {
    return this.request("POST", "/v1/settings/validate", { settings });
  }

  getUserPreferences() {
    return this.request("GET", "/v1/preferences");
  }

  saveUserPreferences({ theme }) {
    return this.request("POST", "/v1/preferences", { theme });
  }

  listOpVaults() {
    return this.request("GET", "/v1/onepassword/vaults");
  }

  listOpItems({ vault }) {
    return this.request("GET", `/v1/onepassword/items?vault=${encodeURIComponent(vault)}`);
  }

  runWorkflow(request) {
    const stream = new EventEmitter();
    const req = httpRequest({
      socketPath: this.socketPath,
      method: "POST",
      path: "/v1/workflows/run",
      headers: {
        "Content-Type": "application/json",
        Accept: "text/event-stream",
      },
    });

    req.on("error", (error) => stream.emit("error", error));

    req.on("response", (res) => {
      if (res.statusCode !== 200) {
        consumeBody(res).then((body) => {
          stream.emit("error", new Error(`workflow run failed (${res.statusCode}): ${body}`));
        });

        return;
      }

      res.setEncoding("utf8");
      let buffer = "";

      res.on("data", (chunk) => {
        buffer += chunk;

        let separator = buffer.indexOf("\n\n");

        while (separator !== -1) {
          const frame = buffer.slice(0, separator);
          buffer = buffer.slice(separator + 2);
          handleFrame(stream, frame);
          separator = buffer.indexOf("\n\n");
        }
      });

      res.on("end", () => stream.emit("end"));
      res.on("error", (error) => stream.emit("error", error));
    });

    req.end(JSON.stringify(request));

    return stream;
  }

  request(method, path, body) {
    return new Promise((resolve, reject) => {
      const headers = { Accept: "application/json" };
      const payload = body === undefined ? null : JSON.stringify(body);

      if (payload !== null) {
        headers["Content-Type"] = "application/json";
        headers["Content-Length"] = Buffer.byteLength(payload);
      }

      const req = httpRequest({ socketPath: this.socketPath, method, path, headers }, (res) => {
        consumeBody(res)
          .then((raw) => {
            if (
              res.statusCode === 200 &&
              (res.headers["content-type"] ?? "").includes("application/json")
            ) {
              try {
                resolve(JSON.parse(raw));
              } catch (error) {
                reject(error);
              }

              return;
            }

            if (res.statusCode === 200) {
              resolve(undefined);

              return;
            }

            const error = new Error(`${method} ${path} failed (${res.statusCode}): ${raw}`);
            error.statusCode = res.statusCode;

            if ((res.headers["content-type"] ?? "").includes("application/json")) {
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
                // Fall through with the generic error message
              }
            }

            reject(error);
          })
          .catch(reject);
      });

      req.on("error", reject);

      if (payload !== null) {
        req.write(payload);
      }

      req.end();
    });
  }
}

function handleFrame(stream, frame) {
  let event = "";
  let data = "";

  for (const line of frame.split("\n")) {
    if (line.startsWith("event: ")) {
      event = line.slice(7);
    } else if (line.startsWith("data: ")) {
      data = data === "" ? line.slice(6) : `${data}\n${line.slice(6)}`;
    }
  }

  if (data === "") {
    return;
  }

  let payload;

  try {
    payload = JSON.parse(data);
  } catch (error) {
    stream.emit("error", error);

    return;
  }

  if (event === "workflow") {
    stream.emit("data", payload);

    return;
  }

  if (event === "end") {
    stream.emit("end-info", payload);

    return;
  }

  if (event === "error") {
    stream.emit("error", new Error(payload?.message ?? "workflow run error"));
  }
}

function consumeBody(res) {
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
