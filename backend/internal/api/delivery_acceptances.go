package api

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"superdevstudio/internal/store"
)

type updateDeliveryAcceptanceRequest struct {
	Status       string `json:"status"`
	ReviewerNote string `json:"reviewer_note"`
}

func (s *Server) handleGetRunDeliveryAcceptance(w http.ResponseWriter, r *http.Request) {
	runID := chi.URLParam(r, "runID")
	if _, err := s.store.GetPipelineRun(r.Context(), runID); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			respondError(w, http.StatusNotFound, err)
			return
		}
		respondError(w, http.StatusInternalServerError, err)
		return
	}
	item, err := s.store.GetDeliveryAcceptanceByRun(r.Context(), runID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			respondJSON(w, http.StatusOK, nil)
			return
		}
		respondError(w, http.StatusInternalServerError, err)
		return
	}
	respondJSON(w, http.StatusOK, item)
}

func (s *Server) handleUpdateRunDeliveryAcceptance(w http.ResponseWriter, r *http.Request) {
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
	var req updateDeliveryAcceptanceRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, err)
		return
	}
	status, err := normalizeDeliveryAcceptanceRequestStatus(req.Status)
	if err != nil {
		respondError(w, http.StatusBadRequest, err)
		return
	}
	if err := s.syncRunFollowups(r.Context(), run); err != nil {
		respondError(w, http.StatusInternalServerError, err)
		return
	}
	if err := s.syncRunPreviewSessions(r.Context(), run); err != nil {
		respondError(w, http.StatusInternalServerError, err)
		return
	}
	if status == "accepted" {
		if reason, readyErr := s.validateRunReadyForFinalAcceptance(r.Context(), run); readyErr != nil {
			respondError(w, http.StatusInternalServerError, readyErr)
			return
		} else if reason != "" {
			respondError(w, http.StatusConflict, errors.New(reason))
			return
		}
	}
	item, err := s.store.UpsertDeliveryAcceptance(r.Context(), store.DeliveryAcceptance{
		ProjectID:     run.ProjectID,
		PipelineRunID: run.ID,
		ChangeBatchID: run.ChangeBatchID,
		Status:        status,
		ReviewerNote:  req.ReviewerNote,
	})
	if err != nil {
		respondError(w, http.StatusInternalServerError, err)
		return
	}
	message := "Final acceptance was reopened for another review."
	if status == "accepted" {
		message = "Final acceptance was recorded for the current release candidate."
	}
	if _, err := s.store.AppendRunEvent(r.Context(), store.RunEvent{
		RunID:   run.ID,
		Stage:   "lifecycle-acceptance",
		Status:  "completed",
		Message: message,
	}); err != nil {
		respondError(w, http.StatusInternalServerError, err)
		return
	}
	respondJSON(w, http.StatusOK, item)
}

func normalizeDeliveryAcceptanceRequestStatus(status string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "accepted":
		return "accepted", nil
	case "revoked":
		return "revoked", nil
	default:
		return "", errors.New("status must be one of: accepted, revoked")
	}
}

func (s *Server) validateRunReadyForFinalAcceptance(ctx context.Context, run store.PipelineRun) (string, error) {
	if !strings.EqualFold(strings.TrimSpace(run.Status), "completed") {
		return "run must be completed before final sign-off", nil
	}
	previewSessions, err := s.store.ListPreviewSessions(ctx, run.ProjectID, run.ID, 20)
	if err != nil {
		return "", err
	}
	if !hasAcceptedPreview(previewSessions) {
		return "preview must be accepted before final sign-off", nil
	}
	approvalGates, err := s.store.ListApprovalGates(ctx, run.ProjectID, run.ID, 100)
	if err != nil {
		return "", err
	}
	if hasOpenApprovalGate(approvalGates) {
		return "resolve open approval gates before final sign-off", nil
	}
	residualItems, err := s.store.ListResidualItems(ctx, run.ProjectID, run.ID, 100)
	if err != nil {
		return "", err
	}
	if countOpenResidualItems(residualItems) > 0 {
		return "resolve open residual items before final sign-off", nil
	}
	completion := buildPipelineCompletionResponse(run)
	if strings.TrimSpace(completion.PreviewURL) == "" {
		return "preview artifact is missing for final sign-off", nil
	}
	events, err := s.store.ListRunEvents(ctx, run.ID)
	if err != nil {
		return "", err
	}
	if !qualityEvidenceReady(run, completion, events) {
		return "quality evidence is not ready for final sign-off", nil
	}
	if len(pickHandoffArtifacts(completion.Artifacts)) == 0 {
		return "handoff artifacts are missing for final sign-off", nil
	}
	return "", nil
}

func countOpenResidualItems(items []store.ResidualItem) int {
	count := 0
	for _, item := range items {
		if strings.EqualFold(strings.TrimSpace(item.Status), "open") {
			count++
		}
	}
	return count
}

func qualityEvidenceReady(run store.PipelineRun, completion pipelineCompletionResponse, events []store.RunEvent) bool {
	for i := len(events) - 1; i >= 0; i-- {
		item := events[i]
		if !strings.Contains(strings.ToLower(item.Stage), "quality") {
			continue
		}
		lowerMessage := strings.ToLower(item.Message)
		if item.Status == "completed" || strings.Contains(lowerMessage, "quality gate passed") {
			return true
		}
		if item.Status == "failed" || strings.Contains(lowerMessage, "not passed") || strings.Contains(lowerMessage, "still failing") {
			return false
		}
	}
	return run.Status == "completed" && hasQualityArtifact(completion.Artifacts)
}

func pickHandoffArtifacts(artifacts []pipelineArtifact) []pipelineArtifact {
	matchers := []func(pipelineArtifact) bool{isPreviewArtifact, hasQualityArtifactForArtifact, isRedteamArtifact, isExecutionArtifact}
	picked := make([]pipelineArtifact, 0, len(matchers))
	seen := make(map[string]struct{}, len(matchers))
	for _, matcher := range matchers {
		for _, artifact := range artifacts {
			if _, ok := seen[artifact.Path]; ok {
				continue
			}
			if matcher(artifact) {
				picked = append(picked, artifact)
				seen[artifact.Path] = struct{}{}
				break
			}
		}
	}
	if len(picked) == 0 && len(artifacts) > 0 {
		limit := len(artifacts)
		if limit > 4 {
			limit = 4
		}
		return artifacts[:limit]
	}
	return picked
}

func isPreviewArtifact(artifact pipelineArtifact) bool {
	lowerPath := strings.ToLower(artifact.Path)
	return artifact.PreviewType == "html" || strings.HasSuffix(lowerPath, "preview.html") || strings.HasSuffix(lowerPath, "frontend/index.html")
}

func hasQualityArtifact(artifacts []pipelineArtifact) bool {
	for _, artifact := range artifacts {
		if hasQualityArtifactForArtifact(artifact) {
			return true
		}
	}
	return false
}

func hasQualityArtifactForArtifact(artifact pipelineArtifact) bool {
	lower := strings.ToLower(artifact.Name + " " + artifact.Path)
	return strings.Contains(lower, "quality-gate")
}

func isRedteamArtifact(artifact pipelineArtifact) bool {
	lower := strings.ToLower(artifact.Name + " " + artifact.Path)
	return strings.Contains(lower, "redteam")
}

func isExecutionArtifact(artifact pipelineArtifact) bool {
	lower := strings.ToLower(artifact.Name + " " + artifact.Path)
	return strings.Contains(lower, "task-execution") || strings.Contains(lower, "execution-report") || strings.Contains(lower, "execution-plan")
}
