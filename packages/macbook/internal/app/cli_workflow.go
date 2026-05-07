package app

import (
	"fmt"
	"strings"

	"github.com/gocanto/mac-os/internal/workflowdomain"
)

func (a app) listWorkflows() int {
	for _, workflow := range a.workflows() {
		fmt.Fprintf(a.stdout, "%s\t%s\n", workflow.ID, workflow.Name)
	}

	return 0
}

func (a app) runWorkflowCLI(args []string) int {
	preview := false
	id := ""

	for _, arg := range args {
		switch {
		case arg == "--preview":
			preview = true
		case strings.HasPrefix(arg, "-"):
			fmt.Fprintf(a.stderr, "unknown flag %q\n", arg)

			return 2
		case id == "":
			id = arg
		default:
			fmt.Fprintf(a.stderr, "unexpected argument %q\n", arg)

			return 2
		}
	}

	if id == "" {
		fmt.Fprintln(a.stderr, "usage: mac-os run-workflow <id> [--preview]")

		return 2
	}

	workflow, err := workflowdomain.Find(a.workflows(), id)

	if err != nil {
		fmt.Fprintf(a.stderr, "%v\n", err)

		return 2
	}

	phases, err := selectWorkflowPhases(workflow, preview)

	if err != nil {
		fmt.Fprintf(a.stderr, "%v\n", err)

		return 2
	}

	for _, phase := range phases {
		if !phase.Enabled {
			fmt.Fprintf(a.stdout, "skipped: %s\n", phase.Name)

			continue
		}

		fmt.Fprintf(a.stdout, "==> %s\n", phase.Name)

		if err := phase.Run(a.stdout); err != nil {
			fmt.Fprintf(a.stderr, "phase %q failed: %v\n", phase.Name, err)

			return 1
		}
	}

	return 0
}

func selectWorkflowPhases(workflow workflowdomain.Workflow, preview bool) ([]workflowdomain.Phase, error) {
	if workflow.Confirmation == nil {
		return workflow.Phases, nil
	}

	wantedID := "run-now"

	if preview {
		wantedID = workflowdomain.ConfirmationOptionPreviewOnly
	}

	for _, option := range workflow.Confirmation.Options {
		if option.ID == wantedID {
			return option.Phases, nil
		}
	}

	return nil, fmt.Errorf("workflow %q has no %q option", workflow.ID, wantedID)
}
