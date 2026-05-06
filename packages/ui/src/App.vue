<script setup lang="ts">
import { AlertTriangle, Inbox } from "lucide-vue-next";
import AppSidebar from "@/components/app/AppSidebar.vue";
import ConfirmationDialog from "@/components/app/ConfirmationDialog.vue";
import DetailToolbar from "@/components/app/DetailToolbar.vue";
import EmptyDetail from "@/components/app/EmptyDetail.vue";
import InitialLoadingSkeleton from "@/components/app/InitialLoadingSkeleton.vue";
import LogsListPanel from "@/components/app/LogsListPanel.vue";
import RunLogDetail from "@/components/app/RunLogDetail.vue";
import SettingsForm from "@/components/app/SettingsForm.vue";
import SettingsListPanel from "@/components/app/SettingsListPanel.vue";
import StepSettingDetail from "@/components/app/StepSettingDetail.vue";
import TemplateFilesDetail from "@/components/app/TemplateFilesDetail.vue";
import WorkflowDetail from "@/components/app/WorkflowDetail.vue";
import WorkflowDetailSkeleton from "@/components/app/WorkflowDetailSkeleton.vue";
import WorkflowListPanel from "@/components/app/WorkflowListPanel.vue";
import { panelFrameClass } from "@/components/app/styles";
import { ResizableHandle, ResizablePanel, ResizablePanelGroup } from "@/components/ui/resizable";
import { Separator } from "@/components/ui/separator";
import { ToastViewport } from "@/components/ui/toast";
import { TooltipProvider } from "@/components/ui/tooltip";
import { useAppController } from "@/composables/useAppController";
import { cn } from "@/lib/utils";

const {
  theme,
  toggleTheme,
  section,
  selectedSettingsKey,
  selectedTemplateFiles,
  macName,
  macHostname,
  selectedWorkflowId,
  selectedRunId,
  selectedRunLog,
  pendingOption,
  running,
  workflowsLoading,
  runsLoading,
  initialLoading,
  runLogLoading,
  loadError,
  navCollapsed,
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
  opVaultsLoading,
  opItemsLoading,
  opVaultsError,
  opItemsError,
  opItemsLoadedFor,
  opSigninLoading,
  opInstallLoading,
  toasts,
  templateFiles,
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
  settingsKeyLabels,
  settingsWorkflows,
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
  runStatus,
  outputText,
  outputSections,
  workflowProgress,
  selectedRunOutputSections,
  detailPaneOpen,
  selectSection,
  selectStepSetting,
  selectTemplateFiles,
  loadTemplateFiles,
  selectTemplateFile,
  saveTemplateFile,
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
  validateSettings,
  saveSettings,
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

      <ResizablePanelGroup
        v-else
        direction="horizontal"
        class="h-screen max-h-screen items-stretch"
      >
        <ResizablePanel
          id="mac-nav"
          :default-size="18"
          :collapsed-size="4"
          collapsible
          :min-size="14"
          :max-size="22"
          :class="cn(navCollapsed && 'min-w-12 transition-all duration-300 ease-in-out')"
          @collapse="navCollapsed = true"
          @expand="navCollapsed = false"
        >
          <AppSidebar
            :collapsed="navCollapsed"
            :section="section"
            :mac-name="macName"
            :mac-hostname="macHostname"
            :theme="theme"
            :step-nav-items="stepNavItems"
            :aux-nav-items="auxNavItems"
            @select-section="selectSection"
            @toggle-theme="toggleTheme"
          />
        </ResizablePanel>

        <ResizableHandle with-handle />

        <ResizablePanel id="mac-list" :default-size="32" :min-size="28">
          <div :class="cn('flex h-full min-h-0 flex-col', panelFrameClass)">
            <WorkflowListPanel
              v-if="stepMeta"
              v-model:search-query="searchQuery"
              :step-meta="stepMeta"
              :workflows="matchingWorkflows"
              :selected-workflow-id="selectedWorkflowId"
              :selected-settings-key="selectedSettingsKey"
              :selected-template-files="selectedTemplateFiles"
              :workflows-loading="workflowsLoading"
              :settings-loading="settingsLoading"
              :settings-response="settingsResponse"
              :settings-form="settingsForm"
              :settings-key-labels="settingsKeyLabels"
              @select-workflow="selectWorkflow"
              @select-step-setting="selectStepSetting"
              @select-template-files="selectTemplateFiles"
              @open-devtools="openDevTools"
            />

            <LogsListPanel
              v-else-if="section === 'logs'"
              v-model:log-tab="logTab"
              v-model:search-query="searchQuery"
              :runs="matchingRuns"
              :selected-run-id="selectedRunId"
              :runs-loading="runsLoading"
              @open-run="openRun"
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
          </div>
        </ResizablePanel>

        <ResizableHandle v-if="detailPaneOpen" with-handle />

        <ResizablePanel v-if="detailPaneOpen" id="mac-detail" :default-size="50" :min-size="35">
          <div class="flex h-full min-h-0 flex-col bg-background">
            <DetailToolbar
              :has-step-meta="Boolean(stepMeta)"
              :selected-workflow="selectedWorkflow"
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
                <p class="mt-2 text-sm text-muted-foreground">{{ loadError }}</p>
              </div>
            </div>

            <StepSettingDetail
              v-else-if="stepMeta && selectedSettingsKey"
              :selected-settings-key="selectedSettingsKey"
              :settings-key-labels="settingsKeyLabels"
              :step-title="stepMeta.title"
              :settings-form="settingsForm"
              :settings-response="settingsResponse"
              @update-setting="updateSetting"
              @choose-directory="chooseDirectory"
            />

            <TemplateFilesDetail
              v-else-if="stepMeta && selectedTemplateFiles"
              v-model:draft="templateFileDraft"
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
            />

            <WorkflowDetailSkeleton v-else-if="stepMeta && workflowsLoading && !selectedWorkflow" />

            <template v-else-if="stepMeta && !selectedWorkflow">
              <div
                class="grid flex-1 place-items-center p-8 text-center text-sm text-muted-foreground"
              >
                <div>
                  <Inbox class="mx-auto mb-3 size-8" />
                  <p>Select a workflow or a step setting to begin.</p>
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
              @close-detail="closeDetailPane"
            />

            <RunLogDetail
              v-else-if="section === 'logs'"
              :run-log-loading="runLogLoading"
              :selected-run-log="selectedRunLog"
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
              @save-settings="saveSettings"
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
  </TooltipProvider>
</template>
