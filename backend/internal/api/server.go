package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"superdevstudio/internal/contextopt"
	"superdevstudio/internal/pipeline"
	"superdevstudio/internal/store"
)

type Server struct {
	store       *store.Store
	pipeline    *pipeline.Manager
	contextOpt  *contextopt.Service
	config      ServerConfig
	rateLimiter *slidingWindowRateLimiter
}

func NewServer(s *store.Store, p *pipeline.Manager, c *contextopt.Service) *Server {
	return NewServerWithConfig(s, p, c, DefaultServerConfig())
}

func NewServerWithConfig(s *store.Store, p *pipeline.Manager, c *contextopt.Service, cfg ServerConfig) *Server {
	limiter := (*slidingWindowRateLimiter)(nil)
	if cfg.RateLimit.Enabled {
		limiter = newSlidingWindowRateLimiter(cfg.RateLimit.Window)
	}
	return &Server{store: s, pipeline: p, contextOpt: c, config: cfg, rateLimiter: limiter}
}

func (s *Server) Router() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Logger)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	r.Get("/api/health", s.handleHealth)

	r.Get("/api/dashboard", s.handleDashboard)

	r.Route("/api/projects", func(r chi.Router) {
		r.Get("/", s.handleListProjects)
		r.With(s.rateLimit(s.mutationPolicy("projects:create"))).Post("/", s.handleCreateProject)
		r.Get("/{projectID}", s.handleGetProject)
		r.With(s.rateLimit(s.mutationPolicy("projects:update"))).Put("/{projectID}", s.handleUpdateProject)
		r.With(s.rateLimit(s.mutationPolicy("projects:delete"))).Delete("/{projectID}", s.handleDeleteProject)
		r.Get("/{projectID}/tasks", s.handleListTasks)
		r.With(s.rateLimit(s.mutationPolicy("tasks:create"))).Post("/{projectID}/tasks", s.handleCreateTask)
		r.With(s.rateLimit(s.expensivePolicy("tasks:auto-schedule"))).Post("/{projectID}/tasks/auto-schedule", s.handleAutoScheduleTasks)
		r.With(s.rateLimit(s.expensivePolicy("projects:advance"))).Post("/{projectID}/advance", s.handleAdvanceProject)
		r.Get("/{projectID}/change-batches", s.handleListChangeBatches)
		r.Get("/{projectID}/agent-bundle", s.handleGetProjectAgentBundle)
		r.With(s.rateLimit(s.mutationPolicy("change-batches:create"))).Post("/{projectID}/change-batches", s.handleCreateChangeBatch)
		r.Get("/{projectID}/memories", s.handleListMemories)
		r.With(s.rateLimit(s.mutationPolicy("memories:create"))).Post("/{projectID}/memories", s.handleCreateMemory)
		r.Get("/{projectID}/knowledge/documents", s.handleListKnowledgeDocuments)
		r.With(s.rateLimit(s.expensivePolicy("knowledge:create"))).Post("/{projectID}/knowledge/documents", s.handleCreateKnowledgeDocument)
		r.Get("/{projectID}/knowledge/search", s.handleSearchKnowledge)
		r.With(s.rateLimit(s.expensivePolicy("context-pack:build"))).Post("/{projectID}/context-pack", s.handleBuildContextPack)
		r.Get("/{projectID}/pipeline-runs", s.handleListPipelineRuns)
	})

	r.With(s.rateLimit(s.mutationPolicy("tasks:update"))).Patch("/api/tasks/{taskID}", s.handleUpdateTask)
	r.With(s.rateLimit(s.pipelinePolicy("runs:start"))).Post("/api/pipeline/runs", s.handleStartPipeline)
	r.With(s.rateLimit(s.pipelinePolicy("runs:retry"))).Post("/api/pipeline/runs/{runID}/retry", s.handleRetryPipeline)
	r.With(s.rateLimit(s.pipelinePolicy("runs:retry"))).Post("/api/pipeline/runs/{runID}/resume", s.handleResumePipeline)
	r.With(s.rateLimit(s.pipelinePolicy("runs:retry"))).Post("/api/pipeline/runs/{runID}/approve-tool", s.handleApprovePipelineTool)
	r.Get("/api/pipeline/runs/{runID}", s.handleGetPipelineRun)
	r.Get("/api/pipeline/runs/{runID}/agent", s.handleGetPipelineRunAgent)
	r.Get("/api/pipeline/runs/{runID}/agent/steps", s.handleListPipelineRunAgentSteps)
	r.Get("/api/pipeline/runs/{runID}/agent/tool-calls", s.handleListPipelineRunAgentToolCalls)
	r.Get("/api/pipeline/runs/{runID}/agent/evidence", s.handleListPipelineRunAgentEvidence)
	r.Get("/api/pipeline/runs/{runID}/agent/evaluations", s.handleListPipelineRunAgentEvaluations)
	r.Get("/api/pipeline/runs/{runID}/completion", s.handleGetPipelineRunCompletion)
	r.Get("/api/pipeline/runs/{runID}/preview", s.handlePreviewPipelineRunOutput)
	r.Get("/api/pipeline/runs/{runID}/preview/*", s.handlePreviewPipelineRunOutput)
	r.Get("/api/pipeline/runs/{runID}/events", s.handleListRunEvents)

	return r
}

type errorResponse struct {
	Error string `json:"error"`
}

const (
	advanceModeStepByStep  = "step_by_step"
	advanceModeFullCycle   = "full_cycle"
	superDevUsageMemoryTag = "super-dev-usage-v2.0.1"
)

func respondJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func respondError(w http.ResponseWriter, status int, err error) {
	respondJSON(w, status, errorResponse{Error: err.Error()})
}

func decodeJSON(r *http.Request, target any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(target)
}

func parseLimit(raw string, defaultValue int) int {
	if raw == "" {
		return defaultValue
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return defaultValue
	}
	return value
}

func normalizeDateUTC(t time.Time) time.Time {
	year, month, day := t.UTC().Date()
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

func parseDateInput(raw string) (time.Time, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return time.Time{}, errors.New("date is required")
	}
	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02",
	}
	for _, layout := range layouts {
		parsed, err := time.Parse(layout, value)
		if err == nil {
			return normalizeDateUTC(parsed), nil
		}
	}
	return time.Time{}, fmt.Errorf("invalid date format: %s", value)
}

