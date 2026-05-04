package app

import (
	"context"
	"io"
	"path/filepath"
	"strings"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/gocanto/mac-os/internal/bridgepb"
)

type workflowEventStream struct {
	grpc.ServerStream

	ctx    context.Context
	events []*bridgepb.WorkflowEvent
}

func (s *workflowEventStream) Send(event *bridgepb.WorkflowEvent) error {
	s.events = append(s.events, event)

	return nil
}

func (s *workflowEventStream) Context() context.Context {
	return s.ctx
}

func (s *workflowEventStream) SetHeader(metadata.MD) error {
	return nil
}

func (s *workflowEventStream) SendHeader(metadata.MD) error {
	return nil
}

func (s *workflowEventStream) SetTrailer(metadata.MD) {}

func TestGRPCListWorkflowsEmitsMetadata(t *testing.T) {
	server := workflowBridgeServer{app: newApp("/Users/gus", "/repo", strings.NewReader(""), io.Discard, io.Discard, stubRunner{})}

	response, err := server.ListWorkflows(context.Background(), &bridgepb.ListWorkflowsRequest{})

	if err != nil {
		t.Fatal(err)
	}

	if len(response.GetWorkflows()) != 8 || response.GetWorkflows()[0].GetId() != "set-up-this-mac" {
		t.Fatalf("workflows = %#v", response.GetWorkflows())
	}
}

func TestGRPCRunPersistsEventsAndRunLog(t *testing.T) {
	home := t.TempDir()
	t.Setenv("MAC_OS_WORKFLOW_DB_PATH", filepath.Join(home, "runs.sqlite3"))

	server := workflowBridgeServer{app: newApp(home, "/repo", strings.NewReader(""), io.Discard, io.Discard, stubRunner{})}
	stream := &workflowEventStream{ctx: context.Background()}

	err := server.RunWorkflow(&bridgepb.RunWorkflowRequest{
		WorkflowId:           "show-homebrew-packages",
		ConfirmationOptionId: "run-now",
		EnabledPhaseIds:      []string{"print-generated-homebrew-package-list"},
	}, stream)

	if err != nil {
		t.Fatal(err)
	}

	runID := firstStreamRunID(t, stream.events)

	runs, err := server.ListRuns(context.Background(), &bridgepb.ListRunsRequest{})

	if err != nil {
		t.Fatal(err)
	}

	if len(runs.GetRuns()) != 1 || runs.GetRuns()[0].GetWorkflowId() != "show-homebrew-packages" {
		t.Fatalf("runs = %#v", runs.GetRuns())
	}

	log, err := server.RunLog(context.Background(), &bridgepb.RunLogRequest{RunId: runID})

	if err != nil {
		t.Fatal(err)
	}

	if len(log.GetEvents()) == 0 || !hasEventType(log.GetEvents(), "phase_output") {
		t.Fatalf("run log = %#v", log.GetEvents())
	}
}

func TestGRPCSettingsValidation(t *testing.T) {
	home := t.TempDir()
	repo := writeSettingsRepo(t)
	server := workflowBridgeServer{app: newApp(home, repo, strings.NewReader(""), io.Discard, io.Discard, stubRunner{})}

	response, err := server.GetSettings(context.Background(), &bridgepb.GetSettingsRequest{})

	if err != nil {
		t.Fatal(err)
	}

	if !response.GetValid() || response.GetSettings().GetRepoRoot() != repo {
		t.Fatalf("settings response = %#v", response)
	}

	validation, err := server.ValidateSettings(context.Background(), &bridgepb.ValidateSettingsRequest{
		Settings: &bridgepb.RuntimeSettings{RepoRoot: filepath.Join(home, "missing")},
	})

	if err != nil {
		t.Fatal(err)
	}

	if validation.GetValid() {
		t.Fatalf("expected invalid settings, got %#v", validation)
	}
}

func firstStreamRunID(t *testing.T, events []*bridgepb.WorkflowEvent) string {
	t.Helper()

	for _, event := range events {
		if event.GetRunId() != "" {
			return event.GetRunId()
		}
	}

	t.Fatalf("no run id in events %#v", events)

	return ""
}

func hasEventType(events []*bridgepb.WorkflowEvent, eventType string) bool {
	for _, event := range events {
		if event.GetType() == eventType {
			return true
		}
	}

	return false
}
