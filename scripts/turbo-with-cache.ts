#!/usr/bin/env tsx

import { spawnSync } from "node:child_process";
import type { Dirent, Stats } from "node:fs";
import {
  existsSync,
  lstatSync,
  mkdirSync,
  readdirSync,
  realpathSync,
  renameSync,
  rmSync,
  statSync,
} from "node:fs";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const rootDir = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const packagesDir = join(rootDir, "packages");
const turboLogCacheDir = join(rootDir, "storage", ".cache", "turbo-logs");
const args = process.argv.slice(2);

function packageDirs(): string[] {
  if (!existsSync(packagesDir)) {
    return [];
  }

  return readdirSync(packagesDir, { withFileTypes: true })
    .filter((entry: Dirent) => entry.isDirectory())
    .map((entry: Dirent) => join(packagesDir, entry.name))
    .filter((packageDir: string) => existsSync(join(packageDir, "package.json")));
}

function moveDirectoryContents(sourceDir: string, destinationDir: string): void {
  mkdirSync(destinationDir, { recursive: true });

  for (const entry of readdirSync(sourceDir)) {
    const from = join(sourceDir, entry);
    const to = join(destinationDir, entry);

    rmSync(to, { recursive: true, force: true });
    renameSync(from, to);
  }
}

function movePackageTurboLogs(): void {
  for (const packageDir of packageDirs()) {
    const turboDir = join(packageDir, ".turbo");
    let stat: Stats;

    try {
      stat = lstatSync(turboDir);
    } catch (error: unknown) {
      if (!isNodeError(error) || error.code !== "ENOENT") {
        throw error;
      }
      continue;
    }

    const packageName = packageDir.split(/[\\/]/u).at(-1) ?? "unknown";
    const destinationDir = join(turboLogCacheDir, packageName);

    if (stat.isSymbolicLink()) {
      let realPath: string;
      try {
        realPath = realpathSync(turboDir);
      } catch {
        rmSync(turboDir, { force: true });
        continue;
      }

      if (statSync(realPath).isDirectory() && resolve(realPath) !== resolve(destinationDir)) {
        moveDirectoryContents(realPath, destinationDir);
      }

      rmSync(turboDir, { force: true });
      continue;
    }

    if (stat.isDirectory()) {
      moveDirectoryContents(turboDir, destinationDir);
      rmSync(turboDir, { recursive: true, force: true });
    }
  }
}

function isNodeError(error: unknown): error is NodeJS.ErrnoException {
  return error instanceof Error && "code" in error;
}

const result = spawnSync("pnpm", ["exec", "turbo", ...args], {
  cwd: rootDir,
  stdio: "inherit",
});

try {
  movePackageTurboLogs();
} catch (error: unknown) {
  const message = error instanceof Error ? error.message : String(error);
  console.error(`Failed to move Turbo logs into storage/.cache: ${message}`);
  process.exitCode = 1;
}

if (result.error) {
  console.error(result.error.message);
  process.exit(result.status ?? 1);
}

if (result.signal) {
  console.error(`Turbo exited after receiving ${result.signal}`);
  process.exit(1);
}

if (process.exitCode) {
  process.exit(process.exitCode);
}

process.exit(result.status ?? 0);