func parseOptionalDate(raw *string) (*time.Time, error) {
	if raw == nil {
		return nil, nil
	}
	trimmed := strings.TrimSpace(*raw)
	if trimmed == "" {
		return nil, nil
	}
	value, err := parseDateInput(trimmed)
	if err != nil {
		return nil, err
	}
	return &value, nil
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	projectID := r.URL.Query().Get("project_id")
	stats, err := s.store.DashboardStats(r.Context(), projectID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err)
		return
	}
	response := map[string]any{"stats": stats}
	if projectID != "" {
		runs, runErr := s.store.ListPipelineRuns(r.Context(), projectID, 5)
		if runErr == nil {
			response["recent_runs"] = runs
		}
	}
	respondJSON(w, http.StatusOK, response)
}

func (s *Server) handleListProjects(w http.ResponseWriter, r *http.Request) {
	projects, err := s.store.ListProjects(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"items": projects})
}

type createProjectRequest struct {
	Name                      string `json:"name"`
	Description               string `json:"description"`
	RepoPath                  string `json:"repo_path"`
	Status                    string `json:"status"`
	DefaultPlatform           string `json:"default_platform"`
	DefaultFrontend           string `json:"default_frontend"`
	DefaultBackend            string `json:"default_backend"`
	DefaultDomain             string `json:"default_domain"`
	DefaultAgentName          string `json:"default_agent_name"`
	DefaultAgentMode          string `json:"default_agent_mode"`
	DefaultContextMode        string `json:"default_context_mode"`
	DefaultContextTokenBudget int    `json:"default_context_token_budget"`
	DefaultContextMaxItems    int    `json:"default_context_max_items"`
	DefaultContextDynamic     *bool  `json:"default_context_dynamic"`
	DefaultMemoryWriteback    *bool  `json:"default_memory_writeback"`
}

func (s *Server) handleCreateProject(w http.ResponseWriter, r *http.Request) {
	var req createProjectRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, err)
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		respondError(w, http.StatusBadRequest, errors.New("name is required"))
		return
	}
	projectPayload := store.Project{
		Name:                      strings.TrimSpace(req.Name),
		Description:               strings.TrimSpace(req.Description),
		RepoPath:                  strings.TrimSpace(req.RepoPath),
		Status:                    strings.TrimSpace(req.Status),
		DefaultPlatform:           strings.TrimSpace(req.DefaultPlatform),
		DefaultFrontend:           strings.TrimSpace(req.DefaultFrontend),
		DefaultBackend:            strings.TrimSpace(req.DefaultBackend),
		DefaultDomain:             strings.TrimSpace(req.DefaultDomain),
		DefaultAgentName:          strings.TrimSpace(req.DefaultAgentName),
		DefaultAgentMode:          strings.TrimSpace(req.DefaultAgentMode),
		DefaultContextMode:        strings.TrimSpace(req.DefaultContextMode),
		DefaultContextTokenBudget: req.DefaultContextTokenBudget,
		DefaultContextMaxItems:    req.DefaultContextMaxItems,
		DefaultContextDynamic:     valueOrDefault(req.DefaultContextDynamic, true),
		DefaultMemoryWriteback:    valueOrDefault(req.DefaultMemoryWriteback, true),
	}
	if err := s.validateProjectAgentDefaults(projectPayload); err != nil {
		respondError(w, http.StatusBadRequest, err)
		return
	}
	project, err := s.store.CreateProject(r.Context(), projectPayload)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err)
		return
	}
	respondJSON(w, http.StatusCreated, project)
}

func (s *Server) handleGetProject(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "projectID")
	project, err := s.store.GetProject(r.Context(), projectID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			respondError(w, http.StatusNotFound, err)
			return
		}
		respondError(w, http.StatusInternalServerError, err)
		return
	}
	respondJSON(w, http.StatusOK, project)
}

func (s *Server) handleUpdateProject(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "projectID")
	var req createProjectRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, err)
		return
	}
	existing, err := s.store.GetProject(r.Context(), projectID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			respondError(w, http.StatusNotFound, err)
			return
		}
		respondError(w, http.StatusInternalServerError, err)
		return
	}
	projectPatch := store.Project{
		Name:                      strings.TrimSpace(req.Name),
		Description:               strings.TrimSpace(req.Description),
		RepoPath:                  strings.TrimSpace(req.RepoPath),
		Status:                    strings.TrimSpace(req.Status),
		DefaultPlatform:           strings.TrimSpace(req.DefaultPlatform),
		DefaultFrontend:           strings.TrimSpace(req.DefaultFrontend),
		DefaultBackend:            strings.TrimSpace(req.DefaultBackend),
		DefaultDomain:             strings.TrimSpace(req.DefaultDomain),
		DefaultAgentName:          strings.TrimSpace(req.DefaultAgentName),
		DefaultAgentMode:          strings.TrimSpace(req.DefaultAgentMode),
		DefaultContextMode:        strings.TrimSpace(req.DefaultContextMode),
		DefaultContextTokenBudget: req.DefaultContextTokenBudget,
		DefaultContextMaxItems:    req.DefaultContextMaxItems,
		DefaultContextDynamic:     valueOrDefault(req.DefaultContextDynamic, existing.DefaultContextDynamic),
		DefaultMemoryWriteback:    valueOrDefault(req.DefaultMemoryWriteback, existing.DefaultMemoryWriteback),
	}
	if strings.TrimSpace(projectPatch.Name) == "" {
		projectPatch.Name = existing.Name
	}
	if strings.TrimSpace(projectPatch.DefaultAgentName) == "" {
		projectPatch.DefaultAgentName = existing.DefaultAgentName
	}
	if strings.TrimSpace(projectPatch.DefaultAgentMode) == "" {
		projectPatch.DefaultAgentMode = existing.DefaultAgentMode
	}
	if err := s.validateProjectAgentDefaults(projectPatch); err != nil {
		respondError(w, http.StatusBadRequest, err)
		return
	}
	project, err := s.store.UpdateProjectWithDefaults(r.Context(), projectID, projectPatch)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			respondError(w, http.StatusNotFound, err)
			return
		}
		respondError(w, http.StatusInternalServerError, err)
		return
	}
	respondJSON(w, http.StatusOK, project)
}

func (s *Server) handleDeleteProject(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "projectID")
	if err := s.store.DeleteProject(r.Context(), projectID); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			respondError(w, http.StatusNotFound, err)
			return
		}
		respondError(w, http.StatusInternalServerError, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

type createChangeBatchRequest struct {
	Title string `json:"title"`
	Goal  string `json:"goal"`
	Mode  string `json:"mode"`
}

func (s *Server) handleListChangeBatches(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "projectID")
	items, err := s.store.ListChangeBatches(r.Context(), projectID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) handleCreateChangeBatch(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "projectID")
	var req createChangeBatchRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, err)
		return
	}
	if strings.TrimSpace(req.Title) == "" {
		respondError(w, http.StatusBadRequest, errors.New("title is required"))
		return
	}
	batch, err := s.store.CreateChangeBatch(r.Context(), store.ChangeBatch{
		ProjectID: projectID,
		Title:     strings.TrimSpace(req.Title),
		Goal:      strings.TrimSpace(req.Goal),
		Mode:      strings.TrimSpace(req.Mode),
		Status:    "draft",
	})
	if err != nil {
		respondError(w, http.StatusInternalServerError, err)
		return
	}
	respondJSON(w, http.StatusCreated, batch)
}

