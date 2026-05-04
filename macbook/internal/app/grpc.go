package app

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net"
	"os"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/gocanto/mac-os/internal/bridgepb"
	"github.com/gocanto/mac-os/internal/storage"
	"github.com/gocanto/mac-os/internal/workflowdomain"
)

type workflowBridgeServer struct {
	bridgepb.UnimplementedWorkflowBridgeServer

	app app
}

func (a app) serveGRPC(args []string) int {
	fs := flag.NewFlagSet("serve-grpc", flag.ContinueOnError)
	fs.SetOutput(a.stderr)

	socketPath := fs.String("socket", "", "Unix socket path")

	if err := fs.Parse(args); err != nil {
		return 2
	}

	if *socketPath == "" {
		fmt.Fprintln(a.stderr, "missing --socket")

		return 2
	}

	if err := os.Remove(*socketPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		fmt.Fprintf(a.stderr, "remove stale grpc socket: %v\n", err)

		return 1
	}

	listener, err := net.Listen("unix", *socketPath)

	if err != nil {
		fmt.Fprintf(a.stderr, "listen on grpc socket: %v\n", err)

		return 1
	}

	defer func() {
		_ = listener.Close()
		_ = os.Remove(*socketPath)
	}()

	server := grpc.NewServer()
	bridgepb.RegisterWorkflowBridgeServer(server, workflowBridgeServer{app: a})

	if err := server.Serve(listener); err != nil {
		fmt.Fprintf(a.stderr, "serve grpc: %v\n", err)

		return 1
	}

	return 0
}

func (s workflowBridgeServer) ListWorkflows(context.Context, *bridgepb.ListWorkflowsRequest) (*bridgepb.ListWorkflowsResponse, error) {
	return &bridgepb.ListWorkflowsResponse{Workflows: pbWorkflows(workflowdomain.Metadata(s.app.workflows()))}, nil
}

func (s workflowBridgeServer) RunWorkflow(req *bridgepb.RunWorkflowRequest, stream grpc.ServerStreamingServer[bridgepb.WorkflowEvent]) error {
	request := workflowdomain.RunRequest{
		WorkflowID:           req.GetWorkflowId(),
		ConfirmationOptionID: req.GetConfirmationOptionId(),
		EnabledPhaseIDs:      req.GetEnabledPhaseIds(),
	}

	plan, err := workflowdomain.BuildRunPlan(s.app.workflows(), request)

	if err != nil {
		return status.Error(codes.InvalidArgument, err.Error())
	}

	ctx := stream.Context()
	store, err := storage.Open(ctx, storage.DefaultPath(s.app.home))

	if err != nil {
		return status.Errorf(codes.Internal, "open workflow log database: %v", err)
	}

	defer store.Close()

	runID := uuid.NewString()
	optionID, optionLabel := confirmationSelection(plan)

	if err := store.CreateRun(ctx, storage.RunStart{
		ID:                      runID,
		WorkflowID:              plan.Workflow.ID,
		WorkflowName:            plan.Workflow.Name,
		ConfirmationOptionID:    optionID,
		ConfirmationOptionLabel: optionLabel,
		Mode:                    plan.Mode,
		Status:                  workflowdomain.RunStatusRunning,
	}); err != nil {
		return status.Errorf(codes.Internal, "create workflow run: %v", err)
	}

	recorder := storage.NewRecorder(store, runID, func(event workflowdomain.Event) error {
		return stream.Send(pbWorkflowEvent(event))
	})

	if err := recorder.Emit(workflowdomain.Event{
		Type:    "run_started",
		Status:  string(workflowdomain.RunStatusRunning),
		Message: plan.Workflow.Name,
	}); err != nil {
		return status.Errorf(codes.Internal, "record workflow start: %v", err)
	}

	runErr := workflowdomain.Executor{Sink: recorder}.Execute(runID, plan)

	statusValue := workflowdomain.RunStatusCompleted
	message := ""

	if plan.Mode == workflowdomain.RunModeStopBeforeRun && runErr == nil {
		statusValue = workflowdomain.RunStatusStopped
	}

	if runErr != nil {
		statusValue = workflowdomain.RunStatusFailed
		message = runErr.Error()

		_ = recorder.Emit(workflowdomain.Event{Type: "run_failed", Status: string(statusValue), Message: message})
	}

	if completeErr := store.CompleteRun(ctx, runID, statusValue, message); completeErr != nil {
		return status.Errorf(codes.Internal, "complete workflow run: %v", completeErr)
	}

	return nil
}

func (s workflowBridgeServer) ListRuns(ctx context.Context, req *bridgepb.ListRunsRequest) (*bridgepb.ListRunsResponse, error) {
	store, err := storage.Open(ctx, storage.DefaultPath(s.app.home))

	if err != nil {
		return nil, status.Errorf(codes.Internal, "open workflow log database: %v", err)
	}

	defer store.Close()

	runs, err := store.ListRuns(ctx, req.GetLimit())

	if err != nil {
		return nil, status.Errorf(codes.Internal, "list workflow runs: %v", err)
	}

	return &bridgepb.ListRunsResponse{Runs: pbRunSummaries(runs)}, nil
}

func (s workflowBridgeServer) RunLog(ctx context.Context, req *bridgepb.RunLogRequest) (*bridgepb.RunLogResponse, error) {
	store, err := storage.Open(ctx, storage.DefaultPath(s.app.home))

	if err != nil {
		return nil, status.Errorf(codes.Internal, "open workflow log database: %v", err)
	}

	defer store.Close()

	log, err := store.RunLog(ctx, req.GetRunId())

	if err != nil {
		return nil, status.Errorf(codes.Internal, "read workflow run log: %v", err)
	}

	return &bridgepb.RunLogResponse{Run: pbRunSummary(log.Run), Events: pbEventRecords(log.Events)}, nil
}

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

func confirmationSelection(plan workflowdomain.RunPlan) (string, string) {
	if plan.ConfirmationOption == nil {
		return "", ""
	}

	return plan.ConfirmationOption.ID, plan.ConfirmationOption.Label
}
