import { failWithMatches } from "#scripts/lib/cli.js";
import { walkFiles } from "#scripts/lib/walk.js";

const roots = ["src", "tests", "electron", "scripts", "../../scripts", "../bridge/src"];
const forbidden = /\.(?:js|jsx|mjs|cjs)$/u;
const matches: string[] = [];

for (const root of roots) {
  for (const path of walkFiles(root)) {
    if (forbidden.test(path)) {
      matches.push(path);
    }
  }
}

failWithMatches(matches);