type createTaskRequest struct {
	Title         string  `json:"title"`
	Description   string  `json:"description"`
	Status        string  `json:"status"`
	Priority      string  `json:"priority"`
	Assignee      string  `json:"assignee"`
	StartDate     *string `json:"start_date"`
	DueDate       *string `json:"due_date"`
	EstimatedDays *int    `json:"estimated_days"`
}

func (s *Server) handleCreateTask(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "projectID")
	var req createTaskRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, err)
		return
	}
	if strings.TrimSpace(req.Title) == "" {
		respondError(w, http.StatusBadRequest, errors.New("title is required"))
		return
	}
	startDate, err := parseOptionalDate(req.StartDate)
	if err != nil {
		respondError(w, http.StatusBadRequest, fmt.Errorf("invalid start_date: %w", err))
		return
	}
	dueDate, err := parseOptionalDate(req.DueDate)
	if err != nil {
		respondError(w, http.StatusBadRequest, fmt.Errorf("invalid due_date: %w", err))
		return
	}
	if startDate != nil && dueDate != nil && dueDate.Before(*startDate) {
		respondError(w, http.StatusBadRequest, errors.New("due_date must be on or after start_date"))
		return
	}
	estimatedDays := 0
	if req.EstimatedDays != nil {
		if *req.EstimatedDays < 0 {
			respondError(w, http.StatusBadRequest, errors.New("estimated_days must be >= 0"))
			return
		}
		estimatedDays = *req.EstimatedDays
	}

	task, err := s.store.CreateTask(r.Context(), store.Task{
		ProjectID:     projectID,
		Title:         strings.TrimSpace(req.Title),
		Description:   strings.TrimSpace(req.Description),
		Status:        strings.TrimSpace(req.Status),
		Priority:      strings.TrimSpace(req.Priority),
		Assignee:      strings.TrimSpace(req.Assignee),
		StartDate:     startDate,
		DueDate:       dueDate,
		EstimatedDays: estimatedDays,
	})
	if err != nil {
		respondError(w, http.StatusInternalServerError, err)
		return
	}
	respondJSON(w, http.StatusCreated, task)
}

func (s *Server) handleListTasks(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "projectID")
	items, err := s.store.ListTasks(r.Context(), projectID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"items": items})
}

type autoScheduleTasksRequest struct {
	StartDate string `json:"start_date"`
}

func (s *Server) handleAutoScheduleTasks(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "projectID")
	scheduleStart := normalizeDateUTC(time.Now().UTC())

	var req autoScheduleTasksRequest
	if r.Body != nil {
		if err := decodeJSON(r, &req); err != nil && !errors.Is(err, io.EOF) {
			respondError(w, http.StatusBadRequest, err)
			return
		}
	}
	if strings.TrimSpace(req.StartDate) != "" {
		parsed, err := parseDateInput(req.StartDate)
		if err != nil {
			respondError(w, http.StatusBadRequest, fmt.Errorf("invalid start_date: %w", err))
			return
		}
		scheduleStart = parsed
	}

	items, scheduledCount, err := s.store.AutoScheduleTasks(r.Context(), projectID, scheduleStart)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"items":           items,
		"scheduled_count": scheduledCount,
		"start_date":      scheduleStart.Format("2006-01-02"),
	})
}

type advanceProjectRequest struct {
	Goal           string `json:"goal"`
	Mode           string `json:"mode"`
	IterationLimit int    `json:"iteration_limit"`
	Platform       string `json:"platform"`
	Frontend       string `json:"frontend"`
	Backend        string `json:"backend"`
	Domain         string `json:"domain"`
}

type advanceProjectResponse struct {
	Run           store.PipelineRun  `json:"run"`
	Mode          string             `json:"mode"`
	MemoryWritten bool               `json:"memory_written"`
	MemoryID      string             `json:"memory_id,omitempty"`
	ChangeBatch   *store.ChangeBatch `json:"change_batch,omitempty"`
}

func (s *Server) handleAdvanceProject(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "projectID")
	project, err := s.store.GetProject(r.Context(), projectID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			respondError(w, http.StatusNotFound, err)
			return
		}
		respondError(w, http.StatusInternalServerError, err)
		return
	}

	var req advanceProjectRequest
	if r.Body != nil {
		if err := decodeJSON(r, &req); err != nil && !errors.Is(err, io.EOF) {
			respondError(w, http.StatusBadRequest, err)
			return
		}
	}
	mode, modeErr := parseAdvanceMode(req.Mode)
	if modeErr != nil {
		respondError(w, http.StatusBadRequest, modeErr)
		return
	}
	iterationLimit := req.IterationLimit
	if iterationLimit <= 0 {
		iterationLimit = 3
	}

	mem, memoryWritten, err := s.ensureSuperDevUsageMemory(r.Context(), projectID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err)
		return
	}

	tasks, err := s.store.ListTasks(r.Context(), projectID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err)
		return
	}
	prompt := buildProjectAdvancePrompt(project, tasks, req.Goal)
	changeBatch, err := s.store.CreateChangeBatch(r.Context(), store.ChangeBatch{
		ProjectID: projectID,
		Title:     buildChangeBatchTitle(project.Name, req.Goal, mode),
		Goal:      strings.TrimSpace(req.Goal),
		Mode:      mode,
		Status:    "queued",
	})
	if err != nil {
		respondError(w, http.StatusInternalServerError, err)
		return
	}

	platform := strings.TrimSpace(req.Platform)
	if platform == "" {
		platform = strings.TrimSpace(project.DefaultPlatform)
	}
	frontend := strings.TrimSpace(req.Frontend)
	if frontend == "" {
		frontend = strings.TrimSpace(project.DefaultFrontend)
	}
	backendValue := strings.TrimSpace(req.Backend)
	if backendValue == "" {
		backendValue = strings.TrimSpace(project.DefaultBackend)
	}

	projectDir := s.resolveProjectDirForAdvance(r.Context(), projectID, project.RepoPath)
	preferredLifecycleMode := func() string {
		if mode == advanceModeFullCycle {
			return advanceModeFullCycle
		}
		if mode == advanceModeStepByStep {
			return advanceModeStepByStep
		}
		return ""
	}()
	agentName, agentMode, selectionErr := s.resolveProjectAgentSelection(
		r,
		project,
		projectDir,
		project.DefaultAgentName,
		resolveProjectLifecycleMode(project, project.DefaultAgentMode, preferredLifecycleMode, ""),
	)
	if selectionErr != nil {
		respondError(w, http.StatusBadRequest, selectionErr)
		return
	}
	startReq := pipeline.StartRequest{
		ProjectID:     projectID,
		ChangeBatchID: changeBatch.ID,
		Prompt:        prompt,
		Simulate:      false,
		Agent: pipeline.AgentOptions{
			Name: agentName,
			Mode: agentMode,
		},
		Context: pipeline.ContextOptions{
			Mode:            pipeline.ContextMode(project.DefaultContextMode),
			Query:           strings.TrimSpace(req.Goal),
			TokenBudget:     project.DefaultContextTokenBudget,
			MaxItems:        project.DefaultContextMaxItems,
			DynamicByPhase:  project.DefaultContextDynamic,
			MemoryWriteback: project.DefaultMemoryWriteback,
		},
		Lifecycle: pipeline.LifecycleOptions{
			OneClickDelivery: mode == advanceModeFullCycle,
			StepByStep:       mode == advanceModeStepByStep,
			IterationLimit:   iterationLimit,
		},
		Options: pipeline.RunRequest{
			Prompt:     prompt,
			ProjectDir: projectDir,
			Platform:   platform,
			Frontend:   frontend,
			Backend:    backendValue,
			Domain:     firstNonEmpty(strings.TrimSpace(req.Domain), strings.TrimSpace(project.DefaultDomain)),
		},
	}

	run, err := s.pipeline.Start(r.Context(), startReq)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err)
		return
	}
	_, _ = s.store.AppendRunEvent(r.Context(), store.RunEvent{
		RunID:   run.ID,
		Stage:   "advance",
		Status:  "log",
		Message: fmt.Sprintf("Task board advance triggered, mode=%s, memory_written=%t", mode, memoryWritten),
	})

	respondJSON(w, http.StatusAccepted, advanceProjectResponse{
		Run:           run,
		Mode:          mode,
		MemoryWritten: memoryWritten,
		MemoryID:      mem.ID,
		ChangeBatch:   &changeBatch,
	})
}

