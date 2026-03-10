package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"superdevstudio/internal/store"
)

type autoAdvancePipelineResponse struct {
	Action      string             `json:"action"`
	Reason      string             `json:"reason"`
	Executed    bool               `json:"executed"`
	Blocking    string             `json:"blocking,omitempty"`
	NextCommand string             `json:"next_command,omitempty"`
	Run         *store.PipelineRun `json:"run,omitempty"`
}

type autoAdvanceState struct {
	run              store.PipelineRun
	latestEvaluation *store.AgentEvaluation
	approvalGates    []store.ApprovalGate
	previewSessions  []store.PreviewSession
	nextCommand      string
}

func (s *Server) handleAutoAdvancePipeline(w http.ResponseWriter, r *http.Request) {
	runID := chi.URLParam(r, "runID")
	run, err := s.store.GetPipelineRun(r.Context(), runID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			respondError(w, http.StatusNotFound, err)
			return
		}
		respondError(w, http.StatusInternalServerError, err)
		return
	}
	result, err := s.autoAdvancePipeline(r.Context(), run)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err)
		return
	}
	status := http.StatusOK
	if result.Executed {
		status = http.StatusAccepted
	}
	respondJSON(w, status, result)
}

func (s *Server) autoAdvancePipeline(ctx context.Context, run store.PipelineRun) (autoAdvancePipelineResponse, error) {
	state, err := s.resolveAutoAdvanceState(ctx, run)
	if err != nil {
		return autoAdvancePipelineResponse{}, err
	}

	if hasOpenApprovalGate(state.approvalGates) {
		return autoAdvancePipelineResponse{
			Action:      firstNonEmpty(state.nextCommand, "await_human"),
			Reason:      "Open approval gates require human confirmation before any automatic advancement.",
			Executed:    false,
			Blocking:    "approval_gate",
			NextCommand: firstNonEmpty(state.nextCommand, "await_human"),
		}, nil
	}

	if strings.EqualFold(state.run.Status, "awaiting_human") {
		reason := "Run is waiting for human confirmation before it can continue."
		if state.latestEvaluation != nil && strings.TrimSpace(state.latestEvaluation.Reason) != "" {
			reason = state.latestEvaluation.Reason
		}
		return autoAdvancePipelineResponse{
			Action:      firstNonEmpty(state.nextCommand, "await_human"),
			Reason:      reason,
			Executed:    false,
			Blocking:    "awaiting_human",
			NextCommand: firstNonEmpty(state.nextCommand, "await_human"),
		}, nil
	}

	if shouldBlockForPreviewReview(state.run, state.previewSessions, state.nextCommand) {
		return autoAdvancePipelineResponse{
			Action:      firstNonEmpty(state.nextCommand, "review_preview"),
			Reason:      "Preview review is still pending; auto advance stops until a reviewer accepts or rejects the preview.",
			Executed:    false,
			Blocking:    "preview_review",
			NextCommand: firstNonEmpty(state.nextCommand, "review_preview"),
		}, nil
	}

	switch state.nextCommand {
	case "rerun_delivery":
		if !strings.EqualFold(state.run.Status, "failed") && !strings.EqualFold(state.run.Status, "completed") {
			return autoAdvancePipelineResponse{
				Action:      state.nextCommand,
				Reason:      "Current run is not terminal, so no rerun was executed.",
				Executed:    false,
				NextCommand: state.nextCommand,
			}, nil
		}
		nextRun, err := s.restartPipelineRun(ctx, state.run)
		if err != nil {
			return autoAdvancePipelineResponse{}, err
		}
		_, _ = s.store.AppendRunEvent(ctx, store.RunEvent{
			RunID:   nextRun.ID,
			Stage:   "auto-advance",
			Status:  "log",
			Message: fmt.Sprintf("Auto advanced from run %s with command %s", state.run.ID, state.nextCommand),
		})
		return autoAdvancePipelineResponse{
			Action:      state.nextCommand,
			Reason:      "A new delivery run has been started automatically.",
			Executed:    true,
			NextCommand: state.nextCommand,
			Run:         &nextRun,
		}, nil
	case "review_preview":
		return autoAdvancePipelineResponse{
			Action:      state.nextCommand,
			Reason:      "Preview review is required before the delivery can be considered complete.",
			Executed:    false,
			Blocking:    "preview_review",
			NextCommand: state.nextCommand,
		}, nil
	case "await_human":
		return autoAdvancePipelineResponse{
			Action:      state.nextCommand,
			Reason:      "Human confirmation is required before continuing.",
			Executed:    false,
			Blocking:    "awaiting_human",
			NextCommand: state.nextCommand,
		}, nil
	case "complete_delivery":
		return autoAdvancePipelineResponse{
			Action:      state.nextCommand,
			Reason:      "Delivery is already complete; no further automatic action is required.",
			Executed:    false,
			NextCommand: state.nextCommand,
		}, nil
	default:
		return autoAdvancePipelineResponse{
			Action:      "noop",
			Reason:      "No safe next command is available for automatic execution.",
			Executed:    false,
			NextCommand: state.nextCommand,
		}, nil
	}
}

