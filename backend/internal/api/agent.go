package api

import (
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"superdevstudio/internal/agentconfig"
	"superdevstudio/internal/store"
)

type pipelineRunAgentResponse struct {
	Run              store.AgentRun         `json:"run"`
	StepCount        int                    `json:"step_count"`
	ToolCallCount    int                    `json:"tool_call_count"`
	EvidenceCount    int                    `json:"evidence_count"`
	EvaluationCount  int                    `json:"evaluation_count"`
	LatestEvaluation *store.AgentEvaluation `json:"latest_evaluation,omitempty"`
}

type projectAgentBundleResponse struct {
	ProjectID        string                    `json:"project_id"`
	ProjectDir       string                    `json:"project_dir"`
	DefaultAgentName string                    `json:"default_agent_name"`
	DefaultAgentMode string                    `json:"default_agent_mode"`
	Agents           []agentconfig.AgentConfig `json:"agents"`
	Modes            []agentconfig.ModeConfig  `json:"modes"`
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
	var latestEvaluation *store.AgentEvaluation
	if len(evaluations) > 0 {
		latestEvaluation = &evaluations[len(evaluations)-1]
	}
	respondJSON(w, http.StatusOK, pipelineRunAgentResponse{
		Run:              agentRun,
		StepCount:        len(steps),
		ToolCallCount:    len(toolCalls),
		EvidenceCount:    len(evidence),
		EvaluationCount:  len(evaluations),
		LatestEvaluation: latestEvaluation,
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

func (s *Server) handleGetProjectAgentBundle(w http.ResponseWriter, r *http.Request) {
	project, bundle, err := s.loadProjectAgentBundle(r)
	if err != nil {
		handleAgentResolveError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, projectAgentBundleResponse{
		ProjectID:        project.ID,
		ProjectDir:       project.RepoPath,
		DefaultAgentName: project.DefaultAgentName,
		DefaultAgentMode: project.DefaultAgentMode,
		Agents:           bundle.Agents,
		Modes:            bundle.Modes,
	})
}

func (s *Server) resolveProjectAgentSelection(r *http.Request, project store.Project, projectDir, requestedAgentName, requestedModeName string) (string, string, error) {
	bundle, err := agentconfig.LoadProjectBundle(strings.TrimSpace(projectDir))
	if err != nil {
		return "", "", err
	}
	agentName := firstNonEmpty(strings.TrimSpace(requestedAgentName), strings.TrimSpace(project.DefaultAgentName), bundle.ResolveAgent("").Name)
	modeName := firstNonEmpty(strings.TrimSpace(requestedModeName), strings.TrimSpace(project.DefaultAgentMode), bundle.ResolveMode("").Name)
	if _, ok := bundle.FindAgent(agentName); !ok {
		return "", "", errors.New("agent_name is not defined in project agent bundle")
	}
	if _, ok := bundle.FindMode(modeName); !ok {
		return "", "", errors.New("agent_mode is not defined in project agent bundle")
	}
	return agentName, modeName, nil
}

func (s *Server) validateProjectAgentDefaults(project store.Project) error {
	_, _, err := s.resolveProjectAgentSelection(nil, project, project.RepoPath, project.DefaultAgentName, project.DefaultAgentMode)
	return err
}

func (s *Server) loadProjectAgentBundle(r *http.Request) (store.Project, agentconfig.Bundle, error) {
	projectID := chi.URLParam(r, "projectID")
	project, err := s.store.GetProject(r.Context(), projectID)
	if err != nil {
		return store.Project{}, agentconfig.Bundle{}, err
	}
	bundle, err := agentconfig.LoadProjectBundle(project.RepoPath)
	if err != nil {
		return store.Project{}, agentconfig.Bundle{}, err
	}
	return project, bundle, nil
}