type updateTaskRequest struct {
	Status   string `json:"status"`
	Priority string `json:"priority"`
	Assignee string `json:"assignee"`
}

func (s *Server) handleUpdateTask(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "taskID")
	var req updateTaskRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, err)
		return
	}
	task, err := s.store.UpdateTask(
		r.Context(),
		taskID,
		strings.TrimSpace(req.Status),
		strings.TrimSpace(req.Priority),
		strings.TrimSpace(req.Assignee),
	)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			respondError(w, http.StatusNotFound, err)
			return
		}
		respondError(w, http.StatusInternalServerError, err)
		return
	}
	respondJSON(w, http.StatusOK, task)
}

type createMemoryRequest struct {
	Role       string   `json:"role"`
	Content    string   `json:"content"`
	Tags       []string `json:"tags"`
	Importance float64  `json:"importance"`
}

func (s *Server) handleCreateMemory(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "projectID")
	var req createMemoryRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, err)
		return
	}
	if strings.TrimSpace(req.Content) == "" {
		respondError(w, http.StatusBadRequest, errors.New("content is required"))
		return
	}
	memory, err := s.store.CreateMemory(r.Context(), store.Memory{
		ProjectID:  projectID,
		Role:       strings.TrimSpace(req.Role),
		Content:    strings.TrimSpace(req.Content),
		Tags:       req.Tags,
		Importance: req.Importance,
	})
	if err != nil {
		respondError(w, http.StatusInternalServerError, err)
		return
	}
	respondJSON(w, http.StatusCreated, memory)
}

func (s *Server) handleListMemories(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "projectID")
	limit := parseLimit(r.URL.Query().Get("limit"), 50)
	items, err := s.store.ListMemories(r.Context(), projectID, limit)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"items": items})
}

type createKnowledgeDocumentRequest struct {
	Title     string `json:"title"`
	Source    string `json:"source"`
	Content   string `json:"content"`
	ChunkSize int    `json:"chunk_size"`
}

func (s *Server) handleCreateKnowledgeDocument(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "projectID")
	var req createKnowledgeDocumentRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, err)
		return
	}
	if strings.TrimSpace(req.Title) == "" || strings.TrimSpace(req.Content) == "" {
		respondError(w, http.StatusBadRequest, errors.New("title and content are required"))
		return
	}
	doc, chunks, err := s.store.AddKnowledgeDocument(
		r.Context(),
		projectID,
		strings.TrimSpace(req.Title),
		strings.TrimSpace(req.Source),
		strings.TrimSpace(req.Content),
		req.ChunkSize,
	)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err)
		return
	}
	respondJSON(w, http.StatusCreated, map[string]any{"document": doc, "chunks": chunks})
}

func (s *Server) handleListKnowledgeDocuments(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "projectID")
	items, err := s.store.ListKnowledgeDocuments(r.Context(), projectID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) handleSearchKnowledge(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "projectID")
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	limit := parseLimit(r.URL.Query().Get("limit"), 8)
	items, err := s.store.SearchKnowledge(r.Context(), projectID, query, limit)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"items": items})
}

type contextPackRequest struct {
	Query       string `json:"query"`
	TokenBudget int    `json:"token_budget"`
	MaxItems    int    `json:"max_items"`
}

func (s *Server) handleBuildContextPack(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "projectID")
	var req contextPackRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, err)
		return
	}
	pack, err := s.contextOpt.BuildContextPack(r.Context(), contextopt.BuildRequest{
		ProjectID:   projectID,
		Query:       strings.TrimSpace(req.Query),
		TokenBudget: req.TokenBudget,
		MaxItems:    req.MaxItems,
	})
	if err != nil {
		respondError(w, http.StatusInternalServerError, err)
		return
	}
	respondJSON(w, http.StatusOK, pack)
}

type startPipelineRequest struct {
	ProjectID          string   `json:"project_id"`
	ChangeBatchID      string   `json:"change_batch_id"`
	Prompt             string   `json:"prompt"`
	LLMEnhancedLoop    bool     `json:"llm_enhanced_loop"`
	MultimodalAssets   []string `json:"multimodal_assets"`
	Simulate           bool     `json:"simulate"`
	FullCycle          bool     `json:"full_cycle"`
	StepByStep         bool     `json:"step_by_step"`
	IterationLimit     int      `json:"iteration_limit"`
	ProjectDir         string   `json:"project_dir"`
	Platform           string   `json:"platform"`
	Frontend           string   `json:"frontend"`
	Backend            string   `json:"backend"`
	Domain             string   `json:"domain"`
	ContextMode        string   `json:"context_mode"`
	ContextQuery       string   `json:"context_query"`
	ContextTokenBudget int      `json:"context_token_budget"`
	ContextMaxItems    int      `json:"context_max_items"`
	ContextDynamic     *bool    `json:"context_dynamic"`
	MemoryWriteback    *bool    `json:"memory_writeback"`
	AgentName          string   `json:"agent_name"`
	AgentMode          string   `json:"agent_mode"`
}

