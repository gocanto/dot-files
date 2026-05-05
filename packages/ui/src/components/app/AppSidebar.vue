<script setup lang="ts">
import { Apple, Moon, Sun } from "lucide-vue-next";
import type { Component } from "vue";
import { Button } from "@/components/ui/button";
import { Separator } from "@/components/ui/separator";
import { Skeleton } from "@/components/ui/skeleton";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";

type SectionId = "template" | "current" | "update" | "settings" | "logs";

interface NavItem {
  id: SectionId;
  label: string;
  icon: Component;
  count: number | null;
}

defineProps<{
  collapsed: boolean;
  section: SectionId;
  macName: string;
  macHostname: string;
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
  <div :class="cn('flex h-12 items-center bg-sidebar', collapsed ? 'justify-center px-2' : 'gap-2 px-3')">
    <Apple class="size-4 shrink-0" />
    <div v-if="!collapsed" class="flex min-w-0 flex-col">
      <span class="truncate text-sm font-medium">Mac: {{ macName }}</span>
      <span class="truncate text-[10px] text-muted-foreground">{{ macHostname }}</span>
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

  <div :data-collapsed="collapsed" class="group flex flex-col gap-4 py-2 data-[collapsed=true]:py-2">
    <nav class="grid gap-1 px-2 group-[[data-collapsed=true]]:justify-center group-[[data-collapsed=true]]:px-2">
      <template v-for="item in stepNavItems" :key="item.id">
        <Tooltip v-if="collapsed">
          <TooltipTrigger as-child>
            <Button
              variant="ghost"
              size="icon"
              :class="cn(section === item.id && 'bg-accent text-accent-foreground hover:bg-accent dark:hover:bg-accent')"
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
          :class="cn('justify-start', section === item.id && 'bg-accent text-accent-foreground hover:bg-accent dark:hover:bg-accent')"
          @click="emit('select-section', item.id)"
        >
          <component :is="item.icon" class="mr-2 size-4" />
          {{ item.label }}
          <Skeleton v-if="item.count === null" class="ml-auto h-4 w-6 rounded" />
          <span v-else class="ml-auto">{{ item.count }}</span>
        </Button>
      </template>
    </nav>
  </div>

  <Separator />

  <div :data-collapsed="collapsed" class="group flex flex-col gap-4 py-2 data-[collapsed=true]:py-2">
    <nav class="grid gap-1 px-2 group-[[data-collapsed=true]]:justify-center group-[[data-collapsed=true]]:px-2">
      <template v-for="item in auxNavItems" :key="item.id">
        <Tooltip v-if="collapsed">
          <TooltipTrigger as-child>
            <Button
              variant="ghost"
              size="icon"
              :class="cn(section === item.id && 'bg-accent text-accent-foreground hover:bg-accent dark:hover:bg-accent')"
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
          :class="cn('justify-start', section === item.id && 'bg-accent text-accent-foreground hover:bg-accent dark:hover:bg-accent')"
          @click="emit('select-section', item.id)"
        >
          <component :is="item.icon" class="mr-2 size-4" />
          {{ item.label }}
        </Button>
      </template>
    </nav>
  </div>
</template>
