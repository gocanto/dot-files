<script setup lang="ts">
import {
  AlertTriangle,
  Database,
  Download,
  KeyRound,
  Loader2,
  RefreshCw,
  Save,
  Settings2,
  ShieldCheck,
} from "lucide-vue-next";
import { nextTick, ref, watch } from "vue";
import { Avatar, AvatarFallback } from "@ui/avatar";
import { Button } from "@ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@ui/card";
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

const props = defineProps<{
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
  opManageActive: boolean;
  opSavedFields: SavedField[];
  scrollTarget: { id: string; nonce: number } | null;
}>();

const highlightedId = ref<string>("");
let highlightTimer: ReturnType<typeof setTimeout> | null = null;

watch(
  () => props.scrollTarget,
  async (target) => {
    if (!target) {
      return;
    }

    await nextTick();

    const element = document.getElementById(`settings-section-${target.id}`);
    if (!element) {
      return;
    }

    element.scrollIntoView({ block: "start", behavior: "smooth" });
    highlightedId.value = target.id;

    if (highlightTimer !== null) {
      clearTimeout(highlightTimer);
    }
    highlightTimer = setTimeout(() => {
      highlightedId.value = "";
      highlightTimer = null;
    }, 1200);
  },
);

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
  (event: "manage-op"): void;
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
      <Card
        id="settings-section-repository"
        :class="
          highlightedId === 'repository'
            ? 'ring-2 ring-ring ring-offset-2 transition-shadow'
            : 'transition-shadow'
        "
      >
        <CardHeader>
          <CardTitle class="text-sm">Repository</CardTitle>
          <CardDescription class="text-xs">
            The dotfiles checkout this Mac is bound to. All other paths and the workflow bridge
            resolve relative to this directory. Configured by the install script and shown read-only
            here so it can't be edited by accident.
          </CardDescription>
        </CardHeader>
        <CardContent class="grid gap-3">
          <div class="grid gap-2">
            <Label for="repo-root">Repo root</Label>
            <Skeleton
              v-if="settingsLoading || settingsPickerField === 'repoRoot'"
              class="h-9 w-full"
            />
            <Input
              v-else
              id="repo-root"
              :model-value="settingsForm.repoRoot"
              data-testid="settings-repo-root"
              readonly
              class="cursor-not-allowed bg-muted"
            />
          </div>
        </CardContent>
      </Card>

      <Card
        id="settings-section-manifests"
        :class="
          highlightedId === 'manifests'
            ? 'ring-2 ring-ring ring-offset-2 transition-shadow'
            : 'transition-shadow'
        "
      >
        <CardHeader>
          <CardTitle class="text-sm">Manifests</CardTitle>
          <CardDescription class="text-xs">
            The YAML files that declare which apps and secret references are tracked, plus the
            generated apps output that workflows compile to. These live in the repo, so they're
            shown read-only — edit them in source.
          </CardDescription>
        </CardHeader>
        <CardContent class="grid gap-3">
          <div class="grid gap-2">
            <Label for="apps-config">Apps manifest</Label>
            <Skeleton
              v-if="settingsLoading || settingsPickerField === 'appsConfigPath'"
              class="h-9 w-full"
            />
            <Input
              v-else
              id="apps-config"
              :model-value="settingsForm.appsConfigPath"
              data-testid="settings-apps-config"
              readonly
              class="cursor-not-allowed bg-muted"
            />
          </div>
          <div class="grid gap-2">
            <Label for="secrets-config">Secrets manifest</Label>
            <Skeleton
              v-if="settingsLoading || settingsPickerField === 'secretsConfigPath'"
              class="h-9 w-full"
            />
            <Input
              v-else
              id="secrets-config"
              :model-value="settingsForm.secretsConfigPath"
              readonly
              class="cursor-not-allowed bg-muted"
            />
          </div>
          <div class="grid gap-2">
            <Label for="generated-apps">Generated apps output</Label>
            <Skeleton
              v-if="settingsLoading || settingsPickerField === 'generatedAppsPath'"
              class="h-9 w-full"
            />
            <Input
              v-else
              id="generated-apps"
              :model-value="settingsForm.generatedAppsPath"
              readonly
              class="cursor-not-allowed bg-muted"
            />
          </div>
        </CardContent>
      </Card>

      <Card
        id="settings-section-storage"
        :class="
          highlightedId === 'storage'
            ? 'ring-2 ring-ring ring-offset-2 transition-shadow'
            : 'transition-shadow'
        "
      >
        <CardHeader>
          <CardTitle class="text-sm">Storage</CardTitle>
          <CardDescription class="text-xs">
            Where workflow runs and logs are persisted. This is the only path you can edit here —
            point it at a SQLite file outside the repo so history survives across checkouts. Saving
            moves the bridge to the new location.
          </CardDescription>
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

      <Card
        id="settings-section-operations"
        :class="
          highlightedId === 'operations'
            ? 'ring-2 ring-ring ring-offset-2 transition-shadow'
            : 'transition-shadow'
        "
      >
        <CardHeader>
          <CardTitle class="text-sm">Operations</CardTitle>
          <CardDescription class="text-xs">
            Where snapshots and one-off backups are written by apply workflows. Lives outside the
            repo and is set during install — shown read-only so a mistyped path doesn't silently
            redirect future archives.
          </CardDescription>
        </CardHeader>
        <CardContent class="grid gap-3">
          <div class="grid gap-2">
            <Label for="archive-root">Archive root</Label>
            <Skeleton
              v-if="settingsLoading || settingsPickerField === 'archiveRoot'"
              class="h-9 w-full"
            />
            <Input
              v-else
              id="archive-root"
              :model-value="settingsForm.archiveRoot"
              readonly
              class="cursor-not-allowed bg-muted"
            />
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader class="flex flex-row items-start justify-between gap-3 space-y-0">
          <div class="grid gap-1">
            <CardTitle class="text-sm">1Password</CardTitle>
            <CardDescription class="text-xs">
              The vault and item the workflow bridge reads secret references from when applying
              templates. Click Manage to sign in to the 1Password CLI and pick which vault/item to
              bind — leave empty if you're not using 1Password-backed secrets.
            </CardDescription>
          </div>
          <Button
            v-if="!opManageActive"
            type="button"
            variant="ghost"
            size="sm"
            data-testid="settings-op-manage"
            @click="emit('manage-op')"
          >
            <Settings2 class="size-3.5" />
            Manage
          </Button>
        </CardHeader>
        <CardContent class="grid gap-3">
          <div
            v-if="opManageActive && opVaultsError"
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
          <div v-if="opManageActive" class="grid grid-cols-2 gap-3">
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
        <CardHeader class="flex flex-row items-start justify-between gap-3 space-y-0">
          <div class="grid gap-1">
            <CardTitle class="text-sm">Validation</CardTitle>
            <CardDescription class="text-xs">
              Checks that every path in your settings exists and is usable — the repository, stow
              directory, manifests, archive, workflow database, and private Git config. Run this
              after changing a path or moving files to confirm the bridge can still find everything.
            </CardDescription>
          </div>
          <Button
            type="button"
            variant="ghost"
            size="sm"
            :disabled="settingsLoading || settingsValidating || settingsSaving"
            @click="emit('validate-settings')"
          >
            <Loader2 v-if="settingsValidating" class="size-4 animate-spin" />
            <ShieldCheck v-else class="size-4" />
            Validate
          </Button>
        </CardHeader>
        <CardContent class="grid gap-3">
          <div class="overflow-hidden rounded-lg border border-section-border bg-section-muted">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Check</TableHead>
                  <TableHead>Path</TableHead>
                  <TableHead class="w-24 text-right">Status</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody v-if="settingsLoading || settingsValidating || settingsSaving">
                <TableRow
                  v-for="i in settingsChecks.length || 8"
                  :key="`skeleton-check-${i}`"
                  data-testid="settings-checks-skeleton"
                >
                  <TableCell>
                    <Skeleton class="h-4 w-32" />
                  </TableCell>
                  <TableCell><Skeleton class="h-3 w-3/4" /></TableCell>
                  <TableCell class="text-right"><Skeleton class="ml-auto h-5 w-12" /></TableCell>
                </TableRow>
              </TableBody>
              <TableBody v-else>
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
                  <TableCell class="max-w-0 text-xs text-muted-foreground">
                    <Tooltip>
                      <TooltipTrigger as-child>
                        <div class="truncate">{{ check.path }}</div>
                      </TooltipTrigger>
                      <TooltipContent side="top" align="start" class="max-w-[90vw] break-all">
                        {{ check.path }}
                      </TooltipContent>
                    </Tooltip>
                  </TableCell>
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