func (s *Server) handleStartPipeline(w http.ResponseWriter, r *http.Request) {
	var req startPipelineRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, err)
		return
	}
	if strings.TrimSpace(req.ProjectID) == "" || strings.TrimSpace(req.Prompt) == "" {
		respondError(w, http.StatusBadRequest, errors.New("project_id and prompt are required"))
		return
	}
	project, err := s.store.GetProject(r.Context(), req.ProjectID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			respondError(w, http.StatusNotFound, err)
			return
		}
		respondError(w, http.StatusInternalServerError, err)
		return
	}
	if strings.TrimSpace(req.ChangeBatchID) != "" {
		batch, batchErr := s.store.GetChangeBatch(r.Context(), strings.TrimSpace(req.ChangeBatchID))
		if batchErr != nil {
			if errors.Is(batchErr, store.ErrNotFound) {
				respondError(w, http.StatusNotFound, batchErr)
				return
			}
			respondError(w, http.StatusInternalServerError, batchErr)
			return
		}
		if batch.ProjectID != req.ProjectID {
			respondError(w, http.StatusBadRequest, errors.New("change_batch_id does not belong to project"))
			return
		}
	}

	rawContextMode := req.ContextMode
	if strings.TrimSpace(rawContextMode) == "" {
		rawContextMode = project.DefaultContextMode
	}
	contextMode, modeErr := parseContextMode(rawContextMode)
	if modeErr != nil {
		respondError(w, http.StatusBadRequest, modeErr)
		return
	}
	if contextMode == pipeline.ContextModeManual && strings.TrimSpace(req.ContextQuery) == "" {
		respondError(w, http.StatusBadRequest, errors.New("context_query is required when context_mode=manual"))
		return
	}
	if req.StepByStep {
		req.Simulate = false
		req.FullCycle = false
	}
	if req.FullCycle {
		req.Simulate = false
		if req.IterationLimit <= 0 {
			req.IterationLimit = 3
		}
	}

	effectiveProjectDir := firstNonEmpty(strings.TrimSpace(req.ProjectDir), strings.TrimSpace(project.RepoPath))
	preferredLifecycleMode := func() string {
		if req.FullCycle {
			return advanceModeFullCycle
		}
		if req.StepByStep {
			return advanceModeStepByStep
		}
		return ""
	}()
	agentName, agentMode, selectionErr := s.resolveProjectAgentSelection(
		r,
		project,
		effectiveProjectDir,
		req.AgentName,
		resolveProjectLifecycleMode(project, req.AgentMode, preferredLifecycleMode, ""),
	)
	if selectionErr != nil {
		respondError(w, http.StatusBadRequest, selectionErr)
		return
	}

	startReq := pipeline.StartRequest{
		ProjectID:     req.ProjectID,
		ChangeBatchID: strings.TrimSpace(req.ChangeBatchID),
		Prompt:        req.Prompt,
		Simulate:      req.Simulate,
		LLM: pipeline.LLMOptions{
			EnhancedLoop:     req.LLMEnhancedLoop,
			MultimodalAssets: compactStrings(req.MultimodalAssets),
		},
		Agent: pipeline.AgentOptions{
			Name: agentName,
			Mode: agentMode,
		},
		Context: pipeline.ContextOptions{
			Mode:            contextMode,
			Query:           strings.TrimSpace(req.ContextQuery),
			TokenBudget:     intOrDefault(req.ContextTokenBudget, project.DefaultContextTokenBudget),
			MaxItems:        intOrDefault(req.ContextMaxItems, project.DefaultContextMaxItems),
			DynamicByPhase:  boolOrDefault(req.ContextDynamic, project.DefaultContextDynamic),
			MemoryWriteback: valueOrDefault(req.MemoryWriteback, project.DefaultMemoryWriteback),
		},
		Lifecycle: pipeline.LifecycleOptions{
			OneClickDelivery: req.FullCycle,
			StepByStep:       req.StepByStep,
			IterationLimit:   req.IterationLimit,
		},
		Options: pipeline.RunRequest{
			Prompt:     req.Prompt,
			ProjectDir: effectiveProjectDir,
			Platform:   firstNonEmpty(strings.TrimSpace(req.Platform), strings.TrimSpace(project.DefaultPlatform)),
			Frontend:   firstNonEmpty(strings.TrimSpace(req.Frontend), strings.TrimSpace(project.DefaultFrontend)),
			Backend:    firstNonEmpty(strings.TrimSpace(req.Backend), strings.TrimSpace(project.DefaultBackend)),
			Domain:     firstNonEmpty(strings.TrimSpace(req.Domain), strings.TrimSpace(project.DefaultDomain)),
		},
	}

	run, err := s.pipeline.Start(r.Context(), startReq)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err)
		return
	}
	respondJSON(w, http.StatusAccepted, run)
}

func (s *Server) handleRetryPipeline(w http.ResponseWriter, r *http.Request) {
	runID := chi.URLParam(r, "runID")
	previousRun, err := s.store.GetPipelineRun(r.Context(), runID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			respondError(w, http.StatusNotFound, err)
			return
		}
		respondError(w, http.StatusInternalServerError, err)
		return
	}
	if previousRun.Status != "failed" {
		respondError(w, http.StatusBadRequest, errors.New("only failed runs can be retried"))
		return
	}
	run, startErr := s.restartPipelineRun(r.Context(), previousRun)
	if startErr != nil {
		respondError(w, http.StatusInternalServerError, startErr)
		return
	}

	_, _ = s.store.AppendRunEvent(r.Context(), store.RunEvent{
		RunID:   run.ID,
		Stage:   "retry",
		Status:  "log",
		Message: fmt.Sprintf("Retried from failed run %s", previousRun.ID),
	})

	respondJSON(w, http.StatusAccepted, run)
}

