import { walkFiles } from "./lib/walk.js";
import { failWithMatches } from "./lib/cli.js";

const roots = ["src", "electron"];
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
