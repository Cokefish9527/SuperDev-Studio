package api

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"superdevstudio/internal/store"
)

func TestSyncRunFollowupsResolvesHistoricalResidualsForLatestRun(t *testing.T) {
	env := newAPITestEnv(t)
	ctx := context.Background()
	projectDir := t.TempDir()
	previewDir := filepath.Join(projectDir, "output", "frontend")
	if err := os.MkdirAll(previewDir, 0o755); err != nil {
		t.Fatalf("create preview dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(previewDir, "index.html"), []byte("<html><body>preview</body></html>"), 0o644); err != nil {
		t.Fatalf("write preview file: %v", err)
	}

	project := createProjectViaAPIWithPayload(t, env.handler, map[string]any{
		"name":        "ResidualResolveProject",
		"description": "test project",
		"repo_path":   projectDir,
	})
	batch, err := env.store.CreateChangeBatch(ctx, store.ChangeBatch{
		ProjectID: project.ID,
		Title:     "Residual Batch",
		Goal:      "Resolve historical residuals",
		Status:    "running",
		Mode:      "step_by_step",
	})
	if err != nil {
		t.Fatalf("create change batch: %v", err)
	}
	oldRun, err := env.store.CreatePipelineRun(ctx, store.PipelineRun{
		ProjectID:     project.ID,
		ChangeBatchID: batch.ID,
		Prompt:        "old run",
		Status:        "failed",
		Progress:      100,
		Stage:         "quality-gate",
		ProjectDir:    projectDir,
	})
	if err != nil {
		t.Fatalf("create old run: %v", err)
	}
	oldResidual, err := env.store.UpsertResidualItem(ctx, store.ResidualItem{
		ProjectID:        project.ID,
		PipelineRunID:    oldRun.ID,
		Stage:            "quality-gate",
		Category:         "quality",
		Severity:         "high",
		Summary:          "Need more tests",
		Evidence:         "Previous run failed quality validation.",
		SuggestedCommand: "POST /api/pipeline/runs/old/auto-advance",
		SourceKey:        "sync:run:" + oldRun.ID + ":missing:0",
		Status:           "open",
	})
	if err != nil {
		t.Fatalf("create old residual: %v", err)
	}
	latestRun, err := env.store.CreatePipelineRun(ctx, store.PipelineRun{
		ProjectID:     project.ID,
		ChangeBatchID: batch.ID,
		Prompt:        "latest run",
		Status:        "completed",
		Progress:      100,
		Stage:         "delivery",
		ProjectDir:    projectDir,
	})
	if err != nil {
		t.Fatalf("create latest run: %v", err)
	}
	if _, err := env.store.UpdateChangeBatch(ctx, batch.ID, batch.Status, latestRun.ID, batch.ExternalChangeID); err != nil {
		t.Fatalf("update change batch latest run: %v", err)
	}

	if err := env.server.syncRunFollowups(ctx, latestRun); err != nil {
		t.Fatalf("sync followups: %v", err)
	}

	updatedResidual, err := env.store.GetResidualItem(ctx, oldResidual.ID)
	if err != nil {
		t.Fatalf("get updated residual: %v", err)
	}
	if updatedResidual.Status != "resolved" {
		t.Fatalf("expected historical residual to resolve, got %s", updatedResidual.Status)
	}
	if !strings.Contains(updatedResidual.ResolutionNote, "Resolved by latest run") {
		t.Fatalf("expected resolution note to mention latest run, got %q", updatedResidual.ResolutionNote)
	}
	events, err := env.store.ListRunEvents(ctx, latestRun.ID)
	if err != nil {
		t.Fatalf("list latest run events: %v", err)
	}
	found := false
	for _, event := range events {
		if event.Stage == "backlog-reconcile" && strings.Contains(event.Message, "1 resolved") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected backlog reconciliation event for latest run")
	}
}

func TestSyncRunFollowupsSupersedesHistoricalResidualsStillPresent(t *testing.T) {
	env := newAPITestEnv(t)
	ctx := context.Background()
	projectDir := t.TempDir()
	project := createProjectViaAPIWithPayload(t, env.handler, map[string]any{
		"name":        "ResidualCarryProject",
		"description": "test project",
		"repo_path":   projectDir,
	})
	batch, err := env.store.CreateChangeBatch(ctx, store.ChangeBatch{
		ProjectID: project.ID,
		Title:     "Carry Batch",
		Goal:      "Carry forward residuals",
		Status:    "running",
		Mode:      "step_by_step",
	})
	if err != nil {
		t.Fatalf("create change batch: %v", err)
	}
	oldRun, err := env.store.CreatePipelineRun(ctx, store.PipelineRun{
		ProjectID:     project.ID,
		ChangeBatchID: batch.ID,
		Prompt:        "old failed run",
		Status:        "failed",
		Progress:      100,
		Stage:         "quality-gate",
		ProjectDir:    projectDir,
	})
	if err != nil {
		t.Fatalf("create old run: %v", err)
	}
	oldResidual, err := env.store.UpsertResidualItem(ctx, store.ResidualItem{
		ProjectID:        project.ID,
		PipelineRunID:    oldRun.ID,
		Stage:            "quality-gate",
		Category:         "quality",
		Severity:         "high",
		Summary:          "?????quality-gate",
		Evidence:         "Old run failed during quality gate.",
		SuggestedCommand: "POST /api/pipeline/runs/old/auto-advance",
		SourceKey:        "sync:run:" + oldRun.ID + ":failed",
		Status:           "open",
	})
	if err != nil {
		t.Fatalf("create old residual: %v", err)
	}
	latestRun, err := env.store.CreatePipelineRun(ctx, store.PipelineRun{
		ProjectID:     project.ID,
		ChangeBatchID: batch.ID,
		Prompt:        "latest failed run",
		Status:        "failed",
		Progress:      100,
		Stage:         "quality-gate",
		ProjectDir:    projectDir,
	})
	if err != nil {
		t.Fatalf("create latest run: %v", err)
	}
	if _, err := env.store.UpdateChangeBatch(ctx, batch.ID, batch.Status, latestRun.ID, batch.ExternalChangeID); err != nil {
		t.Fatalf("update change batch latest run: %v", err)
	}

	if err := env.server.syncRunFollowups(ctx, latestRun); err != nil {
		t.Fatalf("sync followups: %v", err)
	}

	updatedResidual, err := env.store.GetResidualItem(ctx, oldResidual.ID)
	if err != nil {
		t.Fatalf("get updated residual: %v", err)
	}
	if updatedResidual.Status != "resolved" {
		t.Fatalf("expected historical residual to resolve as superseded, got %s", updatedResidual.Status)
	}
	if !strings.Contains(updatedResidual.ResolutionNote, "Superseded by latest run") {
		t.Fatalf("expected superseded resolution note, got %q", updatedResidual.ResolutionNote)
	}
	latestResiduals, err := env.store.ListResidualItems(ctx, project.ID, latestRun.ID, 20)
	if err != nil {
		t.Fatalf("list latest residuals: %v", err)
	}
	foundOpenFailed := false
	for _, item := range latestResiduals {
		if item.Status == "open" && strings.HasSuffix(item.SourceKey, ":failed") {
			foundOpenFailed = true
			break
		}
	}
	if !foundOpenFailed {
		t.Fatalf("expected latest run to keep an open failed residual")
	}
	events, err := env.store.ListRunEvents(ctx, latestRun.ID)
	if err != nil {
		t.Fatalf("list latest run events: %v", err)
	}
	found := false
	for _, event := range events {
		if event.Stage == "backlog-reconcile" && strings.Contains(event.Message, "1 carried forward") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected backlog carry-forward event for latest run")
	}
}