func (s *Server) handleResumePipeline(w http.ResponseWriter, r *http.Request) {
	runID := chi.URLParam(r, "runID")
	previousRun, err := s.store.GetPipelineRun(r.Context(), runID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			respondError(w, http.StatusNotFound, err)
			return
		}
		respondError(w, http.StatusInternalServerError, err)
		return
	}
	if previousRun.Status != "awaiting_human" {
		respondError(w, http.StatusBadRequest, errors.New("only awaiting_human runs can be resumed"))
		return
	}
	run, startErr := s.restartPipelineRun(r.Context(), previousRun)
	if startErr != nil {
		respondError(w, http.StatusInternalServerError, startErr)
		return
	}

	_, _ = s.store.AppendRunEvent(r.Context(), store.RunEvent{
		RunID:   run.ID,
		Stage:   "resume",
		Status:  "log",
		Message: fmt.Sprintf("Resumed from awaiting_human run %s", previousRun.ID),
	})

	respondJSON(w, http.StatusAccepted, run)
}

type approvePipelineToolRequest struct {
	ToolName string `json:"tool_name"`
}

func (s *Server) handleApprovePipelineTool(w http.ResponseWriter, r *http.Request) {
	runID := chi.URLParam(r, "runID")
	var req approvePipelineToolRequest
	if err := decodeJSON(r, &req); err != nil && !errors.Is(err, io.EOF) {
		respondError(w, http.StatusBadRequest, err)
		return
	}
	run, err := s.pipeline.ApprovePendingTool(r.Context(), runID, req.ToolName)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			respondError(w, http.StatusNotFound, err)
			return
		}
		respondError(w, http.StatusBadRequest, err)
		return
	}
	respondJSON(w, http.StatusAccepted, run)
}

func (s *Server) restartPipelineRun(ctx context.Context, previousRun store.PipelineRun) (store.PipelineRun, error) {

	contextMode, modeErr := parseContextMode(previousRun.ContextMode)
	if modeErr != nil {
		return store.PipelineRun{}, modeErr
	}
	if contextMode == pipeline.ContextModeManual && strings.TrimSpace(previousRun.ContextQuery) == "" {
		return store.PipelineRun{}, errors.New("context_query is required when context_mode=manual")
	}

	retryAgentName := ""
	retryAgentMode := ""
	if previousAgentRun, agentErr := s.store.GetAgentRunByPipelineRun(ctx, previousRun.ID); agentErr == nil {
		retryAgentName = previousAgentRun.AgentName
		retryAgentMode = previousAgentRun.ModeName
	}

	startReq := pipeline.StartRequest{
		ProjectID:     previousRun.ProjectID,
		ChangeBatchID: previousRun.ChangeBatchID,
		Prompt:        previousRun.Prompt,
		Simulate:      previousRun.Simulate,
		RetryOf:       previousRun.ID,
		LLM: pipeline.LLMOptions{
			EnhancedLoop:     previousRun.LLMEnhancedLoop,
			MultimodalAssets: compactStrings(previousRun.MultimodalAssets),
		},
		Agent: pipeline.AgentOptions{
			Name: retryAgentName,
			Mode: retryAgentMode,
		},
		Context: pipeline.ContextOptions{
			Mode:            contextMode,
			Query:           strings.TrimSpace(previousRun.ContextQuery),
			TokenBudget:     previousRun.ContextTokenBudget,
			MaxItems:        previousRun.ContextMaxItems,
			DynamicByPhase:  previousRun.ContextDynamic,
			MemoryWriteback: previousRun.MemoryWriteback,
		},
		Lifecycle: pipeline.LifecycleOptions{
			OneClickDelivery: previousRun.FullCycle,
			StepByStep:       previousRun.StepByStep,
			IterationLimit: func() int {
				if previousRun.FullCycle && previousRun.IterationLimit <= 0 {
					return 3
				}
				return previousRun.IterationLimit
			}(),
		},
		Options: pipeline.RunRequest{
			Prompt:     previousRun.Prompt,
			ProjectDir: s.resolveProjectDirForRetry(ctx, previousRun),
			Platform:   strings.TrimSpace(previousRun.Platform),
			Frontend:   strings.TrimSpace(previousRun.Frontend),
			Backend:    strings.TrimSpace(previousRun.Backend),
			Domain:     strings.TrimSpace(previousRun.Domain),
		},
	}

	return s.pipeline.Start(ctx, startReq)
}

func valueOrDefault(value *bool, defaultValue bool) bool {
	if value == nil {
		return defaultValue
	}
	return *value
}

func boolOrDefault(value *bool, defaultValue bool) bool {
	if value == nil {
		return defaultValue
	}
	return *value
}

func intOrDefault(value int, defaultValue int) int {
	if value <= 0 {
		return defaultValue
	}
	return value
}

