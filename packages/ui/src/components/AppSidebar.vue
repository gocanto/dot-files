<script setup lang="ts">
import {
  Activity,
  Apple,
  ClipboardList,
  HardDrive,
  History,
  Settings,
} from "lucide-vue-next";

const props = defineProps<{
  active: string;
  workflowCount: number;
  runCount: number;
}>();

const emit = defineEmits<{
  select: [section: string];
}>();

const items = [
  { id: "this-mac", label: "This Mac", icon: Apple },
  { id: "workflows", label: "Workflows", icon: ClipboardList },
  { id: "snapshots", label: "Snapshots", icon: HardDrive },
  { id: "health", label: "Health Checks", icon: Activity },
  { id: "logs", label: "Logs", icon: History },
  { id: "settings", label: "Settings", icon: Settings },
];
</script>

<template>
  <aside class="flex h-screen w-72 shrink-0 flex-col border-r border-zinc-200 bg-zinc-50">
    <div class="flex h-14 items-center gap-3 border-b border-zinc-200 px-4">
      <div class="grid size-8 place-items-center rounded-lg bg-zinc-900 text-sm font-semibold text-white">
        OS
      </div>
      <div class="min-w-0">
        <p class="truncate text-sm font-semibold text-zinc-950">Mac OS Manager</p>
        <p class="truncate text-xs text-zinc-500">Current Mac</p>
      </div>
    </div>

    <nav class="flex-1 space-y-6 overflow-y-auto px-3 py-4">
      <div>
        <p class="px-2 pb-2 text-xs font-medium text-zinc-500">Computer</p>
        <button
          v-for="item in items"
          :key="item.id"
          class="mb-1 flex h-9 w-full items-center gap-2 rounded-md px-2 text-left text-sm transition"
          :class="
            props.active === item.id
              ? 'bg-white font-medium text-zinc-950 shadow-sm ring-1 ring-zinc-200'
              : 'text-zinc-600 hover:bg-white hover:text-zinc-950'
          "
          @click="emit('select', item.id)"
        >
          <component :is="item.icon" class="size-4" />
          <span class="flex-1 truncate">{{ item.label }}</span>
          <span
            v-if="item.id === 'workflows'"
            class="rounded bg-zinc-100 px-1.5 py-0.5 text-[11px] text-zinc-500"
          >
            {{ props.workflowCount }}
          </span>
          <span
            v-if="item.id === 'logs'"
            class="rounded bg-zinc-100 px-1.5 py-0.5 text-[11px] text-zinc-500"
          >
            {{ props.runCount }}
          </span>
        </button>
      </div>
    </nav>

    <div class="border-t border-zinc-200 p-3 text-xs font-medium text-zinc-500">Local</div>
  </aside>
</template>
