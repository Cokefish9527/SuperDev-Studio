package api

import (
	"context"
	"testing"
	"time"

	"superdevstudio/internal/store"
)

func TestAutoAdvanceWorkerReconcileOnceStartsRetryAndSyncsRequirementSession(t *testing.T) {
	ctx := context.Background()
	env := newAPITestEnv(t)
	repoRoot := t.TempDir()
	project := createProjectViaAPIWithPayload(t, env.handler, map[string]any{
		"name":        "BackgroundAutoAdvanceProject",
		"description": "test project",
		"repo_path":   repoRoot,
	})
	changeBatch, err := env.store.CreateChangeBatch(ctx, store.ChangeBatch{
		ProjectID: project.ID,
		Title:     "Auto advance batch",
		Goal:      "Keep safe delivery moving in the background",
		Status:    "running",
		Mode:      "step_by_step",
	})
	if err != nil {
		t.Fatalf("create change batch: %v", err)
	}
	session, err := env.store.CreateRequirementSession(ctx, store.RequirementSession{
		ProjectID:           project.ID,
		Title:               "Timeline notebook",
		RawInput:            "Build a timeline notebook.",
		Status:              "confirmed",
		LatestChangeBatchID: changeBatch.ID,
	})
	if err != nil {
		t.Fatalf("create requirement session: %v", err)
	}
	run, err := env.store.CreatePipelineRun(ctx, store.PipelineRun{
		ProjectID:          project.ID,
		ChangeBatchID:      changeBatch.ID,
		Prompt:             "Build a timeline notebook.",
		LLMEnhancedLoop:    true,
		Simulate:           true,
		ProjectDir:         repoRoot,
		Platform:           "web",
		Frontend:           "react",
		Backend:            "go",
		Domain:             "saas",
		ContextMode:        "auto",
		ContextTokenBudget: 1200,
		ContextMaxItems:    8,
		ContextDynamic:     true,
		MemoryWriteback:    true,
		StepByStep:         true,
		IterationLimit:     3,
		Status:             "failed",
		Progress:           100,
		Stage:              "done",
	})
	if err != nil {
		t.Fatalf("create failed pipeline run: %v", err)
	}
	session.LatestRunID = run.ID
	if err := env.store.UpdateRequirementSession(ctx, session); err != nil {
		t.Fatalf("update requirement session: %v", err)
	}
	agentRun, err := env.store.CreateAgentRun(ctx, store.AgentRun{
		PipelineRunID: run.ID,
		ProjectID:     project.ID,
		ChangeBatchID: changeBatch.ID,
		AgentName:     "reviewer",
		ModeName:      "step_by_step",
		Status:        "completed",
		CurrentNode:   "done",
	})
	if err != nil {
		t.Fatalf("create agent run: %v", err)
	}
	step, err := env.store.CreateAgentStep(ctx, store.AgentStep{
		AgentRunID: agentRun.ID,
		NodeName:   "agent-evaluate-task-attempt",
		Title:      "Evaluate failed run",
		InputJSON:  "{}",
		Status:     "completed",
	})
	if err != nil {
		t.Fatalf("create agent step: %v", err)
	}
	if _, err := env.store.CreateAgentEvaluation(ctx, store.AgentEvaluation{
		AgentStepID:    step.ID,
		EvaluationType: "step-outcome",
		Verdict:        "fail",
		Reason:         "Retry the delivery in the background.",
		NextAction:     "Start another run.",
		NextCommand:    "rerun_delivery",
	}); err != nil {
		t.Fatalf("create agent evaluation: %v", err)
	}

	worker := NewAutoAdvanceWorker(env.server, AutoAdvanceWorkerConfig{Enabled: true, Interval: time.Millisecond, BatchSize: 10})
	if err := worker.reconcileOnce(ctx); err != nil {
		t.Fatalf("reconcile once: %v", err)
	}

	updatedSession, err := env.store.GetRequirementSession(ctx, session.ID)
	if err != nil {
		t.Fatalf("get updated requirement session: %v", err)
	}
	if updatedSession.LatestRunID == "" || updatedSession.LatestRunID == run.ID {
		t.Fatalf("expected latest run id to move to retry run, got %s", updatedSession.LatestRunID)
	}
	retryRun, err := env.store.GetPipelineRun(ctx, updatedSession.LatestRunID)
	if err != nil {
		t.Fatalf("get retry run: %v", err)
	}
	if retryRun.RetryOf != run.ID {
		t.Fatalf("expected retry run to point to %s, got %s", run.ID, retryRun.RetryOf)
	}

	if err := worker.reconcileOnce(ctx); err != nil {
		t.Fatalf("second reconcile once: %v", err)
	}
	runs, err := env.store.ListPipelineRuns(ctx, project.ID, 20)
	if err != nil {
		t.Fatalf("list project runs: %v", err)
	}
	retryCount := 0
	for _, item := range runs {
		if item.RetryOf == run.ID {
			retryCount++
		}
	}
	if retryCount != 1 {
		t.Fatalf("expected exactly one retry run for %s, got %d", run.ID, retryCount)
	}
}