func compactStrings(items []string) []string {
	result := make([]string, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	for _, raw := range items {
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	return result
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func parseAdvanceMode(raw string) (string, error) {
	mode := strings.ToLower(strings.TrimSpace(raw))
	if mode == "" {
		return advanceModeStepByStep, nil
	}
	switch mode {
	case advanceModeStepByStep, advanceModeFullCycle:
		return mode, nil
	default:
		return "", errors.New("mode must be one of: step_by_step, full_cycle")
	}
}

func resolveProjectLifecycleMode(project store.Project, requestedModeName, preferredLifecycleMode, fallbackMode string) string {
	requested := strings.TrimSpace(requestedModeName)
	projectDefault := strings.TrimSpace(project.DefaultAgentMode)
	preferred := strings.TrimSpace(preferredLifecycleMode)
	if preferred == advanceModeFullCycle && (requested == "" || strings.EqualFold(requested, projectDefault)) {
		return preferred
	}
	return firstNonEmpty(requested, projectDefault, fallbackMode)
}

func buildChangeBatchTitle(projectName, goal, mode string) string {
	base := strings.TrimSpace(goal)
	if base == "" {
		base = strings.TrimSpace(projectName)
	}
	if base == "" {
		base = "项目推进批次"
	}
	if mode == advanceModeFullCycle {
		return base + " / 全流程交付"
	}
	return base + " / 逐步推进"
}

func (s *Server) ensureSuperDevUsageMemory(ctx context.Context, projectID string) (store.Memory, bool, error) {
	memories, err := s.store.ListMemories(ctx, projectID, 200)
	if err != nil {
		return store.Memory{}, false, err
	}
	for _, mem := range memories {
		for _, tag := range mem.Tags {
			if strings.EqualFold(strings.TrimSpace(tag), superDevUsageMemoryTag) {
				return mem, false, nil
			}
		}
	}

	mem, err := s.store.CreateMemory(ctx, store.Memory{
		ProjectID:  projectID,
		Role:       "playbook",
		Content:    buildSuperDevUsageMemoryContent(),
		Tags:       []string{"super-dev", "workflow", "commands", "iteration", superDevUsageMemoryTag},
		Importance: 0.95,
	})
	if err != nil {
		return store.Memory{}, false, err
	}
	return mem, true, nil
}

func buildSuperDevUsageMemoryContent() string {
	return strings.TrimSpace(`super-dev 使用方法（来源：GitHub README + WORKFLOW_GUIDE + CLI 参数定义）:

1) 需求直达模式（推荐）
- super-dev "你的业务需求"
- 自动执行 0-11 阶段：需求增强、文档、Spec、实现骨架、红队、质量门禁、CI/CD、交付包。

2) 增量迭代模式（1-N+1）
- super-dev analyze .
- super-dev spec propose <change_id> --title "标题" --description "描述"
- super-dev spec add-req <change_id> <spec_name> <req_name> "系统 SHALL ..."
- super-dev task status <change_id>
- super-dev task run <change_id> --max-retries 3
- super-dev quality --type all
- 质量未通过时继续 task run + quality 迭代；通过后 super-dev spec archive <change_id>

3) 常用命令参数
- super-dev pipeline "需求" --platform web --frontend react --backend go --cicd all
- super-dev pipeline 支持 --skip-redteam / --skip-scaffold / --skip-quality-gate / --quality-threshold
- super-dev task run 支持 --max-retries

4) 发布门禁
- 红队 critical=0
- 质量门禁建议 >= 80
- Spec 任务闭环后再归档
`)
}

func buildProjectAdvancePrompt(project store.Project, tasks []store.Task, goal string) string {
	target := strings.TrimSpace(goal)
	if target == "" {
		target = strings.TrimSpace(project.Description)
	}
	if target == "" {
		target = "围绕当前任务看板持续推进项目功能与质量闭环"
	}
	taskSummary := summarizeTaskBoard(tasks)
	return strings.TrimSpace(fmt.Sprintf(
		"项目推进目标：%s\n"+
			"项目名称：%s\n"+
			"任务看板摘要：\n%s\n\n"+
			"请按 super-dev 增量迭代方法推进：\n"+
			"1. 分析现状并识别下一批次 change scope\n"+
			"2. 创建/更新 spec 变更并对齐任务闭环\n"+
			"3. 执行 task run，优先清理 todo 和 in_progress\n"+
			"4. 执行 quality --type all，未通过则继续迭代修复\n"+
			"5. 输出明确的下一轮行动项、风险项和验收标准\n"+
			"6. 保持可持续迭代，不做一次性大爆炸改动",
		target,
		project.Name,
		taskSummary,
	))
}

func summarizeTaskBoard(tasks []store.Task) string {
	if len(tasks) == 0 {
		return "- 当前任务看板为空，请先从需求文档生成任务并按模块拆分。"
	}
	todoCount := 0
	inProgressCount := 0
	doneCount := 0
	openItems := make([]string, 0, len(tasks))
	for _, task := range tasks {
		status := strings.ToLower(strings.TrimSpace(task.Status))
		switch status {
		case "done":
			doneCount++
		case "in_progress":
			inProgressCount++
			openItems = append(openItems, fmt.Sprintf("- [in_progress|%s] %s", task.Priority, task.Title))
		default:
			todoCount++
			openItems = append(openItems, fmt.Sprintf("- [todo|%s] %s", task.Priority, task.Title))
		}
	}
	if len(openItems) > 8 {
		openItems = openItems[:8]
	}
	return strings.TrimSpace(fmt.Sprintf(
		"- 总任务: %d\n- todo: %d\n- in_progress: %d\n- done: %d\n%s",
		len(tasks),
		todoCount,
		inProgressCount,
		doneCount,
		strings.Join(openItems, "\n"),
	))
}

func (s *Server) resolveProjectDirForAdvance(ctx context.Context, projectID string, repoPath string) string {
	if candidate := sanitizeProjectDir(strings.TrimSpace(repoPath)); candidate != "" {
		return candidate
	}
	runs, err := s.store.ListPipelineRuns(ctx, projectID, 20)
	if err != nil {
		return ""
	}
	for _, run := range runs {
		candidate := sanitizeProjectDir(strings.TrimSpace(run.ProjectDir))
		if candidate != "" {
			return candidate
		}
	}
	return ""
}

func (s *Server) resolveProjectDirForRetry(ctx context.Context, previousRun store.PipelineRun) string {
	if candidate := sanitizeProjectDir(strings.TrimSpace(previousRun.ProjectDir)); candidate != "" {
		return candidate
	}
	project, err := s.store.GetProject(ctx, previousRun.ProjectID)
	if err == nil {
		if candidate := sanitizeProjectDir(strings.TrimSpace(project.RepoPath)); candidate != "" {
			return candidate
		}
	}
	runs, err := s.store.ListPipelineRuns(ctx, previousRun.ProjectID, 20)
	if err != nil {
		return ""
	}
	for _, run := range runs {
		candidate := sanitizeProjectDir(strings.TrimSpace(run.ProjectDir))
		if candidate != "" {
			return candidate
		}
	}
	return ""
}

func sanitizeProjectDir(projectDir string) string {
	trimmed := strings.TrimSpace(projectDir)
	if trimmed == "" {
		return ""
	}
	info, err := os.Stat(trimmed)
	if err != nil || !info.IsDir() {
		return ""
	}
	return trimmed
}

func parseContextMode(raw string) (pipeline.ContextMode, error) {
	mode := pipeline.ContextMode(strings.ToLower(strings.TrimSpace(raw)))
	if mode == "" {
		return pipeline.ContextModeOff, nil
	}
	switch mode {
	case pipeline.ContextModeOff, pipeline.ContextModeAuto, pipeline.ContextModeManual:
		return mode, nil
	default:
		return "", errors.New("context_mode must be one of: off, auto, manual")
	}
}

func (s *Server) handleGetPipelineRun(w http.ResponseWriter, r *http.Request) {
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
	respondJSON(w, http.StatusOK, run)
}

type pipelineCompletionItem struct {
	Key    string `json:"key"`
	Title  string `json:"title"`
	Status string `json:"status"`
	Note   string `json:"note,omitempty"`
}

type pipelineArtifact struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	Kind        string `json:"kind"`
	SizeBytes   int64  `json:"size_bytes"`
	UpdatedAt   string `json:"updated_at"`
	PreviewURL  string `json:"preview_url,omitempty"`
	PreviewType string `json:"preview_type,omitempty"`
	Stage       string `json:"stage,omitempty"`
}

type pipelineStage struct {
	Key       string             `json:"key"`
	Title     string             `json:"title"`
	Status    string             `json:"status"`
	Artifacts []pipelineArtifact `json:"artifacts"`
}

type pipelineCompletionResponse struct {
	RunID      string                   `json:"run_id"`
	Status     string                   `json:"status"`
	OutputDir  string                   `json:"output_dir"`
	Checklist  []pipelineCompletionItem `json:"checklist"`
	Artifacts  []pipelineArtifact       `json:"artifacts"`
	Stages     []pipelineStage          `json:"stages"`
	PreviewURL string                   `json:"preview_url,omitempty"`
}

func (s *Server) handleGetPipelineRunCompletion(w http.ResponseWriter, r *http.Request) {
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

	resp := buildPipelineCompletionResponse(run)
	respondJSON(w, http.StatusOK, resp)
}

func (s *Server) handlePreviewPipelineRunOutput(w http.ResponseWriter, r *http.Request) {
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

	outputRoot := filepath.Join(resolveRunBaseDir(run), "output")
	requestPath := strings.TrimSpace(chi.URLParam(r, "*"))
	if requestPath == "" {
		requestPath = detectDefaultPreviewPath(outputRoot)
	}
	requestPath = strings.TrimPrefix(requestPath, "/")
	cleanPath := filepath.Clean(requestPath)
	if cleanPath == "." {
		cleanPath = detectDefaultPreviewPath(outputRoot)
	}

	target, resolveErr := secureJoin(outputRoot, cleanPath)
	if resolveErr != nil {
		respondError(w, http.StatusBadRequest, resolveErr)
		return
	}

	info, statErr := os.Stat(target)
	if statErr != nil {
		if os.IsNotExist(statErr) {
			respondError(w, http.StatusNotFound, statErr)
			return
		}
		respondError(w, http.StatusInternalServerError, statErr)
		return
	}
	if info.IsDir() {
		target = filepath.Join(target, "index.html")
		if _, dirErr := os.Stat(target); dirErr != nil {
			respondError(w, http.StatusNotFound, dirErr)
			return
		}
	}

	file, openErr := os.Open(target)
	if openErr != nil {
		respondError(w, http.StatusInternalServerError, openErr)
		return
	}
	defer file.Close()

	http.ServeContent(w, r, info.Name(), info.ModTime(), file)
}

func buildPipelineCompletionResponse(run store.PipelineRun) pipelineCompletionResponse {
	outputDir := filepath.Join(resolveRunBaseDir(run), "output")
	scanned := collectPipelineOutputFiles(run)
	artifacts := make([]pipelineArtifact, 0, len(scanned))
	for _, item := range scanned {
		artifacts = append(artifacts, buildScannedArtifact(run, item))
	}
	sort.SliceStable(artifacts, func(i, j int) bool {
		leftOrder := pipelineStageOrder(artifacts[i].Stage)
		rightOrder := pipelineStageOrder(artifacts[j].Stage)
		if leftOrder == rightOrder {
			return artifacts[i].Path < artifacts[j].Path
		}
		return leftOrder < rightOrder
	})
	stages := buildPipelineStages(run, artifacts)
	previewURL := choosePrimaryPreviewURL(artifacts)
	checklist := []pipelineCompletionItem{
		{
			Key:    "run-status",
			Title:  "流水线状态",
			Status: normalizeRunStatus(run.Status),
			Note:   run.Status,
		},
	}
	for _, stage := range stages {
		checklist = append(checklist, pipelineCompletionItem{
			Key:    "stage-" + stage.Key,
			Title:  stage.Title + "阶段产物",
			Status: stage.Status,
			Note: func() string {
				if len(stage.Artifacts) == 0 {
					return "暂无可预览产物"
				}
				return fmt.Sprintf("已发现 %d 个产物", len(stage.Artifacts))
			}(),
		})
	}
	checklist = append(checklist, pipelineCompletionItem{
		Key:    "frontend-preview",
		Title:  "前端页面预览",
		Status: statusFromBool(previewURL != ""),
		Note: func() string {
			if previewURL == "" {
				return "未检测到 HTML 预览页面"
			}
			return "可在线预览"
		}(),
	})
	return pipelineCompletionResponse{
		RunID:      run.ID,
		Status:     run.Status,
		OutputDir:  outputDir,
		Checklist:  checklist,
		Artifacts:  artifacts,
		Stages:     stages,
		PreviewURL: previewURL,
	}
}

func buildArtifact(name, kind, fullPath, baseDir string) pipelineArtifact {
	info, err := os.Stat(fullPath)
	if err != nil {
		return pipelineArtifact{
			Name: name,
			Path: filepath.ToSlash(fullPath),
			Kind: kind,
		}
	}
	relative, relErr := filepath.Rel(baseDir, fullPath)
	pathValue := fullPath
	if relErr == nil && !strings.HasPrefix(relative, "..") {
		pathValue = relative
	}
	return pipelineArtifact{
		Name:      name,
		Path:      filepath.ToSlash(pathValue),
		Kind:      kind,
		SizeBytes: info.Size(),
		UpdatedAt: info.ModTime().UTC().Format("2006-01-02T15:04:05Z07:00"),
	}
}

func normalizeRunStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "completed":
		return "completed"
	case "failed":
		return "failed"
	default:
		return "in_progress"
	}
}

