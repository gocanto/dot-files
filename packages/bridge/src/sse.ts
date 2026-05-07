import { EventEmitter } from "node:events";
import { request as httpRequest } from "node:http";
import { consumeBody } from "#bridge/http.js";
import type {
  RunWorkflowRequest,
  WorkflowEvent,
  WorkflowRunEndInfo,
  WorkflowRunStream,
} from "#bridge/types.js";

interface ErrorFramePayload {
  message?: unknown;
}

export function runWorkflowStream(
  socketPath: string,
  request: RunWorkflowRequest,
): WorkflowRunStream {
  const stream = new EventEmitter() as WorkflowRunStream;
  const req = httpRequest({
    socketPath,
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

    res.on("data", (chunk: string) => {
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

function handleFrame(stream: WorkflowRunStream, frame: string): void {
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

  let payload: unknown;

  try {
    payload = JSON.parse(data) as unknown;
  } catch (error) {
    stream.emit("error", error instanceof Error ? error : new Error(String(error)));

    return;
  }

  if (event === "workflow") {
    stream.emit("data", payload as WorkflowEvent);

    return;
  }

  if (event === "end") {
    stream.emit("end-info", payload as WorkflowRunEndInfo);

    return;
  }

  if (event === "error") {
    const errorPayload = payload as ErrorFramePayload | null;
    const message =
      errorPayload && typeof errorPayload.message === "string"
        ? errorPayload.message
        : "workflow run error";

    stream.emit("error", new Error(message));
  }
}
