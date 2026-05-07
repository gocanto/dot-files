package workflowdomain

import (
	"fmt"
	"slices"
)

func BuildRunPlan(workflows []Workflow, req RunRequest) (RunPlan, error) {
	workflow, err := Find(workflows, req.WorkflowID)

	if err != nil {
		return RunPlan{}, err
	}

	plan := RunPlan{Workflow: workflow, Phases: clonePhases(workflow.Phases), Mode: RunModeLive}

	if workflow.Confirmation != nil {
		option, err := findOption(workflow.Confirmation.Options, req.ConfirmationOptionID)

		if err != nil {
			return RunPlan{}, err
		}

		plan.ConfirmationOption = option

		if option.Phases != nil {
			plan.Phases = clonePhases(option.Phases)
		}

		switch {
		case option.Back:
			return RunPlan{}, fmt.Errorf("confirmation option %q goes back and cannot run", option.ID)
		case !option.Continue:
			plan.Mode = RunModeStopBeforeRun
		case option.ID == ConfirmationOptionPreviewOnly:
			plan.Mode = RunModePreview
		default:
			plan.Mode = RunModeLive
		}
	}

	enabledIDs := map[string]bool{}

	for _, id := range req.EnabledPhaseIDs {
		enabledIDs[id] = true
	}

	if len(enabledIDs) > 0 {
		for index := range plan.Phases {
			plan.Phases[index].Enabled = enabledIDs[plan.Phases[index].ID]
		}
	}

	return plan, nil
}

func findOption(options []ConfirmationOption, id string) (*ConfirmationOption, error) {
	if id == "" && len(options) > 0 {
		return &options[0], nil
	}

	for index := range options {
		if options[index].ID == id {
			return &options[index], nil
		}
	}

	return nil, fmt.Errorf("unknown confirmation option %q", id)
}

func clonePhases(phases []Phase) []Phase {
	return slices.Clone(phases)
}
