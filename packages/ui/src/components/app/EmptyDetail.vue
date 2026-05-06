<script setup lang="ts">
import { ShieldCheck } from "lucide-vue-next";
import { Avatar, AvatarFallback } from "@ui/avatar";
import { Separator } from "@ui/separator";
import { initials } from "@lib/format";

defineProps<{
  section: string;
}>();
</script>

<template>
  <div class="flex items-start bg-section p-4">
    <div class="flex items-start gap-4 text-sm">
      <Avatar size="sm">
        <AvatarFallback>{{ initials(section.replace("-", " ")) }}</AvatarFallback>
      </Avatar>
      <div class="grid gap-1">
        <div class="font-semibold capitalize">{{ section.replace("-", " ") }}</div>
        <div class="line-clamp-1 text-xs">
          This area is available from the mail-style navigation.
        </div>
        <div class="line-clamp-1 text-xs">
          <span class="font-medium">Profile:</span> Current Mac
        </div>
      </div>
    </div>
  </div>
  <Separator />
  <div class="grid flex-1 place-items-center p-8 text-center text-sm text-muted-foreground">
    <div>
      <ShieldCheck class="mx-auto mb-3 size-8" />
      <p>Workflow execution and log inspection are available from Workflows and Logs.</p>
    </div>
  </div>
</template>
