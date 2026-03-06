package store

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	s, err := New(dbPath)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	t.Cleanup(func() {
		_ = s.Close()
	})
	return s
}

func TestStore_ProjectTaskAndKnowledgeFlow(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	project, err := s.CreateProject(ctx, Project{Name: "Studio"})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}

	task, err := s.CreateTask(ctx, Task{ProjectID: project.ID, Title: "Implement API"})
	if err != nil {
		t.Fatalf("create task: %v", err)
	}
	if task.Status != "todo" {
		t.Fatalf("expected default task status todo, got %s", task.Status)
	}

	_, err = s.CreateMemory(ctx, Memory{
		ProjectID:  project.ID,
		Role:       "note",
		Content:    "Need rollback strategy for failed pipeline",
		Tags:       []string{"pipeline", "risk"},
		Importance: 0.9,
	})
	if err != nil {
		t.Fatalf("create memory: %v", err)
	}

	doc, chunks, err := s.AddKnowledgeDocument(
		ctx,
		project.ID,
		"Delivery Guide",
		"internal",
		"Use retries for transient errors. Add runbook for rollback and incident response.",
		80,
	)
	if err != nil {
		t.Fatalf("add doc: %v", err)
	}
	if doc.ID == "" {
		t.Fatal("expected document id")
	}
	if len(chunks) == 0 {
		t.Fatal("expected chunks")
	}

	results, err := s.SearchKnowledge(ctx, project.ID, "rollback", 5)
	if err != nil {
		t.Fatalf("search knowledge: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected knowledge search results")
	}

	stats, err := s.DashboardStats(ctx, project.ID)
	if err != nil {
		t.Fatalf("dashboard stats: %v", err)
	}
	if stats["tasks"] != 1 {
		t.Fatalf("expected 1 task, got %d", stats["tasks"])
	}
	if stats["memories"] != 1 {
		t.Fatalf("expected 1 memory, got %d", stats["memories"])
	}
	if stats["docs"] != 1 {
		t.Fatalf("expected 1 doc, got %d", stats["docs"])
	}
}

func TestStore_AutoScheduleTasks(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	project, err := s.CreateProject(ctx, Project{Name: "AutoSchedule"})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}

	seedTasks := []Task{
		{ProjectID: project.ID, Title: "正在处理任务", Status: "in_progress", Priority: "low", EstimatedDays: 1},
		{ProjectID: project.ID, Title: "高优先级任务", Status: "todo", Priority: "high"},
		{ProjectID: project.ID, Title: "中优先级任务", Status: "todo", Priority: "medium", EstimatedDays: 5},
		{ProjectID: project.ID, Title: "已完成任务", Status: "done", Priority: "high"},
	}
	for _, task := range seedTasks {
		if _, err := s.CreateTask(ctx, task); err != nil {
			t.Fatalf("create task %q: %v", task.Title, err)
		}
	}

	scheduleStart := time.Date(2026, 3, 1, 8, 30, 0, 0, time.UTC)
	items, scheduledCount, err := s.AutoScheduleTasks(ctx, project.ID, scheduleStart)
	if err != nil {
		t.Fatalf("auto schedule tasks: %v", err)
	}
	if scheduledCount != 3 {
		t.Fatalf("expected 3 scheduled tasks, got %d", scheduledCount)
	}

	taskByTitle := map[string]Task{}
	for _, task := range items {
		taskByTitle[task.Title] = task
	}

	assertTaskSchedule := func(title, expectedStart, expectedDue string, expectedDays int) {
		t.Helper()
		task, ok := taskByTitle[title]
		if !ok {
			t.Fatalf("task %q not found", title)
		}
		if task.StartDate == nil || task.DueDate == nil {
			t.Fatalf("task %q expected non-empty schedule", title)
		}
		if task.StartDate.Format("2006-01-02") != expectedStart {
			t.Fatalf("task %q expected start %s, got %s", title, expectedStart, task.StartDate.Format("2006-01-02"))
		}
		if task.DueDate.Format("2006-01-02") != expectedDue {
			t.Fatalf("task %q expected due %s, got %s", title, expectedDue, task.DueDate.Format("2006-01-02"))
		}
		if task.EstimatedDays != expectedDays {
			t.Fatalf("task %q expected estimated_days=%d, got %d", title, expectedDays, task.EstimatedDays)
		}
	}

	assertTaskSchedule("正在处理任务", "2026-03-01", "2026-03-02", 2)
	assertTaskSchedule("高优先级任务", "2026-03-03", "2026-03-06", 4)
	assertTaskSchedule("中优先级任务", "2026-03-07", "2026-03-11", 5)

	doneTask := taskByTitle["已完成任务"]
	if doneTask.StartDate != nil || doneTask.DueDate != nil {
		t.Fatalf("expected done task to keep empty schedule")
	}
	if doneTask.EstimatedDays != 0 {
		t.Fatalf("expected done task estimated_days=0, got %d", doneTask.EstimatedDays)
	}
}
