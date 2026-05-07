import { readFileSync, readdirSync } from "node:fs";
import { join } from "node:path";

const roots = ["src", "tests"];
const sourceFile = /\.(?:ts|tsx|js|jsx|mjs|cjs|vue)$/u;
const relativeImport = /from\s+['"]\.{1,2}\//u;
const dynamicRelativeImport = /import\(\s*['"]\.{1,2}\//u;
const relativeRequire = /require\(\s*['"]\.{1,2}\//u;
const matches = [];

function walk(dir) {
  for (const entry of readdirSync(dir, { withFileTypes: true })) {
    const path = join(dir, entry.name);

    if (entry.isDirectory()) {
      walk(path);
      continue;
    }

    if (!sourceFile.test(entry.name)) {
      continue;
    }

    const content = readFileSync(path, "utf8");

    if (
      relativeImport.test(content) ||
      dynamicRelativeImport.test(content) ||
      relativeRequire.test(content)
    ) {
      matches.push(path);
    }
  }
}

for (const root of roots) {
  walk(root);
}

if (matches.length > 0) {
  console.error(matches.join("\n"));
  process.exit(1);
}
