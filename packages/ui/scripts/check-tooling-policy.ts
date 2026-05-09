import { existsSync, readdirSync, readFileSync, statSync } from "node:fs";
import { basename, join, relative, resolve } from "node:path";

type PackageJson = {
  scripts?: Record<string, string>;
  dependencies?: Record<string, string>;
  devDependencies?: Record<string, string>;
  optionalDependencies?: Record<string, string>;
  peerDependencies?: Record<string, string>;
};

type PolicyMatch = {
  path: string;
  reason: string;
};

const repoRoot = resolve(process.argv[2] ?? ".");
const matches: PolicyMatch[] = [];
const ignoredDirectories = new Set([
  ".git",
  ".idea",
  ".turbo",
  "dist",
  "dist-electron",
  "node_modules",
  "release",
  "storage",
]);
const forbiddenPackages = [
  "@babel/core",
  "@babel/parser",
  "@babel/preset-env",
  "@babel/preset-typescript",
  "@biomejs/biome",
  "@eslint/js",
  "@swc/core",
  "babel",
  "babel-jest",
  "biome",
  "esbuild",
  "eslint",
  "jest",
  "prettier",
  "rollup",
  "swc",
  "terser",
  "ts-jest",
  "tsup",
  "unbuild",
  "webpack",
];
const forbiddenScriptCommands = [
  "babel",
  "biome",
  "esbuild",
  "eslint",
  "jest",
  "prettier",
  "rollup",
  "swc",
  "terser",
  "ts-jest",
  "tsup",
  "unbuild",
  "webpack",
];
const forbiddenConfigPatterns = [
  /^\.?babel(?:rc)?(?:\..*)?$/u,
  /^\.?biome(?:\..*)?$/u,
  /^\.?eslint(?:rc)?(?:\..*)?$/u,
  /^\.?prettier(?:rc|ignore)?(?:\..*)?$/u,
  /^jest\.config\./u,
  /^rollup\.config\./u,
  /^terser\.config\./u,
  /^webpack\.config\./u,
];
const sourceFile = /\.(?:cjs|cts|js|jsx|mjs|mts|ts|tsx|vue)$/u;

for (const path of walk(repoRoot)) {
  const name = basename(path);

  if (name === "package.json") {
    checkPackageJson(path);
    continue;
  }

  if (forbiddenConfigPatterns.some((pattern) => pattern.test(name))) {
    report(path, "forbidden tooling config");
    continue;
  }

  if (sourceFile.test(name)) {
    checkSourceImports(path);
  }
}

if (matches.length > 0) {
  console.error("Forbidden direct JavaScript tooling usage found:");

  for (const match of matches) {
    console.error(`${match.path}: ${match.reason}`);
  }

  process.exit(1);
}

function checkPackageJson(path: string): void {
  const packageJson = JSON.parse(readFileSync(path, "utf8")) as PackageJson;
  const dependencyGroups = [
    packageJson.dependencies,
    packageJson.devDependencies,
    packageJson.optionalDependencies,
    packageJson.peerDependencies,
  ];

  for (const dependencies of dependencyGroups) {
    for (const name of Object.keys(dependencies ?? {})) {
      if (isForbiddenPackage(name)) {
        report(path, `direct dependency on ${name}`);
      }
    }
  }

  for (const [name, command] of Object.entries(packageJson.scripts ?? {})) {
    const forbidden = forbiddenScriptCommands.find((tool) => commandInvokes(command, tool));

    if (forbidden) {
      report(path, `script "${name}" invokes ${forbidden}`);
    }
  }
}

function checkSourceImports(path: string): void {
  const content = readFileSync(path, "utf8");
  const importPattern =
    /(?:from\s+["']([^"']+)["']|import\(\s*["']([^"']+)["']|require\(\s*["']([^"']+)["'])/gu;
  let match: RegExpExecArray | null;

  while ((match = importPattern.exec(content))) {
    const specifier = match[1] ?? match[2] ?? match[3] ?? "";

    if (isForbiddenPackage(packageName(specifier))) {
      report(path, `imports ${specifier}`);
    }
  }
}

function isForbiddenPackage(name: string): boolean {
  return forbiddenPackages.some(
    (forbidden) => name === forbidden || name.startsWith(`${forbidden}/`),
  );
}

function packageName(specifier: string): string {
  if (specifier.startsWith("@")) {
    const [scope, name] = specifier.split("/");
    return `${scope}/${name ?? ""}`;
  }

  return specifier.split("/")[0] ?? specifier;
}

function commandInvokes(command: string, tool: string): boolean {
  return new RegExp(`(^|[\\s;&|()])${escapeRegex(tool)}($|[\\s;&|()])`, "u").test(command);
}

function escapeRegex(value: string): string {
  return value.replace(/[.*+?^${}()|[\]\\]/gu, "\\$&");
}

function report(path: string, reason: string): void {
  matches.push({
    path: relative(repoRoot, path),
    reason,
  });
}

function* walk(directory: string): Generator<string> {
  if (!existsSync(directory)) {
    return;
  }

  for (const entry of readdirSync(directory)) {
    if (ignoredDirectories.has(entry)) {
      continue;
    }

    const path = join(directory, entry);
    const stat = statSync(path);

    if (stat.isDirectory()) {
      yield* walk(path);
      continue;
    }

    if (stat.isFile()) {
      yield path;
    }
  }
}