func (s *Server) resolveAutoAdvanceState(ctx context.Context, run store.PipelineRun) (autoAdvanceState, error) {
	if err := s.syncRunFollowups(ctx, run); err != nil {
		return autoAdvanceState{}, err
	}
	if err := s.syncRunPreviewSessions(ctx, run); err != nil {
		return autoAdvanceState{}, err
	}

	state := autoAdvanceState{run: run}
	if latestRun, err := s.store.GetPipelineRun(ctx, run.ID); err == nil {
		state.run = latestRun
	} else if !errors.Is(err, store.ErrNotFound) {
		return autoAdvanceState{}, err
	}

	if agentRun, err := s.store.GetAgentRunByPipelineRun(ctx, run.ID); err == nil {
		evaluations, evalErr := s.store.ListAgentEvaluations(ctx, agentRun.ID)
		if evalErr != nil {
			return autoAdvanceState{}, evalErr
		}
		state.latestEvaluation = latestAgentEvaluation(evaluations)
	} else if !errors.Is(err, store.ErrNotFound) {
		return autoAdvanceState{}, err
	}

	approvalGates, err := s.store.ListApprovalGates(ctx, state.run.ProjectID, state.run.ID, 200)
	if err != nil {
		return autoAdvanceState{}, err
	}
	state.approvalGates = approvalGates

	previewSessions, err := s.store.ListPreviewSessions(ctx, state.run.ProjectID, state.run.ID, 200)
	if err != nil {
		return autoAdvanceState{}, err
	}
	state.previewSessions = previewSessions
	state.nextCommand = resolveAutoAdvanceNextCommand(state.run, state.latestEvaluation, state.previewSessions)
	return state, nil
}

func latestAgentEvaluation(items []store.AgentEvaluation) *store.AgentEvaluation {
	if len(items) == 0 {
		return nil
	}
	item := items[len(items)-1]
	return &item
}

func resolveAutoAdvanceNextCommand(run store.PipelineRun, evaluation *store.AgentEvaluation, previewSessions []store.PreviewSession) string {
	if evaluation != nil && strings.TrimSpace(evaluation.NextCommand) != "" {
		candidate := strings.ToLower(strings.TrimSpace(evaluation.NextCommand))
		if candidate == "review_preview" && !hasGeneratedPreview(previewSessions) {
			switch {
			case hasRejectedPreview(previewSessions):
				return "rerun_delivery"
			case hasAcceptedPreview(previewSessions):
				return "complete_delivery"
			}
		}
		return candidate
	}
	switch {
	case strings.EqualFold(run.Status, "failed"):
		return "rerun_delivery"
	case strings.EqualFold(run.Status, "awaiting_human"):
		return "await_human"
	case strings.EqualFold(run.Status, "completed") && hasGeneratedPreview(previewSessions):
		return "review_preview"
	case strings.EqualFold(run.Status, "completed"):
		return "complete_delivery"
	default:
		return ""
	}
}

func hasOpenApprovalGate(items []store.ApprovalGate) bool {
	for _, item := range items {
		if strings.EqualFold(strings.TrimSpace(item.Status), "open") {
			return true
		}
	}
	return false
}

func hasGeneratedPreview(items []store.PreviewSession) bool {
	for _, item := range items {
		if strings.EqualFold(strings.TrimSpace(item.Status), "generated") {
			return true
		}
	}
	return false
}

func hasAcceptedPreview(items []store.PreviewSession) bool {
	for _, item := range items {
		if strings.EqualFold(strings.TrimSpace(item.Status), "accepted") {
			return true
		}
	}
	return false
}

func hasRejectedPreview(items []store.PreviewSession) bool {
	for _, item := range items {
		if strings.EqualFold(strings.TrimSpace(item.Status), "rejected") {
			return true
		}
	}
	return false
}

func shouldBlockForPreviewReview(run store.PipelineRun, previewSessions []store.PreviewSession, nextCommand string) bool {
	if !hasGeneratedPreview(previewSessions) {
		return false
	}
	if strings.EqualFold(strings.TrimSpace(nextCommand), "review_preview") {
		return true
	}
	return strings.EqualFold(run.Status, "completed")
}
