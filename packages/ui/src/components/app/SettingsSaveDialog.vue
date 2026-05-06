<script setup lang="ts">
import { AlertTriangle, Loader2, Save } from "lucide-vue-next";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@ui/alert-dialog";

defineProps<{
  open: boolean;
  saving: boolean;
}>();

const emit = defineEmits<{
  (event: "update:open", open: boolean): void;
  (event: "confirm"): void;
}>();
</script>

<template>
  <AlertDialog
    :open="open"
    @update:open="saving ? emit('update:open', true) : emit('update:open', $event)"
  >
    <AlertDialogContent>
      <AlertDialogHeader>
        <AlertDialogTitle class="flex items-center gap-2">
          <AlertTriangle class="size-5 text-primary" />
          Save settings?
        </AlertDialogTitle>
        <AlertDialogDescription>
          This persists the pending app settings. If the settings are valid, the workflow bridge may
          restart so future workflows use the saved values.
        </AlertDialogDescription>
      </AlertDialogHeader>
      <AlertDialogFooter>
        <AlertDialogCancel :disabled="saving">Cancel</AlertDialogCancel>
        <AlertDialogAction :disabled="saving" @click="emit('confirm')">
          <Loader2 v-if="saving" class="size-4 animate-spin" />
          <Save v-else class="size-4" />
          Save settings
        </AlertDialogAction>
      </AlertDialogFooter>
    </AlertDialogContent>
  </AlertDialog>
</template>
