import { request as httpRequest, type IncomingMessage } from "node:http";
import { request as httpsRequest } from "node:https";

export function requestOk(url: string): Promise<void> {
  return new Promise((resolveRequest, rejectRequest) => {
    const onResponse = (res: IncomingMessage) => {
      res.resume();
      res.on("end", () => {
        if (res.statusCode && res.statusCode >= 200 && res.statusCode < 500) {
          resolveRequest();
          return;
        }

        rejectRequest(new Error(`unexpected status ${res.statusCode}`));
      });
    };

    const req = url.startsWith("https:")
      ? httpsRequest(url, { rejectUnauthorized: false }, onResponse)
      : httpRequest(url, onResponse);

    req.setTimeout(2_000, () => {
      req.destroy(new Error(`timed out waiting for ${url}`));
    });
    req.on("error", rejectRequest);
    req.end();
  });
}
