import * as monaco from "monaco-editor/esm/vs/editor/editor.api.js";
import "monaco-editor/min/vs/editor/editor.main.css";
import "monaco-editor/esm/vs/basic-languages/shell/shell.contribution.js";
import "monaco-editor/esm/vs/basic-languages/yaml/yaml.contribution.js";
import "monaco-editor/esm/vs/language/json/monaco.contribution.js";
import { configureMonacoYaml } from "monaco-yaml";
import YamlWorker from "@lib/yaml.worker?worker";

type MonacoWithEnvironment = typeof globalThis & {
  MonacoEnvironment?: {
    getWorker(_: string, label: string): Worker;
  };
};

type LegacyWebWorkerOptions = {
  createData?: unknown;
  host?: Record<string, (...args: unknown[]) => unknown>;
  keepIdleModels?: boolean;
  label?: string;
};

let configured = false;

function isLegacyWebWorkerOptions(options: unknown): options is LegacyWebWorkerOptions {
  return Boolean(options && typeof options === "object" && !("worker" in options));
}

function createWorker(global: MonacoWithEnvironment, label: string, createData: unknown) {
  const worker = Promise.resolve(
    global.MonacoEnvironment?.getWorker("workerMain.js", label) ??
      new Worker(new URL("monaco-editor/esm/vs/editor/editor.worker.js", import.meta.url), {
        type: "module",
      }),
  );

  return worker.then((resolvedWorker) => {
    resolvedWorker.postMessage("ignore");
    resolvedWorker.postMessage(createData);
    return resolvedWorker;
  });
}

function patchCreateWebWorker(global: MonacoWithEnvironment) {
  const createWebWorker = monaco.editor.createWebWorker.bind(monaco.editor);

  monaco.editor.createWebWorker = ((options: unknown) => {
    if (!isLegacyWebWorkerOptions(options)) {
      return createWebWorker(options as Parameters<typeof createWebWorker>[0]);
    }

    return createWebWorker({
      host: options.host,
      keepIdleModels: options.keepIdleModels,
      worker: createWorker(global, options.label ?? "monaco-editor-worker", options.createData),
    });
  }) as typeof monaco.editor.createWebWorker;
}

export function setupMonaco() {
  if (configured) {
    return monaco;
  }

  const global = globalThis as MonacoWithEnvironment;

  global.MonacoEnvironment = {
    getWorker(_: string, label: string) {
      if (label === "yaml") {
        return new YamlWorker();
      }

      if (label === "json") {
        return new Worker(
          new URL("monaco-editor/esm/vs/language/json/json.worker.js", import.meta.url),
          { type: "module" },
        );
      }

      return new Worker(new URL("monaco-editor/esm/vs/editor/editor.worker.js", import.meta.url), {
        type: "module",
      });
    },
  };

  patchCreateWebWorker(global);

  configureMonacoYaml(monaco, {
    enableSchemaRequest: false,
  });

  configured = true;

  return monaco;
}

export type Monaco = typeof monaco;
