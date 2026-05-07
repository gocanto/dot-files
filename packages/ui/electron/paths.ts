import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

export const electronDir = dirname(fileURLToPath(import.meta.url));
export const repoRoot = resolve(electronDir, "..", "..", "..");
export const macbookDir = join(repoRoot, "packages", "macbook");
