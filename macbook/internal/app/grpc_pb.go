package app

import (
	"github.com/gocanto/mac-os/internal/bridgepb"
	"github.com/gocanto/mac-os/internal/storage"
	"github.com/gocanto/mac-os/internal/workflowdomain"
)

func pbWorkflows(workflows []workflowdomain.WorkflowMetadata) []*bridgepb.Workflow {
	items := make([]*bridgepb.Workflow, 0, len(workflows))

	for _, workflow := range workflows {
		item := &bridgepb.Workflow{
			Id:          workflow.ID,
			Name:        workflow.Name,
			Description: workflow.Description,
			ChangesMac:  workflow.ChangesMac,
			Phases:      pbPhases(workflow.Phases),
		}

		if workflow.Confirmation != nil {
			item.Confirmation = &bridgepb.Confirmation{
				Title:   workflow.Confirmation.Title,
				Message: workflow.Confirmation.Message,
				Options: pbConfirmationOptions(workflow.Confirmation.Options),
			}
		}

		items = append(items, item)
	}

	return items
}

func pbPhases(phases []workflowdomain.PhaseMetadata) []*bridgepb.Phase {
	items := make([]*bridgepb.Phase, 0, len(phases))

	for _, phase := range phases {
		items = append(items, &bridgepb.Phase{Id: phase.ID, Name: phase.Name, Enabled: phase.Enabled})
	}

	return items
}

func pbConfirmationOptions(options []workflowdomain.ConfirmationOptionMetadata) []*bridgepb.ConfirmationOption {
	items := make([]*bridgepb.ConfirmationOption, 0, len(options))

	for _, option := range options {
		items = append(items, &bridgepb.ConfirmationOption{
			Id:          option.ID,
			Label:       option.Label,
			Description: option.Description,
			Continue:    option.Continue,
			Back:        option.Back,
			Phases:      pbPhases(option.Phases),
		})
	}

	return items
}

func pbRunSummaries(runs []storage.RunSummary) []*bridgepb.RunSummary {
	items := make([]*bridgepb.RunSummary, 0, len(runs))

	for _, run := range runs {
		items = append(items, pbRunSummary(run))
	}

	return items
}

func pbRunSummary(run storage.RunSummary) *bridgepb.RunSummary {
	return &bridgepb.RunSummary{
		Id:                      run.ID,
		WorkflowId:              run.WorkflowID,
		WorkflowName:            run.WorkflowName,
		ConfirmationOptionId:    run.ConfirmationOptionID,
		ConfirmationOptionLabel: run.ConfirmationOptionLabel,
		Mode:                    run.Mode,
		Status:                  run.Status,
		StartedAt:               run.StartedAt,
		CompletedAt:             run.CompletedAt,
		ErrorMessage:            run.ErrorMessage,
	}
}

func pbEventRecords(events []storage.EventRecord) []*bridgepb.WorkflowEvent {
	items := make([]*bridgepb.WorkflowEvent, 0, len(events))

	for _, event := range events {
		items = append(items, &bridgepb.WorkflowEvent{
			Id:        event.ID,
			RunId:     event.RunID,
			Seq:       event.Seq,
			Type:      event.Type,
			PhaseId:   event.PhaseID,
			PhaseName: event.PhaseName,
			Status:    event.Status,
			Message:   event.Message,
			CreatedAt: event.CreatedAt,
		})
	}

	return items
}

func pbWorkflowEvent(event workflowdomain.Event) *bridgepb.WorkflowEvent {
	return &bridgepb.WorkflowEvent{
		RunId:     event.RunID,
		Seq:       event.Seq,
		Type:      event.Type,
		PhaseId:   event.PhaseID,
		PhaseName: event.PhaseName,
		Status:    event.Status,
		Message:   event.Message,
	}
}

func pbRuntimeSettings(settings runtimeSettings) *bridgepb.RuntimeSettings {
	return &bridgepb.RuntimeSettings{
		RepoRoot:          settings.RepoRoot,
		AppsConfigPath:    settings.AppsConfigPath,
		SecretsConfigPath: settings.SecretsConfigPath,
		GeneratedAppsPath: settings.GeneratedAppsPath,
		ArchiveRoot:       settings.ArchiveRoot,
		WorkflowDbPath:    settings.WorkflowDBPath,
		OpVault:           settings.OPVault,
		OpItem:            settings.OPItem,
	}
}

func runtimeSettingsFromPB(settings *bridgepb.RuntimeSettings) runtimeSettings {
	if settings == nil {
		return runtimeSettings{}
	}

	return runtimeSettings{
		RepoRoot:          settings.GetRepoRoot(),
		AppsConfigPath:    settings.GetAppsConfigPath(),
		SecretsConfigPath: settings.GetSecretsConfigPath(),
		GeneratedAppsPath: settings.GetGeneratedAppsPath(),
		ArchiveRoot:       settings.GetArchiveRoot(),
		WorkflowDBPath:    settings.GetWorkflowDbPath(),
		OPVault:           settings.GetOpVault(),
		OPItem:            settings.GetOpItem(),
	}
}

func pbSettingsChecks(checks []settingsCheck) []*bridgepb.SettingsCheck {
	items := make([]*bridgepb.SettingsCheck, 0, len(checks))

	for _, check := range checks {
		items = append(items, &bridgepb.SettingsCheck{
			Key:     check.Key,
			Label:   check.Label,
			Path:    check.Path,
			Status:  check.Status,
			Message: check.Message,
		})
	}

	return items
}
