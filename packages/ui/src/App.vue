<script setup lang="ts">
import { AlertTriangle, Inbox } from "lucide-vue-next";
import { defineAsyncComponent } from "vue";
import AppSidebar from "@app/AppSidebar.vue";
import ConfirmationDialog from "@app/ConfirmationDialog.vue";
import DetailToolbar from "@app/DetailToolbar.vue";
import EmptyDetail from "@app/EmptyDetail.vue";
import InitialLoadingSkeleton from "@app/InitialLoadingSkeleton.vue";
import LogsListPanel from "@app/LogsListPanel.vue";
import RunLogDetail from "@app/RunLogDetail.vue";
import SettingsForm from "@app/SettingsForm.vue";
import SettingsListPanel from "@app/SettingsListPanel.vue";
import SettingsSaveDialog from "@app/SettingsSaveDialog.vue";
import SystemStatusPanel from "@app/SystemStatusPanel.vue";
import WorkflowDetail from "@app/WorkflowDetail.vue";
import WorkflowDetailSkeleton from "@app/WorkflowDetailSkeleton.vue";
import WorkflowListPanel from "@app/WorkflowListPanel.vue";
import { panelFrameClass } from "@app/styles";
import { ResizableHandle, ResizablePanel, ResizablePanelGroup } from "@ui/resizable";
import { Separator } from "@ui/separator";
import { ToastViewport } from "@ui/toast";
import { TooltipProvider } from "@ui/tooltip";
import { useAppController } from "@composables/useAppController";
import { cn } from "@lib/utils";

const TemplateFilesDetail = defineAsyncComponent(() =>
  import("@app/TemplateFilesDetail.vue").then((module) => module.default),
);

const {
  theme,
  toggleTheme,
  section,
  selectedTemplateFiles,
  macName,
  macHostname,
  macSystemInfo,
  selectedWorkflowId,
  selectedRunId,
  selectedRunLog,
  selectedAppDiagnosticId,
  pendingOption,
  running,
  workflowsLoading,
  runsLoading,
  initialLoading,
  runLogLoading,
  loadError,
  searchQuery,
  logTab,
  settingsResponse,
  settingsForm,
  settingsSaving,
  settingsLoading,
  settingsValidating,
  settingsPickerField,
  settingsMessage,
  settingsError,
  settingsSaveConfirmationOpen,
  opVaultsLoading,
  opItemsLoading,
  opVaultsError,
  opItemsError,
  opItemsLoadedFor,
  opSigninLoading,
  opInstallLoading,
  toasts,
  templateFiles,
  templateFilesLoaded,
  templateFilesLoading,
  templateFileContentLoading,
  templateFileSaving,
  selectedTemplateFilePath,
  templateFileDraft,
  templateFileError,
  templateFileMessage,
  stepNavItems,
  auxNavItems,
  stepMeta,
  settingsWorkflows,
  workflows,
  selectedWorkflow,
  selectedWorkflowDetail,
  settingsDirty,
  settingsChecks,
  opVaultOptions,
  opItemOptions,
  opItemSelectDisabled,
  opSavedFields,
  settingsGroups,
  displayPhases,
  progressDialogPhases,
  templateFileDirty,
  matchingWorkflows,
  matchingRuns,
  matchingAppDiagnostics,
  runStatus,
  outputText,
  outputSections,
  workflowProgress,
  selectedRunOutputSections,
  selectedAppDiagnostic,
  detailPaneOpen,
  loadAll,
  selectSection,
  selectTemplateFiles,
  closeTemplateFiles,
  loadTemplateFiles,
  selectTemplateFile,
  saveTemplateFile,
  cancelTemplateFileEdit,
  loadOpVaults,
  onOpVaultChange,
  onOpItemChange,
  signinOpCli,
  installOpDependencies,
  openDevTools,
  selectWorkflow,
  closeDetailPane,
  resetEnabledPhases,
  togglePhase,
  openConfirmation,
  updateConfirmationOpen,
  runSelected,
  openRun,
  openAppDiagnostic,
  validateSettings,
  requestSaveSettings,
  updateSettingsSaveConfirmationOpen,
  confirmSaveSettings,
  dismissToast,
  resetSettingsForm,
  updateSetting,
  chooseDirectory,
  chooseFile,
  chooseSaveFile,
} = useAppController();
</script>

