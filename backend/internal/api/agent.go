package api

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"superdevstudio/internal/store"
)

type pipelineRunAgentResponse struct {
	Run             store.AgentRun `json:"run"`
	StepCount       int            `json:"step_count"`
	ToolCallCount   int            `json:"tool_call_count"`
	EvidenceCount   int            `json:"evidence_count"`
	EvaluationCount int            `json:"evaluation_count"`
}

func (s *Server) handleGetPipelineRunAgent(w http.ResponseWriter, r *http.Request) {
	runID := chi.URLParam(r, "runID")
	agentRun, err := s.store.GetAgentRunByPipelineRun(r.Context(), runID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			respondError(w, http.StatusNotFound, err)
			return
		}
		respondError(w, http.StatusInternalServerError, err)
		return
	}
	steps, err := s.store.ListAgentSteps(r.Context(), agentRun.ID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err)
		return
	}
	toolCalls, err := s.store.ListAgentToolCalls(r.Context(), agentRun.ID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err)
		return
	}
	evidence, err := s.store.ListAgentEvidence(r.Context(), agentRun.ID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err)
		return
	}
	evaluations, err := s.store.ListAgentEvaluations(r.Context(), agentRun.ID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err)
		return
	}
	respondJSON(w, http.StatusOK, pipelineRunAgentResponse{
		Run:             agentRun,
		StepCount:       len(steps),
		ToolCallCount:   len(toolCalls),
		EvidenceCount:   len(evidence),
		EvaluationCount: len(evaluations),
	})
}

func (s *Server) handleListPipelineRunAgentSteps(w http.ResponseWriter, r *http.Request) {
	agentRun, err := s.resolvePipelineRunAgent(r)
	if err != nil {
		handleAgentResolveError(w, err)
		return
	}
	items, err := s.store.ListAgentSteps(r.Context(), agentRun.ID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) handleListPipelineRunAgentToolCalls(w http.ResponseWriter, r *http.Request) {
	agentRun, err := s.resolvePipelineRunAgent(r)
	if err != nil {
		handleAgentResolveError(w, err)
		return
	}
	items, err := s.store.ListAgentToolCalls(r.Context(), agentRun.ID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) handleListPipelineRunAgentEvidence(w http.ResponseWriter, r *http.Request) {
	agentRun, err := s.resolvePipelineRunAgent(r)
	if err != nil {
		handleAgentResolveError(w, err)
		return
	}
	items, err := s.store.ListAgentEvidence(r.Context(), agentRun.ID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) handleListPipelineRunAgentEvaluations(w http.ResponseWriter, r *http.Request) {
	agentRun, err := s.resolvePipelineRunAgent(r)
	if err != nil {
		handleAgentResolveError(w, err)
		return
	}
	items, err := s.store.ListAgentEvaluations(r.Context(), agentRun.ID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) resolvePipelineRunAgent(r *http.Request) (store.AgentRun, error) {
	runID := chi.URLParam(r, "runID")
	return s.store.GetAgentRunByPipelineRun(r.Context(), runID)
}

func handleAgentResolveError(w http.ResponseWriter, err error) {
	if errors.Is(err, store.ErrNotFound) {
		respondError(w, http.StatusNotFound, err)
		return
	}
	respondError(w, http.StatusInternalServerError, err)
}
