import { readdirSync } from "node:fs";
import { join } from "node:path";

export function walkFiles(dir: string): string[] {
  const files: string[] = [];

  for (const entry of readdirSync(dir, { withFileTypes: true })) {
    const path = join(dir, entry.name);

    if (entry.isDirectory()) {
      files.push(...walkFiles(path));
      continue;
    }

    files.push(path);
  }

  return files;
}
