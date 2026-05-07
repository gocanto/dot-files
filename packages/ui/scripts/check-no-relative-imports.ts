import { readFileSync } from "node:fs";
import { failWithMatches } from "#scripts/lib/cli.js";
import { walkFiles } from "#scripts/lib/walk.js";

const roots = ["src", "tests", "electron", "scripts", "../../scripts", "../bridge/src"];
const sourceFile = /\.(?:ts|tsx|js|jsx|mjs|cjs|vue)$/u;
const relativeImport = /from\s+['"]\.{1,2}\//u;
const dynamicRelativeImport = /import\(\s*['"]\.{1,2}\//u;
const relativeRequire = /require\(\s*['"]\.{1,2}\//u;
const matches: string[] = [];

for (const root of roots) {
  for (const path of walkFiles(root)) {
    if (!sourceFile.test(path)) {
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

failWithMatches(matches);
