import * as monaco from "monaco-editor/esm/vs/editor/editor.api.js";
import "monaco-editor/min/vs/editor/editor.main.css";
import "monaco-editor/esm/vs/basic-languages/shell/shell.contribution.js";
import "monaco-editor/esm/vs/basic-languages/yaml/yaml.contribution.js";
import "monaco-editor/esm/vs/language/json/monaco.contribution.js";
import { configureMonacoYaml } from "monaco-yaml";

type MonacoWithEnvironment = typeof globalThis & {
  MonacoEnvironment?: {
    getWorker(_: string, label: string): Worker;
  };
};

let configured = false;

export function setupMonaco() {
  if (configured) {
    return monaco;
  }

  const global = globalThis as MonacoWithEnvironment;

  global.MonacoEnvironment = {
    getWorker(_: string, label: string) {
      if (label === "yaml") {
        return new Worker(new URL("monaco-yaml/yaml.worker.js", import.meta.url), {
          type: "module",
        });
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

  configureMonacoYaml(monaco, {
    enableSchemaRequest: false,
  });

  configured = true;

  return monaco;
}

export type Monaco = typeof monaco;
