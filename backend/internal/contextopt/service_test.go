package contextopt

import (
	"context"
	"path/filepath"
	"testing"

	"superdevstudio/internal/store"
)

func newContextTestStore(t *testing.T) *store.Store {
	t.Helper()
	s, err := store.New(filepath.Join(t.TempDir(), "context.db"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	t.Cleanup(func() {
		_ = s.Close()
	})
	return s
}

func TestBuildContextPack(t *testing.T) {
	s := newContextTestStore(t)
	ctx := context.Background()

	project, err := s.CreateProject(ctx, store.Project{Name: "Optimizer"})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}

	_, err = s.CreateMemory(ctx, store.Memory{
		ProjectID:  project.ID,
		Role:       "note",
		Content:    "Pipeline should retry on transient network failures",
		Tags:       []string{"pipeline", "retry"},
		Importance: 0.95,
	})
	if err != nil {
		t.Fatalf("create memory: %v", err)
	}

	_, _, err = s.AddKnowledgeDocument(
		ctx,
		project.ID,
		"Ops Handbook",
		"wiki",
		"Rollback process: stop deployment, restore previous stable release, run smoke tests.",
		120,
	)
	if err != nil {
		t.Fatalf("add knowledge: %v", err)
	}

	service := NewService(s)
	pack, err := service.BuildContextPack(ctx, BuildRequest{
		ProjectID:   project.ID,
		Query:       "rollback pipeline",
		TokenBudget: 300,
		MaxItems:    5,
	})
	if err != nil {
		t.Fatalf("build pack: %v", err)
	}

	if pack.EstimatedTokens <= 0 {
		t.Fatalf("expected tokens > 0, got %d", pack.EstimatedTokens)
	}
	if pack.EstimatedTokens > 300 {
		t.Fatalf("expected tokens <= budget, got %d", pack.EstimatedTokens)
	}
	if len(pack.Memories) == 0 {
		t.Fatal("expected memory recall")
	}
	if len(pack.Knowledge) == 0 {
		t.Fatal("expected knowledge recall")
	}
	if pack.Summary == "" {
		t.Fatal("expected summary")
	}
}