func statusFromBool(ok bool) string {
	if ok {
		return "completed"
	}
	return "missing"
}

func resolveRunBaseDir(run store.PipelineRun) string {
	baseDir := strings.TrimSpace(run.ProjectDir)
	if baseDir == "" {
		baseDir = "."
	}
	abs, err := filepath.Abs(baseDir)
	if err != nil {
		return baseDir
	}
	return abs
}

func detectDefaultPreviewPath(outputRoot string) string {
	if fileExists(filepath.Join(outputRoot, "frontend", "index.html")) {
		return "frontend/index.html"
	}
	if fileExists(filepath.Join(outputRoot, "preview.html")) {
		return "preview.html"
	}
	return "frontend/index.html"
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func secureJoin(root, relativePath string) (string, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	target := filepath.Join(absRoot, relativePath)
	absTarget, err := filepath.Abs(target)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(absRoot, absTarget)
	if err != nil {
		return "", err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", errors.New("invalid preview path")
	}
	return absTarget, nil
}

func changeIDFromPrompt(prompt string) string {
	trimmed := strings.TrimSpace(prompt)
	if trimmed == "" {
		return "pipeline-run"
	}
	var builder strings.Builder
	lastDash := false
	for _, r := range trimmed {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			builder.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			builder.WriteRune('-')
			lastDash = true
		}
	}
	result := strings.Trim(builder.String(), "-")
	if result == "" {
		return "pipeline-run"
	}
	return result
}

func (s *Server) handleListRunEvents(w http.ResponseWriter, r *http.Request) {
	runID := chi.URLParam(r, "runID")
	items, err := s.store.ListRunEvents(r.Context(), runID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) handleListPipelineRuns(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "projectID")
	limit := parseLimit(r.URL.Query().Get("limit"), 20)
	items, err := s.store.ListPipelineRuns(r.Context(), projectID, limit)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"items": items})
}
