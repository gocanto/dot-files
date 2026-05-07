import { readdirSync } from "node:fs";
import { join } from "node:path";

const roots = ["src", "electron"];
const forbidden = /\.(?:js|jsx|mjs|cjs)$/u;
const matches = [];

function walk(dir) {
  for (const entry of readdirSync(dir, { withFileTypes: true })) {
    const path = join(dir, entry.name);

    if (entry.isDirectory()) {
      walk(path);
      continue;
    }

    if (forbidden.test(entry.name)) {
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
