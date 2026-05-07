<script setup lang="ts">
import {
  AlertTriangle,
  Database,
  Download,
  FileText,
  FolderOpen,
  KeyRound,
  Loader2,
  RefreshCw,
  Save,
} from "lucide-vue-next";
import { Avatar, AvatarFallback } from "@ui/avatar";
import { Button } from "@ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@ui/card";
import { Input } from "@ui/input";
import { Label } from "@ui/label";
import { ScrollArea } from "@ui/scroll-area";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@ui/select";
import { Separator } from "@ui/separator";
import { Skeleton } from "@ui/skeleton";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@ui/table";
import { Tooltip, TooltipContent, TooltipTrigger } from "@ui/tooltip";
import StatusBadge from "@components/StatusBadge.vue";
import type { SavedField, SelectOption } from "@app/types";
import type { RuntimeSettings, SettingsCheck, SettingsResponse } from "@api";

type SettingsKey = keyof RuntimeSettings;

defineProps<{
  settingsForm: RuntimeSettings;
  settingsResponse: SettingsResponse | null;
  settingsChecks: SettingsCheck[];
  settingsDirty: boolean;
  settingsLoading: boolean;
  settingsSaving: boolean;
  settingsValidating: boolean;
  settingsPickerField: SettingsKey | null;
  settingsError: string;
  settingsMessage: string;
  opVaultOptions: SelectOption[];
  opItemOptions: SelectOption[];
  opVaultsError: string;
  opItemsError: string;
  opItemsLoadedFor: string;
  opVaultsLoading: boolean;
  opItemsLoading: boolean;
  opItemSelectDisabled: boolean;
  opSigninLoading: boolean;
  opInstallLoading: boolean;
  opSavedFields: SavedField[];
}>();

const emit = defineEmits<{
  (event: "update-setting", key: SettingsKey, value: string): void;
  (event: "choose-directory", key: SettingsKey): void;
  (event: "choose-file", key: SettingsKey): void;
  (event: "choose-save-file", key: SettingsKey): void;
  (event: "validate-settings"): void;
  (event: "reset-settings"): void;
  (event: "request-save-settings"): void;
  (event: "op-vault-change", value: unknown): void;
  (event: "op-item-change", value: unknown): void;
  (event: "signin-op-cli"): void;
  (event: "install-op-dependencies"): void;
  (event: "load-op-vaults"): void;
}>();
</script>

