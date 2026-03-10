package api

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"superdevstudio/internal/store"
)

const syncPreviewPrefix = "sync:preview:"

type updatePreviewSessionRequest struct {
	Status       string `json:"status"`
	ReviewerNote string `json:"reviewer_note"`
}

func (s *Server) handleListProjectPreviewSessions(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "projectID")
	limit := parseLimit(r.URL.Query().Get("limit"), 100)
	items, err := s.store.ListPreviewSessions(r.Context(), projectID, strings.TrimSpace(r.URL.Query().Get("run_id")), limit)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) handleListRunPreviewSessions(w http.ResponseWriter, r *http.Request) {
	runID := chi.URLParam(r, "runID")
	run, err := s.store.GetPipelineRun(r.Context(), runID)
	if err != nil {
		handleAgentResolveError(w, err)
		return
	}
	if err := s.syncRunPreviewSessions(r.Context(), run); err != nil {
		respondError(w, http.StatusInternalServerError, err)
		return
	}
	items, err := s.store.ListPreviewSessions(r.Context(), run.ProjectID, run.ID, parseLimit(r.URL.Query().Get("limit"), 100))
	if err != nil {
		respondError(w, http.StatusInternalServerError, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) handleUpdatePreviewSession(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "sessionID")
	var req updatePreviewSessionRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, err)
		return
	}
	status := strings.ToLower(strings.TrimSpace(req.Status))
	switch status {
	case "generated", "accepted", "rejected":
	default:
		respondError(w, http.StatusBadRequest, errors.New("status must be one of: generated, accepted, rejected"))
		return
	}
	item, err := s.store.UpdatePreviewSessionStatus(r.Context(), sessionID, status, req.ReviewerNote)
	if err != nil {
		handleAgentResolveError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, item)
}

func (s *Server) syncRunPreviewSessions(ctx context.Context, run store.PipelineRun) error {
	completion := buildPipelineCompletionResponse(run)
	for _, item := range derivePreviewSessions(run, completion) {
		if _, err := s.store.UpsertPreviewSession(ctx, item); err != nil {
			return err
		}
	}
	return nil
}

func derivePreviewSessions(run store.PipelineRun, completion pipelineCompletionResponse) []store.PreviewSession {
	if strings.TrimSpace(completion.PreviewURL) == "" {
		return nil
	}
	previewType := "html"
	for _, artifact := range completion.Artifacts {
		if strings.TrimSpace(artifact.PreviewURL) == strings.TrimSpace(completion.PreviewURL) {
			previewType = firstNonEmpty(strings.TrimSpace(artifact.PreviewType), strings.TrimSpace(artifact.Kind), previewType)
			break
		}
	}
	return []store.PreviewSession{{
		ProjectID:     run.ProjectID,
		PipelineRunID: run.ID,
		ChangeBatchID: run.ChangeBatchID,
		PreviewURL:    completion.PreviewURL,
		PreviewType:   previewType,
		Title:         "交付预览版",
		SourceKey:     syncPreviewPrefix + run.ID + ":primary",
		Status:        "generated",
	}}
}
