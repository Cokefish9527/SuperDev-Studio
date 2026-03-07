package eino

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"superdevstudio/internal/agentruntime"
	"superdevstudio/internal/retrieval"
	"superdevstudio/internal/store"
)

func newRuntimeTestDeps(t *testing.T) (context.Context, *store.Store, *retrieval.Service, store.Project, store.PipelineRun, *Runtime) {
	t.Helper()
	ctx := context.Background()
	s, err := store.New(filepath.Join(t.TempDir(), "eino.db"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })

	project, err := s.CreateProject(ctx, store.Project{Name: "Eino Runtime"})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	run, err := s.CreatePipelineRun(ctx, store.PipelineRun{
		ProjectID: project.ID,
		Prompt:    "build runtime",
		Status:    "running",
		Stage:     "plan",
	})
	if err != nil {
		t.Fatalf("create pipeline run: %v", err)
	}
	service := retrieval.NewService(s)
	runtime, err := New(ctx, s, service, Config{})
	if err != nil {
		t.Fatalf("new runtime: %v", err)
	}
	return ctx, s, service, project, run, runtime
}

func TestRuntimePlanPersistsStepAndEvidence(t *testing.T) {
	ctx, s, _, project, run, runtime := newRuntimeTestDeps(t)
	if _, err := s.CreateMemory(ctx, store.Memory{
		ProjectID:  project.ID,
		Role:       "note",
		Content:    "Need delivery traceability for runtime planning",
		Importance: 0.9,
	}); err != nil {
		t.Fatalf("create memory: %v", err)
	}
	if _, err := s.CreateTask(ctx, store.Task{
		ProjectID:   project.ID,
		Title:       "Trace runtime flow",
		Description: "Capture evidence and next action",
		Priority:    "high",
		Status:      "todo",
	}); err != nil {
		t.Fatalf("create task: %v", err)
	}

	agentRun, err := runtime.StartRun(ctx, agentruntime.StartRunRequest{
		PipelineRunID: run.ID,
		ProjectID:     project.ID,
		AgentName:     "delivery-agent",
		ModeName:      "step_by_step",
		CurrentNode:   "plan",
	})
	if err != nil {
		t.Fatalf("start run: %v", err)
	}

	result, err := runtime.Plan(ctx, agentruntime.PlanRequest{
		AgentRunID:    agentRun.ID,
		ProjectID:     project.ID,
		PipelineRunID: run.ID,
		NodeName:      "plan",
		Title:         "Plan next step",
		Goal:          "Improve delivery runtime",
		Query:         "runtime traceability",
		ModeName:      "step_by_step",
		MaxEvidence:   5,
		AllowedTools:  []string{"run_superdev_task_status"},
		Context:       map[string]any{"source": "test"},
	})
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	if result.Step.Status != "completed" {
		t.Fatalf("expected completed step, got %s", result.Step.Status)
	}
	if len(result.Evidence) == 0 {
		t.Fatalf("expected persisted evidence")
	}
	if result.SuggestedTool != "run_superdev_task_status" {
		t.Fatalf("unexpected suggested tool %q", result.SuggestedTool)
	}

	steps, err := s.ListAgentSteps(ctx, agentRun.ID)
	if err != nil {
		t.Fatalf("list steps: %v", err)
	}
	if len(steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(steps))
	}
	evidence, err := s.ListAgentEvidence(ctx, agentRun.ID)
	if err != nil {
		t.Fatalf("list evidence: %v", err)
	}
	if len(evidence) == 0 {
		t.Fatalf("expected evidence to be stored")
	}
}

func TestRuntimePlanFallsBackWithoutEvidence(t *testing.T) {
	ctx, _, _, project, run, runtime := newRuntimeTestDeps(t)
	agentRun, err := runtime.StartRun(ctx, agentruntime.StartRunRequest{
		PipelineRunID: run.ID,
		ProjectID:     project.ID,
		CurrentNode:   "plan",
	})
	if err != nil {
		t.Fatalf("start run: %v", err)
	}

	result, err := runtime.Plan(ctx, agentruntime.PlanRequest{
		AgentRunID:    agentRun.ID,
		ProjectID:     project.ID,
		PipelineRunID: run.ID,
		NodeName:      "plan",
		Goal:          "No evidence path",
		Query:         "missing phrase",
		AllowedTools:  []string{"run_superdev_task_status"},
	})
	if err != nil {
		t.Fatalf("plan without evidence: %v", err)
	}
	if result.Summary == "" || result.NextAction == "" {
		t.Fatalf("expected fallback summary and next action")
	}
	if len(result.Evidence) != 0 {
		t.Fatalf("expected no evidence, got %d", len(result.Evidence))
	}
}

func TestRuntimeEvaluateRecordAndFinishRun(t *testing.T) {
	ctx, s, _, project, run, runtime := newRuntimeTestDeps(t)
	agentRun, err := runtime.StartRun(ctx, agentruntime.StartRunRequest{
		PipelineRunID: run.ID,
		ProjectID:     project.ID,
		CurrentNode:   "evaluate",
	})
	if err != nil {
		t.Fatalf("start run: %v", err)
	}

	passed, err := runtime.Evaluate(ctx, agentruntime.EvaluateRequest{
		AgentRunID:     agentRun.ID,
		NodeName:       "evaluate",
		Title:          "Check quality",
		Goal:           "Ship safely",
		TaskTitle:      "Run quality gate",
		Attempt:        1,
		QualitySummary: "质量门已通过",
	})
	if err != nil {
		t.Fatalf("evaluate pass: %v", err)
	}
	if passed.Verdict != "pass" {
		t.Fatalf("expected pass verdict, got %s", passed.Verdict)
	}

	retry, err := runtime.Evaluate(ctx, agentruntime.EvaluateRequest{
		AgentRunID:     agentRun.ID,
		NodeName:       "evaluate",
		Goal:           "Ship safely",
		TaskTitle:      "Repair issue",
		Attempt:        2,
		QualitySummary: "still unresolved",
	})
	if err != nil {
		t.Fatalf("evaluate retry: %v", err)
	}
	if retry.Verdict != "retry" {
		t.Fatalf("expected retry verdict, got %s", retry.Verdict)
	}

	toolCall, err := runtime.RecordToolCall(ctx, agentruntime.ToolCallRequest{
		AgentStepID: passed.Step.ID,
		ToolName:    "run_superdev_task_status",
		Request:     map[string]any{"change_id": "rate-limit-hardening"},
		Response:    map[string]any{"status": "ok"},
		Success:     true,
		Latency:     25 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("record tool call: %v", err)
	}
	if !toolCall.Success {
		t.Fatalf("expected tool call success to persist")
	}

	if err := runtime.FinishRun(ctx, agentRun.ID, "done", "completed from test"); err != nil {
		t.Fatalf("finish run: %v", err)
	}
	loadedRun, err := s.GetAgentRunByPipelineRun(ctx, run.ID)
	if err != nil {
		t.Fatalf("load agent run: %v", err)
	}
	if loadedRun.Status != "completed" {
		t.Fatalf("expected completed run, got %s", loadedRun.Status)
	}

	evaluations, err := s.ListAgentEvaluations(ctx, agentRun.ID)
	if err != nil {
		t.Fatalf("list evaluations: %v", err)
	}
	if len(evaluations) < 2 {
		t.Fatalf("expected evaluations to be stored, got %d", len(evaluations))
	}
}
