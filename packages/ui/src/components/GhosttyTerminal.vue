<script setup lang="ts">
import { FitAddon, Terminal, init, type IDisposable } from "ghostty-web";
import { nextTick, onBeforeUnmount, onMounted, ref } from "vue";

const emit = defineEmits<{
  data: [value: string];
  error: [error: Error];
  ready: [];
}>();

const terminalContainer = ref<HTMLElement | null>(null);
let terminal: Terminal | null = null;
let fitAddon: FitAddon | null = null;
let dataSubscription: IDisposable | null = null;
let mounted = false;

onMounted(async () => {
  mounted = true;

  try {
    await init();

    if (!mounted || !terminalContainer.value) {
      return;
    }

    terminal = new Terminal({
      cursorBlink: true,
      fontFamily: "'JetBrains Mono Variable', 'JetBrains Mono', monospace",
      fontSize: 13,
      theme: {
        background: "#111111",
        foreground: "#f5f5f5",
        cursor: "#f5f5f5",
        selectionBackground: "#3f3f46",
      },
    });
    fitAddon = new FitAddon();
    terminal.loadAddon(fitAddon);
    dataSubscription = terminal.onData((value) => emit("data", value));
    terminal.open(terminalContainer.value);

    await nextTick();

    if (!mounted) {
      return;
    }

    fitAddon.fit();
    terminal.focus();
    emit("ready");
  } catch (error) {
    emit("error", error instanceof Error ? error : new Error(String(error)));
  }
});

onBeforeUnmount(() => {
  mounted = false;
  dataSubscription?.dispose();
  fitAddon?.dispose();
  terminal?.dispose();
  dataSubscription = null;
  fitAddon = null;
  terminal = null;
});

function write(value: string | Uint8Array) {
  terminal?.write(value);
}

function clear() {
  terminal?.clear();
}

function fit() {
  fitAddon?.fit();
}

function focus() {
  terminal?.focus();
}

defineExpose({ clear, fit, focus, write });
</script>

<template>
  <div ref="terminalContainer" class="ghostty-terminal" />
</template>

<style scoped>
.ghostty-terminal {
  min-height: 260px;
  height: 100%;
  width: 100%;
  overflow: hidden;
  background: #111111;
}
</style>
