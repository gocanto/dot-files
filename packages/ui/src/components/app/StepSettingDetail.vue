<script setup lang="ts">
import { FolderOpen } from "lucide-vue-next";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Separator } from "@/components/ui/separator";
import StatusBadge from "@/components/StatusBadge.vue";
import type { StepSettingsKey } from "@/components/app/types";
import { initials } from "@/lib/format";
import type { RuntimeSettings, SettingsResponse } from "@/types/api";

defineProps<{
  selectedSettingsKey: StepSettingsKey;
  settingsKeyLabels: Record<StepSettingsKey, string>;
  stepTitle: string;
  settingsForm: RuntimeSettings;
  settingsResponse: SettingsResponse | null;
}>();

const emit = defineEmits<{
  (event: "update-setting", key: StepSettingsKey, value: string): void;
  (event: "choose-directory", key: StepSettingsKey): void;
}>();
</script>

<template>
  <div class="flex items-start bg-section p-4">
    <div class="flex items-start gap-4 text-sm">
      <Avatar size="sm">
        <AvatarFallback>{{ initials(settingsKeyLabels[selectedSettingsKey]) }}</AvatarFallback>
      </Avatar>
      <div class="grid gap-1">
        <div class="font-semibold">{{ settingsKeyLabels[selectedSettingsKey] }}</div>
        <div class="line-clamp-1 text-xs text-muted-foreground">Step setting · {{ stepTitle }}</div>
      </div>
    </div>
    <StatusBadge
      class="ml-auto"
      :status="settingsResponse?.valid ? 'ok' : 'failed'"
      :label="settingsResponse?.valid ? 'valid' : 'needs review'"
    />
  </div>
  <Separator />
  <ScrollArea class="min-h-0 flex-1">
    <div class="grid gap-3 p-4">
      <Label :for="`step-setting-${selectedSettingsKey}`">{{
        settingsKeyLabels[selectedSettingsKey]
      }}</Label>
      <div class="flex gap-2">
        <Input
          :id="`step-setting-${selectedSettingsKey}`"
          :model-value="settingsForm[selectedSettingsKey]"
          @update:model-value="emit('update-setting', selectedSettingsKey, String($event))"
        />
        <Button
          type="button"
          variant="outline"
          size="icon"
          @click="emit('choose-directory', selectedSettingsKey)"
        >
          <FolderOpen class="size-4" />
        </Button>
      </div>
      <p class="text-xs text-muted-foreground">
        Edits here apply to the same setting visible in the All settings panel. Save from there to
        persist.
      </p>
    </div>
  </ScrollArea>
</template>
