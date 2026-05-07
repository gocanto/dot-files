import { spawn } from "node:child_process";

export function openTerminalCommand(
  command: string,
): Promise<{ ok: true } | { ok: false; message: string }> {
  return new Promise((resolveResult) => {
    const script = [
      'tell application "Terminal"',
      "  activate",
      `  do script "${appleScriptString(command)}"`,
      "end tell",
    ].join("\n");
    const child = spawn("/usr/bin/osascript", ["-e", script], {
      stdio: ["ignore", "ignore", "pipe"],
    });
    let stderr = "";

    child.stderr?.on("data", (chunk: Buffer) => {
      stderr += chunk.toString("utf8");
    });

    child.on("error", (error) => {
      resolveResult({ ok: false, message: error.message });
    });

    child.on("exit", (code) => {
      if (code === 0) {
        resolveResult({ ok: true });

        return;
      }

      resolveResult({
        ok: false,
        message: stderr.trim() || `osascript exited with code ${code ?? "unknown"}`,
      });
    });
  });
}

function appleScriptString(value: string): string {
  return value.replaceAll("\\", "\\\\").replaceAll('"', '\\"');
}
