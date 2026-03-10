package store

import (
	"context"
	"testing"
)

func TestStore_ListAutoAdvanceCandidateRunsReturnsLeafTerminalRuns(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	project, err := s.CreateProject(ctx, Project{Name: "AutoAdvanceCandidates"})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}

	completedLeaf, err := s.CreatePipelineRun(ctx, PipelineRun{
		ProjectID: project.ID,
		Prompt:    "completed leaf",
		Status:    "completed",
		Progress:  100,
		Stage:     "done",
	})
	if err != nil {
		t.Fatalf("create completed leaf: %v", err)
	}
	failedSource, err := s.CreatePipelineRun(ctx, PipelineRun{
		ProjectID: project.ID,
		Prompt:    "failed source",
		Status:    "failed",
		Progress:  100,
		Stage:     "done",
	})
	if err != nil {
		t.Fatalf("create failed source: %v", err)
	}
	if _, err := s.CreatePipelineRun(ctx, PipelineRun{
		ProjectID: project.ID,
		Prompt:    "retry child",
		RetryOf:   failedSource.ID,
		Status:    "queued",
		Progress:  0,
		Stage:     "queued",
	}); err != nil {
		t.Fatalf("create retry child: %v", err)
	}
	failedLeaf, err := s.CreatePipelineRun(ctx, PipelineRun{
		ProjectID: project.ID,
		Prompt:    "failed leaf",
		Status:    "failed",
		Progress:  100,
		Stage:     "done",
	})
	if err != nil {
		t.Fatalf("create failed leaf: %v", err)
	}
	running, err := s.CreatePipelineRun(ctx, PipelineRun{
		ProjectID: project.ID,
		Prompt:    "running",
		Status:    "running",
		Progress:  50,
		Stage:     "delivery",
	})
	if err != nil {
		t.Fatalf("create running run: %v", err)
	}

	items, err := s.ListAutoAdvanceCandidateRuns(ctx, 10)
	if err != nil {
		t.Fatalf("list auto advance candidates: %v", err)
	}
	candidates := map[string]PipelineRun{}
	for _, item := range items {
		candidates[item.ID] = item
	}

	if _, ok := candidates[completedLeaf.ID]; !ok {
		t.Fatalf("expected completed leaf run to be a candidate")
	}
	if _, ok := candidates[failedLeaf.ID]; !ok {
		t.Fatalf("expected failed leaf run to be a candidate")
	}
	if _, ok := candidates[failedSource.ID]; ok {
		t.Fatalf("expected failed source with retry child to be excluded")
	}
	if _, ok := candidates[running.ID]; ok {
		t.Fatalf("expected non-terminal run to be excluded")
	}
}

func TestStore_SyncRequirementSessionsLatestRunByChangeBatch(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	project, err := s.CreateProject(ctx, Project{Name: "RequirementSessionSync"})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	batch, err := s.CreateChangeBatch(ctx, ChangeBatch{
		ProjectID: project.ID,
		Title:     "Batch",
		Goal:      "Keep latest run in sync",
		Status:    "running",
		Mode:      "step_by_step",
	})
	if err != nil {
		t.Fatalf("create change batch: %v", err)
	}
	session, err := s.CreateRequirementSession(ctx, RequirementSession{
		ProjectID:           project.ID,
		Title:               "Requirement",
		RawInput:            "Build the notebook",
		Status:              "confirmed",
		LatestChangeBatchID: batch.ID,
		LatestRunID:         "run-old",
	})
	if err != nil {
		t.Fatalf("create requirement session: %v", err)
	}

	if err := s.SyncRequirementSessionsLatestRunByChangeBatch(ctx, batch.ID, "run-new"); err != nil {
		t.Fatalf("sync requirement session latest run: %v", err)
	}
	updated, err := s.GetRequirementSession(ctx, session.ID)
	if err != nil {
		t.Fatalf("get requirement session: %v", err)
	}
	if updated.LatestRunID != "run-new" {
		t.Fatalf("expected latest run id run-new, got %s", updated.LatestRunID)
	}
}

func TestStore_ListOpenChangeBatchResidualItems(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	project, err := s.CreateProject(ctx, Project{Name: "ResidualBacklogQuery"})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	batch, err := s.CreateChangeBatch(ctx, ChangeBatch{
		ProjectID: project.ID,
		Title:     "Residual Batch",
		Goal:      "Track historical residuals",
		Status:    "running",
		Mode:      "step_by_step",
	})
	if err != nil {
		t.Fatalf("create change batch: %v", err)
	}
	includedRun, err := s.CreatePipelineRun(ctx, PipelineRun{
		ProjectID:     project.ID,
		ChangeBatchID: batch.ID,
		Prompt:        "included",
		Status:        "failed",
		Progress:      100,
		Stage:         "quality",
	})
	if err != nil {
		t.Fatalf("create included run: %v", err)
	}
	excludedRun, err := s.CreatePipelineRun(ctx, PipelineRun{
		ProjectID:     project.ID,
		ChangeBatchID: batch.ID,
		Prompt:        "excluded",
		Status:        "completed",
		Progress:      100,
		Stage:         "preview",
	})
	if err != nil {
		t.Fatalf("create excluded run: %v", err)
	}
	otherBatch, err := s.CreateChangeBatch(ctx, ChangeBatch{
		ProjectID: project.ID,
		Title:     "Other Batch",
		Goal:      "Other work",
		Status:    "running",
		Mode:      "step_by_step",
	})
	if err != nil {
		t.Fatalf("create other change batch: %v", err)
	}
	otherRun, err := s.CreatePipelineRun(ctx, PipelineRun{
		ProjectID:     project.ID,
		ChangeBatchID: otherBatch.ID,
		Prompt:        "other batch",
		Status:        "failed",
		Progress:      100,
		Stage:         "quality",
	})
	if err != nil {
		t.Fatalf("create other run: %v", err)
	}

	if _, err := s.UpsertResidualItem(ctx, ResidualItem{
		ProjectID:        project.ID,
		PipelineRunID:    includedRun.ID,
		Stage:            "quality",
		Category:         "quality",
		Severity:         "high",
		Summary:          "Need more tests",
		SuggestedCommand: "rerun",
		SourceKey:        "sync:run:included:failed",
		Status:           "open",
	}); err != nil {
		t.Fatalf("create included residual: %v", err)
	}
	if _, err := s.UpsertResidualItem(ctx, ResidualItem{
		ProjectID:        project.ID,
		PipelineRunID:    excludedRun.ID,
		Stage:            "preview",
		Category:         "preview",
		Severity:         "medium",
		Summary:          "Exclude current run",
		SuggestedCommand: "rerun",
		SourceKey:        "sync:run:excluded:preview-missing",
		Status:           "open",
	}); err != nil {
		t.Fatalf("create excluded residual: %v", err)
	}
	if _, err := s.UpsertResidualItem(ctx, ResidualItem{
		ProjectID:        project.ID,
		PipelineRunID:    otherRun.ID,
		Stage:            "quality",
		Category:         "quality",
		Severity:         "high",
		Summary:          "Other batch residual",
		SuggestedCommand: "rerun",
		SourceKey:        "sync:run:other:failed",
		Status:           "open",
	}); err != nil {
		t.Fatalf("create other residual: %v", err)
	}

	items, err := s.ListOpenChangeBatchResidualItems(ctx, batch.ID, excludedRun.ID, 10)
	if err != nil {
		t.Fatalf("list change-batch residual items: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 residual item, got %d", len(items))
	}
	if items[0].PipelineRunID != includedRun.ID {
		t.Fatalf("expected residual from included run, got %s", items[0].PipelineRunID)
	}
}
