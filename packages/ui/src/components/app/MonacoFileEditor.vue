<script setup lang="ts">
import { nextTick, onBeforeUnmount, ref, shallowRef, watch } from "vue";
import type { editor } from "monaco-editor/esm/vs/editor/editor.api.js";
import { setupMonaco } from "@lib/monaco";
import { templateFileLanguage } from "@lib/templateFileLanguage";

const props = defineProps<{
  modelValue: string;
  path: string;
  loading: boolean;
  theme: "light" | "dark";
  readonly?: boolean;
}>();

const emit = defineEmits<{
  (event: "update:modelValue", value: string): void;
}>();

const container = ref<HTMLElement | null>(null);
const editorRef = shallowRef<editor.IStandaloneCodeEditor | null>(null);
const modelRef = shallowRef<editor.ITextModel | null>(null);
const suppressChange = ref(false);
let resizeObserver: ResizeObserver | null = null;

function disposeModel() {
  modelRef.value?.dispose();
  modelRef.value = null;
}

function disposeEditor() {
  resizeObserver?.disconnect();
  resizeObserver = null;
  editorRef.value?.dispose();
  editorRef.value = null;
  disposeModel();
}

function createOrUpdateModel() {
  const monaco = setupMonaco();
  const language = templateFileLanguage(props.path);
  const uri = monaco.Uri.file(props.path || "/template-file.txt");

  disposeModel();
  modelRef.value = monaco.editor.createModel(props.modelValue, language, uri);
  editorRef.value?.setModel(modelRef.value);
}

async function mountEditor() {
  if (!container.value || editorRef.value) {
    return;
  }

  const monaco = setupMonaco();

  await nextTick();

  editorRef.value = monaco.editor.create(container.value, {
    automaticLayout: true,
    fontFamily: "JetBrains Mono Variable, JetBrains Mono, monospace",
    fontSize: 12,
    lineHeight: 20,
    minimap: { enabled: false },
    readOnly: props.readonly || props.loading,
    scrollBeyondLastLine: false,
    theme: props.theme === "dark" ? "vs-dark" : "vs",
    wordWrap: "on",
  });

  createOrUpdateModel();
  editorRef.value.layout();

  resizeObserver = new ResizeObserver(() => {
    editorRef.value?.layout();
  });
  resizeObserver.observe(container.value);

  editorRef.value.onDidChangeModelContent(() => {
    if (suppressChange.value) {
      return;
    }

    emit("update:modelValue", editorRef.value?.getValue() ?? "");
  });
}

watch(container, () => void mountEditor(), { immediate: true });

watch(
  () => props.path,
  () => {
    if (!editorRef.value) {
      return;
    }

    createOrUpdateModel();
  },
);

watch(
  () => props.modelValue,
  (value) => {
    const editor = editorRef.value;

    if (!editor || editor.getValue() === value) {
      return;
    }

    suppressChange.value = true;
    editor.setValue(value);
    suppressChange.value = false;
  },
);

watch(
  () => [props.loading, props.readonly] as const,
  ([loading, readonly]) => {
    editorRef.value?.updateOptions({ readOnly: loading || Boolean(readonly) });
  },
);

watch(
  () => props.theme,
  (theme) => {
    setupMonaco().editor.setTheme(theme === "dark" ? "vs-dark" : "vs");
  },
);

onBeforeUnmount(disposeEditor);
</script>

<template>
  <div
    ref="container"
    class="h-full min-h-0 flex-1 overflow-hidden rounded-md border border-section-border"
  />
</template>
