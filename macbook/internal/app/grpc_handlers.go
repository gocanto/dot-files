package app

import (
	"context"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/gocanto/mac-os/internal/bridgepb"
	"github.com/gocanto/mac-os/internal/storage"
	"github.com/gocanto/mac-os/internal/workflowdomain"
)

func (s workflowBridgeServer) ListWorkflows(context.Context, *bridgepb.ListWorkflowsRequest) (*bridgepb.ListWorkflowsResponse, error) {
	return &bridgepb.ListWorkflowsResponse{Workflows: pbWorkflows(workflowdomain.Metadata(s.app.workflows()))}, nil
}

func (s workflowBridgeServer) RunWorkflow(req *bridgepb.RunWorkflowRequest, stream grpc.ServerStreamingServer[bridgepb.WorkflowEvent]) error {
	plan, err := workflowdomain.BuildRunPlan(s.app.workflows(), workflowdomain.RunRequest{
		WorkflowID:           req.GetWorkflowId(),
		ConfirmationOptionID: req.GetConfirmationOptionId(),
		EnabledPhaseIDs:      req.GetEnabledPhaseIds(),
	})

	if err != nil {
		return status.Error(codes.InvalidArgument, err.Error())
	}

	ctx := stream.Context()
	store, err := s.openStore(ctx)

	if err != nil {
		return err
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
	statusValue, message := finalRunStatus(plan, runErr)

	if runErr != nil {
		_ = recorder.Emit(workflowdomain.Event{Type: "run_failed", Status: string(statusValue), Message: message})
	}

	if completeErr := store.CompleteRun(ctx, runID, statusValue, message); completeErr != nil {
		return status.Errorf(codes.Internal, "complete workflow run: %v", completeErr)
	}

	return nil
}

func (s workflowBridgeServer) ListRuns(ctx context.Context, req *bridgepb.ListRunsRequest) (*bridgepb.ListRunsResponse, error) {
	store, err := s.openStore(ctx)

	if err != nil {
		return nil, err
	}

	defer store.Close()

	runs, err := store.ListRuns(ctx, req.GetLimit())

	if err != nil {
		return nil, status.Errorf(codes.Internal, "list workflow runs: %v", err)
	}

	return &bridgepb.ListRunsResponse{Runs: pbRunSummaries(runs)}, nil
}

func (s workflowBridgeServer) RunLog(ctx context.Context, req *bridgepb.RunLogRequest) (*bridgepb.RunLogResponse, error) {
	store, err := s.openStore(ctx)

	if err != nil {
		return nil, err
	}

	defer store.Close()

	log, err := store.RunLog(ctx, req.GetRunId())

	if err != nil {
		return nil, status.Errorf(codes.Internal, "read workflow run log: %v", err)
	}

	return &bridgepb.RunLogResponse{Run: pbRunSummary(log.Run), Events: pbEventRecords(log.Events)}, nil
}

func (s workflowBridgeServer) GetSettings(context.Context, *bridgepb.GetSettingsRequest) (*bridgepb.SettingsResponse, error) {
	validation := validateRuntimeSettings(s.app.home, s.app.repo, s.app.settings)

	return &bridgepb.SettingsResponse{
		Settings: pbRuntimeSettings(validation.Settings),
		Checks:   pbSettingsChecks(validation.Checks),
		Valid:    validation.Valid,
	}, nil
}

func (s workflowBridgeServer) ValidateSettings(_ context.Context, req *bridgepb.ValidateSettingsRequest) (*bridgepb.SettingsValidationResponse, error) {
	validation := validateRuntimeSettings(s.app.home, s.app.repo, runtimeSettingsFromPB(req.GetSettings()))

	return &bridgepb.SettingsValidationResponse{
		Settings: pbRuntimeSettings(validation.Settings),
		Checks:   pbSettingsChecks(validation.Checks),
		Valid:    validation.Valid,
	}, nil
}

func (s workflowBridgeServer) openStore(ctx context.Context) (*storage.Store, error) {
	store, err := storage.Open(ctx, s.app.settings.WorkflowDBPath)

	if err != nil {
		return nil, status.Errorf(codes.Internal, "open workflow log database: %v", err)
	}

	return store, nil
}

func confirmationSelection(plan workflowdomain.RunPlan) (string, string) {
	if plan.ConfirmationOption == nil {
		return "", ""
	}

	return plan.ConfirmationOption.ID, plan.ConfirmationOption.Label
}

func finalRunStatus(plan workflowdomain.RunPlan, runErr error) (workflowdomain.RunStatus, string) {
	if runErr != nil {
		return workflowdomain.RunStatusFailed, runErr.Error()
	}

	if plan.Mode == workflowdomain.RunModeStopBeforeRun {
		return workflowdomain.RunStatusStopped, ""
	}

	return workflowdomain.RunStatusCompleted, ""
}
