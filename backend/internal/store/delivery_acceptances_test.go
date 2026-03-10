package store

import (
	"context"
	"testing"
)

func TestStore_UpsertDeliveryAcceptanceTracksLatestState(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	project, err := s.CreateProject(ctx, Project{Name: "DeliveryAcceptance"})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	run, err := s.CreatePipelineRun(ctx, PipelineRun{
		ProjectID: project.ID,
		Prompt:    "ship release candidate",
		Status:    "completed",
		Progress:  100,
		Stage:     "done",
	})
	if err != nil {
		t.Fatalf("create run: %v", err)
	}

	accepted, err := s.UpsertDeliveryAcceptance(ctx, DeliveryAcceptance{
		ProjectID:     project.ID,
		PipelineRunID: run.ID,
		Status:        "accepted",
		ReviewerNote:  "Ready to hand off",
	})
	if err != nil {
		t.Fatalf("record accepted delivery acceptance: %v", err)
	}
	if accepted.Status != "accepted" {
		t.Fatalf("expected accepted status, got %q", accepted.Status)
	}
	if accepted.ReviewedAt == nil {
		t.Fatal("expected reviewed_at to be set for accepted record")
	}

	reopened, err := s.UpsertDeliveryAcceptance(ctx, DeliveryAcceptance{
		ProjectID:     project.ID,
		PipelineRunID: run.ID,
		Status:        "revoked",
		ReviewerNote:  "Need another review pass",
	})
	if err != nil {
		t.Fatalf("reopen delivery acceptance: %v", err)
	}
	if reopened.ID != accepted.ID {
		t.Fatalf("expected delivery acceptance record to be updated in place")
	}
	if reopened.Status != "revoked" {
		t.Fatalf("expected revoked status, got %q", reopened.Status)
	}
	if reopened.ReviewerNote != "Need another review pass" {
		t.Fatalf("expected reviewer note to update, got %q", reopened.ReviewerNote)
	}

	stored, err := s.GetDeliveryAcceptanceByRun(ctx, run.ID)
	if err != nil {
		t.Fatalf("get delivery acceptance by run: %v", err)
	}
	if stored.Status != "revoked" {
		t.Fatalf("expected latest stored status revoked, got %q", stored.Status)
	}
	if stored.ReviewerNote != "Need another review pass" {
		t.Fatalf("expected latest reviewer note to persist, got %q", stored.ReviewerNote)
	}
	if stored.ReviewedAt == nil {
		t.Fatal("expected reviewed_at to remain populated")
	}
}
