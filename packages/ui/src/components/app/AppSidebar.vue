<script setup lang="ts">
import { Apple, Moon, Sun } from "lucide-vue-next";
import { Button } from "@ui/button";
import { Separator } from "@ui/separator";
import { Skeleton } from "@ui/skeleton";
import { Tooltip, TooltipContent, TooltipTrigger } from "@ui/tooltip";
import type { NavItem, SectionId } from "@app/types";
import { cn } from "@lib/utils";
import type { MacSystemInfo } from "@api";

defineProps<{
  collapsed: boolean;
  section: SectionId;
  macName: string;
  macHostname: string;
  macSystemInfo: MacSystemInfo;
  theme: string;
  stepNavItems: NavItem[];
  auxNavItems: NavItem[];
}>();

const emit = defineEmits<{
  (event: "select-section", section: SectionId): void;
  (event: "toggle-theme"): void;
}>();
</script>

<template>
  <div
    :class="
      cn(
        'flex min-h-[var(--panel-header-h)] items-center bg-sidebar',
        collapsed ? 'justify-center px-2' : 'gap-3 px-3',
      )
    "
  >
    <div
      class="grid size-10 shrink-0 place-items-center rounded-lg border border-sidebar-border bg-background/70 text-foreground shadow-sm"
    >
      <Apple class="size-5" />
    </div>
    <div v-if="!collapsed" class="flex min-w-0 flex-1 flex-col gap-1">
      <div class="min-w-0">
        <span class="block truncate text-sm font-semibold"
          >Mac: {{ macSystemInfo.name || macName }}</span
        >
        <span class="block truncate text-xs text-muted-foreground">
          {{ macSystemInfo.hostname || macHostname }}
        </span>
      </div>
      <div
        v-if="macSystemInfo.osLabel || macSystemInfo.architectureLabel"
        class="flex min-w-0 flex-wrap items-center gap-1.5 text-[10px] font-medium text-muted-foreground"
      >
        <span
          v-if="macSystemInfo.osLabel"
          class="max-w-full truncate rounded-md border border-sidebar-border bg-background/70 px-1.5 py-0.5"
        >
          {{ macSystemInfo.osLabel }}
        </span>
        <span
          v-if="macSystemInfo.architectureLabel"
          class="max-w-full truncate rounded-md border border-sidebar-border bg-background/70 px-1.5 py-0.5"
        >
          {{ macSystemInfo.architectureLabel }}
        </span>
      </div>
    </div>
    <Tooltip v-if="!collapsed">
      <TooltipTrigger as-child>
        <Button variant="ghost" size="icon-sm" class="ml-auto" @click="emit('toggle-theme')">
          <Sun v-if="theme === 'dark'" class="size-4" />
          <Moon v-else class="size-4" />
          <span class="sr-only">Toggle theme</span>
        </Button>
      </TooltipTrigger>
      <TooltipContent>Toggle theme</TooltipContent>
    </Tooltip>
  </div>

  <Separator />

  <div
    :data-collapsed="collapsed"
    class="group flex flex-col gap-4 py-2 data-[collapsed=true]:py-2"
  >
    <nav
      class="grid gap-1 px-2 group-[[data-collapsed=true]]:justify-center group-[[data-collapsed=true]]:px-2"
    >
      <template v-for="item in stepNavItems" :key="item.id">
        <Tooltip v-if="collapsed">
          <TooltipTrigger as-child>
            <Button
              variant="ghost"
              size="icon"
              :class="
                cn(
                  section === item.id &&
                    'bg-accent text-accent-foreground hover:bg-accent dark:hover:bg-accent',
                )
              "
              @click="emit('select-section', item.id)"
            >
              <component :is="item.icon" class="size-4" />
              <span class="sr-only">{{ item.label }}</span>
            </Button>
          </TooltipTrigger>
          <TooltipContent side="right" class="flex items-center gap-4">
            {{ item.label }}
            <Skeleton v-if="item.count === null" class="ml-auto h-4 w-6 rounded" />
            <span v-else class="ml-auto text-muted-foreground">{{ item.count }}</span>
          </TooltipContent>
        </Tooltip>

        <Button
          v-else
          variant="ghost"
          size="sm"
          :class="
            cn(
              'grid w-full grid-cols-[1.5rem_minmax(0,1fr)_2.5rem] items-center gap-2 justify-self-stretch px-2',
              section === item.id &&
                'bg-accent text-accent-foreground hover:bg-accent dark:hover:bg-accent',
            )
          "
          @click="emit('select-section', item.id)"
        >
          <span class="grid size-6 place-items-center">
            <component :is="item.icon" class="size-4" />
          </span>
          <span class="min-w-0 truncate text-left">{{ item.label }}</span>
          <Skeleton v-if="item.count === null" class="ml-auto h-4 w-6 rounded" />
          <span v-else class="ml-auto justify-self-end tabular-nums">{{ item.count }}</span>
        </Button>
      </template>
    </nav>
  </div>

  <Separator />

  <div
    :data-collapsed="collapsed"
    class="group flex flex-col gap-4 py-2 data-[collapsed=true]:py-2"
  >
    <nav
      class="grid gap-1 px-2 group-[[data-collapsed=true]]:justify-center group-[[data-collapsed=true]]:px-2"
    >
      <template v-for="item in auxNavItems" :key="item.id">
        <Tooltip v-if="collapsed">
          <TooltipTrigger as-child>
            <Button
              variant="ghost"
              size="icon"
              :class="
                cn(
                  section === item.id &&
                    'bg-accent text-accent-foreground hover:bg-accent dark:hover:bg-accent',
                )
              "
              @click="emit('select-section', item.id)"
            >
              <component :is="item.icon" class="size-4" />
              <span class="sr-only">{{ item.label }}</span>
            </Button>
          </TooltipTrigger>
          <TooltipContent side="right">{{ item.label }}</TooltipContent>
        </Tooltip>

        <Button
          v-else
          variant="ghost"
          size="sm"
          :class="
            cn(
              'grid w-full grid-cols-[1.5rem_minmax(0,1fr)_2.5rem] items-center gap-2 justify-self-stretch px-2',
              section === item.id &&
                'bg-accent text-accent-foreground hover:bg-accent dark:hover:bg-accent',
            )
          "
          @click="emit('select-section', item.id)"
        >
          <span class="grid size-6 place-items-center">
            <component :is="item.icon" class="size-4" />
          </span>
          <span class="min-w-0 truncate text-left">{{ item.label }}</span>
          <span aria-hidden="true" class="size-6 justify-self-end" />
        </Button>
      </template>
    </nav>
  </div>
</template>
