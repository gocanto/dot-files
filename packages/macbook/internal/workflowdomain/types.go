package workflowdomain

import (
	"fmt"
	"io"
	"regexp"
	"strings"
)

type Phase struct {
	ID      string
	Name    string
	Run     func(io.Writer) error
	Enabled bool
}

type Workflow struct {
	ID           string
	Name         string
	Description  string
	ChangesMac   string
	Phases       []Phase
	Confirmation *Confirmation
}

type Confirmation struct {
	Title   string
	Message string
	Options []ConfirmationOption
}

type ConfirmationOption struct {
	ID               string
	Label            string
	Description      string
	Continue         bool
	Back             bool
	RequiresApproval bool
	Phases           []Phase
	Run              func(io.Writer) error
	Approve          func(io.Writer) error
}

type WorkflowMetadata struct {
	ID           string                `json:"id"`
	Name         string                `json:"name"`
	Description  string                `json:"description"`
	ChangesMac   string                `json:"changesMac"`
	Phases       []PhaseMetadata       `json:"phases"`
	Confirmation *ConfirmationMetadata `json:"confirmation,omitempty"`
}

type PhaseMetadata struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
}

type ConfirmationMetadata struct {
	Title   string                       `json:"title"`
	Message string                       `json:"message"`
	Options []ConfirmationOptionMetadata `json:"options"`
}

type ConfirmationOptionMetadata struct {
	ID               string          `json:"id"`
	Label            string          `json:"label"`
	Description      string          `json:"description"`
	Continue         bool            `json:"continue"`
	Back             bool            `json:"back"`
	RequiresApproval bool            `json:"requiresApproval"`
	Phases           []PhaseMetadata `json:"phases,omitempty"`
}

var slugPattern = regexp.MustCompile(`[^a-z0-9]+`)

func Normalize(workflows []Workflow) []Workflow {
	seenWorkflows := map[string]int{}

	for workflowIndex := range workflows {
		workflow := &workflows[workflowIndex]
		workflow.ID = uniqueID(workflow.ID, workflow.Name, seenWorkflows)
		seenPhases := map[string]int{}

		for phaseIndex := range workflow.Phases {
			phase := &workflow.Phases[phaseIndex]
			phase.ID = uniqueID(phase.ID, phase.Name, seenPhases)
		}

		if workflow.Confirmation == nil {
			continue
		}

		seenOptions := map[string]int{}

		for optionIndex := range workflow.Confirmation.Options {
			option := &workflow.Confirmation.Options[optionIndex]
			option.ID = uniqueID(option.ID, option.Label, seenOptions)

			optionSeenPhases := map[string]int{}

			for phaseIndex := range option.Phases {
				phase := &option.Phases[phaseIndex]
				phase.ID = uniqueID(phase.ID, phase.Name, optionSeenPhases)
			}
		}
	}

	return workflows
}

func Metadata(workflows []Workflow) []WorkflowMetadata {
	normalized := Normalize(workflows)
	metadata := make([]WorkflowMetadata, 0, len(normalized))

	for _, workflow := range normalized {
		item := WorkflowMetadata{
			ID:          workflow.ID,
			Name:        workflow.Name,
			Description: workflow.Description,
			ChangesMac:  workflow.ChangesMac,
			Phases:      phaseMetadata(workflow.Phases),
		}

		if workflow.Confirmation != nil {
			item.Confirmation = &ConfirmationMetadata{
				Title:   workflow.Confirmation.Title,
				Message: workflow.Confirmation.Message,
				Options: make([]ConfirmationOptionMetadata, 0, len(workflow.Confirmation.Options)),
			}

			for _, option := range workflow.Confirmation.Options {
				item.Confirmation.Options = append(item.Confirmation.Options, ConfirmationOptionMetadata{
					ID:               option.ID,
					Label:            option.Label,
					Description:      option.Description,
					Continue:         option.Continue,
					Back:             option.Back,
					RequiresApproval: option.RequiresApproval,
					Phases:           phaseMetadata(option.Phases),
				})
			}
		}

		metadata = append(metadata, item)
	}

	return metadata
}

func Find(workflows []Workflow, id string) (*Workflow, error) {
	normalized := Normalize(workflows)

	for i := range normalized {
		if normalized[i].ID == id {
			return &normalized[i], nil
		}
	}

	return nil, fmt.Errorf("unknown workflow %q", id)
}

func phaseMetadata(phases []Phase) []PhaseMetadata {
	items := make([]PhaseMetadata, 0, len(phases))

	for _, phase := range phases {
		items = append(items, PhaseMetadata{ID: phase.ID, Name: phase.Name, Enabled: phase.Enabled})
	}

	return items
}

func uniqueID(existing, label string, seen map[string]int) string {
	id := strings.TrimSpace(existing)

	if id == "" {
		id = Slug(label)
	}

	if id == "" {
		id = "item"
	}

	seen[id]++

	if seen[id] == 1 {
		return id
	}

	return fmt.Sprintf("%s-%d", id, seen[id])
}

func Slug(value string) string {
	slug := strings.ToLower(strings.TrimSpace(value))
	slug = slugPattern.ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")

	return slug
}
