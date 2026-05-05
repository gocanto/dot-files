<script setup lang="ts">
import { AlertTriangle, Loader2 } from "lucide-vue-next";
import {
  AlertDialog,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Button } from "@/components/ui/button";
import type { ConfirmationOption } from "@/types/api";

defineProps<{
  pendingOption: ConfirmationOption | null;
  running: boolean;
}>();

const emit = defineEmits<{
  (event: "update:open", open: boolean): void;
  (event: "continue", option: ConfirmationOption): void;
}>();
</script>

<template>
  <AlertDialog :open="pendingOption !== null" @update:open="emit('update:open', $event)">
    <AlertDialogContent>
      <AlertDialogHeader>
        <AlertDialogTitle class="flex items-center gap-2">
          <AlertTriangle class="size-5 text-destructive" />
          {{ pendingOption?.label }}
        </AlertDialogTitle>
        <AlertDialogDescription>{{ pendingOption?.description }}</AlertDialogDescription>
      </AlertDialogHeader>
      <AlertDialogFooter>
        <AlertDialogCancel>Cancel</AlertDialogCancel>
        <Button :disabled="running || !pendingOption" @click="pendingOption && emit('continue', pendingOption)">
          <Loader2 v-if="running" class="size-4 animate-spin" />
          Continue
        </Button>
      </AlertDialogFooter>
    </AlertDialogContent>
  </AlertDialog>
</template>
