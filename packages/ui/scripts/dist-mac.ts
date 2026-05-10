import { spawn } from "node:child_process";

const mode = process.argv[2] === "signed" ? "signed" : "unsigned";
const args = ["exec", "electron-builder", "--mac", "dmg", "zip", "--arm64", "--publish", "never"];

if (mode === "unsigned") {
  args.push("-c.mac.identity=-");
}

const child = spawn("pnpm", args, {
  env: {
    ...process.env,
    CSC_IDENTITY_AUTO_DISCOVERY:
      mode === "unsigned" ? "false" : process.env.CSC_IDENTITY_AUTO_DISCOVERY,
    NODE_OPTIONS: [process.env.NODE_OPTIONS, "--no-deprecation"].filter(Boolean).join(" "),
  },
  stdio: ["ignore", "pipe", "pipe"],
});

let pendingStdout = "";
let pendingStderr = "";

child.stdout.on("data", (chunk: Buffer) => {
  pendingStdout = writeFiltered(process.stdout, pendingStdout + chunk.toString("utf8"));
});

child.stderr.on("data", (chunk: Buffer) => {
  pendingStderr = writeFiltered(process.stderr, pendingStderr + chunk.toString("utf8"));
});

child.on("error", (error) => {
  console.error(error.message);
  process.exitCode = 1;
});

child.on("close", (code, signal) => {
  flush(process.stdout, pendingStdout);
  flush(process.stderr, pendingStderr);

  if (signal) {
    console.error(`electron-builder exited with signal ${signal}`);
    process.exitCode = 1;
    return;
  }

  process.exitCode = code ?? 1;
});

function writeFiltered(stream: NodeJS.WriteStream, value: string) {
  const lines = value.split(/\r?\n/u);
  const rest = lines.pop() ?? "";

  for (const line of lines) {
    if (!isSuppressedElectronBuilderWarning(line)) {
      stream.write(`${line}\n`);
    }
  }

  return rest;
}

function flush(stream: NodeJS.WriteStream, value: string) {
  if (value && !isSuppressedElectronBuilderWarning(value)) {
    stream.write(value);
  }
}

function isSuppressedElectronBuilderWarning(line: string) {
  return line.includes("dependency not found on disk");
}