<template>
  <div class="flex items-start bg-section p-4">
    <div class="flex items-start gap-4 text-sm">
      <Avatar size="sm">
        <AvatarFallback>SE</AvatarFallback>
      </Avatar>
      <div class="grid gap-1">
        <div class="font-semibold">Settings</div>
        <div class="line-clamp-1 text-xs">
          Repository, workflow storage, and operational defaults.
        </div>
        <div class="line-clamp-1 text-xs">
          <span class="font-medium">Status:</span>
          {{ settingsResponse?.valid ? "Valid" : "Needs review" }}
        </div>
      </div>
    </div>
    <Skeleton
      v-if="settingsLoading || settingsSaving || settingsValidating"
      class="ml-auto h-5 w-16"
    />
    <StatusBadge
      v-else
      class="ml-auto"
      :status="settingsResponse?.valid ? 'ok' : 'failed'"
      :label="settingsResponse?.valid ? 'valid' : 'invalid'"
    />
  </div>
  <Separator />
  <ScrollArea class="min-h-0 flex-1">
    <div class="grid gap-6 p-4">
      <Card>
        <CardHeader>
          <CardTitle class="text-sm">Repository</CardTitle>
        </CardHeader>
        <CardContent class="grid gap-3">
          <div class="grid gap-2">
            <Label for="repo-root">Repo root</Label>
            <div class="flex gap-2">
              <Skeleton
                v-if="settingsLoading || settingsPickerField === 'repoRoot'"
                class="h-9 w-full"
              />
              <Input
                v-else
                id="repo-root"
                :model-value="settingsForm.repoRoot"
                data-testid="settings-repo-root"
                @update:model-value="emit('update-setting', 'repoRoot', String($event))"
              />
              <Tooltip>
                <TooltipTrigger as-child>
                  <Button
                    type="button"
                    variant="outline"
                    size="icon"
                    :disabled="settingsLoading || settingsPickerField !== null"
                    @click="emit('choose-directory', 'repoRoot')"
                  >
                    <FolderOpen class="size-4" />
                    <span class="sr-only">Choose repo root</span>
                  </Button>
                </TooltipTrigger>
                <TooltipContent>Choose repo root</TooltipContent>
              </Tooltip>
            </div>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle class="text-sm">Manifests</CardTitle>
        </CardHeader>
        <CardContent class="grid gap-3">
          <div class="grid gap-2">
            <Label for="apps-config">Apps manifest</Label>
            <div class="flex gap-2">
              <Skeleton
                v-if="settingsLoading || settingsPickerField === 'appsConfigPath'"
                class="h-9 w-full"
              />
              <Input
                v-else
                id="apps-config"
                :model-value="settingsForm.appsConfigPath"
                data-testid="settings-apps-config"
                @update:model-value="emit('update-setting', 'appsConfigPath', String($event))"
              />
              <Button
                type="button"
                variant="outline"
                size="icon"
                :disabled="settingsLoading || settingsPickerField !== null"
                @click="emit('choose-file', 'appsConfigPath')"
              >
                <FileText class="size-4" />
                <span class="sr-only">Choose apps manifest</span>
              </Button>
            </div>
          </div>
          <div class="grid gap-2">
            <Label for="secrets-config">Secrets manifest</Label>
            <div class="flex gap-2">
              <Skeleton
                v-if="settingsLoading || settingsPickerField === 'secretsConfigPath'"
                class="h-9 w-full"
              />
              <Input
                v-else
                id="secrets-config"
                :model-value="settingsForm.secretsConfigPath"
                @update:model-value="emit('update-setting', 'secretsConfigPath', String($event))"
              />
              <Button
                type="button"
                variant="outline"
                size="icon"
                :disabled="settingsLoading || settingsPickerField !== null"
                @click="emit('choose-file', 'secretsConfigPath')"
              >
                <FileText class="size-4" />
                <span class="sr-only">Choose secrets manifest</span>
              </Button>
            </div>
          </div>
          <div class="grid gap-2">
            <Label for="generated-apps">Generated apps output</Label>
            <div class="flex gap-2">
              <Skeleton
                v-if="settingsLoading || settingsPickerField === 'generatedAppsPath'"
                class="h-9 w-full"
              />
              <Input
                v-else
                id="generated-apps"
                :model-value="settingsForm.generatedAppsPath"
                @update:model-value="emit('update-setting', 'generatedAppsPath', String($event))"
              />
              <Button
                type="button"
                variant="outline"
                size="icon"
                :disabled="settingsLoading || settingsPickerField !== null"
                @click="emit('choose-save-file', 'generatedAppsPath')"
              >
                <Save class="size-4" />
                <span class="sr-only">Choose generated apps output</span>
              </Button>
            </div>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle class="text-sm">Storage</CardTitle>
        </CardHeader>
        <CardContent class="grid gap-3">
          <div class="grid gap-2">
            <Label for="workflow-db">Workflow SQLite database</Label>
            <div class="flex gap-2">
              <Skeleton
                v-if="settingsLoading || settingsPickerField === 'workflowDbPath'"
                class="h-9 w-full"
              />
              <Input
                v-else
                id="workflow-db"
                :model-value="settingsForm.workflowDbPath"
                data-testid="settings-workflow-db"
                @update:model-value="emit('update-setting', 'workflowDbPath', String($event))"
              />
              <Button
                type="button"
                variant="outline"
                size="icon"
                :disabled="settingsLoading || settingsPickerField !== null"
                @click="emit('choose-save-file', 'workflowDbPath')"
              >
                <Database class="size-4" />
                <span class="sr-only">Choose workflow database</span>
              </Button>
            </div>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle class="text-sm">Operations</CardTitle>
        </CardHeader>
        <CardContent class="grid gap-3">
          <div class="grid gap-2">
            <Label for="archive-root">Archive root</Label>
            <div class="flex gap-2">
              <Skeleton
                v-if="settingsLoading || settingsPickerField === 'archiveRoot'"
                class="h-9 w-full"
              />
              <Input
                v-else
                id="archive-root"
                :model-value="settingsForm.archiveRoot"
                @update:model-value="emit('update-setting', 'archiveRoot', String($event))"
              />
              <Button
                type="button"
                variant="outline"
                size="icon"
                :disabled="settingsLoading || settingsPickerField !== null"
                @click="emit('choose-directory', 'archiveRoot')"
              >
                <FolderOpen class="size-4" />
                <span class="sr-only">Choose archive root</span>
              </Button>
            </div>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle class="text-sm">1Password</CardTitle>
        </CardHeader>
        <CardContent class="grid gap-3">
          <div
            v-if="opVaultsError"
            class="flex items-start justify-between gap-2 rounded-md border border-destructive/40 bg-destructive/10 px-3 py-2 text-xs text-destructive"
          >
            <div class="flex items-start gap-2">
              <AlertTriangle class="mt-0.5 size-4 shrink-0" />
              <span>{{ opVaultsError }}</span>
            </div>
            <div class="flex shrink-0 items-center gap-1">
              <Button
                type="button"
                variant="ghost"
                size="sm"
                :disabled="opInstallLoading"
                @click="emit('install-op-dependencies')"
              >
                <Loader2 v-if="opInstallLoading" class="size-3.5 animate-spin" />
                <Download v-else class="size-3.5" />
                Install
              </Button>
              <Button
                type="button"
                variant="ghost"
                size="sm"
                :disabled="opSigninLoading"
                @click="emit('signin-op-cli')"
              >
                <Loader2 v-if="opSigninLoading" class="size-3.5 animate-spin" />
                <KeyRound v-else class="size-3.5" />
                Sign in
              </Button>
              <Button
                type="button"
                variant="ghost"
                size="sm"
                :disabled="opVaultsLoading"
                @click="emit('load-op-vaults')"
              >
                <Loader2 v-if="opVaultsLoading" class="size-3.5 animate-spin" />
                <RefreshCw v-else class="size-3.5" />
                Retry
              </Button>
            </div>
          </div>
          <div class="grid grid-cols-2 gap-3">
            <div class="grid gap-2">
              <Label for="op-vault">1Password vault</Label>
              <Skeleton v-if="settingsLoading || opVaultsLoading" class="h-9 w-full" />
              <div v-else class="flex gap-2">
                <Select
                  :model-value="settingsForm.opVault"
                  :disabled="opVaultsError !== ''"
                  @update:model-value="emit('op-vault-change', $event)"
                >
                  <SelectTrigger id="op-vault" class="flex-1" data-testid="settings-op-vault">
                    <SelectValue placeholder="Select a vault" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem
                      v-for="option in opVaultOptions"
                      :key="option.value"
                      :value="option.value"
                    >
                      {{ option.label }}
                    </SelectItem>
                  </SelectContent>
                </Select>
                <Button
                  type="button"
                  variant="outline"
                  size="icon"
                  :disabled="opVaultsLoading"
                  @click="emit('load-op-vaults')"
                >
                  <RefreshCw class="size-4" />
                  <span class="sr-only">Refresh vaults</span>
                </Button>
              </div>
            </div>
            <div class="grid gap-2">
              <Label for="op-item">1Password item</Label>
              <Skeleton v-if="settingsLoading || opItemsLoading" class="h-9 w-full" />
              <Select
                v-else
                :model-value="settingsForm.opItem"
                :disabled="opItemSelectDisabled"
                @update:model-value="emit('op-item-change', $event)"
              >
                <SelectTrigger id="op-item" data-testid="settings-op-item">
                  <SelectValue
                    :placeholder="settingsForm.opVault ? 'Select an item' : 'Pick a vault first'"
                  />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem
                    v-for="option in opItemOptions"
                    :key="option.value"
                    :value="option.value"
                  >
                    {{ option.label }}
                  </SelectItem>
                </SelectContent>
              </Select>
              <p
                v-if="opItemsError && opItemsLoadedFor === settingsForm.opVault"
                class="text-xs text-destructive"
              >
                {{ opItemsError }}
              </p>
            </div>
          </div>
          <div class="overflow-hidden rounded-lg border border-section-border bg-section-muted">
            <div
              class="border-b border-section-border px-3 py-2 text-xs font-medium text-muted-foreground"
            >
              Saved fields
            </div>
            <div
              v-for="field in opSavedFields"
              :key="field.key"
              class="grid grid-cols-[8rem_minmax(0,1fr)_minmax(0,1fr)] gap-3 border-b border-section-border px-3 py-2 text-xs last:border-b-0"
            >
              <div class="font-medium">{{ field.label }}</div>
              <div class="min-w-0">
                <div class="text-muted-foreground">Saved</div>
                <div class="truncate">{{ field.saved || "not set" }}</div>
              </div>
              <div class="min-w-0">
                <div class="text-muted-foreground">Pending</div>
                <div class="truncate">{{ field.pending || "not set" }}</div>
              </div>
            </div>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader class="flex flex-row items-center justify-between gap-3 space-y-0">
          <CardTitle class="text-sm">Validation</CardTitle>
          <Button
            type="button"
            variant="ghost"
            size="sm"
            :disabled="settingsLoading || settingsValidating || settingsSaving"
            @click="emit('validate-settings')"
          >
            <Loader2 v-if="settingsValidating" class="size-4 animate-spin" />
            Validate
          </Button>
        </CardHeader>
        <CardContent class="grid gap-3">
          <div class="overflow-hidden rounded-lg border border-section-border bg-section-muted">
            <template v-if="settingsLoading || settingsValidating || settingsSaving">
              <div
                v-for="i in 4"
                :key="`skeleton-check-${i}`"
                class="grid gap-2 border-b px-3 py-3 last:border-b-0"
                data-testid="settings-checks-skeleton"
              >
                <div class="flex items-center gap-2">
                  <Skeleton class="h-4 w-32" />
                  <Skeleton class="ml-auto h-5 w-12" />
                </div>
                <Skeleton class="h-3 w-3/4" />
              </div>
            </template>
            <Table v-else>
              <TableHeader>
                <TableRow>
                  <TableHead>Check</TableHead>
                  <TableHead>Path</TableHead>
                  <TableHead class="w-24 text-right">Status</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                <TableRow v-for="check in settingsChecks" :key="check.key">
                  <TableCell>
                    <div class="font-medium">{{ check.label }}</div>
                    <div
                      v-if="check.message && check.message !== 'ok'"
                      class="mt-1 text-xs text-destructive"
                    >
                      {{ check.message }}
                    </div>
                  </TableCell>
                  <TableCell class="max-w-0 truncate text-xs text-muted-foreground">{{
                    check.path
                  }}</TableCell>
                  <TableCell class="text-right">
                    <StatusBadge :status="check.status" />
                  </TableCell>
                </TableRow>
              </TableBody>
            </Table>
          </div>
          <div
            v-if="settingsError"
            class="rounded-lg border border-destructive/40 bg-section-muted p-3 text-sm text-destructive"
          >
            {{ settingsError }}
          </div>
          <div
            v-if="settingsMessage"
            class="rounded-lg border border-section-border bg-section-muted p-3 text-sm text-muted-foreground"
          >
            {{ settingsMessage }}
          </div>
        </CardContent>
      </Card>
    </div>
  </ScrollArea>
  <Separator />
  <div class="flex items-center gap-2 border-t border-section-border bg-section p-4">
    <Button
      type="button"
      variant="outline"
      :disabled="!settingsDirty || settingsSaving"
      @click="emit('reset-settings')"
    >
      Reset
    </Button>
    <Button
      type="button"
      class="ml-auto"
      :disabled="!settingsDirty || settingsSaving"
      @click="emit('request-save-settings')"
    >
      <Loader2 v-if="settingsSaving" class="size-4 animate-spin" />
      <Save v-else class="size-4" />
      Save settings
    </Button>
  </div>
</template>