<template>
  <TooltipProvider :delay-duration="0">
    <div class="h-screen overflow-hidden bg-background text-foreground">
      <InitialLoadingSkeleton v-if="initialLoading" />

      <div v-else class="flex h-screen max-h-screen items-stretch">
        <aside
          id="mac-nav"
          class="h-full w-[300px] min-w-[300px] max-w-[300px] flex-none border-r border-sidebar-border bg-sidebar"
        >
          <AppSidebar
            :collapsed="false"
            :section="section"
            :mac-name="macName"
            :mac-hostname="macHostname"
            :mac-system-info="macSystemInfo"
            :theme="theme"
            :step-nav-items="stepNavItems"
            :aux-nav-items="auxNavItems"
            @select-section="selectSection"
            @open-devtools="openDevTools"
            @toggle-theme="toggleTheme"
          />
        </aside>

        <main v-if="selectedTemplateFiles" id="template-editor" class="min-w-0 flex-1">
          <div class="flex h-full min-h-0 flex-col bg-background">
            <TemplateFilesDetail
              v-model:draft="templateFileDraft"
              :theme="theme"
              :files="templateFiles"
              :files-loading="templateFilesLoading"
              :content-loading="templateFileContentLoading"
              :saving="templateFileSaving"
              :dirty="templateFileDirty"
              :selected-path="selectedTemplateFilePath"
              :error="templateFileError"
              :message="templateFileMessage"
              @refresh-files="loadTemplateFiles"
              @select-file="selectTemplateFile"
              @save-file="saveTemplateFile"
              @cancel-edit="cancelTemplateFileEdit"
              @back="closeTemplateFiles"
            />
          </div>
        </main>

        <main v-else class="min-w-0 flex-1">
          <ResizablePanelGroup direction="horizontal" class="h-full min-h-0 items-stretch">
            <ResizablePanel
              id="mac-list"
              :default-size="detailPaneOpen ? 38 : 100"
              :min-size="detailPaneOpen ? 28 : 100"
              class="min-w-[320px]"
            >
              <div :class="cn('flex h-full min-h-0 flex-col', panelFrameClass)">
                <WorkflowListPanel
                  v-if="stepMeta"
                  v-model:search-query="searchQuery"
                  :step-meta="stepMeta"
                  :workflows="matchingWorkflows"
                  :selected-workflow-id="selectedWorkflowId"
                  :selected-template-files="selectedTemplateFiles"
                  :template-files-count="templateFiles.length"
                  :template-files-loaded="templateFilesLoaded"
                  :template-files-loading="templateFilesLoading"
                  :workflows-loading="workflowsLoading"
                  @select-workflow="selectWorkflow"
                  @select-template-files="selectTemplateFiles"
                />

                <LogsListPanel
                  v-else-if="section === 'logs'"
                  v-model:log-tab="logTab"
                  v-model:search-query="searchQuery"
                  :runs="matchingRuns"
                  :app-diagnostics="matchingAppDiagnostics"
                  :selected-run-id="selectedRunId"
                  :selected-app-diagnostic-id="selectedAppDiagnosticId"
                  :runs-loading="runsLoading"
                  @open-run="openRun"
                  @open-app-diagnostic="openAppDiagnostic"
                />

                <SettingsListPanel
                  v-else-if="section === 'settings'"
                  :settings-loading="settingsLoading"
                  :settings-response="settingsResponse"
                  :settings-groups="settingsGroups"
                  :workflows-loading="workflowsLoading"
                  :settings-workflows="settingsWorkflows"
                  :selected-workflow-id="selectedWorkflowId"
                  @select-workflow="selectWorkflow"
                />

                <SystemStatusPanel
                  v-else-if="section === 'status'"
                  :settings-loading="settingsLoading"
                  :settings-response="settingsResponse"
                  :workflows="workflows"
                  @refresh="loadAll"
                />
              </div>
            </ResizablePanel>

            <ResizableHandle
              v-if="detailPaneOpen"
              data-testid="workspace-resize-handle"
              with-handle
            />

            <ResizablePanel
              v-if="detailPaneOpen"
              id="mac-detail"
              :default-size="62"
              :min-size="35"
              class="min-w-[420px]"
            >
              <div class="flex h-full min-h-0 flex-col bg-background">
                <DetailToolbar
                  :section="section"
                  :has-step-meta="Boolean(stepMeta)"
                  :selected-workflow="selectedWorkflow"
                  :selected-run-log="selectedRunLog"
                  :selected-app-diagnostic="selectedAppDiagnostic"
                  :run-log-loading="runLogLoading"
                  :run-status="runStatus"
                />

                <Separator />

                <div v-if="loadError" class="grid flex-1 place-items-center p-8">
                  <div
                    class="max-w-xl rounded-lg border border-destructive/40 bg-section p-5 shadow-sm"
                  >
                    <div class="flex items-center gap-2 font-semibold text-destructive">
                      <AlertTriangle class="size-5" />
                      Load failed
                    </div>
                    <p class="mt-2 text-sm text-muted-foreground">
                      {{ loadError }}
                    </p>
                  </div>
                </div>

                <WorkflowDetailSkeleton
                  v-else-if="stepMeta && workflowsLoading && !selectedWorkflow"
                />

                <template v-else-if="stepMeta && !selectedWorkflow">
                  <div
                    class="grid flex-1 place-items-center p-8 text-center text-sm text-muted-foreground"
                  >
                    <div>
                      <Inbox class="mx-auto mb-3 size-8" />
                      <p>Select a workflow to begin.</p>
                    </div>
                  </div>
                </template>

                <WorkflowDetail
                  v-else-if="stepMeta && selectedWorkflow"
                  :selected-workflow="selectedWorkflow"
                  :selected-workflow-detail="selectedWorkflowDetail"
                  :display-phases="displayPhases"
                  :workflow-progress="workflowProgress"
                  @reset-phases="resetEnabledPhases"
                  @toggle-phase="togglePhase"
                  @open-confirmation="openConfirmation"
                  @open-template-files="selectTemplateFiles"
                  @close-detail="closeDetailPane"
                />

                <RunLogDetail
                  v-else-if="section === 'logs'"
                  :run-log-loading="runLogLoading"
                  :selected-run-log="selectedRunLog"
                  :selected-app-diagnostic="selectedAppDiagnostic"
                  :selected-run-output-sections="selectedRunOutputSections"
                />

                <SettingsForm
                  v-else-if="section === 'settings'"
                  :settings-form="settingsForm"
                  :settings-response="settingsResponse"
                  :settings-checks="settingsChecks"
                  :settings-dirty="settingsDirty"
                  :settings-loading="settingsLoading"
                  :settings-saving="settingsSaving"
                  :settings-validating="settingsValidating"
                  :settings-picker-field="settingsPickerField"
                  :settings-error="settingsError"
                  :settings-message="settingsMessage"
                  :op-vault-options="opVaultOptions"
                  :op-item-options="opItemOptions"
                  :op-vaults-error="opVaultsError"
                  :op-items-error="opItemsError"
                  :op-items-loaded-for="opItemsLoadedFor"
                  :op-vaults-loading="opVaultsLoading"
                  :op-items-loading="opItemsLoading"
                  :op-item-select-disabled="opItemSelectDisabled"
                  :op-signin-loading="opSigninLoading"
                  :op-install-loading="opInstallLoading"
                  :op-saved-fields="opSavedFields"
                  @update-setting="updateSetting"
                  @choose-directory="chooseDirectory"
                  @choose-file="chooseFile"
                  @choose-save-file="chooseSaveFile"
                  @validate-settings="validateSettings"
                  @reset-settings="resetSettingsForm"
                  @request-save-settings="requestSaveSettings"
                  @op-vault-change="onOpVaultChange"
                  @op-item-change="onOpItemChange"
                  @signin-op-cli="signinOpCli"
                  @install-op-dependencies="installOpDependencies"
                  @load-op-vaults="loadOpVaults"
                />

                <EmptyDetail v-else :section="section" />
              </div>
            </ResizablePanel>
          </ResizablePanelGroup>
        </main>
      </div>
    </div>

    <ToastViewport :toasts="toasts" @dismiss="dismissToast" />

    <ConfirmationDialog
      :pending-option="pendingOption"
      :title="selectedWorkflow?.confirmation?.title ?? selectedWorkflow?.name ?? ''"
      :summary="selectedWorkflow?.confirmation?.message ?? selectedWorkflow?.description ?? ''"
      :running="running"
      :phases="progressDialogPhases"
      :output-text="outputText"
      :output-sections="outputSections"
      :run-status="runStatus"
      @update:open="updateConfirmationOpen"
      @continue="runSelected"
    />

    <SettingsSaveDialog
      :open="settingsSaveConfirmationOpen"
      :saving="settingsSaving"
      @update:open="updateSettingsSaveConfirmationOpen"
      @confirm="confirmSaveSettings"
    />
  </TooltipProvider>
</template>
